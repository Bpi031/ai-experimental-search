# Implementation Plan: AI Features for go-git

## Goal

Add two AI-powered features to go-git using LangChain Go:

1. **Search Functions** (Semantic + Keyword)
   - Semantic search using vector embeddings
    - Keyword/code pattern search
    - Semantic uses vector DB (embeddings). Keyword is local (regex + symbols) with optional embedding rerank

2. **LLM Code Editor with Timeline**
   - LLM suggests code changes
   - Timeline tracks all edits (like VSCode Copilot)
   - Undo/redo support
   - Git integration for commits

## Architecture

```
┌─────────────────────────────────────────────────────┐
│              Repository (go-git)                     │
│  ┌─────────────────────────────────────────────┐   │
│  │  New AI Methods:                             │   │
│  │  • AISemanticSearch(query)                   │   │
│  │  │  • AIKeywordSearch(pattern)                │   │
│  │  • AIEditCode(request, opts)                 │   │
│  │  • AIUndoEdit() / AIRedoEdit()               │   │
│  └─────────────────────────────────────────────┘   │
│              ↓                                       │
│  ┌─────────────────────────────────────────────┐   │
│  │  LangChainProvider (AIProvider)              │   │
│  │  • VectorStore (Chroma/Qdrant/Pinecone)     │   │
│  │  • Embedder (OpenAI/Hugging Face)           │   │
│  │  • LLM (GPT-4/Claude/Gemini)                │   │
│  │  • EditTimeline (undo/redo stack)           │   │
│  │  • LocalKeywordEngine (regex + symbols)     │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```

## Feature 1: Search Functions

### 1.1 Semantic Search

**Use case:** "Find code that handles JWT authentication"

**Implementation:**
```go
// Semantic search using vector embeddings
results, err := repo.AISemanticSearch(ctx, "JWT authentication logic")

// Returns:
type SearchResult struct {
    FilePath    string
    Snippet     string
    Score       float64  // similarity score 0-1
    LineStart   int
    LineEnd     int
}
```

**How it works:**
1. Index repository files into vector store (one-time)
2. User query converted to embedding
3. Vector store finds similar code chunks
4. Return ranked results with scores

### 1.2 Keyword/Code Search

**Use case:** "Find all functions named `handleRequest`" or "Find error handling patterns"

**Implementation:**
```go
// Code-specific search with filters
results, err := repo.AIKeywordSearch(ctx, AIKeywordOptions{
    Pattern:      "handleRequest",
    FileTypes:    []string{".go", ".js"},
    IncludeDocs:  true,
})

// Returns:
type CodeSearchResult struct {
    FilePath     string
    FunctionName string
    LineStart    int
    LineEnd      int
    CodeSnippet  string
    MatchType    string  // "function", "variable", "comment", etc.
}
```

**How it works (Copilot-style):**
1. Local text scan with regex/globs respecting .gitignore (ripgrep-like)
2. Optional symbol extraction for supported languages (e.g., Go via go/parser) to classify matches (function/variable/comment)
3. Assemble snippets with line ranges and small context windows
4. Optional: rerank top-N candidates using local embeddings (TEI) + vector similarity to the query
5. Return structured results with file path, match type, lines, and snippet

Notes:
- Keyword search does not require a vector DB; it’s fast and fully local
- Rerank step is optional and uses your local GPU embeddings for quality

## Feature 2: LLM Code Editor with Timeline

### 2.1 Code Editing

**Use case:** "Add error handling to all database queries"

**Implementation:**
```go
// Request code changes from LLM
result, err := repo.AIEditCode(ctx, "Add error handling to all DB queries", &AIEditOptions{
    TargetFiles: []string{"db/*.go"}, // storage into go-git storer.go
    DryRun:      false,  // preview before applying
    AutoCommit:  false,  // manual commit control
})

// Returns:
type EditResult struct {
    Edits        []FileEdit
    Timeline     *EditTimeline
    CanUndo      bool
    CanRedo      bool
    CommitHash   plumbing.Hash  // if AutoCommit=true
}

type FileEdit struct {
    FilePath    string
    OldContent  string
    NewContent  string
    Diff        string  // unified diff
    Description string  // LLM explanation
}
```

### 2.2 Timeline & Undo/Redo

**Like VSCode Copilot's edit timeline:**

```go
// Timeline tracks all edits
type EditTimeline struct {
    Edits   []EditEntry
    Current int  // current position in timeline
}

type EditEntry struct {
    ID          string
    Timestamp   time.Time
    Request     string  // original LLM request
    Files       []FileEdit
    CommitHash  plumbing.Hash  // if committed
}

// Undo last edit
err := repo.AIUndoEdit(ctx)

// Redo edit
err := repo.AIRedoEdit(ctx)

// View timeline
timeline, err := repo.AIEditTimeline(ctx)
for _, entry := range timeline.Edits {
    fmt.Printf("%s: %s (%d files)\n", 
        entry.Timestamp, entry.Request, len(entry.Files))
}
```

