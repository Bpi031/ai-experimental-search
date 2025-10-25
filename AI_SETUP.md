# AI Services Setup Guide

This guide helps you set up and run the AI services (TEI embeddings + Chroma vector DB) required for the go-git AI features.

## 🎯 Quick Start

### 1. Start Services

```bash
./ai-services.sh start
```

This will:
- ✅ Check Docker and GPU availability
- ✅ Pull required images (~2-3GB on first run)
- ✅ Start TEI (embeddings) on port 8081
- ✅ Start Chroma (vector DB) on port 9001
- ✅ Wait for services to be ready

### 2. Test Services

```bash
./ai-services.sh test
```

### 3. Run AI Tests

```bash
go test -v ./ai/...
```

Now all tests should pass! 🎉

## 📋 Prerequisites

### Required
- ✅ Docker (with docker-compose)
- ✅ NVIDIA GPU (RTX 4070, RTX 4880, or similar)
- ✅ NVIDIA Docker runtime (nvidia-container-toolkit)

### Installation (if needed)

#### Install Docker
```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
# Log out and back in
```

#### Install NVIDIA Container Toolkit
```bash
# Ubuntu/Debian
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | sudo tee /etc/apt/sources.list.d/nvidia-docker.list

sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker
```

## 🔧 Available Commands

```bash
./ai-services.sh start      # Start all services
./ai-services.sh stop       # Stop services
./ai-services.sh restart    # Restart services
./ai-services.sh status     # Check service status
./ai-services.sh logs       # View all logs
./ai-services.sh logs tei-embeddings  # View TEI logs only
./ai-services.sh test       # Test services are working
./ai-services.sh gpu        # Monitor GPU usage
./ai-services.sh cleanup    # Remove containers and data
./ai-services.sh help       # Show help
```

## 📊 GPU Usage

Expected VRAM usage:
- **TEI (embeddings)**: ~1-2GB
- **Total**: ~2GB / 8GB (RTX 4070) or 48GB (RTX 4880)

Monitor GPU in real-time:
```bash
./ai-services.sh gpu
# or
watch -n 1 nvidia-smi
```

## 🧪 Test Coverage

### Tests that work without services:
- ✅ `TestKeywordSearch_Basic` - Pure text search (no AI required)

### Tests that require services:
- 🔄 `TestSemanticSearch_IndexAndQuery` - Requires TEI + Chroma
- 🔄 `TestIndex_AfterNewCommit` - Requires TEI + Chroma

After starting services, all tests should pass!

## 🐛 Troubleshooting

### Services won't start

**Check Docker:**
```bash
docker info
```

**Check GPU:**
```bash
nvidia-smi
```

**Check NVIDIA Docker runtime:**
```bash
docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi
```

### Port already in use

If ports 9000 or 9001 are in use:
```bash
# Find what's using the ports
sudo lsof -i :8081
sudo lsof -i :9001

# Stop other services or modify docker-compose.local-ai.yml
```

### TEI fails to start

**View logs:**
```bash
./ai-services.sh logs tei-embeddings
```

**Common issues:**
- Insufficient GPU memory (need ~2GB free)
- NVIDIA driver not installed
- Docker can't access GPU

### Chroma fails to start

**View logs:**
```bash
./ai-services.sh logs chroma
```

**Reset Chroma data:**
```bash
./ai-services.sh cleanup
./ai-services.sh start
```

## 🔄 Daily Workflow

```bash
# Start services (once per day or after reboot)
./ai-services.sh start

# Check status anytime
./ai-services.sh status

# Run tests
go test -v ./ai/...

# View logs if issues
./ai-services.sh logs

# Stop when done
./ai-services.sh stop
```

## 📁 File Structure

```
.
├── ai-services.sh                    # Service management script
├── docker-compose.local-ai.yml       # GPU-enabled services
├── docker-compose.cpu.yml            # CPU fallback (slower)
└── ai/                               # AI package
    ├── keyword_test.go               # ✓ Works without services
    ├── semantic_test.go              # ⚠ Requires services
    └── index_incremental_test.go     # ⚠ Requires services
```

## 🚀 Production Deployment

For RTX 4880 production server, same commands work:

```bash
# On production server
./ai-services.sh start
./ai-services.sh test

# Services will auto-restart on reboot
```

The docker-compose file includes `restart: unless-stopped` for automatic restarts.

## 📚 Learn More

- [Quick Start - Local GPU](docs/QUICK_START_LOCAL.md)
- [Quick Start - Hybrid (Local GPU + Cloud LLM)](docs/QUICK_START_HYBRID.md)
- [AI Features Summary](docs/AI_FEATURES_SUMMARY.md)

## ❓ Need Help?

1. Check service status: `./ai-services.sh status`
2. View logs: `./ai-services.sh logs`
3. Test services: `./ai-services.sh test`
4. Check GPU: `./ai-services.sh gpu`
