# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `.golangci.yml` configuration file for linting rules
- CHANGELOG.md for tracking project changes

### Fixed
- **Critical & High Priority Bugs** (All fixed)
  - M12: Conflict resolution strategy now properly applies ReplaceOld and KeepBoth strategies
  - M13: WorkflowExecutor no longer shares outputStore without lock protection
  - M3/M4: Memory leaks fixed with TTL cleanup and LRU eviction
  - M6: Memory operation errors now propagated instead of being silently logged
  - M9: writeBuffer now implements backpressure retry instead of silently dropping data
  - M14: CircuitBreaker.halfOpenInflight leak detection and cleanup added
  - M15: searchPrecision errors now properly propagated
  - M16: Pool.QueryRow error path fixed to return correct connection errors
  - M17: HTTPError type added to support errors.Is()
  - M22: SessionMemory.GetMessages now returns a copy to prevent concurrent modification

- **Security & Configuration**
  - Hardcoded database passwords in cmd/ moved to environment variables
  - Database name inconsistency fixed (styleagent -> goagent across all files)

- **Code Quality**
  - Bubble sort in manager_impl.go replaced with sort.Slice
  - Emoji characters removed from all log messages, using structured logging instead
  - Code duplication eliminated (distillTaskOld/New extracted to distillTaskCommon)
  - Go version number unified across all documentation (Go 1.26+)

### Changed
- Updated bug fix progress: 40/56 bugs fixed (71.4%)
- Comprehensive score improved from 6.9/10 to 7.2/10
- Bug & Reliability score improved from 5.5/10 to 7.0/10

## [0.1.0] - 2026-04-19

### Added
- Initial multi-agent collaboration framework
- Memory management with distillation and retrieval
- Tool calling with ACE (Agent Capability Engine)
- Workflow engine with DAG-based orchestration
- PostgreSQL + pgvector integration
- Support for multiple LLM providers (OpenAI, Ollama, OpenRouter)
