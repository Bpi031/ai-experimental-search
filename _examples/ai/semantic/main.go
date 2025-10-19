package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    "time"

    "github.com/go-git/go-git/v6/ai"
)

func main() {
    var (
        repo  string
        query string
        topK  int
        rerank bool
    )
    flag.StringVar(&repo, "repo", ".", "path to git repository")
    flag.StringVar(&query, "q", "init repository", "semantic query")
    flag.IntVar(&topK, "k", 5, "number of results")
    flag.BoolVar(&rerank, "rerank", true, "use TEI reranker when available")
    flag.Parse()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Index repository (HEAD)
    log.Printf("Indexing repo: %s", repo)
    if err := ai.IndexRepo(ctx, repo); err != nil {
        log.Fatalf("index failed: %v", err)
    }
    log.Printf("Indexing completed")

    // Semantic search
    results, err := ai.SemanticSearchRepo(ctx, repo, query, topK, rerank)
    if err != nil {
        log.Fatalf("search failed: %v", err)
    }
    for i, r := range results {
        fmt.Printf("[%d] %.4f %s:%d-%d (%s)\n", i+1, r.Score, r.Chunk.FilePath, r.Chunk.StartLine, r.Chunk.EndLine, r.Chunk.CommitHash)
        fmt.Println(r.Chunk.Content)
        fmt.Println("---")
    }
}
