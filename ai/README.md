# AI-Powered Git Repository API

This package provides comprehensive AI-powered capabilities for Git repositories, combining code search with intelligent code manipulation:

## Core Capabilities

### 🔍 **Search & Discovery**
1. **Keyword Search** - Fast text-based search (like `grep`)
2. **Semantic Search** - AI-powered understanding of code meaning
3. **File Search** - Fast filename-only search with glob patterns
4. **Grep Search** - Powerful regex search across repository
5. **Symbol Analysis** - Go code intelligence (functions, types, references)

### 📊 **Context & Intelligence**
6. **Workspace Stats** - Language breakdown, LOC, project metrics
7. **Recent Files** - Git history-based recent file tracking
8. **File Metadata** - Comprehensive file information (size, git status, etc.)

### ✏️ **Code Modification**
9. **File Operations** - Safe file creation, editing, and deletion with confirmation
10. **Git Operations** - AI-assisted commits, branch management, diff, blame, history
11. **Git Stash** - Save/restore work in progress
12. **Security Scanning** - Secret detection and dependency analysis

---

## Quick Start

```go
import "github.com/go-git/go-git/v6/ai"

// Open repository
repo, _ := ai.Open("/path/to/repo")

// 🔍 SEARCH - Keyword search (instant, no indexing needed)
results, _ := ai.KeywordSearchRepo(ctx, "/path/to/repo", "OAuth", 10)

// 🔍 SEARCH - Semantic search (auto-indexes on first use)
results, _ := ai.SemanticSearchRepo(ctx, "/path/to/repo", "authentication logic", 10, false)

// 📁 FILE SEARCH - Find files by name (fast filename-only search)
files, _ := repo.FindFiles(ctx, ai.FindFilesOptions{
    Query: "test", UseGlob: true, MaxResults: 10,
})

// 🔎 GREP - Powerful regex search across files
matches, _ := repo.GrepSearch(ctx, ai.GrepOptions{
    Pattern: "TODO|FIXME", UseRegex: true, ContextLines: 2,
})

// 🧠 SYMBOLS - Analyze Go code symbols
symbols, _ := repo.GetSymbols(ctx, ai.GetSymbolsOptions{FilePath: "main.go"})
refs, _ := repo.FindReferences(ctx, ai.FindReferencesOptions{SymbolName: "UserAuth"})

// 📊 STATS - Get workspace statistics
stats, _ := repo.GetWorkspaceStats(ctx)
fmt.Printf("Languages: %v, LOC: %d\n", stats.PrimaryLanguage, stats.TotalLines)

// 📝 RECENT - Track recently modified files
recent, _ := repo.GetRecentFiles(ctx, ai.RecentFilesOptions{MaxFiles: 10})

// 📄 METADATA - Get file information
metadata, _ := repo.GetFileMetadata(ctx, "main.go")
fmt.Printf("Size: %s, Lines: %d, Status: %s\n", metadata.SizeHuman, metadata.LineCount, metadata.GitStatus)

// ✏️ MODIFY - Edit files safely with confirmation
op, _ := repo.EditFile(ctx, ai.EditFileOptions{
    FilePath: "config.go",
    Content:  "package config\n\nconst Version = \"2.0\"",
})
op.Confirm() // Apply after confirmation

// 🌿 BRANCHES - Extended branch operations
branches, _ := repo.ListBranches(ctx)
repo.SwitchBranch(ctx, "feature/new-api", true)
repo.MergeBranch(ctx, "feature/completed")

// 💾 STASH - Save/restore work in progress
stash, _ := repo.StashChanges(ctx, ai.StashOptions{Message: "WIP"})
repo.ApplyStash(ctx, 0, false)

// 🔒 SECURITY - Scan for secrets
result, _ := repo.ScanForSecrets(ctx, ai.ScanOptions{})
fmt.Printf("Found %d secrets\n", len(result.Findings))

// 📝 GIT - Smart commits with AI-generated messages
commitResult, _ := repo.CommitChanges(ctx, ai.CommitOptions{})
fmt.Printf("Committed: %s\n", commitResult.Message)
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

## 4. Symbol Analysis (New!)

Go code intelligence with AST parsing for functions, types, and references.

### Features

- ✅ **Symbol extraction** - Functions, methods, structs, interfaces, variables, constants
- ✅ **Reference finding** - Find all usages of a symbol across repository
- ✅ **Definition lookup** - Jump to symbol definition
- ✅ **Documentation extraction** - Include doc comments
- ✅ **Multi-file analysis** - Scan entire repository or specific files
- ✅ **Generic support** - Handles Go 1.18+ generics

### API

#### Get Symbols

```go
repo, _ := ai.Open("/path/to/repo")

