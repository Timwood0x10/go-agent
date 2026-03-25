# Experience Distillation Demo

This example demonstrates the Experience Distillation System, which automatically learns from task executions and reuses that knowledge for future tasks.

## Features Demonstrated

1. **Task Execution and Distillation**
   - Automatic extraction of reusable experiences from successful tasks
   - LLM-based extraction of Problem, Solution, and Constraints
   - Embedding generation for semantic search

2. **Experience Retrieval with Ranking**
   - Multi-signal ranking (semantic + usage + recency)
   - Conflict resolution to eliminate duplicate experiences
   - Configurable ranking weights

3. **Batch Distillation**
   - Efficient processing of multiple tasks
   - Error handling and logging

4. **Advanced Configuration**
   - Custom ranking weights
   - Configurable conflict resolution thresholds

## Prerequisites

- PostgreSQL database with pgvector extension
- Embedding service (e.g., text-embedding-ada-002)
- OpenAI API key (or compatible LLM service)

## Setup

1. **Set environment variables**:
   ```bash
   export OPENAI_API_KEY=your-api-key
   export CONFIG_PATH=./examples/experience-demo/config/config.yaml
   ```

2. **Configure database connection**:
   Edit `config/config.yaml` to match your PostgreSQL setup.

3. **Run the demo**:
   ```bash
   go run examples/experience-demo/main.go
   ```

## How It Works

### Distillation Flow

```
Task Execution
    ↓
TaskResult (success = true)
    ↓
DistillationService.Distill()
    ↓
LLM Extraction (Problem + Solution + Constraints)
    ↓
Embedding Generation
    ↓
ExperienceRepository.Create()
    ↓
PostgreSQL (experiences_1024 table)
```

### Retrieval Flow

```
User Query
    ↓
RetrievalService.Search()
    ↓
Embedding Generation
    ↓
ExperienceRepository.SearchByVector() [Top 20]
    ↓
RankingService.Rank()
    ↓
ConflictResolver.Resolve()
    ↓
Top 5 Experiences
    ↓
Inject to Agent Prompt
```

## Key Components

### DistillationService

Extracts reusable experiences from task results:
- `ShouldDistill()`: Heuristic filtering
- `Distill()`: Single task distillation
- `DistillBatch()`: Batch processing

### RankingService

Multi-signal ranking algorithm:
```
FinalScore = SemanticScore + UsageBoost + RecencyBoost

UsageBoost = min(log(1 + usage_count) * 0.05, 0.2)
RecencyBoost = exp(-age_days / 30) * 0.05
```

### ConflictResolver

Lazy conflict resolution:
- Groups similar experiences by problem similarity
- Selects best experience per group
- Uses simple clustering O(K²)

## Configuration

### Ranking Weights

```go
weights := &experience.RankingWeights{
    UsageWeight:   0.05, // Default: 5% boost per log usage
    RecencyWeight: 0.05, // Default: 5% boost for recent experiences
    RecencyDays:   30,   // Default: 30-day half-life
}
```

### Conflict Resolution

```go
conflictResolver.Configure(0.9) // 90% similarity threshold
```

## Scenarios

The demo runs 6 scenarios:

1. **Task Execution and Distillation**: Single task distillation
2. **Experience Retrieval with Ranking**: Search with multi-signal ranking
3. **Multiple Task Executions**: Batch distillation
4. **Database Optimization Search**: Specific domain search
5. **Configure Ranking Weights**: Custom ranking configuration
6. **Configure Conflict Resolver**: Custom conflict resolution

## Best Practices

1. **Distill Only Successful Tasks**: The system only distills successful tasks
2. **Use Appropriate Thresholds**: Adjust similarity thresholds based on your use case
3. **Monitor Usage Counts**: Track which experiences are most useful
4. **Regular Cleanup**: Periodically remove low-quality experiences

## Troubleshooting

### No Experiences Distilled

- Check if tasks meet distillation criteria (success, length thresholds)
- Verify LLM client is configured correctly
- Check embedding service availability

### Poor Search Results

- Adjust ranking weights
- Increase ExperienceTopK for more candidates
- Verify embedding quality

### Conflict Detection Issues

- Adjust similarity threshold
- Check embedding quality
- Review problem statements for clarity

## Next Steps

- Integrate with your existing agent system
- Add custom ranking signals
- Implement experience feedback loop
- Monitor and analyze experience quality

## References

- [Experience System Development Plan](../../plan/experience_system_development_plan.md)
- [Code Rules](../../plan/code_rules.md)
- [API Documentation](../../api/experience/)