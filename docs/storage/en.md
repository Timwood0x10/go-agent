# Storage Design Document

## 1. Overview

The Storage module is responsible for data persistence in the Style Agent framework, including PostgreSQL database operations and pgvector vector storage.

## 2. Core Functions

| Function | Description |
|----------|-------------|
| **Session Storage** | Store and manage user sessions |
| **Recommendation History** | Store recommendation result history |
| **Vector Search** | Similarity search based on pgvector |
| **User Profile** | Store and query user profiles |
| **RAG Storage** | Vector storage for fashion knowledge base |

## 3. Table Design

### 3.1 sessions table

```sql
CREATE TABLE sessions (
    session_id   VARCHAR(64) PRIMARY KEY,
    user_id      VARCHAR(64) NOT NULL,
    status       VARCHAR(32) NOT NULL,
    context      JSONB,
    created_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    expired_at   TIMESTAMP,
    
    INDEX idx_user_id (user_id),
    INDEX idx_status (status),
    INDEX idx_expired_at (expired_at)
);
```

### 3.2 recommendations table

```sql
CREATE TABLE recommendations (
    id            SERIAL PRIMARY KEY,
    session_id    VARCHAR(64) NOT NULL,
    user_id       VARCHAR(64) NOT NULL,
    items         JSONB NOT NULL,
    total_price   DECIMAL(10,2),
    score         DECIMAL(5,2),
    summary       TEXT,
    style_tags    VARCHAR(64)[],
    feedback      JSONB,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_session_id (session_id),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
);
```

### 3.3 user_profiles table

```sql
CREATE TABLE user_profiles (
    user_id       VARCHAR(64) PRIMARY KEY,
    name          VARCHAR(128),
    gender        VARCHAR(16),
    age           INTEGER,
    occupation    VARCHAR(64),
    style_tags    VARCHAR(64)[],
    budget_min    DECIMAL(10,2),
    budget_max    DECIMAL(10,2),
    colors        VARCHAR(64)[],
    body_type     VARCHAR(32),
    preferences   JSONB,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW()
);
```

### 3.4 fashion_items vector table

```sql
CREATE TABLE fashion_items (
    item_id       VARCHAR(64) PRIMARY KEY,
    name          VARCHAR(256) NOT NULL,
    category      VARCHAR(64) NOT NULL,
    image_url     TEXT,
    price         DECIMAL(10,2),
    brand         VARCHAR(128),
    tags          VARCHAR(64)[],
    embedding     vector(1536),  -- pgvector dimension
    metadata      JSONB,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_category (category),
    INDEX idx_tags (tags)
);

-- Vector index
CREATE INDEX fashion_items_embedding_idx 
ON fashion_items USING ivfflat (embedding vector_cosine_ops);
```

## 4. Core Interfaces

```go
type Storage interface {
    // Session operations
    CreateSession(ctx context.Context, session *Session) error
    GetSession(ctx context.Context, sessionID string) (*Session, error)
    UpdateSession(ctx context.Context, session *Session) error
    DeleteSession(ctx context.Context, sessionID string) error
    ListSessions(ctx context.Context, userID string) ([]*Session, error)
    
    // Recommendation operations
    SaveRecommendation(ctx context.Context, rec *RecommendResult) error
    GetRecommendation(ctx context.Context, id int) (*RecommendResult, error)
    ListRecommendations(ctx context.Context, userID string, limit int) ([]*RecommendResult, error)
    
    // User profile operations
    SaveUserProfile(ctx context.Context, profile *UserProfile) error
    GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
    
    // Vector search
    SearchSimilar(ctx context.Context, query []float64, category string, limit int) ([]*FashionItem, error)
    AddFashionItem(ctx context.Context, item *FashionItem) error
}
```

## 5. Vector Search

```go
// Vector search
type VectorSearcher interface {
    // Search similarity search
    Search(ctx context.Context, req *VectorSearchRequest) (*VectorSearchResponse, error)
    
    // Add add vectors
    Add(ctx context.Context, items []*FashionItem) error
    
    // Delete delete vectors
    Delete(ctx context.Context, itemIDs []string) error
}

type VectorSearchRequest struct {
    Query      []float64       `json:"query"`       // Query vector
    Category   string          `json:"category"`   // Category filter
    Tags       []string        `json:"tags"`        // Tag filter
    Limit      int             `json:"limit"`       // Return count
    MinScore   float64         `json:"min_score"`  // Min similarity
}

type VectorSearchResponse struct {
    Items   []*FashionItem `json:"items"`
    Total   int            `json:"total"`
}
```

## 6. Connection Pool Configuration

```go
type DBConfig struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int           // Default 25
    MaxIdleConns    int           // Default 10
    ConnMaxLifetime time.Duration // Default 5m
    ConnMaxIdleTime time.Duration // Default 1m
}
```

## 7. Configuration Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| db_host | localhost | Database host |
| db_port | 5432 | Database port |
| max_open_conns | 25 | Max open connections |
| max_idle_conns | 10 | Idle connections |
| vector_dimension | 1536 | Vector dimension |
| search_limit | 20 | Search result limit |
| index_list_count | 100 | IVFFlat list count |
