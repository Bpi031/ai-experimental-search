# AI-Powered Code Search API

This package provides three powerful search capabilities for Git repositories:

1. **Keyword Search** - Fast text-based search (like `grep`)
2. **Semantic Search** - AI-powered understanding of code meaning
3. **Code Embedding** - Vector representations for similarity matching

---

## Quick Start

```go
import "github.com/go-git/go-git/v6/ai"

// Open repository
repo, _ := ai.Open("/path/to/repo")

// Keyword search (instant, no indexing needed)
results, _ := ai.KeywordSearchRepo(ctx, "/path/to/repo", "OAuth", 10)

// Semantic search (auto-indexes on first use)
results, _ := ai.SemanticSearchRepo(ctx, "/path/to/repo", "authentication logic", 10, false)
```

---

## 1. Keyword Search

Fast, local text-based search over repository HEAD. No indexing or external services required.

### Features

- ✅ **Instant** - No indexing delay, works immediately
- ✅ **Regex support** - Use regular expressions for complex patterns
- ✅ **Case-sensitive/insensitive** - Flexible matching
- ✅ **Streaming** - Results emitted as found (Copilot-style)
- ✅ **Context-aware** - Cancellable via context

### API

#### Simple Wrapper (Quick Use)

```go
// Basic keyword search
results, err := ai.KeywordSearchRepo(ctx, repoPath, "CloneOptions", 20)
for _, r := range results {
    fmt.Printf("%s:%d - %s\n", r.Chunk.FilePath, r.Chunk.StartLine, r.Chunk.Content)
}
```

#### Advanced Options

```go
results, err := ai.KeywordSearchRepoWithOptions(ctx, repoPath, ai.KeywordSearchOptions{
    Query:         "(?i)token.*refresh",  // Case-insensitive regex
    CaseSensitive: false,
    UseRegex:      true,
    TopK:          50,
})
```

#### Fluent API (go-git style)

```go
repo, _ := ai.Open("/path/to/repo")

// Iterator pattern - memory efficient
iter, _ := repo.Search().Keyword(ctx, ai.KeywordSearchOptions{
    Query: "authentication",
    TopK:  10,
})
defer iter.Close()

// Process results one by one
for {
    result, err := iter.Next()
    if err == io.EOF {
        break
    }
    fmt.Println(result.Chunk.FilePath)
}

// Or use ForEach
iter.ForEach(func(r *ai.SearchResult) error {
    fmt.Println(r.Chunk.Content)
    return nil
})
```

### Use Cases

- **IDE Find in Files** - Replace built-in search
- **Code Review** - Find all usages of a function
- **Security Audit** - Search for sensitive patterns (`password|secret|token`)
- **Documentation** - Find all TODO/FIXME comments

### Performance

| Repository Size | First Search | Subsequent Searches |
|----------------|--------------|---------------------|
| 100 files | ~50ms | ~50ms (no caching) |
| 1,000 files | ~200ms | ~200ms |
| 10,000 files | ~2s | ~2s |

---

## 2. Semantic Search

AI-powered search that understands code meaning, not just text matching.

### Features

- ✅ **Natural language queries** - "How does authentication work?"
- ✅ **Code understanding** - Finds semantically similar code
- ✅ **Auto-indexing** - Lazy indexing on first search
- ✅ **Incremental updates** - Only re-index changed files
- ✅ **Streaming results** - Progressive rendering (VSCode Copilot pattern)
- ✅ **Reranking** - Optional quality boost with cross-encoder
- ✅ **Background refresh** - Keep index fresh automatically

### API

#### Simple Wrapper (Quick Use)

```go
// Auto-indexes if needed
results, err := ai.SemanticSearchRepo(ctx, repoPath, "OAuth token refresh logic", 10, false)
for _, r := range results {
    fmt.Printf("%.2f - %s:%d\n", r.Score, r.Chunk.FilePath, r.Chunk.StartLine)
}
```

#### With Reranking (Higher Quality)

```go
results, err := ai.SemanticSearchRepo(ctx, repoPath, "database connection pooling", 20, true)
// Results are re-scored by cross-encoder for better relevance
```

#### Fluent API (go-git style)

```go
repo, _ := ai.Open("/path/to/repo")

// Explicit indexing (first time)
repo.Index(ctx)

// Search with options
iter, _ := repo.Search().Semantic(ctx, ai.SemanticSearchOptions{
    Query:  "user authentication flow",
    TopK:   15,
    Rerank: true,
})
defer iter.Close()

// Iterator pattern
for {
    result, err := iter.Next()
    if err == io.EOF {
        break
    }
    fmt.Printf("Score: %.3f\n%s\n\n", result.Score, result.Chunk.Content)
}
```

