# API Reference

Complete API documentation for `github.com/go-git/go-git/v6/ai`

---

## Table of Contents

- [Keyword Search API](#keyword-search-api)
- [Semantic Search API](#semantic-search-api)
- [Code Embedding API](#code-embedding-api)
- [Indexing Management](#indexing-management)
- [Types Reference](#types-reference)

---

## Keyword Search API

### KeywordSearchRepo

Simple wrapper for quick keyword searches.

```go
func KeywordSearchRepo(
    ctx context.Context,
    repoPath string,
    query string,
    topK int,
) ([]SearchResult, error)
```

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `repoPath` - Absolute or relative path to Git repository
- `query` - Text to search for (case-insensitive by default)
- `topK` - Maximum number of results to return

**Returns:**
- `[]SearchResult` - Slice of matching results with scores
- `error` - Any error encountered

**Example:**
```go
results, err := ai.KeywordSearchRepo(ctx, ".", "CloneOptions", 20)
if err != nil {
    log.Fatal(err)
}

for _, r := range results {
    fmt.Printf("%s:%d\n", r.Chunk.FilePath, r.Chunk.StartLine)
}
```

---

### KeywordSearchRepoWithOptions

Advanced keyword search with full control over matching behavior.

```go
func KeywordSearchRepoWithOptions(
    ctx context.Context,
    repoPath string,
    opts KeywordSearchOptions,
) ([]SearchResult, error)
```

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `repoPath` - Path to Git repository
- `opts` - KeywordSearchOptions struct

**KeywordSearchOptions:**
```go
type KeywordSearchOptions struct {
    Query         string  // Search query (text or regex pattern)
    CaseSensitive bool    // Enable case-sensitive matching
    UseRegex      bool    // Treat Query as regex pattern
    TopK          int     // Max results (default: 50)
}
```

**Example:**
```go
results, err := ai.KeywordSearchRepoWithOptions(ctx, ".", ai.KeywordSearchOptions{
    Query:         "(?i)token.*(refresh|renew)",  // Case-insensitive regex
    CaseSensitive: false,
    UseRegex:      true,
    TopK:          100,
})
```

---

### Search().Keyword (Fluent API)

Iterator-based keyword search for memory efficiency and streaming.

```go
func (s *Search) Keyword(
    ctx context.Context,
    opts KeywordSearchOptions,
) (SearchResultIter, error)
```

**Returns:**
- `SearchResultIter` - Iterator over results (implements `Next()`, `ForEach()`, `Close()`)

**Example:**
```go
repo, _ := ai.Open(".")
iter, err := repo.Search().Keyword(ctx, ai.KeywordSearchOptions{
    Query: "authentication",
    TopK:  10,
})
defer iter.Close()

// Method 1: Next()
for {
    result, err := iter.Next()
    if err == io.EOF {
        break
    }
    fmt.Println(result.Chunk.FilePath)
}

// Method 2: ForEach()
iter.ForEach(func(r *ai.SearchResult) error {
    fmt.Println(r.Chunk.Content)
    return nil
})
```

---

## Semantic Search API

### SemanticSearchRepo

Simple wrapper for semantic (AI-powered) search.

```go
func SemanticSearchRepo(
    ctx context.Context,
    repoPath string,
    query string,
    topK int,
    rerank bool,
) ([]SearchResult, error)
```

**Parameters:**
- `ctx` - Context for cancellation and timeout
- `repoPath` - Path to Git repository
- `query` - Natural language query (e.g., "OAuth token refresh logic")
- `topK` - Number of results to return
- `rerank` - Enable cross-encoder reranking for higher quality (slower)

**Returns:**
- `[]SearchResult` - Results sorted by relevance score (higher = better)

**Auto-Indexing:**
- Automatically indexes repository on first use if not already indexed
- Checks staleness (>24h or HEAD changed) and reindexes if needed

**Example:**
```go
// Basic search
results, err := ai.SemanticSearchRepo(ctx, ".", "database connection pooling", 10, false)

// With reranking for higher quality
results, err := ai.SemanticSearchRepo(ctx, ".", "error handling patterns", 20, true)

for _, r := range results {
    fmt.Printf("%.3f - %s:%d\n", r.Score, r.Chunk.FilePath, r.Chunk.StartLine)
    fmt.Printf("  %s\n\n", r.Chunk.Content)
}
```

---

### SemanticSearchRepoStreaming

Progressive semantic search with callback-based emission (VSCode Copilot pattern).

```go
func SemanticSearchRepoStreaming(
    ctx context.Context,
    repoPath string,
    query string,
    topK int,
    rerank bool,
    emit func(SearchResult) error,
) error
```

**Parameters:**
- `emit` - Callback invoked for each result as it's found

**Returns:**
- `error` - Nil on success, context.Canceled if cancelled, other errors on failure

**Example:**
```go
err := ai.SemanticSearchRepoStreaming(ctx, ".", "API rate limiting", 10, false,
    func(result ai.SearchResult) error {
        // Update UI immediately with each result (progressive rendering)
        fmt.Printf("Found: %s (score: %.2f)\n", result.Chunk.FilePath, result.Score)
        updateSearchUI(result)
        return nil
    })
```

---

### Search().Semantic (Fluent API)

Iterator-based semantic search.

```go
func (s *Search) Semantic(
    ctx context.Context,
    opts SemanticSearchOptions,
) (SearchResultIter, error)
```

**SemanticSearchOptions:**
```go
type SemanticSearchOptions struct {
    Query  string  // Natural language query
    TopK   int     // Number of results
    Rerank bool    // Enable cross-encoder reranking
}
```

**Example:**
```go
repo, _ := ai.Open(".")

iter, err := repo.Search().Semantic(ctx, ai.SemanticSearchOptions{
    Query:  "user authentication flow",
    TopK:   15,
    Rerank: true,
})
defer iter.Close()

for {
    result, err := iter.Next()
    if err == io.EOF {
        break
    }
    fmt.Printf("%.3f - %s\n", result.Score, result.Chunk.FilePath)
}
```

---

### VectorStoreRetriever (LangChain-Style API)

LangChain-compatible retrieval interface.

#### Retriever()

```go
func (r *Repository) Retriever() *VectorStoreRetriever
```

Creates a retriever bound to the repository.

**Example:**
```go
repo, _ := ai.Open(".")
retriever := repo.Retriever()
```

---

#### Retrieve

Basic retrieval without reranking.

```go
func (v *VectorStoreRetriever) Retrieve(
    ctx context.Context,
    query string,
    topK int,
) ([]SearchResult, error)
```

**Example:**
```go
docs, err := retriever.Retrieve(ctx, "error handling patterns", 10)
```

---

#### RetrieveWithRerank

Retrieval with cross-encoder reranking for higher quality.

```go
func (v *VectorStoreRetriever) RetrieveWithRerank(
    ctx context.Context,
    query string,
    topK int,
) ([]SearchResult, error)
```

**Example:**
```go
docs, err := retriever.RetrieveWithRerank(ctx, "async task processing", 10)
```

---

#### RetrieveStreaming

Progressive retrieval with callback.

```go
func (v *VectorStoreRetriever) RetrieveStreaming(
    ctx context.Context,
    query string,
    topK int,
    rerank bool,
    emit func(SearchResult) error,
) error
```

**Example:**
```go
err := retriever.RetrieveStreaming(ctx, "caching strategies", 10, false,
    func(result ai.SearchResult) error {
        updateUI(result)
        return nil
    })
```

---

## Code Embedding API

### IndexRepo

Simple wrapper to index a repository.

```go
func IndexRepo(ctx context.Context, repoPath string) error
```

**Example:**
```go
err := ai.IndexRepo(ctx, "/path/to/repo")
if err != nil {
    log.Fatal(err)
}
```

---

### Index

Instance method for indexing with default options.

```go
func (r *Repository) Index(ctx context.Context) error
```

**Example:**
```go
repo, _ := ai.Open(".")
err := repo.Index(ctx)
```

---

### IndexWithOptions

Advanced indexing with full control.

```go
func (r *Repository) IndexWithOptions(
    ctx context.Context,
    opts IndexOptions,
) error
```

**IndexOptions:**
```go
type IndexOptions struct {
    Incremental  bool          // Only index changed files
    Force        bool          // Force reindex even if fresh
    MaxStaleness time.Duration // Reindex if older than this (default: 24h)
    OnProgress   func(file string, current, total int) // Progress callback
    Silent       bool          // Suppress log output
}
```

**Example:**
```go
err := repo.IndexWithOptions(ctx, ai.IndexOptions{
    Incremental:  true,
    MaxStaleness: 12 * time.Hour,
    OnProgress: func(file string, current, total int) {
        fmt.Printf("Indexing %s (%d/%d)\n", file, current, total)
    },
})
```

---

## Indexing Management

### IsIndexed

Check if repository has been indexed.

```go
func (r *Repository) IsIndexed() bool
```

**Example:**
```go
if !repo.IsIndexed() {
    repo.Index(ctx)
}
```

---

### GetIndexState

Get detailed index metadata.

```go
func (r *Repository) GetIndexState() (*IndexState, error)
```

**IndexState:**
```go
type IndexState struct {
    LastCommit  string    // Hash of HEAD when last indexed
    LastIndexed time.Time // Timestamp of last indexing
    FileCount   int       // Number of indexed files
    ChunkCount  int       // Number of indexed chunks
    ModelName   string    // Embedding model used
}
```

**Example:**
```go
state, err := repo.GetIndexState()
if err == nil && state != nil {
    fmt.Printf("Last indexed: %v ago\n", time.Since(state.LastIndexed))
    fmt.Printf("Commit: %s\n", state.LastCommit[:7])
    fmt.Printf("Files: %d\n", state.FileCount)
}
```

---

### NeedsReindex

Determine if index is stale and needs updating.

```go
func (r *Repository) NeedsReindex(maxAge time.Duration) (bool, error)
```

**Returns `true` if:**
- Never indexed before
- HEAD commit changed since last index
- More than `maxAge` has passed since last index

**Example:**
```go
needsReindex, _ := repo.NeedsReindex(24 * time.Hour)
if needsReindex {
    repo.IndexWithOptions(ctx, ai.IndexOptions{Incremental: true})
}
```

---

### GetChangedFiles

Get list of files changed since last index.

```go
func (r *Repository) GetChangedFiles() ([]string, error)
```

**Returns:**
- `[]string` - File paths that changed between last indexed commit and HEAD
- Empty slice if no previous index or same commit

**Example:**
```go
changed, _ := repo.GetChangedFiles()
fmt.Printf("Changed files: %d\n", len(changed))
for _, file := range changed {
    fmt.Println("  -", file)
}
```

---

### SaveIndexState

Manually save index state (usually automatic).

```go
func (r *Repository) SaveIndexState(state *IndexState) error
```

**Example:**
```go
state := &ai.IndexState{
    LastCommit:  commitHash,
    LastIndexed: time.Now(),
    FileCount:   1234,
    ChunkCount:  5678,
    ModelName:   "bge-small-en-v1.5",
}
repo.SaveIndexState(state)
```

---

## Background Watcher

### NewRepoWatcher

Create a background watcher for automatic incremental reindexing.

```go
func NewRepoWatcher(repo *Repository, interval time.Duration) *RepoWatcher
```

**Parameters:**
- `repo` - Repository to watch
- `interval` - How often to check for changes (e.g., 5*time.Minute)

**Example:**
```go
watcher := ai.NewRepoWatcher(repo, 5*time.Minute)
watcher.Silent(true) // Suppress logs
go watcher.Start(ctx)

// Later...
watcher.Stop()
```

---

### Start

Start the background watcher (blocking).

```go
func (w *RepoWatcher) Start(ctx context.Context)
```

**Example:**
```go
ctx, cancel := context.WithCancel(context.Background())
go watcher.Start(ctx)

// Stop after 1 hour
time.AfterFunc(time.Hour, cancel)
```

---

### Stop

Stop the background watcher.

```go
func (w *RepoWatcher) Stop()
```

---

### Silent

Configure log verbosity.

```go
func (w *RepoWatcher) Silent(silent bool) *RepoWatcher
```

**Example:**
```go
watcher := ai.NewRepoWatcher(repo, 5*time.Minute).Silent(true)
```

---

## Types Reference

### SearchResult

A single search result with score and content.

```go
type SearchResult struct {
    Chunk DocumentChunk
    Score float64  // Higher = better (0.0 to 1.0 typically)
}
```

---

### DocumentChunk

Code chunk with metadata.

```go
type DocumentChunk struct {
    ID         string            // Unique chunk identifier
    RepoPath   string            // Repository path
    CommitHash string            // Git commit hash
    FilePath   string            // Relative file path
    StartLine  int               // Starting line number
    EndLine    int               // Ending line number
    BlobHash   string            // Git blob hash
    Language   string            // Programming language
    Content    string            // Code content
    Meta       map[string]string // Additional metadata
    CreatedAt  time.Time         // Indexing timestamp
}
```

---

### SearchResultIter

Iterator over search results (memory-efficient streaming).

```go
type SearchResultIter interface {
    Next() (*SearchResult, error)      // Get next result (io.EOF when done)
    ForEach(func(*SearchResult) error) error  // Process all results
    Close()                            // Release resources
}
```

**Example:**
```go
iter, _ := repo.Search().Keyword(ctx, opts)
defer iter.Close()

// Pattern 1: Next()
for {
    result, err := iter.Next()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    process(result)
}

// Pattern 2: ForEach()
iter.ForEach(func(r *ai.SearchResult) error {
    process(r)
    return nil
})
```

---

## Error Handling

### Common Errors

```go
// Index not found
if err != nil && strings.Contains(err.Error(), "not indexed") {
    repo.Index(ctx)
}

// Connection errors
if err != nil && strings.Contains(err.Error(), "connection") {
    // Check TEI/Chroma services are running
}

// Context cancelled
if errors.Is(err, context.Canceled) {
    // User cancelled operation
}
```

---

## Configuration

### Environment Variables

```bash
# Embedding service
AI_TEI_ENDPOINT=http://localhost:8081

# Vector database
AI_CHROMA_URL=http://localhost:9001

# Optional customization
AI_EMBEDDINGS_SOURCE=local-tei  # or "openai"
AI_VECTOR_DB=chroma             # or "qdrant", "memory"
```

### Programmatic Configuration

```go
cfg := ai.Config{
    TEIEndpoint: "http://custom-tei:8081",
    ChromaURL:   "http://custom-chroma:9001",
}
provider := ai.NewHybridProvider(cfg)
repo, _ := ai.Open(".")
repo.WithProvider(provider)
```

---

## Performance Tips

### Keyword Search
- Use specific queries to reduce result count
- Prefer `TopK < 100` for fast response
- Use regex sparingly (slower than plain text)

### Semantic Search
- Enable `Incremental: true` for faster reindexing
- Use `Rerank: false` unless quality is critical
- Batch queries when possible
- Use streaming for large result sets

### Indexing
- Run initial index during off-hours
- Use background watcher for continuous freshness
- Enable `Silent: true` in production logs
- Cache index state between runs

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/go-git/go-git/v6/ai"
)

func main() {
    // Open repository
    repo, err := ai.Open(".")
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    // Check if indexed
    if !repo.IsIndexed() {
        fmt.Println("Indexing repository...")
        if err := repo.Index(ctx); err != nil {
            log.Fatal(err)
        }
    }

    // Keyword search (fast, local)
    fmt.Println("\n=== Keyword Search ===")
    kwResults, _ := ai.KeywordSearchRepo(ctx, ".", "OAuth", 5)
    for _, r := range kwResults {
        fmt.Printf("%s:%d\n", r.Chunk.FilePath, r.Chunk.StartLine)
    }

    // Semantic search (AI-powered)
    fmt.Println("\n=== Semantic Search ===")
    semResults, _ := ai.SemanticSearchRepo(ctx, ".", "token refresh logic", 5, true)
    for _, r := range semResults {
        fmt.Printf("%.3f - %s:%d\n", r.Score, r.Chunk.FilePath, r.Chunk.StartLine)
    }

    // Start background watcher
    watcher := ai.NewRepoWatcher(repo, 5*time.Minute)
    go watcher.Start(ctx)
    defer watcher.Stop()

    // Searches will always use fresh index
    time.Sleep(10 * time.Minute)
}
```

---

## See Also

- [README.md](./README.md) - Overview and quick start
- [Examples](./_examples/) - Complete working examples
- [AI_SETUP.md](../AI_SETUP.md) - Service setup guide