// Get all symbols from a file
symbols, err := repo.GetSymbols(ctx, ai.GetSymbolsOptions{
    FilePath:    "main.go",
    IncludeDocs: true,
})

for _, sym := range symbols {
    fmt.Printf("%s: %s at %s:%d\n", sym.Type, sym.Name, sym.FilePath, sym.Line)
}

// Filter by symbol type
symbols, _ = repo.GetSymbols(ctx, ai.GetSymbolsOptions{
    SymbolTypes: []ai.SymbolType{ai.SymbolTypeFunction, ai.SymbolTypeMethod},
})

// Scan entire repository
symbols, _ = repo.GetSymbols(ctx, ai.GetSymbolsOptions{}) // All files
```

#### Find References

```go
// Find all usages of a symbol
refs, err := repo.FindReferences(ctx, ai.FindReferencesOptions{
    SymbolName: "AuthenticateUser",
})

for _, ref := range refs {
    fmt.Printf("%s:%d - %s\n", ref.FilePath, ref.Line, ref.Context)
}

// Limit to specific file
refs, _ = repo.FindReferences(ctx, ai.FindReferencesOptions{
    SymbolName: "TokenRefresh",
    FilePath:   "auth/tokens.go",
})
```

#### Get Definition

```go
// Find where a symbol is defined
def, err := repo.GetDefinition(ctx, "UserRepository")
if err == nil {
    fmt.Printf("Defined in %s at line %d\n", def.FilePath, def.Line)
    fmt.Printf("Signature: %s\n", def.Signature)
    fmt.Printf("Documentation: %s\n", def.DocComment)
}
```

### Symbol Types

```go
const (
    SymbolTypeFunction   = "function"
    SymbolTypeMethod     = "method"
    SymbolTypeStruct     = "struct"
    SymbolTypeInterface  = "interface"
    SymbolTypeVariable   = "variable"
    SymbolTypeConstant   = "constant"
    SymbolTypeType       = "type"
    SymbolTypeImport     = "import"
)
```

### Use Cases

- **IDE features** - Go to definition, find references
- **Code navigation** - Explore unfamiliar codebases
- **Refactoring** - Find all usages before renaming
- **Documentation** - Extract API surface
- **Code analysis** - Build dependency graphs

---

## 5. File Modification (New!)

Safe file operations with confirmation workflow for AI agents.

### Features

- ✅ **Create files** - With automatic directory creation
- ✅ **Edit files** - Full content replacement or line-range edits
- ✅ **Delete files** - With safety checks
- ✅ **Confirmation workflow** - Preview changes before applying
- ✅ **Operation descriptions** - Human-readable change summaries

### API

#### Create Files

```go
repo, _ := ai.Open("/path/to/repo")

// Create new file (with confirmation by default)
op, err := repo.CreateFile(ctx, ai.CreateFileOptions{
    FilePath: "config/database.go",
    Content:  "package config\n\nconst DBHost = \"localhost\"",
})

// Preview the operation
fmt.Println(op.Description)
// Output: Create new file: config/database.go
//         Size: 47 bytes
//         Lines: 3

// Apply after confirmation
err = repo.ApplyFileOperation(op)
```

#### Edit Files

```go
// Replace entire file
op, err := repo.EditFile(ctx, ai.EditFileOptions{
    FilePath: "main.go",
    Content:  newFileContent,
})

// Edit specific line ranges
op, err = repo.EditFile(ctx, ai.EditFileOptions{
    FilePath: "auth.go",
    LineRanges: []ai.LineEdit{
        {StartLine: 10, EndLine: 15, NewText: "// Updated implementation\nfunc Auth() { ... }"},
        {StartLine: 25, EndLine: 25, NewText: "const MaxRetries = 5"},
    },
})

// Create file if missing
op, err = repo.EditFile(ctx, ai.EditFileOptions{
    FilePath:        "config.go",
    Content:         defaultConfig,
    CreateIfMissing: true,
})
```

#### Delete Files

```go
// Delete file (with confirmation)
op, err := repo.DeleteFile(ctx, ai.DeleteFileOptions{
    FilePath: "old_config.go",
})

