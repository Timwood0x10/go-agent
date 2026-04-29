# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `.golangci.yml` configuration file for linting rules
- CHANGELOG.md for tracking project changes
- **Triggers mechanism**: SubAgents can define trigger keywords; TaskPlanner filters tasks by matching user input against triggers (word-boundary aware, Unicode-safe)
- **SubAgentConfig.Priority**: Optional priority field for SubAgent configuration, used by SortByPriority aggregation
- **ToolBinder.BridgeFromRegistry**: Sub Agent ToolBinder can now bridge tools from the global Registry
- **ToolBinder.GetTool**: Fallback lookup to bridged Registry when tool not found locally
- **Aggregator sort strategies**: `SortByNone`, `SortByPriority`, `SortByCreatedAt` configurable sorting
- **Planner tests**: 10 test cases covering multi-task dispatch, trigger matching, fallback, maxTasks, and edge cases
- **Aggregator tests**: 9 test cases covering all sort modes, deduplication, maxItems, nil/empty results, and match score
- **Profile tests**: 5 test cases covering default profile, validation with Preferences/Style, empty/nil profiles

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

- **Refactoring: Decouple from fashion/recommendation scenario**
  - TaskPlanner now creates one task per configured SubAgent instead of a single default task
  - Aggregator no longer hardcodes price-based sorting; defaults to SortByNone (preserve original order)
  - Aggregator deduplication no longer uses Price as part of the dedup key
  - ProfileParser default profile no longer injects hardcoded "casual style" values
  - ProfileParser validation checks both Preferences and Style fields (backward compatible)
  - ProfileParser parseResponse no longer injects hardcoded defaults for empty fields
  - Memory operation failures in Leader.Process no longer block the main request flow (warn + continue)
  - Removed 5 hardcoded clothing fallback methods from Sub Agent executor
  - Removed 9 unused fashion-specific constants (AgentTypeShoes/Head/Accessory, StyleCasual/Formal/Street, OccasionDaily/Party/Date)
  - RecommendResult.CalculateScore returns normalised score in [0, 1] instead of raw item count
  - Aggregator now calculates TotalPrice from aggregated items
  - Aggregator logs a warning when items are dropped during deduplication (both ItemID and Name empty)
  - CreatedAt zero-value items sort after non-zero items when using SortByCreatedAt

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
- **Breaking**: `NewResultAggregator` signature changed: added `sortBy string` parameter
- **Breaking**: `TaskPlanner.Plan` signature changed: added `inputText string` parameter
- **Breaking**: `ResultAggregator.Aggregate` signature changed: added `tasks []*models.Task` parameter
- Data model (RecommendResult/RecommendItem) documented as generic Agent output structure; Price/Brand/ImageURL are now optional domain-specific fields

## [0.1.0] - 2026-04-19

### Added
- Initial multi-agent collaboration framework
- Memory management with distillation and retrieval
- Tool calling with ACE (Agent Capability Engine)
- Workflow engine with DAG-based orchestration
- PostgreSQL + pgvector integration
- Support for multiple LLM providers (OpenAI, Ollama, OpenRouter)
