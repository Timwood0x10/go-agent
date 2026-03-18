#!/bin/bash

# Embedding Service Stop Script
# This script stops the embedding service

echo "=========================================="
echo "Stopping Embedding Service"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo "ℹ $1"
}

# Find and kill embedding service processes
print_info "Looking for embedding service processes..."
PIDS=$(pgrep -f "python.*app.py" || true)

if [ -z "$PIDS" ]; then
    print_info "No embedding service processes found"
else
    print_info "Found embedding service processes: $PIDS"
    kill $PIDS
    print_success "Embedding service stopped"
fi

# Optionally stop Ollama
echo ""
read -p "Do you want to stop Ollama as well? (y/n) " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    print_info "Stopping Ollama..."
    pkill -f "ollama serve" || true
    print_success "Ollama stopped"
fi

echo ""
print_success "Cleanup completed"