// Review before applying
if op.NeedsConfirm {
    fmt.Println(op.Description) // "Delete file: old_config.go (1234 bytes)"
    repo.ApplyFileOperation(op)
}
```

#### Skip Confirmation (for batch operations)

```go
// Disable confirmation for automated workflows
fm := ai.NewFileModifier("/path/to/repo")
fm.RequireConfirmation = false

op, _ := fm.CreateFile(ctx, ai.CreateFileOptions{...})
// File created immediately, no manual confirmation needed
```

### Use Cases

- **AI code editors** - GitHub Copilot, Cursor-style editing
- **Code generation** - Create files from templates
- **Refactoring tools** - Safe bulk modifications
- **CI/CD automation** - Generate configuration files

---

## 6. Git Operations (New!)

Git read operations (diff, blame, history) and write operations (commit, branch) with AI assistance.

### Features

- ✅ **Diff** - Compare commits or working tree
- ✅ **Blame** - Line-by-line authorship
- ✅ **History** - Commit logs with filtering
- ✅ **Smart commits** - AI-generated commit messages
- ✅ **Branch management** - Create, delete, checkout branches
- ✅ **Safety checks** - Prevent destructive operations

### API

#### Git Diff

```go
repo, _ := ai.Open("/path/to/repo")

// Diff between commits
diffs, err := repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "abc123",
    ToCommit:   "def456",
})

for _, d := range diffs {
    fmt.Printf("%s: +%d -%d\n", d.FilePath, d.Additions, d.Deletions)
    fmt.Println(d.Diff) // Unified diff format
}

// Diff with working tree
diffs, _ = repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "HEAD",
    // ToCommit empty = working tree
})

// Limit diff size
diffs, _ = repo.GetDiff(ctx, ai.DiffOptions{
    FromCommit: "HEAD~1",
    ToCommit:   "HEAD",
    MaxLines:   50, // Truncate large diffs
})
```

#### Git Blame

```go
// Get line-by-line authorship
blameLines, err := repo.GetBlame(ctx, ai.BlameOptions{
    FilePath: "main.go",
})

for _, line := range blameLines {
    fmt.Printf("Line %d: %s (%s) - %s\n",
        line.LineNumber, line.Author, line.Date.Format("2006-01-02"), line.Text)
}

// Blame at specific commit
blameLines, _ = repo.GetBlame(ctx, ai.BlameOptions{
    FilePath: "config.go",
    Commit:   "abc123",
})
```

#### Git History

```go
// Get commit history
commits, err := repo.GetHistory(ctx, ai.HistoryOptions{
    MaxCount: 50,
})

for _, c := range commits {
    fmt.Printf("%s: %s by %s\n", c.ShortHash, c.Message, c.Author)
}

// Filter by file
commits, _ = repo.GetHistory(ctx, ai.HistoryOptions{
    FilePath: "auth/user.go",
    MaxCount: 20,
})

// Filter by author
commits, _ = repo.GetHistory(ctx, ai.HistoryOptions{
    Author:   "alice@example.com",
    MaxCount: 100,
})
```

#### Smart Commits (AI-Generated Messages)

```go
// Commit with auto-generated message
result, err := repo.CommitChanges(ctx, ai.CommitOptions{})
// Message generated based on file changes:
// "feat: add user authentication module"
// "chore: update dependencies"
// "docs: improve README examples"

fmt.Printf("Committed %s: %s\n", result.Hash, result.Message)
fmt.Printf("Files changed: %v\n", result.FilesChanged)

// Custom commit message
result, _ = repo.CommitChanges(ctx, ai.CommitOptions{
    Message:     "fix: resolve token expiration bug",
    AuthorName:  "Alice",
    AuthorEmail: "alice@example.com",
})

// Allow empty commits
result, _ = repo.CommitChanges(ctx, ai.CommitOptions{
    Message:    "chore: trigger CI",
    AllowEmpty: true,
})
```

#### Branch Management

```go
// Create branch
branchResult, err := repo.CreateBranch(ctx, ai.BranchOptions{
    Name:     "feature/new-auth",
    Checkout: true, // Switch to new branch
})

// Create from specific commit
branchResult, _ = repo.CreateBranch(ctx, ai.BranchOptions{
    Name:       "hotfix/security",
    FromCommit: "abc123",
})