#### LangChain-Style Retriever

```go
repo, _ := ai.Open("/path/to/repo")
retriever := repo.Retriever()

// Simple retrieval
docs, _ := retriever.Retrieve(ctx, "error handling patterns", 10)

// With reranking
docs, _ := retriever.RetrieveWithRerank(ctx, "async task processing", 10)

// Streaming (progressive UI updates)
retriever.RetrieveStreaming(ctx, "caching strategies", 10, false, func(result ai.SearchResult) error {
    updateUI(result) // Render as results arrive
    return nil
})
```

#### Streaming Search (VSCode Copilot Pattern)

```go
// Results emitted progressively as they're computed
err := ai.SemanticSearchRepoStreaming(ctx, repoPath, "API rate limiting", 10, false, 
    func(result ai.SearchResult) error {
        // Update UI immediately with each result
        fmt.Printf("Found: %s (%.2f)\n", result.Chunk.FilePath, result.Score)
        return nil
    })
```

### Use Cases

- **Code exploration** - "Show me all authentication code"
- **Bug hunting** - "Find error handling that might leak resources"
- **Refactoring** - "Find similar implementations to consolidate"
- **Documentation** - "Explain how caching works in this project"
- **Onboarding** - Help new developers understand codebase

### Performance

| Repository Size | Index Time | Search Time | Incremental Reindex |
|----------------|------------|-------------|---------------------|
| 100 files | ~8s | ~200ms | ~1s (10 files) |
| 1,000 files | ~45s | ~300ms | ~2s (50 files) |
| 10,000 files | ~6min | ~500ms | ~5s (100 files) |

---

## 3. Code Embedding

Generate vector representations of code for similarity matching and ML pipelines.

### Features

- ✅ **Batch processing** - Efficient multi-text embedding
- ✅ **Automatic chunking** - Smart code splitting (~1000 chars/chunk)
- ✅ **Metadata tracking** - File path, language, commit hash
- ✅ **Vector storage** - Chroma DB integration
- ✅ **Incremental updates** - Only embed changed code

### API

#### Index Repository (Generate Embeddings)

```go
repo, _ := ai.Open("/path/to/repo")

// Simple indexing (all files)
repo.Index(ctx)

// With options
repo.IndexWithOptions(ctx, ai.IndexOptions{
    Incremental:  true,              // Only changed files
    Force:        false,              // Skip if already indexed
    MaxStaleness: 24 * time.Hour,    // Reindex if older than 1 day
    OnProgress: func(file string, current, total int) {
        fmt.Printf("Indexing %s (%d/%d)\n", file, current, total)
    },
    Silent: false,
})
```

#### Direct Embedding (Low-Level)

```go
// Access embedding provider directly
provider := ai.DefaultProvider()

// Embed code snippets
embeddings, err := provider.Embed(ctx, []string{
    "func authenticate(token string) error { ... }",
    "class UserRepository { ... }",
})

// Each embedding is a float32 vector (typically 384 or 768 dimensions)
fmt.Printf("Embedding dimension: %d\n", len(embeddings[0]))
```

#### Index State Management

```go
repo, _ := ai.Open("/path/to/repo")

// Check if indexed
if repo.IsIndexed() {
    state, _ := repo.GetIndexState()
    fmt.Printf("Last indexed: %v\n", state.LastIndexed)
    fmt.Printf("Commit: %s\n", state.LastCommit)
    fmt.Printf("Files: %d\n", state.FileCount)
}

// Check if stale
needsReindex, _ := repo.NeedsReindex(24 * time.Hour)
if needsReindex {
    repo.Index(ctx)
}

// Get changed files since last index
changedFiles, _ := repo.GetChangedFiles()
fmt.Printf("Changed files: %v\n", changedFiles)
```

### Indexing Strategies

#### 1. First Use (Explicit)

```go
// User explicitly indexes before searching
repo.Index(ctx)
```

#### 2. Lazy (Auto-Index on First Search)

```go
// Search auto-indexes if needed
repo.Search().Semantic(ctx, ai.SemanticSearchOptions{Query: "auth", TopK: 10})
// Index state saved to .git/ai-index.json
```

#### 3. Background Watcher (Keep Fresh)

