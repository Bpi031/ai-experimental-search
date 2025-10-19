#!/bin/bash

# AI Services Manager for go-git AI features
# Manages TEI (Text Embeddings Inference) and Chroma Vector DB

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Determine compose command (prefer Docker Compose V2 "docker compose")
if docker compose version >/dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_CMD="docker-compose"
else
    echo "docker compose or docker-compose not found. Please install Docker Compose."
    echo "See: https://docs.docker.com/compose/install/"
    exit 127
fi

COMPOSE_FILE="docker-compose.local-ai.yml"
TEI_PORT=8081
CHROMA_PORT=8000

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker not found. Please install Docker first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon not running. Please start Docker."
        exit 1
    fi
    
    print_success "Docker is running"
}

check_nvidia() {
    if ! command -v nvidia-smi &> /dev/null; then
        print_warning "nvidia-smi not found. GPU may not be available."
        return 1
    fi
    
    print_info "Checking GPU..."
    nvidia-smi --query-gpu=name,memory.total --format=csv,noheader
    print_success "NVIDIA GPU detected"
    return 0
}

check_nvidia_docker() {
    if docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi &> /dev/null; then
        print_success "NVIDIA Docker runtime is working"
        return 0
    else
        print_warning "NVIDIA Docker runtime may not be configured properly"
        print_info "If you see GPU errors, install nvidia-container-toolkit:"
        print_info "  https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html"
        return 1
    fi
}

is_service_up() {
    local url=$1
    local timeout=${2:-2}
    
    if curl -s -f --max-time "$timeout" "$url" > /dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

start_services() {
    print_info "Starting AI services..."
    
    check_docker
    
    if check_nvidia; then
        check_nvidia_docker
    fi
    
    print_info "Pulling Docker images (may take a few minutes on first run)..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" pull
    
    print_info "Starting containers..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" up -d
    
    print_info "Waiting for services to be ready..."
    
    # Wait for TEI
    print_info "Waiting for TEI (embeddings) on port $TEI_PORT..."
    local count=0
    while ! is_service_up "http://localhost:$TEI_PORT/health" 3; do
        sleep 2
        count=$((count + 1))
        if [ $count -gt 60 ]; then
            print_error "TEI failed to start after 2 minutes"
            print_info "Check logs with: $COMPOSE_CMD -f $COMPOSE_FILE logs tei-embeddings"
            exit 1
        fi
        echo -n "."
    done
    echo ""
    print_success "TEI is ready on http://localhost:$TEI_PORT"
    
    # Wait for Chroma
    print_info "Waiting for Chroma (vector DB) on port $CHROMA_PORT..."
    count=0
    while ! is_service_up "http://localhost:$CHROMA_PORT/api/v1/heartbeat" 3; do
        sleep 2
        count=$((count + 1))
        if [ $count -gt 30 ]; then
            print_error "Chroma failed to start after 1 minute"
            print_info "Check logs with: $COMPOSE_CMD -f $COMPOSE_FILE logs chroma"
            exit 1
        fi
        echo -n "."
    done
    echo ""
    print_success "Chroma is ready on http://localhost:$CHROMA_PORT"
    
    print_success "All services are running!"
    echo ""
    status_services
}

stop_services() {
    print_info "Stopping AI services..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" stop
    print_success "Services stopped"
}

restart_services() {
    print_info "Restarting AI services..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" restart
    print_success "Services restarted"
}

status_services() {
    print_info "Service Status:"
    echo ""
    
    $COMPOSE_CMD -f "$COMPOSE_FILE" ps
    
    echo ""
    print_info "Health Checks:"
    
    if is_service_up "http://localhost:$TEI_PORT/health"; then
        print_success "TEI (embeddings): http://localhost:$TEI_PORT ✓"
    else
        print_error "TEI (embeddings): http://localhost:$TEI_PORT ✗"
    fi
    
    if is_service_up "http://localhost:$CHROMA_PORT/api/v1/heartbeat"; then
        print_success "Chroma (vector DB): http://localhost:$CHROMA_PORT ✓"
    else
        print_error "Chroma (vector DB): http://localhost:$CHROMA_PORT ✗"
    fi
}

logs_services() {
    local service=${1:-}
    if [ -z "$service" ]; then
        $COMPOSE_CMD -f "$COMPOSE_FILE" logs -f --tail=100
    else
        $COMPOSE_CMD -f "$COMPOSE_FILE" logs -f --tail=100 "$service"
    fi
}

test_services() {
    print_info "Testing AI services..."
    echo ""
    
    # Test TEI
    print_info "Testing TEI embeddings..."
    if curl -s -X POST "http://localhost:$TEI_PORT/embed" \
        -H "Content-Type: application/json" \
        -d '{"inputs":["test embedding"]}' | grep -q "embeddings"; then
        print_success "TEI is responding correctly"
    else
        print_error "TEI test failed"
    fi
    
    # Test Chroma
    print_info "Testing Chroma vector DB..."
    if curl -s "http://localhost:$CHROMA_PORT/api/v1/heartbeat" | grep -q "heartbeat"; then
        print_success "Chroma is responding correctly"
    else
        print_error "Chroma test failed"
    fi
    
    echo ""
    print_success "All tests passed! Ready to run Go tests."
    echo ""
    print_info "Run the AI tests with:"
    echo "  go test -v ./ai/..."
}

cleanup() {
    print_warning "This will remove all containers and volumes (data will be lost)"
    read -p "Are you sure? (y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Cleaning up..."
    $COMPOSE_CMD -f "$COMPOSE_FILE" down -v
        print_success "Cleanup complete"
    else
        print_info "Cleanup cancelled"
    fi
}

gpu_stats() {
    if command -v nvidia-smi &> /dev/null; then
        watch -n 1 nvidia-smi
    else
        print_error "nvidia-smi not found"
    fi
}

usage() {
    cat << EOF
${GREEN}AI Services Manager for go-git${NC}

${BLUE}Usage:${NC}
  $0 <command>

${BLUE}Commands:${NC}
  start       Start TEI and Chroma services
  stop        Stop services
  restart     Restart services
  status      Show service status
  logs        Show logs (add service name: tei-embeddings or chroma)
  test        Test services are working
  cleanup     Remove containers and volumes
  gpu         Monitor GPU usage (requires nvidia-smi)
  help        Show this help

${BLUE}Examples:${NC}
  $0 start              # Start all services
  $0 logs               # View all logs
  $0 logs tei-embeddings # View TEI logs only
  $0 test               # Test services
  $0 gpu                # Monitor GPU usage

${BLUE}After starting services:${NC}
  - TEI embeddings: http://localhost:$TEI_PORT
  - Chroma vector DB: http://localhost:$CHROMA_PORT
  - Run tests: go test -v ./ai/...

${BLUE}GPU Requirements:${NC}
  - RTX 4070: ~2GB VRAM for embeddings
  - RTX 4880: ~2GB VRAM for embeddings
EOF
}

# Main
case "${1:-}" in
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    status)
        status_services
        ;;
    logs)
        logs_services "${2:-}"
        ;;
    test)
        test_services
        ;;
    cleanup)
        cleanup
        ;;
    gpu)
        gpu_stats
        ;;
    help|--help|-h)
        usage
        ;;
    *)
        usage
        exit 1
        ;;
esac
