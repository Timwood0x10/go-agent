# Memory System 设计文档

## 1. 概述

Memory System 模块负责管理系统中的各类内存数据，包括会话内存、用户内存和任务内存，实现短期会话、长期用户画像和任务蒸馏。

## 2. 内存类型

| 类型 | 生命周期 | 存储方式 | 用途 |
|------|----------|----------|------|
| SessionMemory | 会话期间 | In-Memory + Redis | 短期会话状态 |
| UserMemory | 长期 | PostgreSQL | 用户画像 |
| TaskMemory | 任务期间 | In-Memory | 任务蒸馏 |

## 3. SessionMemory

会话内存，存储当前会话的上下文信息。

```go
type SessionMemory struct {
    SessionID    string                 `json:"session_id"`
    UserID      string                 `json:"user_id"`
    Context     map[string]interface{} `json:"context"`      // 会话上下文
    History     []Message              `json:"history"`       // 对话历史
    TempData    map[string]interface{} `json:"temp_data"`    // 临时数据
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    TTL         time.Duration          `json:"ttl"`          // 过期时间
}
```

### 操作接口

```go
type SessionMemoryStore interface {
    // Get 获取会话内存
    Get(ctx context.Context, sessionID string) (*SessionMemory, error)
    
    // Set 设置会话内存
    Set(ctx context.Context, memory *SessionMemory) error
    
    // Delete 删除会话内存
    Delete(ctx context.Context, sessionID string) error
    
    // Update 更新会话内存
    Update(ctx context.Context, sessionID string, update func(*SessionMemory) error) error
    
    // AddMessage 添加对话消息
    AddMessage(ctx context.Context, sessionID string, msg Message) error
    
    // GetHistory 获取对话历史
    GetHistory(ctx context.Context, sessionID string, limit int) ([]Message, error)
}
```

## 4. UserMemory

用户内存，存储用户的长期偏好信息。

```go
type UserMemory struct {
    UserID        string                 `json:"user_id"`
    Profile       *UserProfile           `json:"profile"`        // 用户画像
    Preferences   map[string]interface{} `json:"preferences"`   // 偏好设置
    LikedItems    []string               `json:"liked_items"`   // 喜欢的商品
    DislikedItems []string               `json:"disliked_items"` // 不喜欢的商品
    FeedbackHistory []UserFeedback       `json:"feedback_history"` // 反馈历史
    StyleEvolution []StyleTag            `json:"style_evolution"` // 风格演变
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

### 操作接口

```go
type UserMemoryStore interface {
    // Get 获取用户内存
    Get(ctx context.Context, userID string) (*UserMemory, error)
    
    // Save 保存用户内存
    Save(ctx context.Context, memory *UserMemory) error
    
    // UpdatePreference 更新偏好
    UpdatePreference(ctx context.Context, userID string, key string, value interface{}) error
    
    // AddFeedback 添加反馈
    AddFeedback(ctx context.Context, userID string, feedback *UserFeedback) error
    
    // GetSimilarUsers 获取相似用户 (用于协同过滤)
    GetSimilarUsers(ctx context.Context, userID string, limit int) ([]string, error)
}
```

## 5. TaskMemory

任务内存，用于任务执行过程中的中间数据存储。

```go
type TaskMemory struct {
    TaskID    string                 `json:"task_id"`
    SessionID string                 `json:"session_id"`
    Input     map[string]interface{} `json:"input"`     // 输入数据
    Output    map[string]interface{} `json:"output"`    // 输出数据
    Context   map[string]interface{} `json:"context"`   // 执行上下文
    Artifacts map[string]interface{} `json:"artifacts"` // 产生的中间结果
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
}
```

## 6. RAG 知识库

基于向量存储的服装知识库。

```go
type KnowledgeBase struct {
    // 服装搭配知识
    OutfitKnowledge []OutfitDoc
    
    // 品牌知识
    BrandKnowledge []BrandDoc
    
    // 趋势知识
    TrendKnowledge []TrendDoc
}

type OutfitDoc struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Embedding []float64 `json:"embedding"`
    Metadata  map[string]interface{} `json:"metadata"`
}
```

## 7. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| session_ttl | 24h | 会话过期时间 |
| history_limit | 100 | 历史消息数量限制 |
| memory_backend | memory | 存储后端 (memory/redis) |
| vector_dimension | 1536 | 向量维度 |
