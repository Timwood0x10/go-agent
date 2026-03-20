# Storage 模块 API 文档

## 概述

Storage模块是goagent的核心数据持久化层，基于PostgreSQL 16 + pgvector实现，提供高性能的向量存储、检索和多租户隔离能力。

### 核心能力

- **向量存储与检索**: 基于pgvector的高性能向量相似度搜索
- **多租户隔离**: RLS + Tenant Guard双重保护
- **混合检索**: 向量检索 + BM25全文检索
- **智能缓存**: 嵌入向量缓存、结果缓存
- **安全加密**: AES-256-GCM加密敏感数据
- **限流熔断**: 保护系统稳定性

## 架构设计

### 分层架构

```
┌─────────────────────────────────────────────────────────┐
│                   应用层 (Application)                    │
│  知识库应用 | 代理经验存储 | 工具管理 | 任务结果存储      │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                   服务层 (Services)                       │
│  RetrievalService | EmbeddingClient | Reconciler         │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                 数据访问层 (Repositories)                 │
│  KnowledgeRepository | SecretRepository | ...            │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                   核心层 (Core)                          │
│  Pool | TenantGuard | RetrievalGuard | Security         │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│              PostgreSQL 16 + pgvector                    │
│  knowledge_chunks_1024 | experiences_1024 | tools | ...   │
└─────────────────────────────────────────────────────────┘
```

### 核心组件

#### 1. Pool (连接池)

数据库连接池，管理所有数据库连接。

**接口:**
```go
type Pool struct {
    db  *sql.DB
    cfg *Config
}

// 创建连接池
func NewPool(cfg *Config) (*Pool, error)

// 获取底层连接
func (p *Pool) DB() *sql.DB

// 获取配置
func (p *Pool) Config() *Config

// 关闭连接池
func (p *Pool) Close() error
```

**配置:**
```go
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int           // 最大连接数 (默认25)
    MaxIdleConns    int           // 最大空闲连接 (默认10)
    ConnMaxLifetime time.Duration // 连接最大生命周期 (默认5分钟)
    QueryTimeout    time.Duration // 查询超时 (默认30秒)
    Embedding       *EmbeddingConfig
}
```

#### 2. TenantGuard (租户守卫)

实现多租户数据隔离，通过PostgreSQL的RLS和Tenant Context双重保护。

**接口:**
```go
type TenantGuard struct {
    pool *Pool
}

// 创建租户守卫
func NewTenantGuard(pool *Pool) *TenantGuard

// 设置租户上下文
func (tg *TenantGuard) SetTenantContext(ctx context.Context, tenantID string) error

// 获取当前租户ID
func (tg *TenantGuard) GetCurrentTenantID(ctx context.Context) (string, error)

// 验证租户权限
func (tg *TenantGuard) ValidateTenantAccess(ctx context.Context, tenantID string) error
```

**工作原理:**
1. 在每个请求开始时设置 `app.tenant_id` 会话变量
2. PostgreSQL RLS策略自动过滤非当前租户的数据
3. Repository层自动应用租户隔离

#### 3. RetrievalGuard (检索守卫)

提供限流、熔断、超时保护，防止检索服务过载。

**接口:**
```go
type RetrievalGuard struct {
    rateLimiter    *RateLimiter
    circuitBreaker *CircuitBreaker
    dbTimeout      time.Duration
}

// 创建检索守卫
func NewRetrievalGuard() *RetrievalGuard

// 检查限流
func (rg *RetrievalGuard) AllowRateLimit() error

// 检查熔断器
func (rg *RetrievalGuard) CheckEmbeddingCircuitBreaker() error

// 记录嵌入服务成功
func (rg *RetrievalGuard) RecordEmbeddingSuccess()

// 记录嵌入服务失败
func (rg *RetrievalGuard) RecordEmbeddingFailure()

// 设置数据库超时
func (rg *RetrievalGuard) WithDBTimeout(ctx context.Context) (context.Context, context.CancelFunc)
```

