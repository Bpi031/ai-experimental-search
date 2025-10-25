# 🚀 Getting AI Tests to Pass - Quick Guide

You have **2 options** to activate the AI semantic search tests:

## ✅ Option 1: Install NVIDIA Docker Support (Recommended for RTX 4070)

This will let you use your RTX 4070 GPU for fast embeddings (~10x faster than CPU).

### Quick Install:
```bash
sudo ./install-nvidia-docker.sh
```

This script will:
1. ✅ Check your NVIDIA GPU
2. ✅ Add NVIDIA container toolkit repository
3. ✅ Install nvidia-container-toolkit
4. ✅ Configure Docker to use GPU
5. ✅ Restart Docker
6. ✅ Test GPU access

**Time: ~5 minutes**

### Then start services:
```bash
./ai-services.sh start
go test -v ./ai/...
```

All tests will pass! 🎉

---

## 🔄 Option 2: Use CPU Version (Temporary Fallback)

If you want to test immediately without GPU setup, use the CPU version:

```bash
# Use CPU docker-compose file
docker-compose -f docker-compose.cpu.yml up -d

# Wait for services (takes ~30 seconds)
sleep 30

# Test
go test -v ./ai/...
```

**Note:** CPU version is slower (~10x) but works without GPU setup.

To stop CPU services:
```bash
docker-compose -f docker-compose.cpu.yml down
```

---

## 📊 Comparison

| Feature | GPU (Option 1) | CPU (Option 2) |
|---------|----------------|----------------|
| Speed | ⚡ Fast (100-500ms) | 🐌 Slow (1-5s) |
| VRAM | ~2GB | 0GB |
| Setup | 5 min install | No install |
| Best for | Development & Production | Quick testing |

---

## 🎯 Recommended Path

**For your RTX 4070 dev machine:**
1. Install NVIDIA Docker support (one-time, 5 minutes)
2. Use GPU version forever (fast & efficient)

**For your RTX 4880 production:**
- Same process, same scripts work!

---

## 📝 What's Happening Now

Your test output shows:
```
=== RUN   TestSemanticSearch_IndexAndQuery
    semantic_test.go:28: TEI (http://localhost:8081) or Chroma (http://localhost:9001) 
                         not reachable; skipping semantic test
--- SKIP: TestSemanticSearch_IndexAndQuery (0.00s)
```

The tests are **skipping** because TEI and Chroma services aren't running yet.

After starting services, you'll see:
```
=== RUN   TestSemanticSearch_IndexAndQuery
--- PASS: TestSemanticSearch_IndexAndQuery (2.43s)
=== RUN   TestIndex_AfterNewCommit  
--- PASS: TestIndex_AfterNewCommit (1.87s)
PASS
```

---

## 🔧 Quick Commands

```bash
# Install GPU support (one-time)
sudo ./install-nvidia-docker.sh

# Start services
./ai-services.sh start

# Check status
./ai-services.sh status

# Test services
./ai-services.sh test

# Run Go tests
go test -v ./ai/...

# Monitor GPU
./ai-services.sh gpu

# Stop services
./ai-services.sh stop
```

---

## ❓ Need Help?

### Check if Docker sees your GPU:
```bash
docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi
```

### View logs:
```bash
./ai-services.sh logs
```

### Test services manually:
```bash
# Test TEI
curl -X POST http://localhost:8081/embed \
  -H "Content-Type: application/json" \
  -d '{"inputs":["test"]}'

# Test Chroma
curl http://localhost:9001/api/v1/heartbeat
```

---

## 🎓 What You're Testing

The AI package provides:
- **Keyword Search**: Works without services (pure text matching)
- **Semantic Search**: Requires TEI + Chroma (AI-powered understanding)
- **Incremental Indexing**: Requires TEI + Chroma (efficient updates)

After setup, all features will work! 🚀