// Force recreate branch
branchResult, _ = repo.CreateBranch(ctx, ai.BranchOptions{
    Name:  "develop",
    Force: true, // Overwrite if exists
})

// Delete branch
err = repo.DeleteBranch(ctx, "old-feature", false)
```

### Convenience Methods

```go
// Quick commit (auto-message, no confirmation)
hash, err := repo.QuickCommit(ctx, "")

// Quick history (last 10 commits)
commits, err := repo.QuickHistory(ctx)
```

### Use Cases

- **AI coding assistants** - GitHub Copilot commit messages
- **Code review tools** - Automated diff analysis
- **Git automation** - Smart branch workflows
- **Developer tools** - Enhanced git blame, history

---

## 7. Security Scanning (New!)

Detect secrets, credentials, and analyze dependencies.

### Features

- ✅ **Secret detection** - 11+ common secret patterns
- ✅ **Secret redaction** - Safe display of findings
- ✅ **Dependency analysis** - Parse go.mod, package.json, requirements.txt
- ✅ **Severity levels** - High, medium, low
- ✅ **File filtering** - Scan specific files or patterns

### API

#### Scan for Secrets

```go
repo, _ := ai.Open("/path/to/repo")

// Scan entire repository
result, err := repo.ScanForSecrets(ctx, ai.ScanOptions{})

fmt.Printf("Found %d secrets\n", result.SecretsFound)
fmt.Printf("High severity: %d\n", result.HighSeverity)

for _, finding := range result.Findings {
    fmt.Printf("[%s] %s in %s:%d\n",
        finding.Severity, finding.Type, finding.FilePath, finding.Line)
    fmt.Printf("Redacted: %s\n", finding.Redacted) // Safe to display
}

// Scan specific file
result, _ = repo.ScanForSecrets(ctx, ai.ScanOptions{
    FilePath: "config.yaml",
})

// Exclude directories
result, _ = repo.ScanForSecrets(ctx, ai.ScanOptions{
    ExcludePatterns: []string{"vendor/", "node_modules/", ".git/"},
})

// Add custom patterns
result, _ = repo.ScanForSecrets(ctx, ai.ScanOptions{
    CustomPatterns: []ai.SecretPattern{
        {
            Name:     "Custom API Key",
            Pattern:  regexp.MustCompile(`CUSTOM_KEY_[A-Z0-9]{32}`),
            Severity: "high",
        },
    },
})
```

#### Secret Patterns Detected

- AWS Access Keys (AKIA...)
- AWS Secret Keys
- GitHub Tokens (ghp_..., gho_...)
- OpenAI API Keys (sk-...)
- Private Keys (-----BEGIN PRIVATE KEY-----)
- JWT Tokens
- Database URLs (postgres://, mysql://)
- Generic API Keys
- Passwords in configs
- Google API Keys (AIza...)
- Slack Tokens (xoxb-...)

#### Analyze Dependencies

```go
// Parse dependency files
deps, err := repo.AnalyzeDependencies(ctx)

for _, dep := range deps {
    fmt.Printf("%s@%s (%s from %s)\n",
        dep.Name, dep.Version, dep.Type, dep.Source)
}
```

**Supported formats:**
- `go.mod` (Go modules)
- `package.json` (npm/Node.js)
- `requirements.txt` (Python pip)

#### Convenience Method

```go
// Quick security scan
summary, err := repo.QuickSecurityScan(ctx)
fmt.Printf("Secrets: %d | Dependencies: %d\n",
    len(summary.Secrets), len(summary.Dependencies))
```

### Use Cases

- **Security audits** - Find leaked credentials
- **CI/CD gates** - Block commits with secrets
- **Compliance** - Ensure no sensitive data in repos
- **Dependency tracking** - Monitor external packages

---

## 8. File Search

Fast filename-only search optimized for finding files quickly, separate from content search.

### Features

- ✅ **Glob pattern support** - `*.go`, `test*`, `*_test.go`
- ✅ **Multiple matching strategies** - Exact, prefix, suffix, contains, fuzzy
- ✅ **Relevance scoring** - Better matches ranked higher
- ✅ **Extension filtering** - Find all `.py`, `.js`, `.go` files
- ✅ **Directory exclusions** - Skip `node_modules`, `.git`, `vendor`
- ✅ **Case-insensitive** - Works like GitHub Copilot's `#` file references