**限流策略:**
- 默认每秒100次检索请求
- 使用滑动窗口算法
- 超出限制返回错误

**熔断策略:**
- 默认失败阈值5次
- 熔断后等待30秒再尝试
- 半开启状态允许少量测试请求

#### 4. Repository (数据访问层)

提供统一的数据访问接口，支持事务。

**接口:**
```go
type Repository struct {
    Session   *SessionRepository
    Recommend *RecommendRepository
    Profile   *ProfileRepository
    Vector    *VectorSearcher
    pool      *Pool
}

// 创建Repository
func NewRepository(pool *Pool) *Repository

// 事务操作
func (r *Repository) Transaction(ctx context.Context, fn func(repo *Repository) error) error

// 获取连接池
func (r *Repository) Pool() *Pool

// 关闭Repository
func (r *Repository) Close() error
```

## 功能模块

### 1. 知识库 (Knowledge Repository)

管理文档知识的存储、检索。

**核心功能:**
- 文档分块存储
- 向量相似度检索
- BM25全文检索
- 批量导入
- 时间衰减

**接口:**
```go
type KnowledgeRepository struct {
    db     DBTX
    dbPool *sql.DB
}

// 创建知识块
func (r *KnowledgeRepository) Create(ctx context.Context, chunk *KnowledgeChunk) error

// 批量创建
func (r *KnowledgeRepository) CreateBatch(ctx context.Context, chunks []*KnowledgeChunk) error

// 根据ID查询
func (r *KnowledgeRepository) GetByID(ctx context.Context, id string) (*KnowledgeChunk, error)

// 向量检索
func (r *KnowledgeRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*KnowledgeChunk, error)

// 关键词检索
func (r *KnowledgeRepository) SearchByKeyword(ctx context.Context, query, tenantID string, limit int) ([]*KnowledgeChunk, error)

// 按文档列出所有块
func (r *KnowledgeRepository) ListByDocument(ctx context.Context, documentID, tenantID string) ([]*KnowledgeChunk, error)

// 更新嵌入向量
func (r *KnowledgeRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string, version int) error

// 删除
func (r *KnowledgeRepository) Delete(ctx context.Context, id string) error

// 清理过期数据
func (r *KnowledgeRepository) CleanupExpired(ctx context.Context, olderThan time.Time) (int64, error)
```

**数据模型:**
```go
type KnowledgeChunk struct {
    ID               string                 `json:"id"`
    TenantID         string                 `json:"tenant_id"`
    Content          string                 `json:"content"`
    Embedding        []float64              `json:"embedding"`
    EmbeddingModel   string                 `json:"embedding_model"`
    EmbeddingVersion int                    `json:"embedding_version"`
    EmbeddingStatus  string                 `json:"embedding_status"`
    SourceType       string                 `json:"source_type"`
    Source           string                 `json:"source"`
    Metadata         map[string]interface{} `json:"metadata"`
    DocumentID       string                 `json:"document_id"`
    ChunkIndex       int                    `json:"chunk_index"`
    ContentHash      string                 `json:"content_hash"`
    AccessCount      int                    `json:"access_count"`
    CreatedAt        time.Time              `json:"created_at"`
    UpdatedAt        time.Time              `json:"updated_at"`
}
```

### 2. 密钥管理 (Secret Repository)

管理API密钥、密码等敏感信息，支持AES-256-GCM加密。

**核心功能:**
- 密钥加密存储
- 多格式导入 (JSON/YAML/CSV)
- 批量导入
- 密钥轮换

**接口:**
```go
type SecretRepository struct {
    db     DBTX
    dbPool *sql.DB
}

// 创建密钥
func (r *SecretRepository) Create(ctx context.Context, secret *Secret) error

// 批量导入
func (r *SecretRepository) Import(ctx context.Context, items []*SecretImportItem) error

// 获取密钥
func (r *SecretRepository) Get(ctx context.Context, tenantID, key string) (*Secret, error)

// 列出所有密钥
func (r *SecretRepository) List(ctx context.Context, tenantID string) ([]*Secret, error)

// 更新密钥
func (r *SecretRepository) Update(ctx context.Context, secret *Secret) error

// 删除密钥
func (r *SecretRepository) Delete(ctx context.Context, tenantID, key string) error
```

