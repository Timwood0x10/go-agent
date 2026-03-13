# Makefile for Style Agent Framework

.PHONY: all lint test test-race check check-core check-tools help clean install

# Default target
all: lint test

# Install dependencies
install:
	go mod download
	go get ./...

# Format code
fmt:
	goimports -w .
	gofmt -s -w .

# Lint targets
lint: lint-vet lint-staticcheck lint-golangci

lint-vet:
	go vet ./...

lint-staticcheck:
	staticcheck ./...

lint-golangci:
	golangci-lint run

# Test targets
test:
	go test -cover ./...

test-race:
	go test -race -cover ./...

# Core modules require 90%+ coverage
test-core:
	go test -cover -coverprofile=coverage.out ./internal/core/...
	@echo "Checking core module coverage..."
	@COVERAGE=$$(go tool cover -func=coverage.out | grep "internal/core" | grep -v "test" | awk -F'=' '{gsub(/[[:space:]]+/,"",$$3); print $$3}' | sort -n | head -1); \
	if [ "$$COVERAGE" -lt 90 ]; then \
		echo "ERROR: Core module coverage is $$COVERAGE%, expected >= 90%"; \
		exit 1; \
	fi
	@echo "Core module coverage: $$COVERAGE%"

# Other modules require 80%+ coverage
test-tools:
	go test -cover -coverprofile=coverage.out ./internal/llm/... ./internal/workflow/... ./internal/memory/... ./internal/shutdown/... ./internal/ratelimit/... ./internal/tools/... ./internal/storage/... ./internal/agents/...
	@echo "Checking tools coverage..."
	@COVERAGE=$$(go tool cover -func=coverage.out | awk -F'=' '{gsub(/[[:space:]]+/,"",$$3); print $$3}' | sort -n | head -1); \
	if [ "$$COVERAGE" -lt 80 ]; then \
		echo "ERROR: Tools coverage is $$COVERAGE%, expected >= 80%"; \
		exit 1; \
	fi
	@echo "Tools coverage: $$COVERAGE%"

# All checks
check: lint test

# Combined check with coverage
check-all: lint test-race test-core test-tools

# Quick check (lint + basic test)
check-quick: lint test

# Build targets
build:
	go build -o bin/server ./cmd/server

build-all:
	go build -o bin/ ./cmd/...

# Clean targets
clean:
	rm -rf bin/
	rm -f coverage.out

# Help
help:
	@echo "Available targets:"
	@echo "  install       - Download and install dependencies"
	@echo "  fmt           - Format code with goimports and gofmt"
	@echo "  lint          - Run all linters (vet, staticcheck, golangci-lint)"
	@echo "  lint-vet      - Run go vet"
	@echo "  lint-staticcheck  - Run staticcheck"
	@echo "  lint-golangci    - Run golangci-lint"
	@echo "  test          - Run tests with coverage"
	@echo "  test-race     - Run tests with race detection"
	@echo "  test-core     - Run tests for core modules (requires 90%+ coverage)"
	@echo "  test-tools    - Run tests for tools modules (requires 80%+ coverage)"
	@echo "  check         - Run lint and test"
	@echo "  check-all     - Run lint, tests with race detection, and coverage checks"
	@echo "  check-quick   - Quick check (lint + basic test)"
	@echo "  build         - Build server binary"
	@echo "  build-all     - Build all binaries"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Show this help message"

# CI target (used in CI pipelines)
ci: install fmt lint test-race