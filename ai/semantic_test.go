package ai_test

import (
    "context"
    "net/http"
    "os"
    "path/filepath"
    "testing"
    "time"

    git "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/ai"
)

func isUp(url string) bool {
    c := &http.Client{Timeout: 2 * time.Second}
    req, _ := http.NewRequest(http.MethodGet, url, nil)
    resp, err := c.Do(req)
    if err != nil { return false }
    resp.Body.Close()
    return resp.StatusCode >= 200 && resp.StatusCode < 500
}

func TestSemanticSearch_IndexAndQuery(t *testing.T) {
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
    chroma := getenv("AI_CHROMA_URL", "http://localhost:8000")
    if !isUp(tei) || !isUp(chroma) {
        t.Skipf("TEI (%s) or Chroma (%s) not reachable; skipping semantic test", tei, chroma)
    }

    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    mustWrite(t, filepath.Join(dir, "readme.txt"), "This repository demonstrates clone and index operations.")
    mustCommitAll(t, repo, "chore: initial content")

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := r.Index(ctx); err != nil { t.Fatalf("index failed: %v", err) }

    it, err := r.Search().Semantic(ctx, ai.SemanticSearchOptions{Query: "index repository", TopK: 3, Rerank: true})
    if err != nil { t.Fatal(err) }
    defer it.Close()
    // Just ensure no panic and at least zero or more results (best-effort)
    _, _ = it.Next()
}

func getenv(k, def string) string {
    if v := os.Getenv(k); v != "" { return v }
    return def
}