**How it works:**
1. Each edit stored in timeline (in-memory + optional persistence)
2. Undo: revert file changes, move timeline pointer back
3. Redo: re-apply changes, move pointer forward
4. Timeline can be saved to `.git/ai-timeline.json`
5. Optionally create git commits for each edit

## Implementation Steps

### Phase 1: Core Infrastructure (~3 hours)

**Files to create:**
1. `ai_provider.go` - AIProvider interface
2. `ai_provider_langchain.go` - LangChain implementation
3. `ai_search.go` - Search methods on Repository
4. `ai_edit.go` - Edit methods on Repository
5. `ai_timeline.go` - Timeline tracking

**Key tasks:**
- [ ] Define AIProvider interface
- [ ] Create LangChainProvider struct
- [ ] Implement IndexRepository() method
- [ ] Add AI methods to Repository struct

### Phase 2: Search Functions (~4 hours)

**Semantic Search:**
- [ ] Implement `AISemanticSearch()` using vector store
- [ ] Add score threshold filtering
- [ ] Return ranked results with snippets

**Keyword Search:**
- [ ] Implement `AIKeywordSearch()` as local scan (regex) respecting .gitignore
- [ ] Add metadata-based filtering (file types, size limits, exclude paths)
- [ ] Extract function/variable names for Go using go/parser; fallback to regex for other languages
- [ ] Optional: add embedding-based rerank for top-N results using local TEI

### Phase 3: Edit Functions (~5 hours)

**LLM Editing:**
- [ ] Implement `AIEditCode()` with LLM integration
- [ ] Add dry-run preview
- [ ] Generate diffs for review
- [ ] Apply changes to worktree

**Timeline:**
- [ ] Create EditTimeline struct
- [ ] Implement undo/redo logic
- [ ] Persist timeline to disk
- [ ] Integrate with git commits

### Phase 4: Testing & Examples (~3 hours)

- [ ] Unit tests for search functions
- [ ] Unit tests for edit functions
- [ ] Integration test with Chroma
- [ ] Example: Semantic search demo
- [ ] Example: LLM code editor demo

### Phase 5: Documentation (~2 hours)

- [ ] API documentation
- [ ] Usage examples
- [ ] Setup instructions (Chroma Docker, API keys)
- [ ] Update READMEDEV.md

## File Structure

```
go-git/
├── ai_provider.go              # AIProvider interface (100 lines)
├── ai_provider_langchain.go    # LangChain implementation (300 lines)
├── ai_search.go                # Search methods (200 lines)
├── ai_edit.go                  # Edit methods (250 lines)
├── ai_timeline.go              # Timeline tracking (150 lines)
├── ai_test.go                  # Tests (300 lines)
├── _examples/
│   ├── ai-search/
│   │   └── main.go             # Search demo
│   └── ai-edit/
│       └── main.go             # Edit demo
└── IMPLEMENTATION_PLAN.md      # This file
```

**Total estimated code: ~1,300 lines**

## Dependencies

Add to `go.mod`:
```go
require (
    github.com/tmc/langchaingo v0.1.13
    // LangChain already includes:
    // - Vector stores (Chroma, Qdrant, etc.)
    // - Embeddings (OpenAI, Hugging Face, or custom HTTP endpoint)
    // - LLMs (OpenAI, Anthropic, Ollama, etc.)
)
```

**Note:** For local GPU setup, LangChain Go supports:
- Custom embeddings endpoints (TEI via HTTP)
- Ollama for local LLMs
- All major vector stores work with local embeddings

## Quick Start Setup

### Cloud Setup (OpenAI + Chroma)
```bash
# Start Chroma
docker run -p 8000:8000 chromadb/chroma
```

### Local GPU Setup (RTX 5880) - **RECOMMENDED**
```bash
# See LOCAL_GPU_SETUP.md for full guide

# Start local embedding service (GPU accelerated)
docker run -d --gpus all -p 9000:80 \
  -v ~/text-embeddings-models:/data \
  ghcr.io/huggingface/text-embeddings-inference:89-1.8 \
  --model-id nomic-ai/nomic-embed-text-v1.5

# Start local vector DB
docker run -d -p 8000:8000 -v ~/chroma-data:/chroma/chroma chromadb/chroma

```

### 2. Use in Code

**Option A: Cloud Setup (with API keys)**
```go
package main

import (
    "context"
    "fmt"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    
    // Open repo
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Create AI provider (LangChain + OpenAI)
    provider, _ := git.NewLangChainProvider(
        "your-openai-key",
        "http://localhost:8000",
    )
    
    // Index repository (one-time, ~1 min for medium repo)
    fmt.Println("Indexing repository...")
    provider.IndexRepository(ctx, "/path/to/repo")
    
    // Set provider
    repo.SetAIProvider(provider)
}
```

