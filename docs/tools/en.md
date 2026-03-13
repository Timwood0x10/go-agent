# Tools Design Document

## 1. Overview

The Tools module provides external capabilities needed for Agent task execution, including fashion search, weather query, style recommendation, etc.

## 2. Tool List

| Tool | Description | Dependencies |
|------|-------------|--------------|
| fashion_search | Fashion product search | Vector DB |
| weather_check | Weather query | Weather API |
| style_recomm | Style recommendation | LLM |
| outfit_match | Outfit matching | Vector DB |
| price_compare | Price comparison | External API |

## 3. Core Interfaces

```go
type Tool interface {
    // Name tool name
    Name() string
    
    // Description tool description
    Description() string
    
    // Parameters parameter schema
    Parameters() *schema.Schema
    
    // Execute execute tool
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

type ToolResult struct {
    Success bool                   `json:"success"`
    Data    interface{}            `json:"data"`
    Error   string                 `json:"error,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## 4. Tool Implementations

### 4.1 FashionSearch

```go
type FashionSearchTool struct {
    vectorSearcher VectorSearcher
    llm           LLM
}

func (t *FashionSearchTool) Name() string {
    return "fashion_search"
}

func (t *FashionSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
    category := params["category"].(string)
    style := params["style"].([]string)
    budget := params["budget"].(map[string]float64)
    limit := params["limit"].(int)
    
    // 1. Search similar products
    items, err := t.vectorSearcher.Search(ctx, &VectorSearchRequest{
        Category: category,
        Tags: style,
        Limit: limit,
    })
    if err != nil {
        return nil, err
    }
    
    // 2. Filter by budget
    items = filterByBudget(items, budget)
    
    // 3. Sort and return
    return &ToolResult{
        Success: true,
        Data:    items,
    }, nil
}
```

### 4.2 WeatherCheck

```go
type WeatherCheckTool struct {
    weatherAPI string
}

func (t *WeatherCheckTool) Name() string {
    return "weather_check"
}

func (t *WeatherCheckTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
    location := params["location"].(string)
    date := params["date"].(string) // YYYY-MM-DD
    
    // Call weather API
    weather, err := t.getWeather(location, date)
    if err != nil {
        return &ToolResult{Success: false, Error: err.Error()}, nil
    }
    
    return &ToolResult{
        Success: true,
        Data:    weather,
        Metadata: map[string]interface{}{
            "source": "weather_api",
        },
    }, nil
}
```

### 4.3 StyleRecommend

```go
type StyleRecommendTool struct {
    llm LLM
}

func (t *StyleRecommendTool) Name() string {
    return "style_recomm"
}

func (t *StyleRecommendTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
    profile := params["profile"].(*UserProfile)
    occasion := params["occasion"].(string)
    season := params["season"].(string)
    
    // Call LLM for recommendation
    messages := []Message{
        {Role: "system", Content: "You are a fashion consultant..."},
        {Role: "user", Content: fmt.Sprintf("Recommend style based on profile: %+v", profile)},
    }
    
    resp, err := t.llm.Chat(ctx, &ChatRequest{
        Model:    "gpt-oss:20b",
        Messages: messages,
    })
    if err != nil {
        return nil, err
    }
    
    return &ToolResult{
        Success: true,
        Data:    resp.Message.Content,
    }, nil
}
```

## 5. Tool Registration

```go
type ToolRegistry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func (r *ToolRegistry) Register(tool Tool) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if _, ok := r.tools[tool.Name()]; ok {
        return fmt.Errorf("tool %s already registered", tool.Name())
    }
    r.tools[tool.Name()] = tool
    return nil
}

func (r *ToolRegistry) Get(name string) (Tool, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    tool, ok := r.tools[name]
    if !ok {
        return nil, fmt.Errorf("tool %s not found", name)
    }
    return tool, nil
}
```

## 6. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| weather_api_key | - | Weather API key |
| search_limit | 20 | Search result limit |
| enable_tools | all | Enabled tools list |
