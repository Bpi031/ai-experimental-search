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
    var repoPath string
    var keyword string
    var semantic string
    flag.StringVar(&repoPath, "repo", ".", "path to repo")
    flag.StringVar(&keyword, "kw", "IndexRepository", "keyword query")
    flag.StringVar(&semantic, "sem", "index repository", "semantic query")
    flag.Parse()

    r, err := ai.Open(repoPath)
    if err != nil { log.Fatal(err) }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Index (semantic)
    if err := r.Index(ctx); err != nil { log.Fatalf("index: %v", err) }

    // Keyword search
    kwIter, err := r.Search().Keyword(ctx, ai.KeywordSearchOptions{Query: keyword, CaseSensitive: false, TopK: 10})
    if err != nil { log.Fatalf("keyword: %v", err) }
    defer kwIter.Close()
    fmt.Println("Keyword results:")
    _ = kwIter.ForEach(func(sr *ai.SearchResult) error {
        fmt.Printf("- %s:%d %q\n", sr.Chunk.FilePath, sr.Chunk.StartLine, sr.Chunk.Content)
        return nil
    })

    // Semantic search
    semIter, err := r.Search().Semantic(ctx, ai.SemanticSearchOptions{Query: semantic, TopK: 5, Rerank: true})
    if err != nil { log.Fatalf("semantic: %v", err) }
    defer semIter.Close()
    fmt.Println("\nSemantic results:")
    _ = semIter.ForEach(func(sr *ai.SearchResult) error {
        fmt.Printf("- %.3f %s:%d-%d\n", sr.Score, sr.Chunk.FilePath, sr.Chunk.StartLine, sr.Chunk.EndLine)
        return nil
    })
}
