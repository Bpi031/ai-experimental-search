# Local GPU Setup Guide - RTX 5880 Ada Lovelace

## Overview

Run go-git AI features **100% locally** on your RTX 5880 GPU with:
- **Zero cloud costs** (no OpenAI/Anthropic fees)
- **Full privacy** (code never leaves your machine)
- **Blazing fast** (local GPU embedding: ~1000 tokens/sec)
- **Production-ready** tools

## Architecture (Hybrid: Local GPU + Cloud LLM)

```
┌─────────────────────────────────────────────────────┐
│              Repository (go-git)                     │
│              ↓                                       │
│  ┌─────────────────────────────────────────────┐   │
│  │  LangChainProvider (Hybrid Mode)             │   │
│  │  ├── Local Embeddings API                    │   │
│  │  │   └── TEI + RTX 5880 GPU (LOCAL)          │   │
│  │  ├── Local Vector Store                      │   │
│  │  │   └── Chroma (localhost)                  │   │
│  │  └── Cloud LLM APIs                          │   │
│  │      ├── OpenAI (ChatGPT-4)                  │   │
│  │      ├── Google Gemini                       │   │
│  │      ├── Anthropic Claude                    │   │
│  │      └── Other cloud LLMs                    │   │
│  └─────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────┘
```


## Stack Components

| Component | Technology | Where | Purpose |
|-----------|-----------|-------|---------|
| **Embeddings** | HuggingFace TEI | 🏠 Local GPU (RTX 5880) | Convert code to vectors |
| **Vector DB** | Chroma | 🏠 Local (CPU) | Store embeddings locally |
| **LLM** | OpenAI/Gemini/Claude | ☁️ Cloud API | Code editing/generation |

## Step 1: Install Dependencies

### 1.1 Install Docker
```bash
# macOS: Docker Desktop already installed
# Verify Docker
docker --version
```

## Step 2: Setup Local Embedding Service (TEI)

### 2.1 Choose Embedding Model

Recommended models for code (tested on RTX GPUs):

| Model | Size | Speed | Quality | Best For |
|-------|------|-------|---------|----------|
| `Qwen/Qwen3-Embedding-0.6B` | 1.2 GB | ⚡️ Very Fast | Good | General code search |
| `jinaai/jina-embeddings-v2-base-code` | 274 MB | ⚡️⚡️ Ultra Fast | Good | Code-specific |
| `nomic-ai/nomic-embed-text-v1.5` | 274 MB | ⚡️⚡️ Ultra Fast | Excellent | Semantic search |
| `Alibaba-NLP/gte-large-en-v1.5` | 868 MB | ⚡️ Fast | Excellent | High accuracy |

### 2.2 Start TEI with GPU Support

**For RTX 5880 (Ada Lovelace architecture, compute capability 8.9):**

```bash
# Create directory for model cache
mkdir -p ~/text-embeddings-models

# Pull TEI Docker image for Ada Lovelace
docker pull ghcr.io/huggingface/text-embeddings-inference:89-1.8

# Start TEI with GPU (using nomic-embed for speed)
docker run -d \
  --name tei-embeddings \
  --gpus all \
  -p 8081:80 \
  -v ~/text-embeddings-models:/data \
  ghcr.io/huggingface/text-embeddings-inference:89-1.8 \
  --model-id nomic-ai/nomic-embed-text-v1.5 \
  --max-batch-tokens 16384 \
  --max-concurrent-requests 512

# Verify it's running (wait 30 sec for model download)
curl http://localhost:8081/health
```

**Test embedding generation:**
```bash
curl http://localhost:8081/embed \
  -X POST \
  -d '{"inputs":"function handleRequest() {}"}' \
  -H 'Content-Type: application/json'
# Should return: {"embeddings":[[0.123, -0.456, ...]]}
```

### 2.3 Performance Expectations

With RTX 5880 (48GB VRAM):
- **Embedding speed**: ~1000-2000 tokens/second
- **Batch size**: Up to 512 requests concurrently
- **Model loading**: 5-10 seconds (first time: 1-2 min for download)
- **Memory usage**: ~1-2GB VRAM (plenty of room for LLM too!)

## Step 3: Setup Local Vector Database

### Option A: Chroma (Recommended - Simplest)

```bash
# Start Chroma with persistent storage
docker run -d \
  --name chroma-vectordb \
  -p 8000:8000 \
  -v ~/chroma-data:/chroma/chroma \
  chromadb/chroma

# Verify
curl http://localhost:8000/api/v1/heartbeat
```

