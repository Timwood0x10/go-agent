# Tools 设计文档

## 1. 概述

Tools 模块提供 Agent 执行任务时所需的外部能力，包括时尚搜索、天气查询、风格推荐等。

## 2. 工具列表

| 工具 | 说明 | 依赖 |
|------|------|------|
| fashion_search | 时尚商品搜索 | Vector DB |
| weather_check | 天气查询 | 天气 API |
| style_recomm | 风格推荐 | LLM |
| outfit_match | 搭配匹配 | Vector DB |
| price_compare | 价格比较 | 外部 API |

## 3. 核心接口

```go
type Tool interface {
    // Name 工具名称
    Name() string
    
    // Description 工具描述
    Description() string
    
    // Parameters 参数 schema
    Parameters() *schema.Schema
    
    // Execute 执行工具
    Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
}

type ToolResult struct {
    Success bool                   `json:"success"`
    Data    interface{}            `json:"data"`
    Error   string                 `json:"error,omitempty"`
    Metadata map[string]interface{} `json:"metadata,omitempty"`
}
```

## 4. 工具实现

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
    
    // 1. 搜索相似商品
    items, err := t.vectorSearcher.Search(ctx, &VectorSearchRequest{
        Category: category,
        Tags: style,
        Limit: limit,
    })
    if err != nil {
        return nil, err
    }
    
    // 2. 过滤预算
    items = filterByBudget(items, budget)
    
    // 3. 排序返回
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
    
    // 调用天气 API
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
    
    // 调用 LLM 生成推荐
    messages := []Message{
        {Role: "system", Content: "你是一个时尚顾问..."},
        {Role: "user", Content: fmt.Sprintf("根据用户画像推荐风格: %+v", profile)},
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

## 5. 工具注册

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

## 6. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| weather_api_key | - | 天气 API 密钥 |
| search_limit | 20 | 搜索返回数量 |
| enable_tools | all | 启用的工具列表 |