### API

```go
// Find files by name
matches, err := repo.FindFiles(ctx, ai.FindFilesOptions{
    Query:      "test",
    MaxResults: 20,
})

// Find with glob patterns
matches, err := repo.FindFiles(ctx, ai.FindFilesOptions{
    Query:   "*.go",
    UseGlob: true,
})

// Filter by extension
goFiles, err := repo.GetFilesByExtension(ctx, []string{".go", ".mod"})

// List all files
allFiles, err := repo.ListAllFiles(ctx)
```

### Match Scoring

Files are ranked by relevance:
- **1.0** - Exact filename match
- **0.85** - Glob pattern match
- **0.8** - Prefix match
- **0.75** - Suffix match
- **0.7** - Contains in filename
- **0.6** - Contains in path
- **0.5** - Fuzzy match

### Use Cases

- **LLM file references** - Like GitHub Copilot's `#filename` feature
- **Quick navigation** - Jump to files by name
- **IDE autocomplete** - File path suggestions
- **Build tools** - Find test files, config files

---

## 9. Workspace Statistics

Comprehensive project metrics and language analysis.

### Features

- ✅ **Language detection** - 50+ programming languages
- ✅ **LOC counting** - Lines of code per language
- ✅ **File statistics** - Counts by language and extension
- ✅ **Directory analysis** - Top directories by file count
- ✅ **Size metrics** - Total size and size per language
- ✅ **Project insights** - Primary language, depth, structure

### API

```go
// Get comprehensive stats
stats, err := repo.GetWorkspaceStats(ctx)

fmt.Printf("Primary Language: %s\n", stats.PrimaryLanguage)
fmt.Printf("Total Files: %d\n", stats.TotalFiles)
fmt.Printf("Total Lines: %d\n", stats.TotalLines)
fmt.Printf("Total Size: %s\n", formatBytes(stats.TotalSize))

// Language breakdown
for lang, count := range stats.FilesByLanguage {
    fmt.Printf("%s: %d files (%.1f%%)\n", 
        lang, count, stats.LanguagePercent[lang])
}

// Get detailed language info (sorted by percentage)
languages, err := repo.GetLanguageBreakdown(ctx)
for _, lang := range languages {
    fmt.Printf("%s: %d files, %d lines, %.1f%%\n",
        lang.Name, lang.Files, lang.Lines, lang.Percentage)
}
```

### Supported Languages

Go, Python, JavaScript, TypeScript, Java, C, C++, C#, Ruby, PHP, Swift, Kotlin, Rust, Scala, Shell, Perl, R, Objective-C, SQL, HTML, CSS, YAML, JSON, Markdown, and 25+ more.

### Use Cases

- **Project overview** - Understand codebase composition
- **Tech stack discovery** - Identify languages used
- **Code metrics** - LOC tracking and reporting
- **LLM context** - Provide project info to AI assistants

---

## 10. Recent Files Tracking

Git history-based tracking of recently modified files.

### Features

- ✅ **Git history analysis** - Find files from recent commits
- ✅ **Time-based filtering** - Last 7 days, 30 days, custom
- ✅ **Author filtering** - Find changes by specific developers
- ✅ **Extension filtering** - Track only `.go`, `.py`, etc.
- ✅ **Change types** - Added, modified, deleted
- ✅ **Sorted by recency** - Most recent first

### API

```go
// Get recent files (last 7 days)
recent, err := repo.GetRecentFiles(ctx, ai.RecentFilesOptions{
    MaxFiles: 20,
    MaxAge:   7 * 24 * time.Hour,
})

for _, file := range recent {
    fmt.Printf("%s - %s by %s (%s)\n",
        file.FilePath,
        file.LastModified.Format("2006-01-02"),
        file.Author,
        file.ChangeType)
}

// Filter by extension
goRecent, err := repo.GetRecentFiles(ctx, ai.RecentFilesOptions{
    MaxFiles:  10,
    Extension: []string{".go"},
})

// Get files from last N commits
files, err := repo.GetRecentlyModifiedFiles(ctx, 10)

// Get full history for a file
history, err := repo.GetFileHistory(ctx, "main.go", 50)
```

### Use Cases

- **Context awareness** - Show what user has been working on
- **Quick access** - Jump to recently edited files
- **Activity tracking** - Monitor project activity
- **LLM context** - Provide recent work history to AI

