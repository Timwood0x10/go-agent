# Memory System Design Document

## 1. Overview

The Memory System module manages various types of memory data in the system, including Session Memory, User Memory, and Task Memory, implementing short-term sessions, long-term user profiles, and task distillation.

## 2. Memory Types

| Type | Lifecycle | Storage | Use Case |
|------|-----------|---------|----------|
| SessionMemory | Session duration | In-Memory + Redis | Short-term session state |
| UserMemory | Long-term | PostgreSQL | User profile |
| TaskMemory | Task duration | In-Memory | Task distillation |

## 3. SessionMemory

Session memory storing current session context information.

```go
type SessionMemory struct {
    SessionID    string                 `json:"session_id"`
    UserID      string                 `json:"user_id"`
    Context     map[string]interface{} `json:"context"`      // Session context
    History     []Message              `json:"history"`       // Conversation history
    TempData    map[string]interface{} `json:"temp_data"`    // Temporary data
    CreatedAt   time.Time              `json:"created_at"`
    UpdatedAt   time.Time              `json:"updated_at"`
    TTL         time.Duration          `json:"ttl"`          // Expiration time
}
```

### Operation Interface

```go
type SessionMemoryStore interface {
    // Get get session memory
    Get(ctx context.Context, sessionID string) (*SessionMemory, error)
    
    // Set set session memory
    Set(ctx context.Context, memory *SessionMemory) error
    
    // Delete delete session memory
    Delete(ctx context.Context, sessionID string) error
    
    // Update update session memory
    Update(ctx context.Context, sessionID string, update func(*SessionMemory) error) error
    
    // AddMessage add conversation message
    AddMessage(ctx context.Context, sessionID string, msg Message) error
    
    // GetHistory get conversation history
    GetHistory(ctx context.Context, sessionID string, limit int) ([]Message, error)
}
```

## 4. UserMemory

User memory storing long-term user preference information.

```go
type UserMemory struct {
    UserID        string                 `json:"user_id"`
    Profile       *UserProfile           `json:"profile"`        // User profile
    Preferences   map[string]interface{} `json:"preferences"`   // Preference settings
    LikedItems    []string               `json:"liked_items"`   // Liked items
    DislikedItems []string               `json:"disliked_items"` // Disliked items
    FeedbackHistory []UserFeedback       `json:"feedback_history"` // Feedback history
    StyleEvolution []StyleTag            `json:"style_evolution"` // Style evolution
    CreatedAt    time.Time              `json:"created_at"`
    UpdatedAt    time.Time              `json:"updated_at"`
}
```

### Operation Interface

```go
type UserMemoryStore interface {
    // Get get user memory
    Get(ctx context.Context, userID string) (*UserMemory, error)
    
    // Save save user memory
    Save(ctx context.Context, memory *UserMemory) error
    
    // UpdatePreference update preference
    UpdatePreference(ctx context.Context, userID string, key string, value interface{}) error
    
    // AddFeedback add feedback
    AddFeedback(ctx context.Context, userID string, feedback *UserFeedback) error
    
    // GetSimilarUsers get similar users (for collaborative filtering)
    GetSimilarUsers(ctx context.Context, userID string, limit int) ([]string, error)
}
```

## 5. TaskMemory

Task memory for intermediate data storage during task execution.

```go
type TaskMemory struct {
    TaskID    string                 `json:"task_id"`
    SessionID string                 `json:"session_id"`
    Input     map[string]interface{} `json:"input"`     // Input data
    Output    map[string]interface{} `json:"output"`    // Output data
    Context   map[string]interface{} `json:"context"`   // Execution context
    Artifacts map[string]interface{} `json:"artifacts"` // Generated intermediate results
    CreatedAt time.Time              `json:"created_at"`
    UpdatedAt time.Time              `json:"updated_at"`
}
```

## 6. RAG Knowledge Base

Vector storage based fashion knowledge base.

```go
type KnowledgeBase struct {
    // Outfit knowledge
    OutfitKnowledge []OutfitDoc
    
    // Brand knowledge
    BrandKnowledge []BrandDoc
    
    // Trend knowledge
    TrendKnowledge []TrendDoc
}

type OutfitDoc struct {
    ID        string    `json:"id"`
    Content   string    `json:"content"`
    Embedding []float64 `json:"embedding"`
    Metadata  map[string]interface{} `json:"metadata"`
}
```

## 7. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| session_ttl | 24h | Session expiration time |
| history_limit | 100 | History message limit |
| memory_backend | memory | Storage backend (memory/redis) |
| vector_dimension | 1536 | Vector dimension |
