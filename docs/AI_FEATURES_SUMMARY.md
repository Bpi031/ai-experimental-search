# AI Features Summary — go-git

## What We're Building

Two AI-powered features for go-git using LangChain Go:

### 1. **Search Functions** 
- **Semantic Search** - "Find JWT authentication code" → finds relevant code by meaning
- **Keyword Search** - "Find all `handleRequest` functions" → finds specific patterns

### 2. **LLM Code Editor with Timeline** (like VSCode Copilot)
- **AI Edits** - "Add logging to all handlers" → LLM modifies code
- **Timeline** - Track all edits with timestamps
- **Undo/Redo** - Step backward/forward through edit history
- **Git Integration** - Optionally commit each edit

## Quick Start

### Setup Option A: Cloud (OpenAI) - 5 minutes

```bash
# 1. Start vector database (Chroma)
docker run -p 9001:9001 chromadb/chroma

# 2. Install dependencies
cd /path/to/your/go/project
go get github.com/tmc/langchaingo@v0.1.13
go get github.com/go-git/go-git/v6
```

### Setup Option B: Local GPU (RTX 5880) - 15 minutes ⚡️

**Zero API costs! Full privacy! See `LOCAL_GPU_SETUP.md` for details.**

```bash
# 1. Start local embedding service (GPU accelerated)
docker run -d --gpus all -p 9000:80 \
  -v ~/text-embeddings-models:/data \
  ghcr.io/huggingface/text-embeddings-inference:89-1.8 \
  --model-id nomic-ai/nomic-embed-text-v1.5

# 2. Start vector database
docker run -d -p 9001:9001 -v ~/chroma-data:/chroma/chroma chromadb/chroma

# 3. Install and start Ollama (local LLM)
brew install ollama
ollama serve &
ollama pull qwen2.5-coder:7b

# 4. Install dependencies
go get github.com/tmc/langchaingo@v0.1.13
go get github.com/go-git/go-git/v6
```

### Usage Example (Cloud)

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Setup AI provider (Cloud)
    provider, _ := git.NewLangChainProvider(
        "your-openai-api-key",
        "http://localhost:9001",  // Chroma URL
    )
    repo.SetAIProvider(provider)
    
    // Index repository (one-time)
    provider.IndexRepository(ctx, "/path/to/repo")
    
    // ... use features below
}
```

### Usage Example (Local GPU - Recommended!)

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Setup LOCAL AI provider (NO API keys!)
    provider, _ := git.NewLocalLangChainProvider(git.LocalProviderOptions{})
    repo.SetAIProvider(provider)
    
    // Index repository with local GPU (fast!)
    provider.IndexRepository(ctx, "/path/to/repo")
    
    // ... use features below (same API!)
}
```

### Common Features (works with both cloud and local)

```go
// 1. Semantic search
results, _ := repo.AISemanticSearch(ctx, "JWT authentication")
    for _, r := range results {
        fmt.Printf("%.2f: %s (%s)\n", r.Score, r.FilePath, r.Snippet[:50])
    }
    
    // 2. Keyword search
    code, _ := repo.AIKeywordSearch(ctx, git.AIKeywordOptions{
        Pattern: "handleRequest",
        FileTypes: []string{".go"},
    })
    
    // 3. AI edit code
    edit, _ := repo.AIEditCode(ctx, "Add logging to handlers", &git.AIEditOptions{
        DryRun: true,  // preview first
    })
    fmt.Println(edit.Edits[0].Diff)  // review changes
    
    // 4. Apply edit (if satisfied)
    repo.AIEditCode(ctx, "Add logging", &git.AIEditOptions{
        DryRun: false,
        AutoCommit: true,
    })
    
    // 5. Undo if needed
    repo.AIUndoEdit(ctx)
}
```

## Architecture

```
Repository (go-git)
    ↓
LangChainProvider
    ├── Vector Store (Chroma/Qdrant/Pinecone)
    ├── Embedder (OpenAI/Hugging Face)
    ├── LLM (GPT-4/Claude)
    └── EditTimeline (undo/redo)
```

## Key Features

### Search
✅ Semantic search using vector embeddings
✅ Keyword/pattern search with metadata filters
✅ File type filtering (`.go`, `.js`, etc.)
✅ Score-based ranking
✅ Code snippet extraction

### Editing
✅ LLM-powered code modifications
✅ Dry-run preview mode
✅ Diff generation
✅ Timeline tracking (all edits saved)
✅ Undo/redo functionality
✅ Optional auto-commit
✅ Timeline persistence (`.git/ai-timeline.json`)

## Implementation Status

See `IMPLEMENTATION_PLAN.md` for detailed plan.

**Phases:**
1. ✅ Design complete
2. ⏳ Core infrastructure (next)
3. ⏳ Search functions
4. ⏳ Edit functions + timeline
5. ⏳ Tests & examples
6. ⏳ Documentation

**Estimated completion: 17 hours**

## Files

- `IMPLEMENTATION_PLAN.md` - Detailed implementation plan
- `docs/archive/` - Old design documents (for reference)
- Implementation files (to be created):
  - `ai_provider.go`
  - `ai_provider_langchain.go`
  - `ai_search.go`
  - `ai_edit.go`
  - `ai_timeline.go`

## Dependencies

**LangChain Go** (v0.1.13) provides:
- 10+ vector stores (Chroma, Qdrant, Pinecone, etc.)
- Multiple embeddings providers (OpenAI, Hugging Face, etc.)
- LLM integrations (GPT-4, Claude, Gemini, Ollama)
- Document loaders and text splitters
- Production-ready with 7.8k ⭐

## Cost Estimates

### Cloud Setup (OpenAI)

**Per repository indexing (one-time):**
- Medium repo (~500 files): ~$0.10 in OpenAI API costs
- Large repo (~5000 files): ~$1.00 in API costs

**Per search query:** ~$0.001 (very cheap)

**Per code edit:** ~$0.02 - $0.10 (depends on code size)

**Vector store:**
- Chroma (local Docker): Free
- Qdrant Cloud: Free tier available
- Pinecone: Free tier (1M vectors)

**Monthly (medium usage): ~$50-200**

### Local GPU Setup (RTX 5880) ⚡️

**Hardware:** $0 (you already have it!)

**Electricity:** ~$5-10/month (24/7 running)

**API costs:** $0 (zero!)

**Total:** ~$5-10/month 🎉

**Break-even vs cloud: 1 month**

## Next Steps

Ready to implement? Let me know and I'll start with Phase 1 (core infrastructure).

The implementation will create:
- Working search functions (semantic + keyword)
- LLM code editor with undo/redo
- Complete examples
- Tests
- Documentation

Total: ~1,300 lines of code across 5-6 files.