---

## 11. File Metadata

Comprehensive file information including Git status and content analysis.

### Features

- ✅ **Basic info** - Size, timestamps, permissions
- ✅ **Git integration** - Status, blob hash, last commit
- ✅ **Content analysis** - Line count, binary detection, encoding
- ✅ **Language detection** - Auto-detect programming language
- ✅ **Bulk operations** - Get metadata for multiple files efficiently

### API

```go
// Get file metadata
metadata, err := repo.GetFileMetadata(ctx, "main.go")

fmt.Printf("File: %s\n", metadata.FilePath)
fmt.Printf("Size: %s (%d bytes)\n", metadata.SizeHuman, metadata.Size)
fmt.Printf("Lines: %d\n", metadata.LineCount)
fmt.Printf("Language: %s\n", metadata.Language)
fmt.Printf("Git Status: %s\n", metadata.GitStatus)
fmt.Printf("Last Modified: %s\n", metadata.LastModified)

// Last commit info
if metadata.LastCommit != nil {
    fmt.Printf("Last Commit: %s by %s\n",
        metadata.LastCommit.ShortHash,
        metadata.LastCommit.Author)
}

// Bulk metadata
files := []string{"main.go", "utils.go", "config.go"}
metadata, err := repo.GetBulkMetadata(ctx, files)
```

### Git Status Values

- `tracked` - File is committed and unchanged
- `modified` - File has uncommitted changes
- `staged-modified` - Changes are staged
- `staged-new` - New file staged for commit
- `untracked` - File not in Git
- `deleted` - File deleted but not committed

### Use Cases

- **File information panels** - Show file details in IDEs
- **LLM context** - Provide file info to AI assistants
- **Build systems** - Check file sizes and types
- **Git status UI** - Display working tree status

---

## 12. Grep Search

Powerful text/regex search across all repository files.

### Features

- ✅ **Regex support** - Full regular expression matching
- ✅ **Context lines** - Show N lines before/after matches
- ✅ **Advanced filtering** - Include/exclude patterns
- ✅ **Case sensitivity** - Optional case-sensitive matching
- ✅ **Whole word** - Match complete words only
- ✅ **Invert match** - Find lines NOT matching pattern
- ✅ **Multiple modes** - Full matches, count only, files only

### API

```go
// Basic text search
matches, err := repo.GrepSearch(ctx, ai.GrepOptions{
    Pattern:    "TODO",
    MaxMatches: 100,
})

// Regex search with context
matches, err := repo.GrepSearch(ctx, ai.GrepOptions{
    Pattern:      "func.*Error.*{",
    UseRegex:     true,
    ContextLines: 3,
})

// Filter by file type
matches, err := repo.GrepSearch(ctx, ai.GrepOptions{
    Pattern: "authenticate",
    Include: []string{"*.go", "*.py"},
    Exclude: []string{"*_test.go"},
})

// Get just count
count, err := repo.GrepCount(ctx, ai.GrepOptions{
    Pattern: "FIXME",
})

// Get just filenames containing matches
files, err := repo.GrepFiles(ctx, ai.GrepOptions{
    Pattern: "password",
})
```

### Use Cases

- **Code search** - Find patterns across codebase
- **Refactoring** - Find all usages before changes
- **Code review** - Search for problematic patterns
- **Documentation** - Find TODO/FIXME comments

---

## 13. Git Stash Operations

Save and restore work in progress.

### Features

- ✅ **Save changes** - Stash uncommitted work
- ✅ **Custom messages** - Descriptive stash names
- ✅ **List stashes** - View all saved stashes
- ✅ **Apply stash** - Restore without deleting
- ✅ **Pop stash** - Restore and delete
- ✅ **Drop stash** - Delete without applying
- ✅ **Clear all** - Remove all stashes
- ✅ **Confirmation workflow** - Safe for AI agents

### API

```go
// Stash current changes
result, err := repo.StashChanges(ctx, ai.StashOptions{
    Message: "WIP: refactoring auth module",
})
result.Confirm() // Apply after confirmation

// List all stashes
stashes, err := repo.ListStashes(ctx)
for i, stash := range stashes {
    fmt.Printf("stash@{%d}: %s (%s)\n",
        i, stash.Message, stash.CreatedAt.Format("2006-01-02"))
}

// Apply stash (keep it in list)
err := repo.ApplyStash(ctx, 0, false)

// Pop stash (apply and remove)
err := repo.PopStash(ctx, 0)

// Drop specific stash
err := repo.DropStash(ctx, 0)

// Clear all stashes
err := repo.ClearStashes(ctx)
```

