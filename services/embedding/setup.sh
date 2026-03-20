#!/bin/bash

# Embedding Service Setup Script with uv
# This script helps set up the embedding service environment with Ollama and uv

set -e

echo "=========================================="
echo "Embedding Service Setup (uv + Ollama)"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Check if running on macOS
OS="$(uname -s)"
if [[ "$OS" != "Darwin" && "$OS" != "Linux" ]]; then
    print_error "Unsupported operating system: $OS"
    exit 1
fi

# Check if uv is installed
print_info "Checking for uv installation..."
if command -v uv &> /dev/null; then
    UV_VERSION=$(uv --version)
    print_success "uv is installed: $UV_VERSION"
else
    print_warning "uv is not installed"
    echo ""
    print_info "Installing uv (fast Python package manager)..."
    curl -LsSf https://astral.sh/uv/install.sh | sh
    print_success "uv installed successfully"
    
    # Add uv to PATH for current session
    export PATH="$HOME/.local/bin:$PATH"
fi

echo ""
print_info "Checking for Python..."
if command -v python3 &> /dev/null; then
    PYTHON_VERSION=$(python3 --version)
    print_success "Python is installed: $PYTHON_VERSION"
else
    print_error "Python 3 is not installed"
    exit 1
fi

echo ""
print_info "Checking pyproject.toml..."
if [ ! -f "$SCRIPT_DIR/pyproject.toml" ]; then
    print_error "pyproject.toml not found"
    exit 1
fi
print_success "pyproject.toml found"

echo ""
print_info "Setting up Python virtual environment with uv..."
cd "$SCRIPT_DIR"

if [ ! -d ".venv" ]; then
    uv venv
    print_success "Virtual environment created with uv"
else
    print_success "Virtual environment already exists"
fi

echo ""
print_info "Syncing dependencies with uv..."
uv sync
print_success "Dependencies synced"

echo ""
print_info "Checking for Ollama installation..."
if command -v ollama &> /dev/null; then
    print_success "Ollama is already installed"
    OLLAMA_VERSION=$(ollama --version)
    echo "  Version: $OLLAMA_VERSION"
else
    print_warning "Ollama is not installed"
    echo ""
    print_info "To install Ollama on macOS:"
    echo "  curl -fsSL https://ollama.com/install.sh | sh"
    echo ""
    print_info "To install Ollama on Linux:"
    echo "  curl -fsSL https://ollama.com/install.sh | sh"
    echo ""
    read -p "Would you like to install Ollama now? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "Installing Ollama..."
        curl -fsSL https://ollama.com/install.sh | sh
        print_success "Ollama installed successfully"
    else
        print_warning "Ollama is required for the embedding service (if using Ollama backend)"
        echo "You can install it later with: curl -fsSL https://ollama.com/install.sh | sh"
    fi
fi

echo ""
print_info "Downloading embedding model..."
MODEL_NAME="hf.co/ChristianAzinn/e5-large-v2-gguf:Q8_0"

# Check if ollama is available before trying to pull model
if command -v ollama &> /dev/null; then
    # Check if model is already pulled
    if ollama list 2>/dev/null | grep -q "e5-large-v2"; then
        print_success "Model e5-large-v2 is already available"
    else
        print_info "Pulling model: $MODEL_NAME"
        ollama pull "$MODEL_NAME"
        print_success "Model downloaded successfully"
    fi
else
    print_warning "Ollama is not available. Model will be downloaded when you install Ollama."
fi

echo ""
print_info "Creating environment configuration..."
if [ ! -f "$SCRIPT_DIR/.env" ]; then
    if [ -f "$SCRIPT_DIR/.env.example" ]; then
        cp "$SCRIPT_DIR/.env.example" "$SCRIPT_DIR/.env"
        print_success "Environment configuration created from .env.example"
    else
        cat > "$SCRIPT_DIR/.env" << 'EOF'
# Backend Configuration
BACKEND_TYPE=ollama  # Options: ollama, transformers

# Ollama Configuration (when BACKEND_TYPE=ollama)
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=hf.co/ChristianAzinn/e5-large-v2-gguf:Q8_0

# Model Configuration (when BACKEND_TYPE=transformers)
MODEL_NAME=intfloat/e5-large
EMBEDDING_DIM=1024
BATCH_SIZE=32
MAX_LENGTH=512

# Redis Configuration (optional)
REDIS_URL=redis://localhost:6379
REDIS_PASSWORD=
CACHE_TTL=86400

# Server Configuration
HOST=0.0.0.0
PORT=8000
LOG_LEVEL=INFO

# Timeout Configuration
REQUEST_TIMEOUT=30
EOF
        print_success "Environment configuration created"
    fi
else
    print_success "Environment configuration already exists"
fi

echo ""
echo "=========================================="
print_success "Setup completed successfully!"
echo "=========================================="
echo ""
print_info "To start the embedding service:"
echo "  1. Start Ollama (if not running): ollama serve"
echo "  2. Run: ./start.sh"
echo ""
print_info "Or manually:"
echo "  1. Start Ollama: ollama serve"
echo "  2. Activate venv: source .venv/bin/activate"
echo "  3. Start service: uv run python app.py"
echo ""
print_info "Service will be available at: http://localhost:8000"
print_info "API documentation: http://localhost:8000/docs"
echo ""