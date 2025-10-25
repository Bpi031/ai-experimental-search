# Quick Start - Local GPU Setup (RTX 5880)

**Setup time: 15 minutes | Zero API costs | Full privacy**

## Prerequisites

✅ RTX 5880 GPU (or any NVIDIA GPU with CUDA 12.2+)  
✅ Docker with NVIDIA Container Toolkit  
✅ 10GB+ free disk space  

## Step-by-Step Setup

### 1. Start Docker Services (2 minutes)

```bash
cd /Users/m4air/Documents/go-git

# Start embedding service + vector database
docker-compose -f docker-compose.local-ai.yml up -d

# Wait for services to start (check logs)
docker-compose -f docker-compose.local-ai.yml logs -f
# Press Ctrl+C when you see "Model loaded successfully"
```

### 2. Install Ollama (3 minutes)

```bash
# Install Ollama (if not already installed)
brew install ollama

# Start Ollama service (in background)
ollama serve > /dev/null 2>&1 &

# Download code model (~4GB, takes 2-3 minutes)
ollama pull qwen2.5-coder:7b
```

### 3. Verify Everything Works (1 minute)

```bash
# Test embedding service
curl http://localhost:9000/embed \
  -X POST \
  -d '{"inputs":"test code"}' \
  -H 'Content-Type: application/json'
# Should return: {"embeddings":[[0.123, -0.456, ...]]}

# Test vector database
curl http://localhost:8000/api/v1/heartbeat
# Should return: {"nanosecond heartbeat": ...}

# Test LLM
curl http://localhost:11434/api/tags
# Should show: qwen2.5-coder:7b model
```

### 4. Monitor GPU Usage (optional)

```bash
# Watch GPU memory usage in real-time
watch -n 1 nvidia-smi

# Expected usage:
# - TEI (embeddings): ~1-2GB VRAM
# - Ollama (when running): ~7-10GB VRAM
# - Total: ~10-12GB / 48GB (plenty left!)
```

## Usage in Go Code

Create `test_local_ai.go`:

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    
    // Open your repository
    repo, err := git.PlainOpen(".")
    if err != nil {
        log.Fatal(err)
    }
    
    // Create LOCAL AI provider (no API keys!)
    provider, err := git.NewLocalLangChainProvider(git.LocalProviderOptions{
        TEIEndpoint:   "http://localhost:9000",
        ChromaURL:     "http://localhost:8000",
        OllamaURL:     "http://localhost:11434",
        OllamaModel:   "qwen2.5-coder:7b",
        Namespace:     "my-repo",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Index repository (one-time, uses local GPU)
    fmt.Println("Indexing repository with local GPU...")
    err = provider.IndexRepository(ctx, ".")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Indexing complete!")
    
    // Set provider
    repo.SetAIProvider(provider)
    
    // === NOW USE IT ===
    
    // Semantic search (local GPU)
    fmt.Println("\nSearching for 'JWT authentication'...")
    results, err := repo.AISemanticSearch(ctx, "JWT authentication")
    if err != nil {
        log.Fatal(err)
    }
    
    for i, r := range results {
        fmt.Printf("%d. %s (score: %.2f)\n", i+1, r.FilePath, r.Score)
        fmt.Printf("   %s\n", r.Snippet[:80])
    }
    
    // AI code editing (local LLM)
    fmt.Println("\nRequesting code edit...")
    edit, err := repo.AIEditCode(ctx, "Add error handling", &git.AIEditOptions{
        TargetFiles: []string{"*.go"},
        DryRun: true,  // preview first
    })
    if err != nil {
        log.Fatal(err)
    }
    
    for _, e := range edit.Edits {
        fmt.Printf("Edit: %s\n%s\n", e.FilePath, e.Description)
    }
}
```

Run it:
```bash
go run test_local_ai.go
```

## Expected Performance

| Operation | Local GPU (RTX 5880) | Cloud API |
|-----------|---------------------|-----------|
| **Embedding 1000 tokens** | ~5-10ms | ~100-200ms |
| **Semantic search** | ~50-100ms | ~500-800ms |
| **Index 500 files** | ~45 seconds | ~2-3 minutes |
| **LLM edit (200 tokens)** | ~3-5 seconds | ~5-10 seconds |

**Your local setup is 5-10x faster than cloud APIs!** ⚡️

## Cost Comparison

### Cloud (per month)
- Embeddings (OpenAI): ~$20-50
- LLM edits (GPT-4): ~$100-300
- **Total: $120-350/month** 💸

### Local (per month)
- Electricity (~300W GPU): ~$5-10
- API costs: $0
- **Total: $5-10/month** 🎉

**You save ~$110-340/month!**

## Troubleshooting

### Services not starting?

```bash
# Check Docker logs
docker-compose -f docker-compose.local-ai.yml logs

# Restart services
docker-compose -f docker-compose.local-ai.yml restart
```

### GPU not detected?

```bash
# Check NVIDIA drivers
nvidia-smi

# Check Docker GPU access
docker run --rm --gpus all nvidia/cuda:12.2.0-base-ubuntu22.04 nvidia-smi

# If fails, install NVIDIA Container Toolkit:
# https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html
```

### Ollama not responding?

```bash
# Check if running
curl http://localhost:11434/api/tags

# If not, restart Ollama
pkill ollama
ollama serve > /dev/null 2>&1 &
```

### Out of VRAM?

```bash
# Check VRAM usage
nvidia-smi

# Option 1: Use smaller Ollama model
ollama pull qwen2.5-coder:1.5b  # Uses ~3GB instead of ~7GB

# Option 2: Use smaller embedding model
# Edit docker-compose.local-ai.yml:
# Change: --model-id=jinaai/jina-embeddings-v2-base-code
# Restart: docker-compose -f docker-compose.local-ai.yml restart tei-embeddings
```

## Stop Services

```bash
# Stop Docker services (keeps data)
docker-compose -f docker-compose.local-ai.yml stop

# Or completely remove (deletes data)
docker-compose -f docker-compose.local-ai.yml down -v

# Stop Ollama
pkill ollama
```

## Next Steps

1. ✅ Services running? → Test with example code above
2. ✅ Example works? → Index your actual repositories
3. ✅ Ready for more? → See `IMPLEMENTATION_PLAN.md` for full API

## Tips

💡 **Startup time**: First time takes ~2-3 min to download models, then 10-30 seconds  
💡 **Index once**: After indexing, search is instant (~50ms)  
💡 **Batch operations**: Index multiple repos overnight for instant search later  
💡 **Privacy**: All code stays on your machine, never sent to cloud  
💡 **Offline**: Works without internet (after initial model downloads)  

## Alternative Models

### Faster Embeddings (for speed)
```bash
# Edit docker-compose.local-ai.yml, change to:
# --model-id=jinaai/jina-embeddings-v2-base-code

docker-compose -f docker-compose.local-ai.yml restart tei-embeddings
```

### Better Quality Embeddings (for accuracy)
```bash
# Edit docker-compose.local-ai.yml, change to:
# --model-id=Alibaba-NLP/gte-large-en-v1.5

docker-compose -f docker-compose.local-ai.yml restart tei-embeddings
```

### Smaller LLM (for speed)
```bash
ollama pull qwen2.5-coder:1.5b
# Update OllamaModel in your Go code
```

### Larger LLM (for quality)
```bash
ollama pull qwen2.5-coder:32b  # Needs ~20GB VRAM
# Update OllamaModel in your Go code
```

See `LOCAL_GPU_SETUP.md` for detailed model comparisons!

---

**You're all set!** 🚀 Enjoy blazing fast, private, local AI code search and editing!