### Use Cases

- **Context switching** - Save work when switching branches
- **Experimentation** - Try changes without committing
- **Emergency fixes** - Quickly save unfinished work
- **AI agents** - Safe WIP management

---

## 14. Extended Branch Operations

Comprehensive branch management beyond create/delete.

### Features

- ✅ **List branches** - All local and remote branches
- ✅ **Branch metadata** - Last commit, author, date
- ✅ **Switch branches** - Checkout existing or create new
- ✅ **Merge branches** - Merge with conflict detection
- ✅ **Current branch** - Identify active branch
- ✅ **Ahead/behind** - Track divergence (future feature)

### API

```go
// List all branches
branches, err := repo.ListBranches(ctx)
for _, branch := range branches {
    current := ""
    if branch.IsCurrent {
        current = " *"
    }
    fmt.Printf("%s%s - %s (%s)\n",
        current,
        branch.Name,
        branch.ShortHash,
        branch.LastCommit.Message)
}

// Switch to existing branch
err := repo.SwitchBranch(ctx, "feature/new-api", false)

// Create and switch to new branch
err := repo.SwitchBranch(ctx, "feature/experimental", true)

// Merge branch into current
result, err := repo.MergeBranch(ctx, "feature/completed")
if result.Success {
    fmt.Printf("Merged %d files\n", len(result.MergedFiles))
} else {
    fmt.Printf("Conflicts in: %v\n", result.Conflicts)
}
```

### Use Cases

- **Branch management** - View and manage branches
- **Workflow automation** - Automated branch operations
- **AI-assisted merging** - Smart merge operations
- **IDE branch UI** - Display branch information

---

## Architecture

```
┌────────────────────────────────────────────────────────────────────┐
│                        Application Layer                            │
│  (CLI, IDE Extension, Web API, AI Coding Assistant, RAG System)   │
└────────────────────────────────────────────────────────────────────┘
                          │
         ┌────────────────┼──────────────────────────┐
         │                │                          │
         ▼                ▼                          ▼
┌─────────────────┐  ┌─────────────────┐  ┌──────────────────┐
│  SEARCH & AI    │  │  CODE MODIFY    │  │  GIT OPS         │
├─────────────────┤  ├─────────────────┤  ├──────────────────┤
│ • Keyword       │  │ • File Ops      │  │ • Diff/Blame     │
│ • Semantic      │  │ • Symbol        │  │ • Smart Commit   │
│ • Embeddings    │  │   Analysis      │  │ • Branch Mgmt    │
│ • Vector DB     │  │ • Security      │  │ • History        │
└─────────────────┘  └─────────────────┘  └──────────────────┘
         │                │                          │
         └────────────────┼──────────────────────────┘
                          ▼
        ┌──────────────────────────────────────┐
        │    ai.Repository (Unified Interface) │
        └──────────────────────────────────────┘
                          ▼
        ┌──────────────────────────────────────┐
        │    go-git (Git Operations Layer)     │
        └──────────────────────────────────────┘
                          ▼
        ┌──────────────────────────────────────┐
        │  External Services (Optional)         │
        │  • TEI (Embeddings)                   │
        │  • Chroma DB (Vector Store)           │
        └──────────────────────────────────────┘
```

### Components

**Search & AI:**
- **Keyword Search** - Pure Go, no dependencies
- **Semantic Search** - TEI + Chroma DB for vector similarity
- **Symbol Analysis** - Go AST parsing for code intelligence

**Code Modification:**
- **File Modifier** - Safe file operations with confirmation
- **Security Scanner** - Secret detection, dependency analysis

**Git Operations:**
- **Git Reader** - Diff, blame, history
- **Git Writer** - Smart commits, branch management

**Infrastructure:**
- **TEI (Text Embeddings Inference)** - HuggingFace embedding server
- **Chroma DB** - Vector database for semantic search
- **go-git** - Core Git repository access

---

## Configuration

### Environment Variables