**数据模型:**
```go
type Secret struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Key       string    `json:"key"`
    Value     string    `json:"value"` // 加密存储
    Type      string    `json:"type"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type SecretImportItem struct {
    Key   string `json:"key"`
    Value string `json:"value"`
    Type  string `json:"type"`
}
```

**适配器支持:**
```go
type SecretAdapter struct{}

// 从不同格式解析
func (a *SecretAdapter) ParseFrom(data []byte) ([]*SecretImportItem, error)

// 格式检测
func (a *SecretAdapter) DetectFormat(data []byte) Format

type Format string

const (
    FormatJSON Format = "json"
    FormatYAML Format = "yaml"
    FormatCSV  Format = "csv"
)
```

### 3. 检索服务 (Retrieval Service)

智能检索服务，支持多源混合检索。

**核心功能:**
- 混合检索 (向量 + BM25)
- 多源检索 (知识库、经验、工具)
- 查询重写
- 时间衰减
- 结果合并排序

**接口:**
```go
type RetrievalService struct {
    db              *Pool
    embeddingClient *EmbeddingClient
    tenantGuard     *TenantGuard
    retrievalGuard  *RetrievalGuard
    logger          *slog.Logger
}

// 创建检索服务
func NewRetrievalService(pool *Pool, embeddingClient *EmbeddingClient, tenantGuard *TenantGuard, retrievalGuard *RetrievalGuard) *RetrievalService

// 执行检索
func (s *RetrievalService) Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error)
```

**检索请求:**
```go
type SearchRequest struct {
    Query       string          `json:"query"`           // 检索查询
    TenantID    string          `json:"tenant_id"`       // 租户ID
    TopK        int             `json:"top_k"`           // 返回结果数
    MinScore    float64         `json:"min_score"`       // 最小相似度
    Plan        *RetrievalPlan  `json:"plan"`            // 检索策略
    EnableTrace bool            `json:"enable_trace"`    // 启用追踪
    Trace       *RetrievalTrace `json:"trace,omitempty"` // 追踪信息
}
```

**检索策略:**
```go
type RetrievalPlan struct {
    SearchKnowledge   bool    `json:"search_knowledge"`    // 检索知识库
    SearchExperience  bool    `json:"search_experience"`   // 检索经验
    SearchTools       bool    `json:"search_tools"`        // 检索工具
    SearchTaskResults bool    `json:"search_task_results"` // 检索任务结果

    KnowledgeWeight   float64 `json:"knowledge_weight"`    // 知识库权重 (默认0.4)
    ExperienceWeight  float64 `json:"experience_weight"`   // 经验权重 (默认0.3)
    ToolsWeight       float64 `json:"tools_weight"`        // 工具权重 (默认0.2)
    TaskResultsWeight float64 `json:"task_results_weight"` // 任务结果权重 (默认0.1)

    EnableQueryRewrite  bool `json:"enable_query_rewrite"`  // 启用查询重写
    EnableKeywordSearch bool `json:"enable_keyword_search"` // 启用关键词搜索
    EnableTimeDecay     bool `json:"enable_time_decay"`     // 启用时间衰减

    TopK int `json:"top_k"` // 每个源返回的最大结果数
}
```

**检索结果:**
```go
type SearchResult struct {
    ID        string                 `json:"id"`
    Content   string                 `json:"content"`
    Score     float64                `json:"score"`
    Source    string                 `json:"source"`   // knowledge, experience, tool, task_result
    Type      string                 `json:"type"`     // 结果类型
    Metadata  map[string]interface{} `json:"metadata"` // 附加元数据
    CreatedAt time.Time              `json:"created_at"`
}
```

**默认检索策略:**
```go
func DefaultRetrievalPlan() *RetrievalPlan {
    return &RetrievalPlan{
        SearchKnowledge:     true,
        SearchExperience:    true,
        SearchTools:         true,
        SearchTaskResults:   false,
        KnowledgeWeight:     0.4,
        ExperienceWeight:    0.3,
        ToolsWeight:         0.2,
        TaskResultsWeight:   0.1,
        EnableQueryRewrite:  false,
        EnableKeywordSearch: true,
        EnableTimeDecay:     true,
        TopK:                10,
    }
}
```

### 4. 嵌入服务 (Embedding Client)

提供嵌入向量生成服务，支持多种嵌入模型。

**核心功能:**
- 嵌入向量生成
- 批量处理
- 缓存支持
- 超时保护
- 重试机制

**接口:**
```go
type EmbeddingClient struct {
    serviceURL string
    model      string
    httpClient *http.Client
    cache      *cache.Cache
    enabled    bool
}