```go
// Start background watcher (checks every 5 minutes)
watcher := ai.NewRepoWatcher(repo, 5*time.Minute)
go watcher.Start(ctx)

// Searches always use fresh index
repo.Search().Semantic(...)
```

#### 4. Incremental (After git pull)

```go
// Only reindex changed files
repo.IndexWithOptions(ctx, ai.IndexOptions{Incremental: true})
```

#### 5. CI/CD (Explicit, Deterministic)

```go
// In CI pipeline after merge
if branchName == "main" {
    repo.Index(ctx)
    // Cache embeddings for next run
}
```

### Use Cases

- **Code search engines** - Build Sourcegraph-like tools
- **IDE extensions** - Power semantic features
- **RAG systems** - Context retrieval for AI coding assistants
- **Code similarity** - Detect duplicates or near-duplicates
- **ML pipelines** - Train models on code representations

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
│  (CLI, IDE Extension, Web API, RAG System)              │
└─────────────────────────────────────────────────────────┘
                          │
         ┌────────────────┼────────────────┐
         │                │                │
         ▼                ▼                ▼
┌─────────────┐  ┌──────────────┐  ┌──────────────┐
│   Keyword   │  │   Semantic   │  │  Embedding   │
│   Search    │  │    Search    │  │   Provider   │
│  (Local)    │  │ (Chroma DB)  │  │    (TEI)     │
└─────────────┘  └──────────────┘  └──────────────┘
         │                │                │
         └────────────────┼────────────────┘
                          ▼
                  ┌──────────────┐
                  │   go-git     │
                  │  Repository  │
                  └──────────────┘
```

### Components

- **Keyword Search** - Pure Go, no dependencies
- **TEI (Text Embeddings Inference)** - HuggingFace embedding server
- **Chroma DB** - Vector database for semantic search
- **go-git** - Git repository access

---

## Configuration

### Environment Variables

```bash
# Embedding service (required for semantic search)
export AI_TEI_ENDPOINT=http://localhost:8081

# Vector database (required for semantic search)
export AI_CHROMA_URL=http://localhost:8000

# Optional: Customize embedding model
export AI_EMBEDDINGS_SOURCE=local-tei  # or "openai"
export AI_VECTOR_DB=chroma              # or "qdrant", "memory"
```

### Start Services (Docker Compose)

```bash
# Start TEI + Chroma
./ai-services.sh start

# Check status
./ai-services.sh status

# Stop services
./ai-services.sh stop
```

---

## Examples

### Example 1: Smart Code Search

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-git/go-git/v6/ai"
)

func main() {
    repo, _ := ai.Open(".")
    ctx := context.Background()

    // Try semantic search first (understands meaning)
    results, _ := repo.Search().Semantic(ctx, ai.SemanticSearchOptions{
        Query:  "token expiration handling",
        TopK:   5,
        Rerank: true,
    })

    // Fallback to keyword if no semantic results
    if len(results) == 0 {
        results, _ = ai.KeywordSearchRepo(ctx, ".", "token.*expir", 5)
    }

    for _, r := range results {
        fmt.Printf("%s:%d (%.2f)\n%s\n\n",
            r.Chunk.FilePath, r.Chunk.StartLine, r.Score, r.Chunk.Content)
    }
}
```

### Example 2: Progressive Search UI

```go
// Stream results to UI as they arrive (VSCode Copilot style)
func searchWithProgress(query string) {
    repo, _ := ai.Open(".")
    ctx := context.Background()

    results := make(chan ai.SearchResult, 10)

    go func() {
        defer close(results)
        ai.SemanticSearchRepoStreaming(ctx, ".", query, 20, false,
            func(r ai.SearchResult) error {
                results <- r
                return nil
            })
    }()

    // Update UI progressively
    for result := range results {
        updateSearchUI(result) // Immediate feedback
    }
}
```

### Example 3: CLI Tool

See `_examples/git-ai/main.go` for a complete CLI implementation:

```bash
# Initialize semantic search
git-ai init

# Check index status
git-ai status

# Incremental reindex after git pull
git-ai reindex --incremental

# Search
git-ai search "database connection pooling"

# Background watcher
git-ai watch --interval=5m
```

---

## Comparison Matrix