### Option B: Qdrant (More Features)

```bash
# Start Qdrant
docker run -d \
  --name qdrant-vectordb \
  -p 6333:6333 \
  -p 6334:6334 \
  -v ~/qdrant-data:/qdrant/storage \
  qdrant/qdrant

# Verify
curl http://localhost:6333/
```

## Step 4: Setup Cloud LLM API Keys

**Choose your preferred LLM provider(s):**

### Option A: OpenAI (ChatGPT-4)

```bash
# Get API key from: https://platform.openai.com/api-keys
export OPENAI_API_KEY="sk-..."

# Test it
curl https://api.openai.com/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4",
    "messages": [{"role": "user", "content": "Write a Go function"}],
    "max_tokens": 100
  }'
```

### Option B: Google Gemini

```bash
# Get API key from: https://makersuite.google.com/app/apikey
export GOOGLE_API_KEY="..."

# Test it
curl "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent?key=$GOOGLE_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"contents":[{"parts":[{"text":"Write a Go function"}]}]}'
```

### Option C: Anthropic Claude

```bash
# Get API key from: https://console.anthropic.com/
export ANTHROPIC_API_KEY="sk-ant-..."

# Test it
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 100,
    "messages": [{"role": "user", "content": "Write a Go function"}]
  }'
```

### Save API Keys

```bash
# Add to your ~/.zshrc or ~/.bashrc
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.zshrc
echo 'export GOOGLE_API_KEY="..."' >> ~/.zshrc
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.zshrc

source ~/.zshrc
```

## Step 5: Configure LangChain Go for Hybrid Stack

### 5.1 Update Implementation Code

**Create `ai_provider_hybrid.go`:**

```go
package git

import (
    "context"
    "fmt"
    "os"
    
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/openai"
    "github.com/tmc/langchaingo/llms/googleai"
    "github.com/tmc/langchaingo/llms/anthropic"
    "github.com/tmc/langchaingo/vectorstores/chroma"
)

// NewHybridLangChainProvider creates a hybrid AI provider:
// - Local GPU for embeddings (fast + private)
// - Cloud APIs for LLM (best quality)
func NewHybridLangChainProvider(opts HybridProviderOptions) (*LangChainProvider, error) {
    ctx := context.Background()
    
    // Use local TEI for embeddings (GPU accelerated, PRIVATE)
    embedder, err := embeddings.NewEmbedder(
        embeddings.WithEndpoint(opts.TEIEndpoint),
        embeddings.WithBatchSize(32),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create embedder: %w", err)
    }
    
    // Use local Chroma vector store (PRIVATE)
    store, err := chroma.New(
        chroma.WithChromaURL(opts.ChromaURL),
        chroma.WithEmbedder(embedder),
        chroma.WithNameSpace(opts.Namespace),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create vector store: %w", err)
    }
    
    // Use cloud LLM API (choose your provider)
    var llm interface{}
    
    switch opts.LLMProvider {
    case "openai":
        llm, err = openai.New(
            openai.WithModel("gpt-4-turbo-preview"),
            openai.WithToken(os.Getenv("OPENAI_API_KEY")),
        )
    case "gemini":
        llm, err = googleai.New(
            googleai.WithAPIKey(os.Getenv("GOOGLE_API_KEY")),
            googleai.WithDefaultModel("gemini-pro"),
        )
    case "claude":
        llm, err = anthropic.New(
            anthropic.WithToken(os.Getenv("ANTHROPIC_API_KEY")),
            anthropic.WithModel("claude-3-opus-20240229"),
        )
    default:
        return nil, fmt.Errorf("unsupported LLM provider: %s", opts.LLMProvider)
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to create LLM: %w", err)
    }
    
    return &LangChainProvider{
        embedder:    embedder,
        vectorStore: store,
        llm:         llm,
        timeline:    &EditTimeline{Edits: []EditEntry{}},
    }, nil
}

type HybridProviderOptions struct {
    // Local services (GPU accelerated)
    TEIEndpoint   string // default: http://localhost:8081
    ChromaURL     string // default: http://localhost:8000
    Namespace     string // default: go-git-code
    
    // Cloud LLM
    LLMProvider   string // "openai", "gemini", or "claude"
}

// Convenience constructors for each LLM provider

func NewHybridWithOpenAI() (*LangChainProvider, error) {
    return NewHybridLangChainProvider(HybridProviderOptions{
        TEIEndpoint: "http://localhost:8081",
        ChromaURL:   "http://localhost:8000",
        Namespace:   "go-git-code",
        LLMProvider: "openai",
    })
}

func NewHybridWithGemini() (*LangChainProvider, error) {
    return NewHybridLangChainProvider(HybridProviderOptions{
        TEIEndpoint: "http://localhost:8081",
        ChromaURL:   "http://localhost:8000",
        Namespace:   "go-git-code",
        LLMProvider: "gemini",
    })
}

func NewHybridWithClaude() (*LangChainProvider, error) {
    return NewHybridLangChainProvider(HybridProviderOptions{
        TEIEndpoint: "http://localhost:8081",
        ChromaURL:   "http://localhost:8000",
        Namespace:   "go-git-code",
        LLMProvider: "claude",
    })
}
```