// 创建嵌入客户端
func NewEmbeddingClient(serviceURL, model string) *EmbeddingClient

// 生成嵌入向量
func (c *EmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error)

// 批量生成嵌入向量
func (c *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)

// 检查是否启用
func (c *EmbeddingClient) IsEnabled() bool

// 启用客户端
func (c *EmbeddingClient) Enable()

// 禁用客户端
func (c *EmbeddingClient) Disable()
```

## 使用指南

### 基础使用

#### 1. 初始化连接池

```go
import "goagent/internal/storage/postgres"

config := &postgres.Config{
    Host:     "localhost",
    Port:     5433,
    User:     "postgres",
    Password: "postgres",
    Database: "goagent",
    Embedding: &postgres.EmbeddingConfig{
        ServiceURL: "http://localhost:11434",
        Model:      "nomic-embed-text",
    },
}

pool, err := postgres.NewPool(config)
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

#### 2. 创建Repository

```go
repo := postgres.NewRepository(pool)
defer repo.Close()
```

#### 3. 使用知识库

```go
import "goagent/internal/storage/postgres/repositories"
import "goagent/internal/storage/postgres/models"

// 创建知识库Repository
kbRepo := repositories.NewKnowledgeRepository(pool.DB(), pool.DB())

// 创建知识块
chunk := &models.KnowledgeChunk{
    TenantID:        "tenant-001",
    Content:         "这是知识内容",
    Embedding:       []float64{0.1, 0.2, ...}, // 1024维向量
    EmbeddingModel:  "nomic-embed-text",
    EmbeddingVersion: 1,
    EmbeddingStatus: "completed",
    SourceType:      "document",
    Source:          "doc.pdf",
    ChunkIndex:      0,
    ContentHash:     "hash123",
}

err := kbRepo.Create(ctx, chunk)
```

#### 4. 向量检索

```go
// 查询向量
queryEmbedding := []float64{0.1, 0.2, ...}

// 向量检索
results, err := kbRepo.SearchByVector(ctx, queryEmbedding, "tenant-001", 10)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("ID: %s, Content: %s\n", result.ID, result.Content)
}
```

#### 5. 使用检索服务

```go
import "goagent/internal/storage/postgres/services"
import "goagent/internal/storage/postgres/embedding"

// 创建嵌入客户端
embeddingClient := embedding.NewEmbeddingClient(
    "http://localhost:11434",
    "nomic-embed-text",
)

// 创建租户守卫
tenantGuard := postgres.NewTenantGuard(pool)

// 创建检索守卫
retrievalGuard := postgres.NewRetrievalGuard()

// 创建检索服务
retrievalService := services.NewRetrievalService(
    pool,
    embeddingClient,
    tenantGuard,
    retrievalGuard,
)

// 执行检索
req := &services.SearchRequest{
    Query:    "什么是RAG？",
    TenantID: "tenant-001",
    TopK:     5,
    MinScore: 0.6,
    Plan:     services.DefaultRetrievalPlan(),
}

results, err := retrievalService.Search(ctx, req)
if err != nil {
    log.Fatal(err)
}
```

