package ai_test

import (
    "context"
    "path/filepath"
    "testing"
    "time"

    git "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/ai"
)

func TestIndex_AfterNewCommit(t *testing.T) {
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
    chroma := getenv("AI_CHROMA_URL", "http://localhost:8000")
    if !isUp(tei) || !isUp(chroma) {
        t.Skip("TEI/Chroma not available")
    }

    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    mustWrite(t, filepath.Join(dir, "a.txt"), "alpha content")
    mustCommitAll(t, repo, "initial")

    r, _ := ai.Open(dir)
    ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
    defer cancel()
    if err := r.Index(ctx); err != nil { t.Fatal(err) }

    // New commit
    mustWrite(t, filepath.Join(dir, "b.txt"), "beta index content")
    mustCommitAll(t, repo, "add b.txt")

    // Re-index (full for now)
    if err := r.Index(ctx); err != nil { t.Fatal(err) }

    // Try a semantic query hitting 'index'
    it, err := r.Search().Semantic(ctx, ai.SemanticSearchOptions{Query: "index", TopK: 5, Rerank: false})
    if err != nil { t.Fatal(err) }
    defer it.Close()
    _, _ = it.Next()
}