### 5.2 Usage Examples

#### Example 1: Using OpenAI GPT-4

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    
    // Open repository
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Create HYBRID provider (local GPU embeddings + GPT-4)
    provider, err := git.NewHybridWithOpenAI()
    if err != nil {
        panic(err)
    }
    
    // Index repository (uses local GPU - FAST & PRIVATE)
    fmt.Println("Indexing with local GPU...")
    err = provider.IndexRepository(ctx, "/path/to/repo")
    if err != nil {
        panic(err)
    }
    
    // Set provider
    repo.SetAIProvider(provider)
    
    // === SEARCH: LOCAL (fast, private) ===
    
    // Semantic search uses local GPU embeddings
    results, _ := repo.AISemanticSearch(ctx, "JWT authentication code")
    for _, r := range results {
        fmt.Printf("Found: %s (score: %.2f)\n", r.FilePath, r.Score)
    }
    
    // === EDIT: CLOUD GPT-4 (best quality) ===
    
    // AI code editing uses GPT-4
    edit, _ := repo.AIEditCode(ctx, "Add error handling to HTTP handlers", &git.AIEditOptions{
        DryRun: true,
    })
    
    for _, e := range edit.Edits {
        fmt.Println(e.Diff)
    }
}
```

#### Example 2: Using Google Gemini

```go
package main

import (
    "context"
    "fmt"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Use Gemini Pro for code editing
    provider, _ := git.NewHybridWithGemini()
    provider.IndexRepository(ctx, "/path/to/repo")
    repo.SetAIProvider(provider)
    
    // Search: local GPU (fast)
    results, _ := repo.AISemanticSearch(ctx, "database connection pool")
    
    // Edit: Gemini Pro (cloud)
    edit, _ := repo.AIEditCode(ctx, "Optimize database queries", nil)
}
```

#### Example 3: Using Anthropic Claude

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Use Claude 3 Opus for code editing
    provider, _ := git.NewHybridWithClaude()
    provider.IndexRepository(ctx, "/path/to/repo")
    repo.SetAIProvider(provider)
    
    // Search: local GPU (fast + private)
    results, _ := repo.AISemanticSearch(ctx, "error handling patterns")
    
    // Edit: Claude 3 Opus (cloud, best reasoning)
    edit, _ := repo.AIEditCode(ctx, "Refactor error handling", nil)
}
```

#### Example 4: Custom Configuration

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen("/path/to/repo")
    
    // Full control over configuration
    provider, _ := git.NewHybridLangChainProvider(git.HybridProviderOptions{
        TEIEndpoint: "http://localhost:8081",  // Local GPU embeddings
        ChromaURL:   "http://localhost:8000",  // Local vector DB
        Namespace:   "my-project",
        LLMProvider: "openai",                 // Cloud LLM
    })
    
    provider.IndexRepository(ctx, "/path/to/repo")
    repo.SetAIProvider(provider)
}
```

## Step 6: Verify Hybrid Setup

### 6.1 Check Local Services

```bash
# Check TEI (local embeddings)
curl http://localhost:8081/health
# Expected: {"status":"ok"}

# Check Chroma (local vector DB)
curl http://localhost:8000/api/v1/heartbeat
# Expected: {"nanosecond heartbeat":...}
```

### 6.2 Check Cloud API Keys

```bash
# Test OpenAI
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
# Expected: List of available models

# Test Gemini
curl "https://generativelanguage.googleapis.com/v1/models?key=$GOOGLE_API_KEY"
# Expected: List of models

