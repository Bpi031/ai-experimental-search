# Quick Reference — AI Features for go-git

## 🎯 Two Main Features

### 1. Search (Semantic + Keyword)
Find code using natural language or patterns

### 2. Edit with Timeline (like VSCode Copilot)
LLM modifies code, track changes, undo/redo

---

## 📦 Setup (One-time)

```bash
# Start Chroma vector database
docker run -p 8000:8000 chromadb/chroma

# Set OpenAI API key
export OPENAI_API_KEY="your-key-here"
```

---

## 🔍 Search APIs

### Semantic Search
```go
results, err := repo.AISemanticSearch(ctx, "JWT authentication logic")
// Returns code ranked by similarity (0-1 score)
```

### Keyword Search
```go
results, err := repo.AIKeywordSearch(ctx, AIKeywordOptions{
    Pattern:   "handleRequest",
    FileTypes: []string{".go"},
})
// Returns specific functions/patterns
```

---

## ✏️ Edit APIs

### Request Code Changes
```go
// Preview first (dry-run)
result, err := repo.AIEditCode(ctx, "Add logging to all HTTP handlers", &AIEditOptions{
    DryRun: true,
})
fmt.Println(result.Edits[0].Diff)  // review diff

// Apply changes
result, err = repo.AIEditCode(ctx, "Add logging to all HTTP handlers", &AIEditOptions{
    DryRun:     false,
    AutoCommit: true,  // optional git commit
})
```

### Timeline Management
```go
// Undo last edit
err := repo.AIUndoEdit(ctx)

// Redo edit
err := repo.AIRedoEdit(ctx)

// View history
timeline, err := repo.AIEditTimeline(ctx)
for i, entry := range timeline.Edits {
    fmt.Printf("%d. %s: %s\n", i+1, entry.Timestamp, entry.Request)
}
```

---

## 📊 Return Types

```go
// Search result
type SearchResult struct {
    FilePath  string
    Snippet   string
    Score     float64  // 0-1 similarity
    LineStart int
    LineEnd   int
}

// Code search result
type CodeSearchResult struct {
    FilePath     string
    FunctionName string
    LineStart    int
    CodeSnippet  string
    MatchType    string
}

// Edit result
type EditResult struct {
    Edits      []FileEdit
    Timeline   *EditTimeline
    CanUndo    bool
    CanRedo    bool
    CommitHash plumbing.Hash
}

// File edit
type FileEdit struct {
    FilePath    string
    OldContent  string
    NewContent  string
    Diff        string  // unified diff
    Description string  // LLM explanation
}
```

---

## 🔧 Configuration

```go
// Create provider
provider, err := git.NewLangChainProvider(
    os.Getenv("OPENAI_API_KEY"),
    "http://localhost:8000",  // Chroma URL
)

// Index repository (one-time, ~1 min for medium repo)
err = provider.IndexRepository(ctx, "/path/to/repo")

// Set provider on repository
repo.SetAIProvider(provider)
```

---

## 💾 Timeline Storage

Timeline automatically saved to `.git/ai-timeline.json`:

```json
{
  "edits": [
    {
      "id": "edit-123",
      "timestamp": "2025-10-09T10:30:00Z",
      "request": "Add logging",
      "files": [...],
      "commit_hash": "abc123"
    }
  ],
  "current": 0
}
```

Load/save manually:
```go
repo.AISaveTimeline(ctx, "/path/to/timeline.json")
repo.AILoadTimeline(ctx, "/path/to/timeline.json")
```

---

## 💰 Cost Estimates

| Operation | Cost (OpenAI) |
|-----------|---------------|
| Index repo (500 files) | ~$0.10 one-time |
| Semantic search | ~$0.001 per query |
| Code edit | ~$0.05 per edit |

Use local Chroma = $0 for vector storage!

---

## 🚀 Complete Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen(".")
    
    // Setup
    provider, _ := git.NewLangChainProvider(
        os.Getenv("OPENAI_API_KEY"),
        "http://localhost:8000",
    )
    provider.IndexRepository(ctx, ".")
    repo.SetAIProvider(provider)
    
    // Search
    results, _ := repo.AISemanticSearch(ctx, "error handling")
    fmt.Printf("Found %d results\n", len(results))
    
    // Edit
    edit, _ := repo.AIEditCode(ctx, "Add comments", &git.AIEditOptions{
        DryRun: true,
    })
    fmt.Println("Preview:", edit.Edits[0].Description)
    
    // Apply
    edit, _ = repo.AIEditCode(ctx, "Add comments", &git.AIEditOptions{
        DryRun: false,
        AutoCommit: true,
    })
    fmt.Println("Applied:", edit.CommitHash)
    
    // Undo if needed
    repo.AIUndoEdit(ctx)
}
```

---

## 📚 Documentation Files

- `AI_FEATURES_SUMMARY.md` - Overview
- `IMPLEMENTATION_PLAN.md` - Detailed plan
- `QUICK_REFERENCE.md` - This file
- `READMEDEV.md` - Developer guide

---

## ✅ Implementation Status

- [x] Design & planning
- [ ] Core infrastructure (Phase 1)
- [ ] Search functions (Phase 2)  
- [ ] Edit + timeline (Phase 3)
- [ ] Tests & examples (Phase 4)
- [ ] Documentation (Phase 5)

**Ready to implement!**