### 高级使用

#### 1. 事务操作

```go
err := repo.Transaction(ctx, func(txRepo *postgres.Repository) error {
    // 创建会话
    session := &models.Session{
        ID:       "session-001",
        TenantID: "tenant-001",
        Status:   "active",
    }
    if err := txRepo.Session.Create(ctx, session); err != nil {
        return err
    }

    // 创建推荐结果
    result := &models.RecommendResult{
        SessionID: "session-001",
        Items:     []string{"item1", "item2"},
    }
    if err := txRepo.Recommend.Create(ctx, result); err != nil {
        return err
    }

    return nil
})
```

#### 2. 批量导入

```go
chunks := []*models.KnowledgeChunk{
    {Content: "内容1", ...},
    {Content: "内容2", ...},
    {Content: "内容3", ...},
}

err := kbRepo.CreateBatch(ctx, chunks)
```

#### 3. 密钥管理

```go
import "goagent/internal/storage/postgres/adapters"

// 创建密钥Repository
secretRepo := repositories.NewSecretRepository(pool.DB(), pool.DB())

// 导入密钥 (支持JSON/YAML/CSV)
data := []byte(`
openai_api_key: sk-xxx123
anthropic_api_key: sk-yyy456
`)

adapter := &adapters.SecretAdapter{}
items, err := adapter.ParseFrom(data)
if err != nil {
    log.Fatal(err)
}

// 批量导入
err = secretRepo.Import(ctx, "tenant-001", items)
```

#### 4. 自定义检索策略

```go
// 自定义检索计划
customPlan := &services.RetrievalPlan{
    SearchKnowledge:     true,
    SearchExperience:    false,  // 不检索经验
    SearchTools:         false,  // 不检索工具
    KnowledgeWeight:     1.0,    // 只考虑知识库
    EnableKeywordSearch: true,
    EnableTimeDecay:     false,  // 不使用时间衰减
    TopK:                3,
}

req := &services.SearchRequest{
    Query:    "查询内容",
    TenantID: "tenant-001",
    Plan:     customPlan,
}

results, err := retrievalService.Search(ctx, req)
```

#### 5. 检索追踪

```go
req := &services.SearchRequest{
    Query:       "查询内容",
    TenantID:    "tenant-001",
    EnableTrace: true,  // 启用追踪
}

results, err := retrievalService.Search(ctx, req)

// 查看追踪信息
if req.Trace != nil {
    log.Printf("Original query: %s", req.Trace.OriginalQuery)
    log.Printf("Vector results: %d", req.Trace.VectorResults)
    log.Printf("Keyword results: %d", req.Trace.KeywordResults)
    log.Printf("Final results: %d", req.Trace.FinalResults)
    log.Printf("Execution time: %v", req.Trace.ExecutionTime)
}
```

## 最佳实践

### 1. 连接池配置

```go
config := &postgres.Config{
    MaxOpenConns:    25,  // 根据并发需求调整
    MaxIdleConns:    10,  // 保持一定空闲连接
    ConnMaxLifetime: 5 * time.Minute,  // 定期刷新连接
    QueryTimeout:    30 * time.Second,  // 设置合理超时
}
```

### 2. 错误处理

```go
results, err := kbRepo.SearchByVector(ctx, embedding, tenantID, limit)
if err != nil {
    if errors.Is(err, errors.ErrRecordNotFound) {
        // 记录不存在
        return nil, nil
    }
    // 其他错误
    return nil, fmt.Errorf("search failed: %w", err)
}
```

### 3. 上下文管理

```go
// 设置超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results, err := kbRepo.SearchByVector(ctx, embedding, tenantID, limit)
```

### 4. 多租户隔离

