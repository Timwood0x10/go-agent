# Storage 设计文档

## 1. 概述

Storage 模块负责 Style Agent 框架的数据持久化，包括 PostgreSQL 数据库操作和 pgvector 向量存储。

## 2. 核心功能

| 功能 | 说明 |
|------|------|
| **会话存储** | 存储和管理用户会话 |
| **推荐历史** | 存储推荐结果历史 |
| **向量搜索** | 基于 pgvector 的相似度搜索 |
| **用户画像** | 存储和查询用户画像 |
| **RAG 存储** | 服装知识库的向量存储 |

## 3. 数据表设计

### 3.1 sessions 表

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

### 3.2 recommendations 表

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

### 3.3 user_profiles 表

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

### 3.4 fashion_items 向量表

```sql
CREATE TABLE fashion_items (
    item_id       VARCHAR(64) PRIMARY KEY,
    name          VARCHAR(256) NOT NULL,
    category      VARCHAR(64) NOT NULL,
    image_url     TEXT,
    price         DECIMAL(10,2),
    brand         VARCHAR(128),
    tags          VARCHAR(64)[],
    embedding     vector(1536),  -- pgvector 维度
    metadata      JSONB,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    
    INDEX idx_category (category),
    INDEX idx_tags (tags)
);

-- 向量索引
CREATE INDEX fashion_items_embedding_idx 
ON fashion_items USING ivfflat (embedding vector_cosine_ops);
```

## 4. 核心接口

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

## 5. 向量搜索

```go
// 向量搜索
type VectorSearcher interface {
    // Search 相似度搜索
    Search(ctx context.Context, req *VectorSearchRequest) (*VectorSearchResponse, error)
    
    // Add 添加向量
    Add(ctx context.Context, items []*FashionItem) error
    
    // Delete 删除向量
    Delete(ctx context.Context, itemIDs []string) error
}

type VectorSearchRequest struct {
    Query      []float float64    `json:"query"`       // 查询向量
    Category   string             `json:"category"`   // 分类过滤
    Tags       []string          `json:"tags"`        // 标签过滤
    Limit      int                `json:"limit"`       // 返回数量
    MinScore   float64            `json:"min_score"`  // 最小相似度
}

type VectorSearchResponse struct {
    Items   []*FashionItem `json:"items"`
    Total   int            `json:"total"`
}
```

## 6. 连接池配置

```go
type DBConfig struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int           // 默认 25
    MaxIdleConns    int           // 默认 10
    ConnMaxLifetime time.Duration // 默认 5m
    ConnMaxIdleTime time.Duration // 默认 1m
}
```

## 7. 配置参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| db_host | localhost | 数据库地址 |
| db_port | 5432 | 数据库端口 |
| max_open_conns | 25 | 最大连接数 |
| max_idle_conns | 10 | 空闲连接数 |
| vector_dimension | 1536 | 向量维度 |
| search_limit | 20 | 搜索返回数量 |
| index_list_count | 100 | IVFFlat 列表数 |
