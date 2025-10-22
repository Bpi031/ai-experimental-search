package ai_test

import (
    "context"
    "io"
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

// TestKeywordSearchRepo_WrapperAPI validates the top-level wrapper functions.
func TestKeywordSearchRepo_WrapperAPI(t *testing.T) {
    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    
    mustWrite(t, filepath.Join(dir, "main.go"), "package main\n\nfunc commit() {\n\t// git commit logic\n}\n")
    mustWrite(t, filepath.Join(dir, "remote.go"), "package main\n\nfunc pull() {\n\t// git pull logic\n}\n")
    mustCommitAll(t, repo, "add: git-like methods")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Test simple wrapper
    results, err := ai.KeywordSearchRepo(ctx, dir, "commit", 10)
    if err != nil { t.Fatal(err) }
    if len(results) == 0 { t.Fatal("expected at least 1 result for 'commit'") }
    
    foundMain := false
    for _, r := range results {
        if filepath.Base(r.Chunk.FilePath) == "main.go" {
            foundMain = true
        }
    }
    if !foundMain { t.Fatal("expected to find match in main.go") }

    // Test options wrapper with case-sensitive and regex
    results, err = ai.KeywordSearchRepoWithOptions(ctx, dir, ai.KeywordSearchOptions{
        Query:         "(?i)PULL",
        CaseSensitive: false,
        UseRegex:      true,
        TopK:          5,
    })
    if err != nil { t.Fatal(err) }
    if len(results) == 0 { t.Fatal("expected at least 1 result for regex '(?i)PULL'") }
}

// TestKeywordSearch_StreamingBehavior ensures results emit as they're found (Copilot-style).
func TestKeywordSearch_StreamingBehavior(t *testing.T) {
    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    
    // Add multiple files with the target query
    for i := 1; i <= 5; i++ {
        content := "package test\n\n// This is a streaming test\nfunc f" + string(rune('0'+i)) + "() {}\n"
        mustWrite(t, filepath.Join(dir, "file"+string(rune('0'+i))+".go"), content)
    }
    mustCommitAll(t, repo, "add: multiple files for streaming")

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    it, err := r.Search().Keyword(ctx, ai.KeywordSearchOptions{Query: "streaming", TopK: 10})
    if err != nil { t.Fatal(err) }
    defer it.Close()

    count := 0
    for {
        sr, err := it.Next()
        if err == io.EOF { break }
        if err != nil { t.Fatal(err) }
        if sr.Chunk.Content == "" { t.Fatal("empty content in streamed result") }
        count++
    }
    if count < 5 { t.Fatalf("expected at least 5 results, got %d", count) }
}
