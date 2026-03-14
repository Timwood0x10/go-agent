# Travel Planning Agent Example

A multi-agent travel assistant powered by LLMs, built on the GoAgent framework.

## What is this?

This is a **demo application** showcasing the GoAgent multi-agent framework. It demonstrates:

- **Profile Parsing**: Extracts travel preferences from natural language input using LLM
- **Dynamic Task Planning**: Leader agent decides which sub-agents to call based on user profile triggers
- **Parallel Execution**: Multiple sub-agents work concurrently
- **Result Aggregation**: Combines results from all agents

## Quick Start

### 1. Set API Key

```bash
# Using OpenRouter (recommended)
export OPENROUTER_API_KEY="your-api-key"

# Or use other providers: openai, ollama
```

### 2. Run

```bash
cd /Users/scc/go/src/styleagent
go run ./examples/travel/main.go
```

### 3. Try It

```
=== Request 1: 我想去日本东京旅游，5天4晚，预算10000元，喜欢美食和购物 ===
```

---

## Configuration Reference

All configuration is in `config/server.yaml`. Below is a complete reference:

### LLM Settings

```yaml
llm:
  provider: "openrouter"      # Provider: "openai", "ollama", "openrouter"
  api_key: ""                 # API key (use env var: OPENROUTER_API_KEY)
  base_url: "https://openrouter.ai/api/v1"  # API endpoint
  model: "meta-llama/llama-3.1-8b-instruct"  # Model name
  timeout: 60                 # Timeout in seconds
  max_tokens: 4096            # Max response tokens
```

### Agent Settings

```yaml
agents:
  leader:
    id: "leader-travel"           # Leader agent ID
    max_steps: 10                 # Max execution steps
    max_parallel_tasks: 4          # Max parallel sub-agents
    max_validation_retry: 3        # Max retries on validation failure
    enable_cache: true            # Enable result caching

  sub:
    - id: "agent-destination"     # Sub-agent ID
      type: "destination"        # Agent type (used in templates as {{.Category}})
      category: "destination"    # Category for recommendation
      triggers: ["destination"]   # Keywords to trigger this agent
      max_retries: 3              # Max retries on LLM failure
      timeout: 30                 # Execution timeout (seconds)
      model: "..."                # Optional: override model for this agent
      provider: "..."             # Optional: override provider
```

### Prompt Templates

The most important part! Templates define what data is passed to the LLM.

#### Profile Extraction Template

Used to parse user input into structured data:

```yaml
prompts:
  profile_extraction: |
    你是一位旅行助手。请从用户的输入中提取旅行偏好信息。
    
    用户输入: {{.input}}    # <-- User's raw input
    
    ...
```

**Available Variables:**
| Variable | Description |
|----------|-------------|
| `{{.input}}` | Raw user input text |

**Expected Output Format (JSON):**
The LLM should return JSON with these fields (can be customized):
```json
{
  "destination": "东京",
  "duration": "5天4晚",
  "budget": 10000,
  "preferences": ["美食", "购物"],
  "travel_style": "休闲"
}
```

#### Recommendation Template

Used by sub-agents to generate recommendations:

```yaml
prompts:
  recommendation: |
    你是一位专业的旅行顾问。请根据以下信息推荐 {{.Category}}：
    
    用户目的地: {{if index . "destination"}}{{index . "destination"}}{{else}}<no value>{{end}}
    旅行天数: {{if index . "duration"}}{{index . "duration"}}{{else}}<no value>{{end}}
    预算范围: {{if index . "budget"}}{{index . "budget"}}{{else}}<no value>{{end}}
    用户偏好: {{if index . "preferences"}}{{index . "preferences"}}{{else}}<no value>{{end}}
    旅行风格: {{if index . "travel_style"}}{{index . "travel_style"}}{{else}}<no value>{{end}}
```

**Available Variables for Recommendation:**

| Variable | Description | Example |
|----------|-------------|---------|
| `{{.Category}}` | Agent type | "destination", "food", "hotel" |
| `{{index . "destination"}}` | Destination from profile | "东京", "清迈" |
| `{{index . "duration"}}` | Trip duration | "5天4晚", "3天2夜" |
| `{{index . "budget"}}` | Budget (number) | 10000, 3000 |
| `{{index . "preferences"}}` | User preferences (array) | ["美食", "购物"] |
| `{{index . "travel_style"}}` | Travel style | "休闲", "经济", "奢华" |

**Template Syntax Notes:**

- Use `{{index . "key"}}` to access map values
- Use `{{if index . "key"}}...{{else}}...{{end}}` for conditional rendering
- All keys in templates must match keys passed from code (case-sensitive!)

### Output Settings

```yaml
output:
  format: "table"              # Output format: "table", "json", "simple"
  item_template: "{{.Name}} - {{.Location}} (¥{{.Price}})"
  summary_template: "为您推荐了 {{.Count}} 个选项"
```

### Workflow Settings (Future)

```yaml
workflow:
  definition_path: "./configs/workflow.yaml"
  auto_reload: false
  reload_interval: 60
```

### Storage Settings (Future)

```yaml
storage:
  enabled: false
  type: "postgres"       # "postgres" or "sqlite"
  host: "localhost"
  port: 5432
  username: "postgres"
  password: "postgres"
  database: "travel_agent"
  ssl_mode: "disable"
  pgvector:
    enabled: false
    dimension: 1536
    table_name: "embeddings"
```

### Memory Settings (Future)

```yaml
memory:
  enabled: false
  session:
    enabled: true
    max_history: 50
  user_profile:
    enabled: false
    storage: "memory"    # "memory" or "postgres"
    vector_db: false
  task_distillation:
    enabled: false
    storage: "memory"
    vector_store: false
    prompt: "请简洁总结以下旅行规划的关键信息..."
```

---

## Available Agent Types

| Type | Description | Default Triggers |
|------|-------------|------------------|
| `destination` | Destination recommendations | "destination" |
| `food` | Dining recommendations | "food", "美食" |
| `hotel` | Hotel recommendations | "hotel", "住宿" |
| `itinerary` | Trip planning | "itinerary", "行程" |

---

## Architecture

```
User Input
    │
    ▼
┌─────────────────┐
│ Leader Agent   │ ── Parse Profile (LLM + profile_extraction template)
│                │ ── Plan Tasks (based on triggers)
└────────┬────────┘
         │ Dispatch tasks in parallel
         ▼
┌────────┴────────┐
│ Sub Agents       │ ── recommendation template receives:
│ (Parallel)       │     {{.Category}}, {{index . "destination"}}, etc.
└────────┬────────┘
         │ Results
         ▼
┌─────────────────┐
│ Aggregation     │ ── Combine all results
└─────────────────┘
```

## Project Structure

```
examples/travel/
├── main.go              # Entry point
├── README.md            # This file
└── config/
    └── server.yaml      # Configuration file
```