**Option B: Local GPU Setup (NO API keys needed!)**
```go
package main

import (
    "context"
    "fmt"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    
    // Open repo
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Create LOCAL AI provider (uses your RTX 5880!)
    provider, _ := git.NewLocalLangChainProvider(git.LocalProviderOptions{})
    
    // Index repository (uses local GPU)
    fmt.Println("Indexing with local GPU...")
    provider.IndexRepository(ctx, "/path/to/repo")
    
    // Set provider
    repo.SetAIProvider(provider)
}
```

### 3. Use the Features (same for both cloud and local!)

```go
// === SEARCH FUNCTIONS ===

// 1. Semantic search
results, _ := repo.AISemanticSearch(ctx, "JWT authentication")
for _, r := range results {
    fmt.Printf("Found in %s (%.2f): %s\n", r.FilePath, r.Score, r.Snippet)
}

// 2. Keyword search
codeResults, _ := repo.AIKeywordSearch(ctx, git.AIKeywordOptions{
    Pattern: "handleRequest",
    FileTypes: []string{".go"}, // local regex + optional symbol extraction
})
for _, r := range codeResults {
    fmt.Printf("Function %s in %s at line %d\n", 
        r.FunctionName, r.FilePath, r.LineStart)
}

// === EDIT FUNCTIONS ===

// 3. LLM code editing
editResult, _ := repo.AIEditCode(ctx, "Add logging to all HTTP handlers", &git.AIEditOptions{
    TargetFiles: []string{"handlers/*.go"},
    DryRun: true,  // preview first
})

// Review edits
for _, edit := range editResult.Edits {
    fmt.Printf("Will edit %s: %s\n", edit.FilePath, edit.Description)
    fmt.Println(edit.Diff)
}

// Apply if satisfied
if userConfirms() {
    editResult, _ = repo.AIEditCode(ctx, "Add logging to all HTTP handlers", &git.AIEditOptions{
        TargetFiles: []string{"handlers/*.go"},
        DryRun: false,
        AutoCommit: true,
    })
    fmt.Printf("Changes committed: %s\n", editResult.CommitHash)
}

// 4. Undo last edit
repo.AIUndoEdit(ctx)

// 5. View timeline
timeline, _ := repo.AIEditTimeline(ctx)
for i, entry := range timeline.Edits {
    fmt.Printf("%d. %s: %s (%d files)\n", 
        i+1, entry.Timestamp, entry.Request, len(entry.Files))
}
```

## API Summary

### Search APIs
```go
// Semantic search using embeddings
AISemanticSearch(ctx, query string) ([]SearchResult, error)

// Keyword/code pattern search
AIKeywordSearch(ctx, opts AIKeywordOptions) ([]CodeSearchResult, error)
```

### Edit APIs
```go
// Request LLM code changes
AIEditCode(ctx, request string, opts *AIEditOptions) (*EditResult, error)

// Undo last edit
AIUndoEdit(ctx) error

// Redo edit
AIRedoEdit(ctx) error

// View edit timeline
AIEditTimeline(ctx) (*EditTimeline, error)

// Save timeline to disk
AISaveTimeline(ctx, path string) error

// Load timeline from disk
AILoadTimeline(ctx, path string) error
```

## Options Structs

```go
type AIKeywordOptions struct {
    Pattern      string
    FileTypes    []string  // e.g., [".go", ".js"]
    IncludeDocs  bool      // include comments/docs in results
    MaxResults   int       // default 20
    ExcludePaths []string  // e.g., vendor/, node_modules/, dist/
    MaxFileSize  int64     // bytes; skip giant files (default: 1MB)
    Rerank       bool      // if true, rerank top-N using embeddings
}

type AIEditOptions struct {
    TargetFiles  []string  // files to edit (glob patterns)
    DryRun       bool      // preview without applying
    AutoCommit   bool      // auto-commit changes
    CommitMsg    string    // custom commit message
}
```

## Timeline Storage Format

Timeline saved to `.git/ai-timeline.json`:
```json
{
  "edits": [
    {
      "id": "edit-1234567890",
      "timestamp": "2025-10-09T10:30:00Z",
      "request": "Add error handling to all DB queries",
      "files": [
        {
          "path": "db/users.go",
          "old_content": "...",
          "new_content": "...",
          "diff": "...",
          "description": "Added error wrapping"
        }
      ],
      "commit_hash": "abc123..."
    }
  ],
  "current": 0
}
```

## Copilot Parity Notes (Keyword Search)

- VS Code Copilot relies on local workspace search (ripgrep) and language server symbol indexes for keyword and structural queries; it does not require a vector database for keyword search
- Our design mirrors this: fast local scan + optional symbol parsing for higher precision; vector embeddings are only used for semantic search or optional reranking
- Benefits: predictable latency, no network cost for keyword queries, respects .gitignore by default

## Next Steps

1. **Review this plan** - Confirm approach is correct
2. **Start implementation** - Begin with Phase 1 (core infrastructure)
3. **Test incrementally** - Test each phase before moving to next
4. **Create examples** - Working demos for documentation

**Estimated total time: 17 hours of implementation**

Ready to start? I can implement Phase 1 right now!