| Feature | Keyword Search | Semantic Search |
|---------|---------------|-----------------|
| **Speed** | Instant (~50ms) | Fast (~300ms after index) |
| **Accuracy** | Exact matches only | Understands meaning |
| **Setup** | None | Requires TEI + Chroma |
| **Indexing** | Not needed | First-time index (~1min/1k files) |
| **Query Style** | Exact text/regex | Natural language |
| **Use Case** | Known function names | Exploratory search |
| **Cost** | Free (local) | TEI server required |

---

## Best Practices

### When to Use Keyword Search

- ✅ You know the exact function/variable name
- ✅ Need instant results without indexing delay
- ✅ Searching for specific patterns (URLs, IDs, constants)
- ✅ No AI services available

### When to Use Semantic Search

- ✅ Exploratory search ("show me auth code")
- ✅ Natural language queries
- ✅ Finding similar implementations
- ✅ Cross-language concept search

### Hybrid Approach (Recommended)

```go
func smartSearch(query string) []ai.SearchResult {
    // Try semantic search first (better understanding)
    semanticResults := semanticSearch(query)
    if len(semanticResults) > 0 {
        return semanticResults
    }

    // Fallback to keyword (always works)
    return keywordSearch(query)
}
```

---

## Troubleshooting

### "Index not found" Error

```go
// Solution 1: Explicit indexing
repo.Index(ctx)

// Solution 2: Auto-index on first search
// (semantic search does this automatically)
repo.Search().Semantic(...)
```

### "Chroma connection failed"

```bash
# Check services are running
docker ps | grep chroma

# Restart if needed
./ai-services.sh restart
```

### Slow Indexing

```go
// Use incremental indexing
repo.IndexWithOptions(ctx, ai.IndexOptions{
    Incremental: true,  // Only changed files
    Silent:      false, // Show progress
})
```

---

## API Reference

### Core Types

```go
type SearchResult struct {
    Chunk DocumentChunk
    Score float64
}

type DocumentChunk struct {
    FilePath   string
    StartLine  int
    EndLine    int
    Content    string
    Language   string
    CommitHash string
    // ...
}

type IndexOptions struct {
    Incremental  bool
    Force        bool
    MaxStaleness time.Duration
    OnProgress   func(file string, current, total int)
    Silent       bool
}
```

### Top-Level Functions

```go
// Keyword search
func KeywordSearchRepo(ctx, repoPath, query string, topK int) ([]SearchResult, error)
func KeywordSearchRepoWithOptions(ctx, repoPath string, opts KeywordSearchOptions) ([]SearchResult, error)

// Semantic search
func SemanticSearchRepo(ctx, repoPath, query string, topK int, rerank bool) ([]SearchResult, error)
func SemanticSearchRepoStreaming(ctx, repoPath, query string, topK int, rerank bool, emit func(SearchResult) error) error

// Indexing
func IndexRepo(ctx context.Context, repoPath string) error
```

### Repository Methods

```go
type Repository struct { ... }

// Indexing
func (r *Repository) Index(ctx) error
func (r *Repository) IndexWithOptions(ctx, opts IndexOptions) error
func (r *Repository) IsIndexed() bool
func (r *Repository) NeedsReindex(maxAge time.Duration) (bool, error)
func (r *Repository) GetIndexState() (*IndexState, error)

// Search
func (r *Repository) Search() *Search
func (r *Repository) Retriever() *VectorStoreRetriever
```

---

## Performance Tuning

### Keyword Search

```go
// Use specific file patterns to reduce search space
opts := ai.KeywordSearchOptions{
    Query: "TODO",
    TopK:  100,
    // Filter by extension in your code
}
```

### Semantic Search

```go
// Batch queries for efficiency
queries := []string{"auth", "cache", "rate limit"}
for _, q := range queries {
    go func(query string) {
        results, _ := ai.SemanticSearchRepo(ctx, ".", query, 10, false)
        processResults(results)
    }(q)
}

// Use reranking only when quality matters
repo.Search().Semantic(ctx, ai.SemanticSearchOptions{
    Query:  query,
    TopK:   50,    // Get more candidates
    Rerank: true,  // Then rerank top results
})
```

### Indexing

```go
// Incremental indexing after git pull
watcher := ai.NewRepoWatcher(repo, 5*time.Minute)
watcher.Silent(true) // Reduce log noise
go watcher.Start(ctx)
```

---

## License

See [LICENSE](../LICENSE) for details.

## Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## Support

- 📖 [Full Documentation](./docs/)
- 🐛 [Issue Tracker](https://github.com/go-git/go-git/issues)
- 💬 [Discussions](https://github.com/go-git/go-git/discussions)
