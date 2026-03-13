# GoAgent Framework

A modern multi-agent framework for building AI-powered fashion recommendation systems in Go.

## Overview

GoAgent is a lightweight, modular multi-agent framework designed for building sophisticated AI applications. Originally developed for fashion recommendations, the framework provides a robust foundation for orchestrating multiple AI agents with features like workflow management, memory systems, and graceful shutdown handling.

## Features

- **Multi-Agent Architecture**: Leader agent orchestrates multiple sub-agents for parallel task execution
- **AHP Protocol**: Custom Agent Heartbeat Protocol for inter-agent communication
- **Workflow Engine**: Dynamic DAG-based workflow orchestration with hot-reload support
- **LLM Integration**: Unified adapters for OpenAI, Ollama, and other LLM providers
- **Memory System**: Three-tier memory management (session, user, task) with RAG support
- **Graceful Shutdown**: Five-phase shutdown with callback registration
- **Rate Limiting**: Token bucket, sliding window, and semaphore-based limiting
- **Tool System**: Extensible tool registry for agent capabilities

## Architecture

```
goagent/
├── cmd/                  # Application entry points
├── configs/             # Configuration files
├── docs/                # Architecture and API documentation
├── internal/
│   ├── agents/
│   │   ├── base/       # Base agent interfaces
│   │   ├── leader/      # Leader agent implementation
│   │   └── sub/         # Sub-agent implementation
│   ├── core/
│   │   ├── errors/      # Error handling and codes
│   │   └── models/       # Core data models
│   ├── llm/
│   │   └── output/       # LLM adapters and output standardization
│   ├── memory/
│   │   └── context/      # Memory management
│   ├── protocol/
│   │   └── ahp/          # Agent Heartbeat Protocol
│   ├── ratelimit/        # Rate limiting implementations
│   ├── shutdown/          # Graceful shutdown management
│   ├── storage/
│   │   └── postgres/      # PostgreSQL persistence
│   ├── tools/
│   │   └── resources/    # Tool definitions
│   └── workflow/
│       └── engine/        # Workflow orchestration
└── pkg/                   # Reusable utilities
```

## Installation

```bash
# Clone the repository
git clone https://github.com/yourorg/goagent.git
cd goagent

# Install dependencies
make install

# Run tests
make test
```

## Quick Start

```go
package main

import (
    "context"
    "log"

    "goagent/internal/agents/leader"
    "goagent/internal/llm/output"
)

func main() {
    ctx := context.Background()

    // Create LLM adapter
    factory := output.NewFactory()
    adapter, err := factory.Create("openai", &output.Config{
        Model:   "gpt-4",
        APIKey:  "your-api-key",
    })
    if err != nil {
        log.Fatal(err)
    }

    // Use the adapter
    response, err := adapter.Generate(ctx, "Suggest a casual outfit")
    if err != nil {
        log.Fatal(err)
    }

    log.Println(response)
}
```

## Documentation

Detailed documentation is available in the `docs/` directory:

- [Architecture Overview](docs/arch.md)
- [Agent Definitions](docs/agents/)
- [Core Components](docs/core/)
- [LLM Integration](docs/llm/)
- [Protocol](docs/protocol/)
- [Storage](docs/storage/)

## Development

### Code Quality

This project follows strict coding standards:

- All code must pass `go vet`, `staticcheck`, and `golangci-lint`
- Unit tests required with coverage ≥80% (core modules ≥90%)
- Use `gofmt` / `goimports` for formatting
- 4-space indentation, line length ≤120 characters

### Makefile Targets

```bash
make help          # Show available targets
make lint          # Run all linters
make test          # Run tests with coverage
make test-race    # Run tests with race detection
make check         # Run lint and tests
make build         # Build binaries
make clean         # Clean build artifacts
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with race detection
make test-race

# Run tests for core modules (requires 90%+ coverage)
make test-core

# Run linting
make lint
```

## Modules

### Core Modules (P0)

| Module | Description |
|--------|-------------|
| `core/models` | Core data types and models |
| `core/errors` | Error handling and codes |
| `protocol/ahp` | Agent communication protocol |
| `storage/postgres` | PostgreSQL persistence |

### Agent Modules (P1)

| Module | Description |
|--------|-------------|
| `agents/base` | Base agent interfaces |
| `agents/leader` | Leader agent orchestration |
| `agents/sub` | Sub-agent implementation |
| `workflow/engine` | Workflow orchestration |

### Infrastructure (P2)

| Module | Description |
|--------|-------------|
| `llm/output` | LLM adapters and output standardization |
| `memory/context` | Three-tier memory management |
| `shutdown` | Graceful shutdown handling |
| `ratelimit` | Rate limiting implementations |
| `tools/resources` | Tool system |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Run tests and linting (`make check`)
4. Commit your changes (`git commit -m 'Add amazing feature'`)
5. Push to the branch (`git push origin feature/amazing-feature`)
6. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Acknowledgments

- Built with Go 1.26+
- Inspired by [golang-standards/project-layout](https://github.com/golang-standards/project-layout)
