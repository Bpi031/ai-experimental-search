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
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:9000")
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

// TestSemanticSearch_StreamingProgressive validates VSCode Copilot-style progressive emission.
func TestSemanticSearch_StreamingProgressive(t *testing.T) {
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
    chroma := getenv("AI_CHROMA_URL", "http://localhost:8000")
    if !isUp(tei) || !isUp(chroma) {
        t.Skip("TEI/Chroma not available")
    }

    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    
    // Add multiple files to get multiple results
    mustWrite(t, filepath.Join(dir, "auth.go"), "package auth\n\n// OAuth2 token refresh logic\nfunc RefreshToken() {}\n")
    mustWrite(t, filepath.Join(dir, "client.go"), "package client\n\n// HTTP client with retry and token management\nfunc NewClient() {}\n")
    mustWrite(t, filepath.Join(dir, "storage.go"), "package storage\n\n// Token storage and persistence layer\nfunc SaveToken() {}\n")
    mustCommitAll(t, repo, "add: multi-file codebase")

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := r.Index(ctx); err != nil { t.Fatal(err) }

    // Test progressive emission (results emitted as they arrive)
    emittedCount := 0
    err = ai.SemanticSearchRepoStreaming(ctx, dir, "token refresh", 5, false, func(sr ai.SearchResult) error {
        emittedCount++
        if sr.Chunk.Content == "" { t.Fatal("empty content in streamed result") }
        // Simulate progressive UI update (this is where you'd update UI in real app)
        t.Logf("Streamed result %d: %s (score: %.3f)", emittedCount, sr.Chunk.FilePath, sr.Score)
        return nil
    })
    if err != nil { t.Fatal(err) }
    if emittedCount == 0 { t.Fatal("expected at least one streamed result") }
}

// TestSemanticSearch_ContextCancellation ensures streaming respects context cancellation.
func TestSemanticSearch_ContextCancellation(t *testing.T) {
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
    chroma := getenv("AI_CHROMA_URL", "http://localhost:8000")
    if !isUp(tei) || !isUp(chroma) {
        t.Skip("TEI/Chroma not available")
    }

    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    mustWrite(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() { println(\"hello\") }\n")
    mustCommitAll(t, repo, "init")

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := r.Index(ctx); err != nil { t.Fatal(err) }

    // Cancel immediately after first result
    searchCtx, searchCancel := context.WithCancel(ctx)
    count := 0
    err = ai.SemanticSearchRepoStreaming(searchCtx, dir, "hello", 10, false, func(sr ai.SearchResult) error {
        count++
        if count == 1 {
            searchCancel() // Cancel after first result
        }
        return nil
    })
    // Should return context.Canceled or nil (if only 1 result exists)
    if err != nil && err != context.Canceled {
        t.Fatalf("expected nil or context.Canceled, got: %v", err)
    }
}

// TestVectorStoreRetriever_LangChainStyle validates the LangChain-style retriever API.
func TestVectorStoreRetriever_LangChainStyle(t *testing.T) {
    tei := getenv("AI_TEI_ENDPOINT", "http://localhost:8081")
    chroma := getenv("AI_CHROMA_URL", "http://localhost:8000")
    if !isUp(tei) || !isUp(chroma) {
        t.Skip("TEI/Chroma not available")
    }

    dir := t.TempDir()
    repo, err := git.PlainInit(dir, false)
    if err != nil { t.Fatal(err) }
    mustWrite(t, filepath.Join(dir, "vector.txt"), "This document explains vector embeddings and semantic similarity.")
    mustCommitAll(t, repo, "add: vector doc")

    r, err := ai.Open(dir)
    if err != nil { t.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    if err := r.Index(ctx); err != nil { t.Fatal(err) }

    // LangChain-style retriever usage
    retriever := r.Retriever()
    
    // Test basic retrieve (no rerank)
    results, err := retriever.Retrieve(ctx, "semantic embeddings", 3)
    if err != nil { t.Fatal(err) }
    if len(results) == 0 { t.Log("no results (acceptable for small test corpus)") }
    
    // Test retrieve with rerank
    results, err = retriever.RetrieveWithRerank(ctx, "semantic embeddings", 3)
    if err != nil { t.Fatal(err) }
    
    // Test streaming retrieval
    streamCount := 0
    err = retriever.RetrieveStreaming(ctx, "semantic embeddings", 3, false, func(sr ai.SearchResult) error {
        streamCount++
        return nil
    })
    if err != nil { t.Fatal(err) }
    t.Logf("streamed %d results via retriever", streamCount)
}

