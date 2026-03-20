#!/bin/bash

# Embedding Service Start Script with uv
# This script starts the embedding service using uv package manager

set -e

echo "=========================================="
echo "Starting Embedding Service (uv)"
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

# Check if uv is installed
print_info "Checking for uv installation..."
if ! command -v uv &> /dev/null; then
    print_error "uv is not installed"
    echo ""
    echo "Please install uv first:"
    echo "  curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi

UV_VERSION=$(uv --version)
print_success "uv is installed: $UV_VERSION"

# Check if pyproject.toml exists
if [ ! -f "$SCRIPT_DIR/pyproject.toml" ]; then
    print_error "pyproject.toml not found"
    echo "Please run setup.sh first"
    exit 1
fi

# Check if .env exists
if [ ! -f "$SCRIPT_DIR/.env" ]; then
    print_error "Environment configuration not found"
    echo "Please run setup.sh first"
    exit 1
fi

# Sync dependencies with uv
print_info "Syncing dependencies with uv..."
uv sync
print_success "Dependencies synced"

# Check if Ollama is running (if using Ollama backend)
BACKEND_TYPE=$(grep "^BACKEND_TYPE=" "$SCRIPT_DIR/.env" | cut -d '=' -f2)
BACKEND_TYPE=${BACKEND_TYPE:-"ollama"}

if [ "$BACKEND_TYPE" = "ollama" ]; then
    print_info "Checking if Ollama is running..."
    if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        print_warning "Ollama is not running"
        echo ""
        print_info "Starting Ollama in background..."
        ollama serve > /dev/null 2>&1 &
        OLLAMA_PID=$!
        
        # Wait for Ollama to start
        print_info "Waiting for Ollama to start..."
        for i in {1..30}; do
            if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
                print_success "Ollama started successfully (PID: $OLLAMA_PID)"
                break
            fi
            sleep 1
            if [ $i -eq 30 ]; then
                print_error "Ollama failed to start"
                exit 1
            fi
        done
    else
        print_success "Ollama is already running"
    fi
    
    # Check if model is available
    OLLAMA_MODEL=$(grep "^OLLAMA_MODEL=" "$SCRIPT_DIR/.env" | cut -d '=' -f2)
    OLLAMA_MODEL=${OLLAMA_MODEL:-"hf.co/ChristianAzinn/e5-large-v2-gguf:Q8_0"}
    
    print_info "Checking if embedding model is available..."
    if ! ollama list | grep -q "e5-large-v2"; then
        print_warning "Model e5-large-v2 not found"
        echo ""
        print_info "Downloading model: $OLLAMA_MODEL"
        ollama pull "$OLLAMA_MODEL"
        print_success "Model downloaded successfully"
    else
        print_success "Model e5-large-v2 is available"
    fi
else
    print_info "Using sentence-transformers backend"
fi

# Load environment variables
print_info "Loading environment configuration..."
cd "$SCRIPT_DIR"
set -a
source .env
set +a

# Start the service
HOST=${HOST:-"0.0.0.0"}
PORT=${PORT:-"8000"}
MODEL_NAME=${MODEL_NAME:-"e5-large-v2"}
EMBEDDING_DIM=${EMBEDDING_DIM:-"1024"}

print_info "Starting embedding service..."
echo ""
print_info "Service URL: http://$HOST:$PORT"
print_info "Model: $MODEL_NAME"
print_info "Dimension: $EMBEDDING_DIM"
print_info "Backend: $BACKEND_TYPE"
echo ""

# Create PID file
PID_FILE="$SCRIPT_DIR/.service.pid"

# Start with uv run
uv run python app.py &
SERVICE_PID=$!
echo $SERVICE_PID > "$PID_FILE"
print_success "Service started with PID: $SERVICE_PID"

# Wait for service to be ready
print_info "Waiting for service to be ready..."
for i in {1..10}; do
    if curl -s "http://$HOST:$PORT/health" > /dev/null 2>&1; then
        print_success "Service is ready!"
        break
    fi
    sleep 1
    if [ $i -eq 10 ]; then
        print_warning "Service started but health check timeout"
    fi
done

echo ""
print_info "Service is running. Press Ctrl+C to stop."
print_info "Health check: http://$HOST:$PORT/health"
print_info "API docs: http://$HOST:$PORT/docs"
echo ""

# Handle cleanup on exit
cleanup() {
    echo ""
    print_info "Stopping service..."
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        kill $PID 2>/dev/null || true
        rm -f "$PID_FILE"
        print_success "Service stopped (PID: $PID)"
    fi
    
    # Cleanup Ollama if we started it
    if [ ! -z "$OLLAMA_PID" ]; then
        print_info "Stopping Ollama (PID: $OLLAMA_PID)..."
        kill $OLLAMA_PID 2>/dev/null || true
        print_success "Ollama stopped"
    fi
    
    exit 0
}

trap cleanup SIGINT SIGTERM

# Wait for service process
wait $SERVICE_PID