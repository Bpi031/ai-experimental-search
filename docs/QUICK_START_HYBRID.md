# Quick Start - Hybrid Setup (Local GPU + Cloud LLM)

**Setup time: 10 minutes | Best of both worlds: Fast local search + Quality cloud LLMs**

## Why Hybrid?

✅ **10x faster search** - Local GPU embeddings  
✅ **Private embeddings** - Code vectors stay local  
✅ **Best LLMs** - Use GPT-4, Claude, Gemini  
✅ **Cost effective** - Save $70-200/month vs full cloud  
✅ **Minimal VRAM** - Only ~2GB vs 10-20GB for local LLM  

## Prerequisites

✅ RTX 5880 GPU (or any NVIDIA GPU with CUDA 12.2+)  
✅ Docker with NVIDIA Container Toolkit  
✅ API key from OpenAI, Google, or Anthropic  

## Step-by-Step Setup

### 1. Get Cloud LLM API Key (3 minutes)

**Choose ONE of these:**

**Option A: OpenAI GPT-4** (Best overall)
```bash
# Get key: https://platform.openai.com/api-keys
export OPENAI_API_KEY="sk-..."
echo 'export OPENAI_API_KEY="sk-..."' >> ~/.zshrc
```

**Option B: Google Gemini** (Good quality, cheaper)
```bash
# Get key: https://makersuite.google.com/app/apikey
export GOOGLE_API_KEY="..."
echo 'export GOOGLE_API_KEY="..."' >> ~/.zshrc
```

**Option C: Anthropic Claude** (Best reasoning)
```bash
# Get key: https://console.anthropic.com/
export ANTHROPIC_API_KEY="sk-ant-..."
echo 'export ANTHROPIC_API_KEY="sk-ant-..."' >> ~/.zshrc
```

### 2. Start Local Services (2 minutes)

```bash
cd /Users/m4air/Documents/go-git

# Start embedding service + vector database
docker-compose -f docker-compose.local-ai.yml up -d

# Wait for services (check logs)
docker-compose -f docker-compose.local-ai.yml logs -f
# Press Ctrl+C when you see "Model loaded successfully"
```

### 3. Verify Everything Works (1 minute)

```bash
# Test local embedding service
curl http://localhost:8081/embed \
  -X POST \
  -d '{"inputs":"test code"}' \
  -H 'Content-Type: application/json'
# Should return: {"embeddings":[[0.123, -0.456, ...]]}

# Test local vector database
curl http://localhost:8000/api/v1/heartbeat
# Should return: {"nanosecond heartbeat": ...}

# Test cloud LLM (OpenAI example)
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
# Should return: List of models including gpt-4
```

### 4. Monitor GPU Usage (optional)

```bash
# Watch GPU memory usage
watch -n 1 nvidia-smi

# Expected usage:
# - TEI (embeddings): ~1-2GB VRAM
# - No LLM (runs in cloud)
# - Total: ~1-2GB / 48GB (minimal!)
```

## Usage in Go Code

### Example 1: With OpenAI GPT-4

Create `test_hybrid.go`:

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
    
    // Create HYBRID provider (local GPU + GPT-4)
    provider, err := git.NewHybridWithOpenAI()
    if err != nil {
        log.Fatal(err)
    }
    
    // Index repository (uses local GPU - FAST!)
    fmt.Println("Indexing with local GPU...")
    err = provider.IndexRepository(ctx, ".")
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Indexing complete!")
    
    // Set provider
    repo.SetAIProvider(provider)
    
    // === SEARCH: LOCAL GPU (fast + private) ===
    
    fmt.Println("\nSearching for 'JWT authentication'...")
    results, err := repo.AISemanticSearch(ctx, "JWT authentication")
    if err != nil {
        log.Fatal(err)
    }
    
    for i, r := range results {
        fmt.Printf("%d. %s (score: %.2f)\n", i+1, r.FilePath, r.Score)
        fmt.Printf("   %s\n", r.Snippet[:80])
    }
    
    // === EDIT: CLOUD GPT-4 (best quality) ===
    
    fmt.Println("\nRequesting code edit with GPT-4...")
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
go run test_hybrid.go
```

### Example 2: With Google Gemini

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen(".")
    
    // Use Gemini Pro (cheaper than GPT-4)
    provider, _ := git.NewHybridWithGemini()
    provider.IndexRepository(ctx, ".")
    repo.SetAIProvider(provider)
    
    // Search: local GPU (fast)
    results, _ := repo.AISemanticSearch(ctx, "database queries")
    
    // Edit: Gemini Pro (cloud)
    edit, _ := repo.AIEditCode(ctx, "Optimize database queries", nil)
}
```

### Example 3: With Anthropic Claude

```go
package main

import (
    "context"
    "github.com/go-git/go-git/v6"
)

func main() {
    ctx := context.Background()
    repo, _ := git.PlainOpen(".")
    
    // Use Claude 3 (best reasoning)
    provider, _ := git.NewHybridWithClaude()
    provider.IndexRepository(ctx, ".")
    repo.SetAIProvider(provider)
    
    // Search: local GPU (fast + private)
    results, _ := repo.AISemanticSearch(ctx, "error handling")
    
    // Edit: Claude 3 (cloud, excellent reasoning)
    edit, _ := repo.AIEditCode(ctx, "Refactor error handling", nil)
}
```

