# API Reference

Complete API documentation for `github.com/go-git/go-git/v6/ai`

---

## Table of Contents

- [Keyword Search API](#keyword-search-api)
- [Semantic Search API](#semantic-search-api)
- [Code Embedding API](#code-embedding-api)
- [Indexing Management](#indexing-management)
- [Symbol Analysis API](#symbol-analysis-api)
- [File Modification API](#file-modification-api)
- [Git Operations API](#git-operations-api)
- [Security Scanning API](#security-scanning-api)
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

## Symbol Analysis API

Code intelligence features for navigating and understanding Go codebases.

### GetSymbols

Extract all symbols from Go files in the repository.

```go
func (r *Repository) GetSymbols(
    ctx context.Context,
    opts GetSymbolsOptions,
) ([]Symbol, error)
```

**GetSymbolsOptions:**
```go
type GetSymbolsOptions struct {
    FilePath    string       // Optional: limit to specific file
    SymbolTypes []SymbolType // Optional: filter by type
    IncludeDocs bool         // Include doc comments
}

// SymbolType constants
const (
    SymbolTypeFunction   SymbolType = "function"
    SymbolTypeMethod     SymbolType = "method"
    SymbolTypeStruct     SymbolType = "struct"
    SymbolTypeInterface  SymbolType = "interface"
    SymbolTypeVariable   SymbolType = "variable"
    SymbolTypeConstant   SymbolType = "constant"
    SymbolTypeType       SymbolType = "type"
)
```

**Returns:**
- `[]Symbol` - List of symbols with metadata
- `error` - Any error encountered

**Example:**
```go
repo, _ := ai.Open(".")

// Get all symbols
symbols, err := repo.GetSymbols(ctx, ai.GetSymbolsOptions{
    IncludeDocs: true,
})

// Get only functions from specific file
functions, err := repo.GetSymbols(ctx, ai.GetSymbolsOptions{
    FilePath:    "main.go",
    SymbolTypes: []ai.SymbolType{ai.SymbolTypeFunction},
})

for _, sym := range symbols {
    fmt.Printf("%s %s at %s:%d\n", sym.Type, sym.Name, sym.FilePath, sym.Line)
    if sym.DocComment != "" {
        fmt.Printf("  // %s\n", sym.DocComment)
    }
}
```

---

### FindReferences

Find all references to a symbol across the repository.

```go
func (r *Repository) FindReferences(
    ctx context.Context,
    opts FindReferencesOptions,
) ([]Reference, error)
```

**FindReferencesOptions:**
```go
type FindReferencesOptions struct {
    SymbolName string // Symbol to find references for
    FilePath   string // Optional: limit to specific file
}
```

**Returns:**
- `[]Reference` - List of locations where symbol is referenced
- `error` - Any error encountered

**Example:**
```go
// Find all usages of a function
refs, err := repo.FindReferences(ctx, ai.FindReferencesOptions{
    SymbolName: "ProcessPayment",
})

for _, ref := range refs {
    fmt.Printf("%s:%d:%d - %s\n", 
        ref.FilePath, ref.Line, ref.Column, ref.Context)
}

// Find references in specific file
refs, err := repo.FindReferences(ctx, ai.FindReferencesOptions{
    SymbolName: "User",
    FilePath:   "handlers/auth.go",
})
```

---

### GetDefinition

Get the definition location of a symbol.

```go
func (r *Repository) GetDefinition(
    ctx context.Context,
    symbolName string,
) (*Symbol, error)
```

**Parameters:**
- `symbolName` - Name of the symbol to find

**Returns:**
- `*Symbol` - Symbol definition with location and metadata
- `error` - ErrSymbolNotFound if not found, or other errors

**Example:**
```go
// Jump to definition (IDE feature)
def, err := repo.GetDefinition(ctx, "UserRepository")
if err != nil {
    if err == ai.ErrSymbolNotFound {
        fmt.Println("Symbol not found")
    }
    return err
}

fmt.Printf("Defined at %s:%d\n", def.FilePath, def.Line)
fmt.Printf("Signature: %s\n", def.Signature)

// Open file at definition line
openInEditor(def.FilePath, def.Line)
```

---

## File Modification API

Safe file operations with confirmation workflow for AI agents.

### CreateFile

Create a new file with content.

```go
func (r *Repository) CreateFile(
    ctx context.Context,
    opts CreateFileOptions,
) (*FileOperation, error)
```

**CreateFileOptions:**
```go
type CreateFileOptions struct {
    FilePath  string // Relative path from repository root
    Content   string // File content
    Overwrite bool   // Allow overwriting existing file
}
```

**Returns:**
- `*FileOperation` - Pending operation (call `.Confirm()` to execute)
- `error` - Any error encountered

**Example:**
```go
repo, _ := ai.Open(".")

// Create new file (requires confirmation)
op, err := repo.CreateFile(ctx, ai.CreateFileOptions{
    FilePath: "internal/models/user.go",
    Content:  `package models\n\ntype User struct {\n\tID string\n}`,
})
if err != nil {
    return err
}

// Show operation to user
fmt.Printf("AI wants to create: %s\n", op.FilePath)
fmt.Printf("Content:\n%s\n", op.NewContent)

// User confirms
if userConfirms() {
    if err := op.Confirm(); err != nil {
        return err
    }
}

// Or bypass confirmation (careful!)
fm := ai.NewFileModifier(repo.Path())
fm.RequireConfirmation = false
op, _ = fm.CreateFile(ctx, opts)
// Automatically applied
```

---

### EditFile

Modify existing file content (full or line-range edits).

```go
func (r *Repository) EditFile(
    ctx context.Context,
    opts EditFileOptions,
) (*FileOperation, error)
```

**EditFileOptions:**
```go
type EditFileOptions struct {
    FilePath        string     // File to edit
    Content         string     // Replace entire file (if no LineRanges)
    LineRanges      []LineEdit // Or edit specific line ranges
    CreateIfMissing bool       // Create file if doesn't exist
}

type LineEdit struct {
    StartLine int    // 1-based line number
    EndLine   int    // 1-based, inclusive
    NewText   string // Replacement text
}
```

**Returns:**
- `*FileOperation` - Pending operation with old/new content diff
- `error` - Any error encountered

**Example:**
```go
// Replace entire file
op, err := repo.EditFile(ctx, ai.EditFileOptions{
    FilePath: "config.json",
    Content:  newConfigJSON,
})

// Edit specific line ranges (AI precision editing)
op, err := repo.EditFile(ctx, ai.EditFileOptions{
    FilePath: "main.go",
    LineRanges: []ai.LineEdit{
        {
            StartLine: 15,
            EndLine:   20,
            NewText:   "// Updated implementation\nfunc process() error {\n\treturn nil\n}",
        },
        {
            StartLine: 50,
            EndLine:   50,
            NewText:   "import \"github.com/pkg/errors\"",
        },
    },
})

// Show diff to user
fmt.Printf("Old:\n%s\n", op.OldContent)
fmt.Printf("New:\n%s\n", op.NewContent)

if userConfirms() {
    op.Confirm()
}
```

---

### DeleteFile

Delete a file from the repository.

```go
func (r *Repository) DeleteFile(
    ctx context.Context,
    opts DeleteFileOptions,
) (*FileOperation, error)
```

**DeleteFileOptions:**
```go
type DeleteFileOptions struct {
    FilePath string // File to delete
}
```

**Returns:**
- `*FileOperation` - Pending deletion operation
- `error` - Any error encountered

**Example:**
```go
op, err := repo.DeleteFile(ctx, ai.DeleteFileOptions{
    FilePath: "deprecated/old_handler.go",
})

fmt.Printf("AI wants to delete: %s\n", op.FilePath)
fmt.Printf("Current content:\n%s\n", op.OldContent)

if userConfirms() {
    op.Confirm()
}
```

---

## Git Operations API

Read and write Git operations for AI-powered workflows.

### GetDiff

Get diff between commits or working tree.

```go
func (r *Repository) GetDiff(
    ctx context.Context,
    opts DiffOptions,
) ([]DiffResult, error)
```

**DiffOptions:**
```go
type DiffOptions struct {
    FromCommit string // Starting commit hash
    ToCommit   string // Ending commit (empty = working tree)
    FilePath   string // Optional: specific file only
    MaxLines   int    // Truncate large diffs (0 = unlimited)
}
```

**Returns:**
- `[]DiffResult` - List of file changes with unified diff
- `error` - Any error encountered

**Example:**
```go
// Diff between commits
diffs, err := repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "HEAD~1",
    ToCommit:   "HEAD",
})

// Working tree changes (staged + unstaged)
diffs, err := repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "HEAD",
    ToCommit:   "", // Empty = working tree
})

// Specific file, truncated
diffs, err := repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "main",
    ToCommit:   "feature-branch",
    FilePath:   "src/api/handler.go",
    MaxLines:   100,
})

for _, d := range diffs {
    fmt.Printf("%s: %s (+%d -%d)\n", 
        d.Status, d.FilePath, d.Additions, d.Deletions)
    fmt.Println(d.Diff)
}
```

---

### GetBlame

Get line-by-line authorship information.

```go
func (r *Repository) GetBlame(
    ctx context.Context,
    opts BlameOptions,
) ([]BlameLine, error)
```

**BlameOptions:**
```go
type BlameOptions struct {
    FilePath string // File to blame
    Commit   string // Optional: blame at specific commit
}
```

**Returns:**
- `[]BlameLine` - One entry per line with commit/author info
- `error` - Any error encountered

**Example:**
```go
// Blame current version
blame, err := repo.GetBlame(ctx, ai.BlameOptions{
    FilePath: "src/auth.go",
})

// Blame at specific commit
blame, err := repo.GetBlame(ctx, ai.BlameOptions{
    FilePath: "config.yaml",
    Commit:   "abc123",
})

for _, line := range blame {
    fmt.Printf("%d: %s (%s, %s) - %s\n",
        line.LineNumber,
        line.CommitHash[:7],
        line.Author,
        line.Date.Format("2006-01-02"),
        line.Text)
}
```

---

### GetHistory

Get commit history with filtering.

```go
func (r *Repository) GetHistory(
    ctx context.Context,
    opts HistoryOptions,
) ([]CommitInfo, error)
```

**HistoryOptions:**
```go
type HistoryOptions struct {
    MaxCount int       // Limit number of commits
    FilePath string    // Optional: commits touching file
    Author   string    // Optional: filter by author
    Since    time.Time // Optional: commits after date
    Until    time.Time // Optional: commits before date
}
```

**Returns:**
- `[]CommitInfo` - List of commits with metadata
- `error` - Any error encountered

**Example:**
```go
// Recent 10 commits
history, err := repo.GetHistory(ctx, ai.HistoryOptions{
    MaxCount: 10,
})

// File history
history, err := repo.GetHistory(ctx, ai.HistoryOptions{
    FilePath: "README.md",
    MaxCount: 50,
})

// Commits by author in date range
history, err := repo.GetHistory(ctx, ai.HistoryOptions{
    Author:   "john@example.com",
    Since:    time.Now().AddDate(0, -1, 0), // Last month
    MaxCount: 100,
})

for _, commit := range history {
    fmt.Printf("%s - %s (%s)\n", 
        commit.ShortHash, commit.Message, commit.Author)
    for _, file := range commit.Files {
        fmt.Printf("  - %s\n", file)
    }
}
```

---

### CommitChanges

Create a commit with AI-generated or custom message.

```go
func (r *Repository) CommitChanges(
    ctx context.Context,
    opts CommitOptions,
) (*CommitResult, error)
```

**CommitOptions:**
```go
type CommitOptions struct {
    Message       string   // Commit message (auto-generated if empty)
    Files         []string // Specific files (empty = all changes)
    AuthorName    string   // Override author name
    AuthorEmail   string   // Override author email
    AllowEmpty    bool     // Allow commit with no changes
    AmendPrevious bool     // Amend previous commit
}
```

**Returns:**
- `*CommitResult` - Pending commit (call `.Confirm()` to execute)
- `error` - Any error encountered

**Example:**
```go
// AI-generated commit message from diff
result, err := repo.CommitChanges(ctx, ai.CommitOptions{})

fmt.Printf("AI commit message:\n%s\n", result.Message)
fmt.Printf("Files: %v\n", result.FilesChanged)

if userConfirms() {
    result.Confirm()
    fmt.Printf("Committed: %s\n", result.Hash)
}

// Custom message
result, err := repo.CommitChanges(ctx, ai.CommitOptions{
    Message: "feat: add user authentication",
    Files:   []string{"auth.go", "middleware.go"},
})

// Amend previous commit
result, err := repo.CommitChanges(ctx, ai.CommitOptions{
    Message:       "Updated documentation",
    AmendPrevious: true,
})
```

---

### CreateBranch

Create a new Git branch.

```go
func (r *Repository) CreateBranch(
    ctx context.Context,
    opts BranchOptions,
) (*BranchResult, error)
```

**BranchOptions:**
```go
type BranchOptions struct {
    Name       string // Branch name
    FromCommit string // Starting point (empty = HEAD)
    Checkout   bool   // Switch to branch after creation
    Force      bool   // Overwrite if exists
}
```

**Returns:**
- `*BranchResult` - Branch creation result
- `error` - Any error encountered

**Example:**
```go
// Create feature branch
result, err := repo.CreateBranch(ctx, ai.BranchOptions{
    Name:     "feature/new-api",
    Checkout: true,
})

// Create from specific commit
result, err := repo.CreateBranch(ctx, ai.BranchOptions{
    Name:       "hotfix/bug-123",
    FromCommit: "v1.0.0",
    Checkout:   true,
})

fmt.Printf("Created branch: %s at %s\n", result.Name, result.Hash[:7])
```

---

### DeleteBranch

Delete a Git branch.

```go
func (r *Repository) DeleteBranch(
    ctx context.Context,
    branchName string,
) error
```

**Parameters:**
- `branchName` - Name of branch to delete

**Returns:**
- `error` - Any error (cannot delete current branch)

**Example:**
```go
// Delete merged feature branch
err := repo.DeleteBranch(ctx, "feature/completed")
if err != nil {
    fmt.Printf("Cannot delete: %v\n", err)
}
```

---

## Security Scanning API

Detect secrets and analyze dependencies for security risks.

### ScanForSecrets

Scan repository for exposed secrets and credentials.

```go
func (r *Repository) ScanForSecrets(
    ctx context.Context,
    opts ScanOptions,
) (*ScanResult, error)
```

**ScanOptions:**
```go
type ScanOptions struct {
    FilePath        string          // Optional: scan specific file
    CustomPatterns  []SecretPattern // Additional patterns
    ExcludePatterns []string        // Files to exclude
}

type SecretPattern struct {
    Name        string         // Pattern name
    Pattern     *regexp.Regexp // Regex pattern
    Description string         // Description
    Severity    string         // "high", "medium", "low"
}
```

**Returns:**
- `*ScanResult` - Findings summary and details
- `error` - Any error encountered

**Built-in Patterns (11 types):**
- AWS Access Keys & Secret Keys
- GitHub Personal Access Tokens
- Google API Keys & OAuth
- Slack Tokens & Webhooks
- Private Keys (RSA, SSH)
- Generic API Keys
- Database Connection Strings
- JWT Tokens

**Example:**
```go
// Scan entire repository
result, err := repo.ScanForSecrets(ctx, ai.ScanOptions{})

fmt.Printf("Scanned %d files\n", result.FilesScanned)
fmt.Printf("Found %d secrets:\n", result.SecretsFound)
fmt.Printf("  High: %d, Medium: %d, Low: %d\n",
    result.HighSeverity, result.MediumSeverity, result.LowSeverity)

for _, finding := range result.Findings {
    fmt.Printf("\n[%s] %s\n", finding.Severity, finding.Type)
    fmt.Printf("  File: %s:%d\n", finding.FilePath, finding.Line)
    fmt.Printf("  Match: %s\n", finding.Redacted)
    fmt.Printf("  Context: %s\n", finding.Context)
}

// Scan specific file
result, err := repo.ScanForSecrets(ctx, ai.ScanOptions{
    FilePath: "config/credentials.yaml",
})

// Custom patterns
result, err := repo.ScanForSecrets(ctx, ai.ScanOptions{
    CustomPatterns: []ai.SecretPattern{
        {
            Name:        "Internal API Key",
            Pattern:     regexp.MustCompile(`INTERNAL_KEY_[A-Za-z0-9]{32}`),
            Description: "Internal service API key",
            Severity:    "high",
        },
    },
})
```

---

### AnalyzeDependencies

Parse and analyze project dependencies.

```go
func (r *Repository) AnalyzeDependencies(
    ctx context.Context,
) ([]DependencyInfo, error)
```

**Returns:**
- `[]DependencyInfo` - List of dependencies from manifest files
- `error` - Any error encountered

**Supported Formats:**
- Go: `go.mod`
- Node.js: `package.json`
- Python: `requirements.txt`, `Pipfile`

**Example:**
```go
deps, err := repo.AnalyzeDependencies(ctx)

fmt.Printf("Found %d dependencies:\n", len(deps))

for _, dep := range deps {
    fmt.Printf("%s: %s@%s (%s)\n",
        dep.Type, dep.Name, dep.Version, dep.Source)
}

// Group by type
byType := make(map[string][]ai.DependencyInfo)
for _, dep := range deps {
    byType[dep.Type] = append(byType[dep.Type], dep)
}

for typ, deps := range byType {
    fmt.Printf("\n%s dependencies: %d\n", typ, len(deps))
}
```

---

## Types Reference

### Search Types

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

### Symbol Analysis Types

```go
type Symbol struct {
    Name       string     // Symbol name
    Type       SymbolType // function, method, struct, etc.
    FilePath   string     // Source file
    Line       int        // Line number
    Column     int        // Column number
    Signature  string     // Full signature
    DocComment string     // Documentation comment
    Receiver   string     // For methods only
}

type Reference struct {
    FilePath string // File containing reference
    Line     int    // Line number
    Column   int    // Column number
    Context  string // Line of code
}

type GetSymbolsOptions struct {
    FilePath    string
    SymbolTypes []SymbolType
    IncludeDocs bool
}

type FindReferencesOptions struct {
    SymbolName string
    FilePath   string
}
```

---

### File Modification Types

```go
type FileOperation struct {
    Type        string // "create", "edit", "delete"
    FilePath    string
    OldContent  string
    NewContent  string
    Description string
    NeedsConfirm bool
}

func (op *FileOperation) Confirm() error // Execute operation

type CreateFileOptions struct {
    FilePath  string
    Content   string
    Overwrite bool
}

type EditFileOptions struct {
    FilePath        string
    Content         string
    LineRanges      []LineEdit
    CreateIfMissing bool
}

type LineEdit struct {
    StartLine int    // 1-based
    EndLine   int    // 1-based, inclusive
    NewText   string
}

type DeleteFileOptions struct {
    FilePath string
}
```

---

### Git Operations Types

```go
type DiffOptions struct {
    FromCommit string
    ToCommit   string
    FilePath   string
    MaxLines   int
}

type DiffResult struct {
    FilePath  string
    OldPath   string // For renames
    Status    string // Added, Deleted, Modified, Renamed
    Additions int
    Deletions int
    Diff      string // Unified diff format
}

type BlameOptions struct {
    FilePath string
    Commit   string
}

type BlameLine struct {
    LineNumber int
    CommitHash string
    Author     string
    Date       time.Time
    Text       string
}

type HistoryOptions struct {
    MaxCount int
    FilePath string
    Author   string
    Since    time.Time
    Until    time.Time
}

type CommitInfo struct {
    Hash      string
    ShortHash string
    Author    string
    Email     string
    Date      time.Time
    Message   string
    Files     []string
}

type CommitOptions struct {
    Message       string
    Files         []string
    AuthorName    string
    AuthorEmail   string
    AllowEmpty    bool
    AmendPrevious bool
}

type CommitResult struct {
    Hash         string
    Message      string
    FilesChanged []string
    NeedsConfirm bool
}

func (cr *CommitResult) Confirm() error // Execute commit

type BranchOptions struct {
    Name       string
    FromCommit string
    Checkout   bool
    Force      bool
}

type BranchResult struct {
    Name    string
    Hash    string
    Message string
}
```

---

### Security Scanning Types

```go
type ScanOptions struct {
    FilePath        string
    CustomPatterns  []SecretPattern
    ExcludePatterns []string
}

type ScanResult struct {
    Findings       []SecretFinding
    FilesScanned   int
    SecretsFound   int
    HighSeverity   int
    MediumSeverity int
    LowSeverity    int
}

type SecretFinding struct {
    Type        string // "aws_access_key", "github_token", etc.
    Description string
    Severity    string // "high", "medium", "low"
    FilePath    string
    Line        int
    Column      int
    Match       string // Raw match (use carefully!)
    Redacted    string // Safe to display
    Context     string // Surrounding code
}

type SecretPattern struct {
    Name        string
    Pattern     *regexp.Regexp
    Description string
    Severity    string
}

type DependencyInfo struct {
    Name    string // Package name
    Version string // Version string
    Type    string // "go", "npm", "python"
    Source  string // "go.mod", "package.json", etc.
}
```

---

## Error Handling

### Common Errors

```go
// Index not found
if err != nil && strings.Contains(err.Error(), "not indexed") {
    repo.Index(ctx)
}

// Connection errors (semantic search only)
if err != nil && strings.Contains(err.Error(), "connection") {
    // Check TEI/Chroma services are running
}

// Context cancelled
if errors.Is(err, context.Canceled) {
    // User cancelled operation
}

// Symbol not found
if errors.Is(err, ai.ErrSymbolNotFound) {
    fmt.Println("Symbol doesn't exist in codebase")
}

// File operation errors
if err != nil && strings.Contains(err.Error(), "already exists") {
    // Use Overwrite: true or choose different name
}

// Git operation errors
if err != nil && strings.Contains(err.Error(), "no changes") {
    // Nothing to commit
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

    // === Search Features ===
    
    // Keyword search (fast, local)
    fmt.Println("\n=== Keyword Search ===")
    kwResults, _ := ai.KeywordSearchRepo(ctx, ".", "OAuth", 5)
    for _, r := range kwResults {
        fmt.Printf("%s:%d\n", r.Chunk.FilePath, r.Chunk.StartLine)
    }

    // Semantic search (AI-powered, requires TEI/Chroma)
    if repo.IsIndexed() {
        fmt.Println("\n=== Semantic Search ===")
        semResults, _ := ai.SemanticSearchRepo(ctx, ".", "token refresh logic", 5, true)
        for _, r := range semResults {
            fmt.Printf("%.3f - %s:%d\n", r.Score, r.Chunk.FilePath, r.Chunk.StartLine)
        }
    }

    // === Symbol Analysis (Go code intelligence) ===
    
    fmt.Println("\n=== Symbol Analysis ===")
    
    // Get all functions
    symbols, _ := repo.GetSymbols(ctx, ai.GetSymbolsOptions{
        SymbolTypes: []ai.SymbolType{ai.SymbolTypeFunction},
        IncludeDocs: true,
    })
    fmt.Printf("Found %d functions\n", len(symbols))
    
    // Find references to a symbol
    refs, _ := repo.FindReferences(ctx, ai.FindReferencesOptions{
        SymbolName: "ProcessPayment",
    })
    fmt.Printf("Found %d references\n", len(refs))
    
    // Get definition
    def, _ := repo.GetDefinition(ctx, "UserRepository")
    if def != nil {
        fmt.Printf("Defined at %s:%d\n", def.FilePath, def.Line)
    }

    // === File Modification (with confirmation) ===
    
    fmt.Println("\n=== File Modification ===")
    
    // Create new file
    op, _ := repo.CreateFile(ctx, ai.CreateFileOptions{
        FilePath: "internal/utils/helper.go",
        Content:  "package utils\n\nfunc Helper() {}\n",
    })
    fmt.Printf("AI wants to create: %s\n", op.FilePath)
    // op.Confirm() to apply
    
    // Edit specific lines
    op, _ = repo.EditFile(ctx, ai.EditFileOptions{
        FilePath: "main.go",
        LineRanges: []ai.LineEdit{
            {StartLine: 10, EndLine: 15, NewText: "// Updated code"},
        },
    })
    fmt.Printf("AI wants to edit lines %d-%d\n", 10, 15)

    // === Git Operations ===
    
    fmt.Println("\n=== Git Operations ===")
    
    // Get diff
    diffs, _ := repo.GetDiff(ctx, ai.DiffOptions{
        FromCommit: "HEAD~1",
        ToCommit:   "HEAD",
    })
    fmt.Printf("Changed files: %d\n", len(diffs))
    
    // Get blame
    blame, _ := repo.GetBlame(ctx, ai.BlameOptions{
        FilePath: "README.md",
    })
    fmt.Printf("Blame lines: %d\n", len(blame))
    
    // Get history
    history, _ := repo.GetHistory(ctx, ai.HistoryOptions{
        MaxCount: 10,
    })
    for _, commit := range history {
        fmt.Printf("%s - %s\n", commit.ShortHash, commit.Message)
    }
    
    // AI-generated commit
    result, _ := repo.CommitChanges(ctx, ai.CommitOptions{})
    fmt.Printf("AI commit message: %s\n", result.Message)
    // result.Confirm() to apply
    
    // Create branch
    branch, _ := repo.CreateBranch(ctx, ai.BranchOptions{
        Name:     "feature/new-api",
        Checkout: true,
    })
    fmt.Printf("Created branch: %s\n", branch.Name)

    // === Security Scanning ===
    
    fmt.Println("\n=== Security Scanning ===")
    
    // Scan for secrets
    scanResult, _ := repo.ScanForSecrets(ctx, ai.ScanOptions{})
    fmt.Printf("Found %d secrets (High: %d, Medium: %d, Low: %d)\n",
        scanResult.SecretsFound,
        scanResult.HighSeverity,
        scanResult.MediumSeverity,
        scanResult.LowSeverity)
    
    for _, finding := range scanResult.Findings {
        fmt.Printf("[%s] %s at %s:%d\n",
            finding.Severity, finding.Type, finding.FilePath, finding.Line)
    }
    
    // Analyze dependencies
    deps, _ := repo.AnalyzeDependencies(ctx)
    fmt.Printf("Dependencies: %d\n", len(deps))
    for _, dep := range deps {
        fmt.Printf("  %s@%s (%s)\n", dep.Name, dep.Version, dep.Type)
    }

    // === Background Watcher (optional) ===
    
    watcher := ai.NewRepoWatcher(repo, 5*time.Minute)
    go watcher.Start(ctx)
    defer watcher.Stop()
}
```

---

## See Also

- [README.md](./README.md) - Overview and quick start
- [Examples](./_examples/) - Complete working examples
- [AI_SETUP.md](../AI_SETUP.md) - Service setup guide