```go
// 创建租户守卫
tenantGuard := postgres.NewTenantGuard(pool)

// 在每个请求开始时设置租户上下文
ctx = context.WithValue(ctx, "tenant_id", "tenant-001")
if err := tenantGuard.SetTenantContext(ctx, "tenant-001"); err != nil {
    return err
}

// 后续所有操作自动应用租户隔离
results, err := kbRepo.SearchByVector(ctx, embedding, "tenant-001", limit)
```

### 5. 批量操作

```go
// 批量导入优于逐个导入
chunks := make([]*models.KnowledgeChunk, 0, 100)
for _, doc := range documents {
    chunks = append(chunks, doc.Chunks...)
}
err := kbRepo.CreateBatch(ctx, chunks)
```

## 性能优化

### 1. 向量检索优化

- 使用合适的TopK值 (5-10)
- 设置合理的最小相似度 (0.6-0.8)
- 利用时间衰减避免过期数据干扰

### 2. 缓存策略

- 启用嵌入向量缓存
- 缓存常用查询结果
- 设置合理的缓存TTL

### 3. 批量处理

- 使用批量导入接口
- 批量生成嵌入向量
- 批量更新操作

### 4. 索引优化

- 确保tenant_id有索引
- 确保embedding_status有索引
- 确保content_hash有索引

## 错误处理

### 错误类型

```go
// 核心错误
var (
    ErrInvalidArgument   = errors.New("invalid argument")
    ErrRecordNotFound    = errors.New("record not found")
    ErrDuplicateKey      = errors.New("duplicate key")
    ErrNoTransaction     = errors.New("no transaction")
    ErrTenantNotAllowed  = errors.New("tenant not allowed")
    ErrRateLimitExceeded = errors.New("rate limit exceeded")
    ErrCircuitBreakerOpen = errors.New("circuit breaker open")
)
```

### 错误处理示例

```go
// 检查错误类型
if errors.Is(err, errors.ErrRecordNotFound) {
    // 记录不存在
    return nil, nil
}

if errors.Is(err, errors.ErrRateLimitExceeded) {
    // 限流错误，等待后重试
    time.Sleep(time.Second)
    return nil, err
}

if errors.Is(err, errors.ErrCircuitBreakerOpen) {
    // 熔断器打开，降级处理
    return fallbackSearch(ctx, query)
}
```

## 安全考虑

### 1. 数据加密

- 敏感数据使用AES-256-GCM加密
- 密钥管理使用专用Secret Repository
- 定期轮换加密密钥

### 2. 多租户隔离

- 使用RLS策略自动隔离
- Tenant Guard双重保护
- 验证所有跨租户访问

### 3. SQL注入防护

- 使用参数化查询
- 避免字符串拼接SQL
- 验证所有输入参数

### 4. 访问控制

- 实施最小权限原则
- 定期审计访问日志
- 监控异常访问模式

## 监控与日志

### 1. 性能监控

```go
// 启用检索追踪
req.EnableTrace = true
results, err := retrievalService.Search(ctx, req)

// 记录性能指标
log.Printf("Execution time: %v", req.Trace.ExecutionTime)
log.Printf("Vector results: %d", req.Trace.VectorResults)
log.Printf("Keyword results: %d", req.Trace.KeywordResults)
```

### 2. 错误日志

```go
// 记录详细错误信息
slog.Error("Failed to search knowledge base",
    "error", err,
    "tenant_id", tenantID,
    "query", query,
    "top_k", topK,
)
```

### 3. 指标收集

```go
// 记录检索次数
metrics.RecordSearch("knowledge", tenantID)

// 记录检索耗时
metrics.RecordSearchDuration("knowledge", time.Since(start))

// 记录检索结果数
metrics.RecordSearchResults("knowledge", len(results))
```

## 参考资料

- [PostgreSQL文档](https://www.postgresql.org/docs/16/)
- [pgvector文档](https://github.com/pgvector/pgvector)
- [RLS策略](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [向量检索最佳实践](https://github.com/pgvector/pgvector#best-practices)