## Performance Comparison

| Operation | Hybrid | Full Cloud | Full Local |
|-----------|--------|------------|------------|
| **Search** | 50-100ms | 500-800ms | 50-100ms |
| **Embedding** | 5-10ms | 100-200ms | 5-10ms |
| **LLM Quality** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| **VRAM Usage** | ~2GB | 0GB | ~10-20GB |
| **Privacy** | Embeddings local | ❌ All cloud | ✅ All local |

**Hybrid gives you the best of both worlds!** ⚡️

## Cost Comparison (per month)

| Setup | Embeddings | Vector DB | LLM | Electricity | Total |
|-------|-----------|-----------|-----|-------------|-------|
| **Full Cloud** | $20-50 | $0-70 | $100-300 | $0 | **$120-420** 💸 |
| **Hybrid** | $0 (local) | $0 (local) | $50-150 | $3-5 | **$53-155** 💰 |
| **Full Local** | $0 (local) | $0 (local) | $0 (local) | $5-10 | **$5-10** 🎉 |

**Hybrid saves ~$65-265/month vs full cloud!**

**Why Hybrid Wins:**
- ✅ Search is fast (local GPU)
- ✅ Search is private (embeddings stay local)
- ✅ LLM is best quality (GPT-4, Claude, Gemini)
- ✅ LLM doesn't use VRAM (cloud-based)
- ✅ Cost effective (pay only for LLM)

## Troubleshooting

### Local services not starting?

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

### API key not working?

```bash
# Check if environment variable is set
echo $OPENAI_API_KEY

# Re-export if needed
export OPENAI_API_KEY="sk-..."

# Test the API
curl https://api.openai.com/v1/models \
  -H "Authorization: Bearer $OPENAI_API_KEY"
```

### Want to switch LLM providers?

```go
// Just change the provider in your code:

// From OpenAI:
provider, _ := git.NewHybridWithOpenAI()

// To Gemini:
provider, _ := git.NewHybridWithGemini()

// To Claude:
provider, _ := git.NewHybridWithClaude()
```

## Stop Services

```bash
# Stop Docker services (keeps data)
docker-compose -f docker-compose.local-ai.yml stop

# Or completely remove (deletes data)
docker-compose -f docker-compose.local-ai.yml down -v
```

## LLM Comparison

| Provider | Model | Quality | Speed | Cost (per 1M tokens) |
|----------|-------|---------|-------|---------------------|
| **OpenAI** | GPT-4 Turbo | ⭐⭐⭐⭐⭐ | Fast | $10 / $30 |
| **OpenAI** | GPT-3.5 Turbo | ⭐⭐⭐⭐ | Very Fast | $0.50 / $1.50 |
| **Google** | Gemini Pro | ⭐⭐⭐⭐ | Fast | $0.50 / $1.50 |
| **Anthropic** | Claude 3 Opus | ⭐⭐⭐⭐⭐ | Medium | $15 / $75 |
| **Anthropic** | Claude 3 Sonnet | ⭐⭐⭐⭐ | Fast | $3 / $15 |

**Recommendation for code:**
- **Best quality**: Claude 3 Opus or GPT-4 Turbo
- **Best value**: Gemini Pro or Claude 3 Sonnet
- **Fastest**: GPT-3.5 Turbo

## Next Steps

1. ✅ Services running? → Test with example code above
2. ✅ Example works? → Index your actual repositories
3. ✅ Want more control? → See `LOCAL_GPU_SETUP.md` for advanced options

## Tips

💡 **Privacy**: Your code embeddings never leave your machine during search  
💡 **LLM calls**: Only code snippets sent to LLM for editing (you can review first with DryRun)  
💡 **Batch indexing**: Index multiple repos overnight for instant search  
💡 **Switch providers**: Easy to change from GPT-4 → Gemini → Claude  
💡 **Cost control**: Use DryRun to preview edits before paying for LLM calls  

## Comparison with Full Local

**When to use Hybrid (Local GPU + Cloud LLM):**
- ✅ You want the best LLM quality (GPT-4, Claude 3)
- ✅ You don't want to manage large LLM models
- ✅ You have limited VRAM (< 24GB)
- ✅ You're okay with API costs (~$50-150/month)

**When to use Full Local:**
- ✅ You need 100% offline capability
- ✅ You have plenty of VRAM (48GB+)
- ✅ You want zero API costs
- ✅ You're okay with slightly lower LLM quality

**Hybrid is recommended for most users!** 🎯

---

**You're all set!** 🚀 Enjoy fast local search with the best cloud LLMs!

For more details, see:
- `LOCAL_GPU_SETUP.md` - Complete technical guide
- `IMPLEMENTATION_PLAN.md` - Full implementation details
- `AI_FEATURES_SUMMARY.md` - Feature overview
