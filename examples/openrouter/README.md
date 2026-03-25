# OpenRouter API Example

A multi-agent example using OpenRouter API as the LLM provider, demonstrating how to integrate with different LLM services and leverage OpenRouter's model selection capabilities.

## Tech Stack and Components

### Technologies Used
- **Language**: Go 1.21+
- **LLM Provider**: OpenRouter API (unified access to multiple models)
- **Configuration Format**: YAML
- **Concurrency Control**: errgroup
- **Template Engine**: Go text/template
- **HTTP Client**: OpenRouter API calls

### Core Components Used

| Component | Purpose | Code Location |
|-----------|---------|---------------|
| **Leader Agent** | Task analysis, config parsing, sub-agent coordination | `internal/agents/leader/` |
| **Sub Agents** | Parallel execution of specific tasks | `internal/agents/sub/` |
| **AHP Protocol** | Inter-agent communication (message queue) | `internal/protocol/ahp/` |
| **LLM Client** | OpenRouter API interaction | `internal/llm/client.go` |
| **Configuration Management** | YAML config file parsing | `internal/config/config.go` |
| **Template Engine** | Prompt template rendering | `internal/llm/template.go` |
| **Memory System** | Session memory management | `internal/memory/context/` |

### OpenRouter Features

| Feature | Description | Configuration Location |
|---------|-------------|------------------------|
| **Multi-Model Support** | Unified interface to access multiple LLM models | `config/server.yaml:llm.model` |
| **API Key Management** | Via environment variable or config file | `OPENROUTER_API_KEY` |
| **Request Routing** | Auto-route to best available model | OpenRouter platform |
| **Cost Optimization** | Auto-select most cost-effective model | OpenRouter platform |

### Key Feature Implementations

**Code Location References**:
- OpenRouter configuration loading: `examples/openrouter/main.go:30-50`
- Leader Agent creation: `examples/openrouter/main.go:100-120`
- Sub Agents creation: `examples/openrouter/main.go:125-145`
- API calls: `internal/llm/client.go:80-120`
- Error handling: `internal/llm/client.go:150-180`

## OpenRouter API Integration

### Get API Key