```bash
# Embedding service (required for semantic search)
export AI_TEI_ENDPOINT=http://localhost:8081

# Vector database (required for semantic search)
export AI_CHROMA_URL=http://localhost:9001

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

## Feature Comparison

### Search Capabilities

| Feature | Keyword Search | Semantic Search | Symbol Analysis |
|---------|---------------|-----------------|-----------------|
| **Speed** | Instant (~50ms) | Fast (~300ms) | Fast (~100ms) |
| **Accuracy** | Exact matches | Meaning-based | Syntax-aware |
| **Setup** | None | TEI + Chroma | None |
| **Indexing** | Not needed | First-time index | Not needed |
| **Query Style** | Text/regex | Natural language | Symbol names |
| **Language** | Any | Any | Go only |
| **Use Case** | Text search | Concept search | Code navigation |

### Code Operations

| Feature | Availability | Safety | Use Case |
|---------|--------------|--------|----------|
| **File Modification** | ✅ Production | Confirmation required | AI code editors |
| **Git Diff/Blame** | ✅ Production | Read-only | Code review |
| **Smart Commits** | ✅ Production | Optional confirmation | Automated workflows |
| **Security Scan** | ✅ Production | Read-only | CI/CD gates |
| **Symbol Analysis** | ✅ Production | Read-only | IDE features |

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

#### Search Types

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
}

type IndexOptions struct {
    Incremental  bool
    Force        bool
    MaxStaleness time.Duration
    OnProgress   func(file string, current, total int)
    Silent       bool
}
```

#### Symbol Analysis Types

```go
type Symbol struct {
    Name       string
    Type       SymbolType // function, method, struct, interface, etc.
    FilePath   string
    Line       int
    Column     int
    Signature  string
    DocComment string
    Receiver   string // For methods
}

type Reference struct {
    FilePath string
    Line     int
    Column   int
    Context  string // Line of code containing reference
}

type GetSymbolsOptions struct {
    FilePath    string
    SymbolTypes []SymbolType
    IncludeDocs bool
}

type FindReferencesOptions struct {
    SymbolName string
    FilePath   string // Optional: limit to file
}
```

#### File Modification Types

```go
type FileOperation struct {
    Type        string // "edit", "create", "delete"
    FilePath    string
    OldContent  string
    NewContent  string
    Description string
    NeedsConfirm bool
}

type EditFileOptions struct {
    FilePath        string
    Content         string      // Replace entire file
    LineRanges      []LineEdit  // Or edit specific lines
    CreateIfMissing bool
}

type LineEdit struct {
    StartLine int    // 1-based
    EndLine   int    // 1-based, inclusive
    NewText   string
}

type CreateFileOptions struct {
    FilePath  string
    Content   string
    Overwrite bool
}

type DeleteFileOptions struct {
    FilePath string
}
```

#### Git Operations Types

```go
type DiffOptions struct {
    FromCommit string
    ToCommit   string // Empty = working tree
    FilePath   string // Optional: specific file
    MaxLines   int    // Truncate large diffs
}

type DiffResult struct {
    FilePath   string
    OldPath    string // For renames
    Status     string // Added, Deleted, Modified, Renamed
    Additions  int
    Deletions  int
    Diff       string // Unified diff format
}

type BlameOptions struct {
    FilePath string
    Commit   string // Optional: blame at specific commit
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
    FilePath string // Optional: filter by file
    Author   string // Optional: filter by author
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
    Message       string // Auto-generated if empty
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

type BranchOptions struct {
    Name       string
    FromCommit string // Empty = HEAD
    Checkout   bool
    Force      bool   // Overwrite if exists
}

type BranchResult struct {
    Name    string
    Hash    string
    Message string
}
```

#### Security Scanning Types

```go
type ScanOptions struct {
    FilePath        string
    CustomPatterns  []SecretPattern
    ExcludePatterns []string
}

type ScanResult struct {
    Findings         []SecretFinding
    FilesScanned     int
    SecretsFound     int
    HighSeverity     int
    MediumSeverity   int
    LowSeverity      int
}

type SecretFinding struct {
    Type        string
    Description string
    Severity    string // "high", "medium", "low"
    FilePath    string
    Line        int
    Column      int
    Match       string // Raw match
    Redacted    string // Safe to display
    Context     string
}

type DependencyInfo struct {
    Name    string
    Version string
    Type    string // "go", "npm", "python"
    Source  string // "go.mod", "package.json", etc.
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