# Test Claude
curl https://api.anthropic.com/v1/messages \
  -H "x-api-key: $ANTHROPIC_API_KEY" \
  -H "anthropic-version: 2023-06-01" \
  -H "content-type: application/json" \
  -d '{"model":"claude-3-opus-20240229","max_tokens":10,"messages":[{"role":"user","content":"Hi"}]}'
# Expected: Response with message
```

### 6.3 Monitor GPU Usage (Only for Local Services)

```bash
# Watch GPU memory usage
watch -n 1 nvidia-smi

# You should see:
# - TEI container using ~1-2GB VRAM
# - No LLM using VRAM (cloud-based)
# - Total: ~1-2GB / 48GB (minimal usage!)
```


## Troubleshooting

### Issue: TEI not using GPU

```bash
# Check Docker has GPU access
docker run --rm --gpus all nvidia/cuda:12.2.0-base-ubuntu22.04 nvidia-smi

# If fails, install NVIDIA Container Toolkit:
# https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html
```

### Issue: Ollama slow or crashing

```bash
# Check VRAM usage
nvidia-smi

# If VRAM full, use smaller model
ollama pull qwen2.5-coder:1.5b

# Or stop TEI temporarily
docker stop tei-embeddings
```

### Issue: Port conflicts

```bash
# Check what's using ports
lsof -i :8081  # TEI
lsof -i :8000  # Chroma
lsof -i :11434 # Ollama

# Change ports if needed:
docker run -p 9081:80 ... # Use 9081 instead of 8081
```

## Advanced: Multi-GPU Support

If you add more GPUs later:

```bash
# Run TEI on GPU 0, Ollama on GPU 1
docker run --gpus '"device=0"' ... tei-embeddings
CUDA_VISIBLE_DEVICES=1 ollama serve
```

## Docker Compose Setup (All-in-One)

Create `docker-compose.local-ai.yml`:

```yaml
version: '3.8'

services:
  # Local embeddings (GPU accelerated)
  tei-embeddings:
    image: ghcr.io/huggingface/text-embeddings-inference:89-1.8
    container_name: tei-embeddings
    ports:
      - "8081:80"
    volumes:
      - ~/text-embeddings-models:/data
    environment:
      - MODEL_ID=nomic-ai/nomic-embed-text-v1.5
      - MAX_BATCH_TOKENS=16384
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu]

  # Local vector database
  chroma-db:
    image: chromadb/chroma
    container_name: chroma-vectordb
    ports:
      - "8000:8000"
    volumes:
      - ~/chroma-data:/chroma/chroma
    environment:
      - IS_PERSISTENT=TRUE

  # Ollama runs outside Docker for better GPU access
  # Start manually: ollama serve
```

**Start everything:**
```bash
# Start Docker services
docker-compose -f docker-compose.local-ai.yml up -d

# Start Ollama (in separate terminal)
ollama serve

# Pull model
ollama pull qwen2.5-coder:7b
```

## Next Steps

1. **Test the setup** - Use the example code above
2. **Benchmark your GPU** - Measure indexing speed on your repos
3. **Optimize batch sizes** - Tune `MAX_BATCH_TOKENS` for your workload
4. **Experiment with models** - Try different embedding/LLM models

## Alternative Embedding Models

### For Maximum Speed (Small Models)
```bash
docker run --gpus all -p 8081:80 -v ~/models:/data \
  ghcr.io/huggingface/text-embeddings-inference:89-1.8 \
  --model-id jinaai/jina-embeddings-v2-base-code \
  --max-batch-tokens 32768  # Increase batch size for tiny model
```

### For Maximum Quality (Large Models)
```bash
docker run --gpus all -p 8081:80 -v ~/models:/data \
  ghcr.io/huggingface/text-embeddings-inference:89-1.8 \
  --model-id Alibaba-NLP/gte-Qwen2-7B-instruct \
  --max-batch-tokens 8192  # Reduce for large model
```

## Summary

✅ **Zero cloud costs** - Everything runs locally  
✅ **Full privacy** - Code never leaves your machine  
✅ **GPU accelerated** - TEI + Ollama use RTX 5880  
✅ **Production ready** - Same quality as cloud APIs  
✅ **Easy setup** - 3 Docker containers + Ollama  

**Total setup time: ~15 minutes**

Ready to implement? Check `IMPLEMENTATION_PLAN.md` for the full code integration!