1. Visit [OpenRouter](https://openrouter.ai/)
2. Register account and log in
3. Get your API key

### Supported Models

OpenRouter supports multiple models, commonly used ones include:

| Model | Description | Use Cases |
|-------|-------------|-----------|
| `meta-llama/llama-3.1-8b-instruct` | Llama 3.1 8B | General tasks, fast response |
| `meta-llama/llama-3.1-70b-instruct` | Llama 3.1 70B | Complex tasks, high-quality output |
| `google/gemini-flash-1.5` | Gemini Flash | Fast response, cost-optimized |
| `anthropic/claude-3-haiku` | Claude 3 Haiku | Lightweight tasks |
| `openai/gpt-4o` | GPT-4o | High-quality generation |

## Quick Start

### 1. Set API Key

```bash
export OPENROUTER_API_KEY="sk-or-v1-your-api-key"
```

Or set in `config/server.yaml`:

```yaml
llm:
  api_key: "sk-or-v1-your-api-key"
```

### 2. Configure Model

Edit `config/server.yaml`:

```yaml
llm:
  provider: "openrouter"
  api_key: "${OPENROUTER_API_KEY}"  # Use environment variable
  base_url: "https://openrouter.ai/api/v1"
  model: "meta-llama/llama-3.1-8b-instruct"
  timeout: 60
  max_tokens: 2048
```

### 3. Run Example

```bash
cd examples/openrouter
go run main.go
```

### 4. Example Output

```
=== Style Agent (OpenRouter Example) ===
Starting Style Agent Example

[INFO] Loaded configuration from ./examples/openrouter/config/server.yaml
[INFO] Using OpenRouter API with model: meta-llama/llama-3.1-8b-instruct
[INFO] Initialized Leader Agent: leader-openrouter
[INFO] Initialized 3 Sub Agents

=== Processing Sample Request ===
Input: I want to travel to Tokyo, Japan for 5 days and 4 nights, with a budget of 10,000 yuan

[Leader Agent] Parsing profile using OpenRouter...
Profile: {"destination": "Tokyo", "duration": "5 days and 4 nights", "budget": 10000}

[Leader Agent] Dispatching tasks to 3 sub-agents...

[Agent: destination] Processing with OpenRouter...
[Agent: food] Processing with OpenRouter...
[Agent: hotel] Processing with OpenRouter...

=== Aggregated Results ===
Destination Recommendations: ...
Food Recommendations: ...
Hotel Recommendations: ...

[INFO] Example completed successfully
```

## Configuration

### config/server.yaml

```yaml
llm:
  provider: "openrouter"
  api_key: "${OPENROUTER_API_KEY}"  # Recommended: use environment variable
  base_url: "https://openrouter.ai/api/v1"
  model: "meta-llama/llama-3.1-8b-instruct"
  timeout: 60
  max_tokens: 2048

agents:
  leader:
    id: "leader-openrouter"
    max_steps: 5
    max_parallel_tasks: 3

  sub:
    - id: "agent-destination"
      type: "destination"
      triggers: ["destination", "go", "travel"]
      max_retries: 2

prompts:
  profile_extraction: |
    Extract travel information from user input: {{.input}}
```

## OpenRouter Features

### 1. Model Selection

You can choose different models in configuration:

```yaml
llm:
  model: "meta-llama/llama-3.1-70b-instruct"  # More powerful model
```

### 2. Cost Tracking

OpenRouter provides detailed cost tracking, you can view costs for each request:

```bash
# View usage in OpenRouter console
# https://openrouter.ai/activity
```

### 3. Error Handling

The code includes comprehensive error handling:

```go
// internal/llm/client.go:150-180
if err != nil {
    if strings.Contains(err.Error(), "401") {
        return fmt.Errorf("API key invalid")
    }
    if strings.Contains(err.Error(), "429") {
        return fmt.Errorf("rate limit exceeded")
    }
    return fmt.Errorf("API error: %w", err)
}
```

## Differences from Other Examples

| Feature | simple Example | openrouter Example |
|---------|----------------|---------------------|
| **LLM Provider** | Configurable (default Ollama) | Dedicated to OpenRouter |
| **API Key** | Optional | Required |
| **Model Selection** | Single model | Multiple models available |
| **Cost Tracking** | Not supported | Supported |
| **Use Case** | Local development | Production environment |

## Troubleshooting

### Issue 1: Invalid API Key

```
Error: API key invalid
```

**Solution**:
- Check `OPENROUTER_API_KEY` environment variable
- Verify API key is correct
- Confirm API key is activated

### Issue 2: Model Not Available

```
Error: model not found or not available
```

**Solution**:
- Check model name is correct
- Visit [OpenRouter Models](https://openrouter.ai/models) to view available models
- Try using a different model

### Issue 3: Rate Limit

```
Error: rate limit exceeded
```

**Solution**:
- Wait and retry after some time
- Upgrade OpenRouter plan
- Reduce request frequency

### Issue 4: Network Connection Failed

```
Error: connection refused
```

**Solution**:
- Check network connection
- Verify OpenRouter service status
- Check proxy settings

## Extending

### Use Different Models for Different Agents

```yaml
agents:
  sub:
    - id: "agent-destination"
      type: "destination"
      model: "meta-llama/llama-3.1-70b-instruct"  # Use more powerful model
    
    - id: "agent-food"
      type: "food"
      model: "meta-llama/llama-3.1-8b-instruct"  # Use fast model
```

### Add Request Retry

```yaml
llm:
  retry:
    max_attempts: 3
    initial_delay: 1s
    backoff_multiplier: 2.0
```

## References

- [Main README](../../README.md)
- [OpenRouter Documentation](https://openrouter.ai/docs)
- [OpenRouter Models](https://openrouter.ai/models)
- [LLM Documentation](../../docs/llm/)
- [Quick Start](../../docs/quick_start_en.md)

## License

MIT License

---

**Created**: 2026-03-23  
**Example Type**: OpenRouter API Integration Demonstration  
**Code Location**: `examples/openrouter/main.go:1-409`