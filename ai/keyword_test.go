package ai_test

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"

    git "github.com/go-git/go-git/v6"
    "github.com/go-git/go-git/v6/ai"
)

func TestKeywordSearch_Basic(t *testing.T) {
    dir := t.TempDir()
    if err := os.MkdirAll(filepath.Join(dir, "foo"), 0o755); err != nil {
        t.Fatal(err)
    }

    // Init repo and commit a file
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    mustWrite(t, filepath.Join(dir, "foo", "bar.go"), "package foo\n\n// CloneOptions here\nfunc Hello() {}\n")
    mustCommitAll(t, repo, "feat: add bar.go with CloneOptions comment")

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    it, err := r.Search().Keyword(ctx, ai.KeywordSearchOptions{Query: "CloneOptions", TopK: 10})
    if err != nil { t.Fatal(err) }
    defer it.Close()

    found := false
    _ = it.ForEach(func(sr *ai.SearchResult) error {
        if filepath.Base(sr.Chunk.FilePath) == "bar.go" {
            found = true
        }
        return nil
    })
    if !found { t.Fatalf("expected to find match in bar.go") }
}
