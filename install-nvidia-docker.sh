#!/bin/bash

# NVIDIA Container Toolkit Installation Script for Debian/Ubuntu
# This enables Docker to use your NVIDIA GPU

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}  NVIDIA Container Toolkit Installer${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    print_error "This script needs sudo privileges"
    echo "Please run: sudo $0"
    exit 1
fi

# Check for NVIDIA driver
if ! command -v nvidia-smi &> /dev/null; then
    print_error "NVIDIA driver not found. Please install NVIDIA driver first."
    exit 1
fi

print_success "NVIDIA driver detected:"
nvidia-smi --query-gpu=name,driver_version --format=csv,noheader
echo ""

# Detect distribution
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS=$ID
    VER=$VERSION_ID
else
    print_error "Cannot detect OS distribution"
    exit 1
fi

print_info "Detected OS: $OS $VER"
echo ""

add_repo() {
    # Remove any broken file first
    rm -f /etc/apt/sources.list.d/nvidia-container-toolkit.list || true

    # Add GPG key
    print_info "Adding NVIDIA GPG key..."
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
      | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg

    # Try distribution-specific repo first (e.g., ubuntu22.04, debian12)
    distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
    print_info "Trying distro-specific repo: $distribution"
    if curl -fsSL "https://nvidia.github.io/libnvidia-container/${distribution}/libnvidia-container.list" \
        | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
        | tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null; then
        # Validate the file begins with 'deb '
        if head -n1 /etc/apt/sources.list.d/nvidia-container-toolkit.list | grep -q '^deb '; then
            print_success "Added distro-specific NVIDIA repo"
            return 0
        fi
    fi

    # Fallback to stable repo (works for most Debian/Ubuntu)
    print_warning "Falling back to stable repo"
    curl -fsSL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
      | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
      | tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null

    # Validate file content
    if ! head -n1 /etc/apt/sources.list.d/nvidia-container-toolkit.list | grep -q '^deb '; then
        print_error "Failed to configure NVIDIA repo (unexpected content)"
        print_info "Open the file to inspect: /etc/apt/sources.list.d/nvidia-container-toolkit.list"
        exit 1
    fi
    print_success "Added stable NVIDIA repo"
}

# Install based on distribution
case $OS in
    debian|ubuntu)
        print_info "Installing for Debian/Ubuntu..."
        add_repo
        
        # Update and install
        print_info "Updating package list..."
        apt-get update
        
        print_info "Installing nvidia-container-toolkit..."
        apt-get install -y nvidia-container-toolkit
        
        print_success "Installation complete!"
        ;;
        
    *)
        print_error "Unsupported OS: $OS"
        print_info "Please visit: https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html"
        exit 1
        ;;
esac

echo ""
print_info "Configuring Docker to use NVIDIA runtime..."
nvidia-ctk runtime configure --runtime=docker
print_success "Docker configured"

echo ""
print_info "Restarting Docker daemon..."
systemctl restart docker
print_success "Docker restarted"

echo ""
print_info "Testing NVIDIA Docker integration..."
if docker run --rm --gpus all nvidia/cuda:12.0.0-base-ubuntu22.04 nvidia-smi; then
    echo ""
    print_success "✓ NVIDIA Container Toolkit is working!"
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}  Installation Successful!${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════${NC}"
    echo ""
    print_info "You can now start the AI services:"
    echo "  ./ai-services.sh start"
else
    print_error "Test failed. Please check the logs above."
    exit 1
fi
