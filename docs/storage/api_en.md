# Storage Module API Documentation

## Overview

The Storage module is the core data persistence layer of goagent, implemented based on PostgreSQL 16 + pgvector, providing high-performance vector storage, retrieval, and multi-tenant isolation capabilities.

### Core Capabilities

- **Vector Storage and Retrieval**: High-performance vector similarity search based on pgvector
- **Multi-Tenant Isolation**: RLS + Tenant Guard dual-layer protection
- **Hybrid Retrieval**: Vector search + BM25 full-text search
- **Intelligent Caching**: Embedding vector caching, result caching
- **Security Encryption**: AES-256-GCM encryption for sensitive data
- **Rate Limiting and Circuit Breaking**: Protect system stability

## Architecture Design

### Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Application Layer                       │
│  Knowledge Base | Agent Experience | Tool Management      │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                   Service Layer                          │
│  RetrievalService | EmbeddingClient | Reconciler         │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                 Data Access Layer (Repositories)          │
│  KnowledgeRepository | SecretRepository | ...            │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│                   Core Layer                             │
│  Pool | TenantGuard | RetrievalGuard | Security         │
└─────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────┐
│              PostgreSQL 16 + pgvector                    │
│  knowledge_chunks_1024 | experiences_1024 | tools | ...   │
└─────────────────────────────────────────────────────────┘
```

### Core Components

#### 1. Pool (Connection Pool)

Database connection pool that manages all database connections.

**Interface:**
```go
type Pool struct {
    db  *sql.DB
    cfg *Config
}

// Create connection pool
func NewPool(cfg *Config) (*Pool, error)

// Get underlying connection
func (p *Pool) DB() *sql.DB

// Get configuration
func (p *Pool) Config() *Config

// Close connection pool
func (p *Pool) Close() error
```

**Configuration:**
```go
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    Database        string
    MaxOpenConns    int           // Maximum connections (default 25)
    MaxIdleConns    int           // Maximum idle connections (default 10)
    ConnMaxLifetime time.Duration // Connection maximum lifetime (default 5 minutes)
    QueryTimeout    time.Duration // Query timeout (default 30 seconds)
    Embedding       *EmbeddingConfig
}
```

#### 2. TenantGuard (Tenant Guard)

Implements multi-tenant data isolation through PostgreSQL RLS and Tenant Context dual protection.

**Interface:**
```go
type TenantGuard struct {
    pool *Pool
}

// Create tenant guard
func NewTenantGuard(pool *Pool) *TenantGuard

// Set tenant context
func (tg *TenantGuard) SetTenantContext(ctx context.Context, tenantID string) error

// Get current tenant ID
func (tg *TenantGuard) GetCurrentTenantID(ctx context.Context) (string, error)

// Validate tenant permission
func (tg *TenantGuard) ValidateTenantAccess(ctx context.Context, tenantID string) error
```

**How It Works:**
1. Set `app.tenant_id` session variable at the start of each request
2. PostgreSQL RLS policies automatically filter out non-current tenant data
3. Repository layer automatically applies tenant isolation

#### 3. RetrievalGuard (Retrieval Guard)

Provides rate limiting, circuit breaking, and timeout protection to prevent retrieval service overload.

**Interface:**
```go
type RetrievalGuard struct {
    rateLimiter    *RateLimiter
    circuitBreaker *CircuitBreaker
    dbTimeout      time.Duration
}

// Create retrieval guard
func NewRetrievalGuard() *RetrievalGuard

// Check rate limiting
func (rg *RetrievalGuard) AllowRateLimit() error

// Check circuit breaker
func (rg *RetrievalGuard) CheckEmbeddingCircuitBreaker() error

// Record embedding service success
func (rg *RetrievalGuard) RecordEmbeddingSuccess()

// Record embedding service failure
func (rg *RetrievalGuard) RecordEmbeddingFailure()

// Set database timeout
func (rg *RetrievalGuard) WithDBTimeout(ctx context.Context) (context.Context, context.CancelFunc)
```

**Rate Limiting Strategy:**
- Default 100 retrieval requests per second
- Uses sliding window algorithm
- Returns error when limit exceeded

**Circuit Breaking Strategy:**
- Default failure threshold 5 times
- Wait 30 seconds before retry after opening
- Half-open state allows a small number of test requests

#### 4. Repository (Data Access Layer)

Provides unified data access interface with transaction support.

**Interface:**
```go
type Repository struct {
    Session   *SessionRepository
    Recommend *RecommendRepository
    Profile   *ProfileRepository
    Vector    *VectorSearcher
    pool      *Pool
}

// Create repository
func NewRepository(pool *Pool) *Repository

// Transaction operation
func (r *Repository) Transaction(ctx context.Context, fn func(repo *Repository) error) error

// Get connection pool
func (r *Repository) Pool() *Pool

// Close repository
func (r *Repository) Close() error
```

## Functional Modules

### 1. Knowledge Repository

Manages document knowledge storage and retrieval.

**Core Features:**
- Document chunking storage
- Vector similarity retrieval
- BM25 full-text retrieval
- Batch import
- Time decay

**Interface:**
```go
type KnowledgeRepository struct {
    db     DBTX
    dbPool *sql.DB
}

// Create knowledge chunk
func (r *KnowledgeRepository) Create(ctx context.Context, chunk *KnowledgeChunk) error

// Batch create
func (r *KnowledgeRepository) CreateBatch(ctx context.Context, chunks []*KnowledgeChunk) error

// Query by ID
func (r *KnowledgeRepository) GetByID(ctx context.Context, id string) (*KnowledgeChunk, error)

// Vector retrieval
func (r *KnowledgeRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*KnowledgeChunk, error)

// Keyword retrieval
func (r *KnowledgeRepository) SearchByKeyword(ctx context.Context, query, tenantID string, limit int) ([]*KnowledgeChunk, error)

// List all chunks by document
func (r *KnowledgeRepository) ListByDocument(ctx context.Context, documentID, tenantID string) ([]*KnowledgeChunk, error)

// Update embedding vector
func (r *KnowledgeRepository) UpdateEmbedding(ctx context.Context, id string, embedding []float64, model string, version int) error

// Delete
func (r *KnowledgeRepository) Delete(ctx context.Context, id string) error

// Cleanup expired data
func (r *KnowledgeRepository) CleanupExpired(ctx context.Context, olderThan time.Time) (int64, error)
```

**Data Model:**
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

### 2. Secret Repository

Manages API keys, passwords and other sensitive information with AES-256-GCM encryption.

**Core Features:**
- Key encryption storage
- Multi-format import (JSON/YAML/CSV)
- Batch import
- Key rotation

**Interface:**
```go
type SecretRepository struct {
    db     DBTX
    dbPool *sql.DB
}

// Create key
func (r *SecretRepository) Create(ctx context.Context, secret *Secret) error

// Batch import
func (r *SecretRepository) Import(ctx context.Context, items []*SecretImportItem) error

// Get key
func (r *SecretRepository) Get(ctx context.Context, tenantID, key string) (*Secret, error)

// List all keys
func (r *SecretRepository) List(ctx context.Context, tenantID string) ([]*Secret, error)

// Update key
func (r *SecretRepository) Update(ctx context.Context, secret *Secret) error

// Delete key
func (r *SecretRepository) Delete(ctx context.Context, tenantID, key string) error
```

**Data Model:**
```go
type Secret struct {
    ID        string    `json:"id"`
    TenantID  string    `json:"tenant_id"`
    Key       string    `json:"key"`
    Value     string    `json:"value"` // Encrypted storage
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

**Adapter Support:**
```go
type SecretAdapter struct{}

// Parse from different formats
func (a *SecretAdapter) ParseFrom(data []byte) ([]*SecretImportItem, error)

// Format detection
func (a *SecretAdapter) DetectFormat(data []byte) Format

type Format string

const (
    FormatJSON Format = "json"
    FormatYAML Format = "yaml"
    FormatCSV  Format = "csv"
)
```

### 3. Retrieval Service

Intelligent retrieval service supporting multi-source hybrid retrieval.

**Core Features:**
- Hybrid retrieval (vector + BM25)
- Multi-source retrieval (knowledge base, experience, tools)
- Query rewriting
- Time decay
- Result merging and ranking

**Interface:**
```go
type RetrievalService struct {
    db              *Pool
    embeddingClient *EmbeddingClient
    tenantGuard     *TenantGuard
    retrievalGuard  *RetrievalGuard
    logger          *slog.Logger
}

// Create retrieval service
func NewRetrievalService(pool *Pool, embeddingClient *EmbeddingClient, tenantGuard *TenantGuard, retrievalGuard *RetrievalGuard) *RetrievalService

// Execute retrieval
func (s *RetrievalService) Search(ctx context.Context, req *SearchRequest) ([]*SearchResult, error)
```

**Retrieval Request:**
```go
type SearchRequest struct {
    Query       string          `json:"query"`           // Retrieval query
    TenantID    string          `json:"tenant_id"`       // Tenant ID
    TopK        int             `json:"top_k"`           // Number of results to return
    MinScore    float64         `json:"min_score"`       // Minimum similarity
    Plan        *RetrievalPlan  `json:"plan"`            // Retrieval strategy
    EnableTrace bool            `json:"enable_trace"`    // Enable tracing
    Trace       *RetrievalTrace `json:"trace,omitempty"` // Trace information
}
```

**Retrieval Strategy:**
```go
type RetrievalPlan struct {
    SearchKnowledge   bool    `json:"search_knowledge"`    // Search knowledge base
    SearchExperience  bool    `json:"search_experience"`   // Search experience
    SearchTools       bool    `json:"search_tools"`        // Search tools
    SearchTaskResults bool    `json:"search_task_results"` // Search task results

    KnowledgeWeight   float64 `json:"knowledge_weight"`    // Knowledge base weight (default 0.4)
    ExperienceWeight  float64 `json:"experience_weight"`   // Experience weight (default 0.3)
    ToolsWeight       float64 `json:"tools_weight"`        // Tools weight (default 0.2)
    TaskResultsWeight float64 `json:"task_results_weight"` // Task results weight (default 0.1)

    EnableQueryRewrite  bool `json:"enable_query_rewrite"`  // Enable query rewriting
    EnableKeywordSearch bool `json:"enable_keyword_search"` // Enable keyword search
    EnableTimeDecay     bool `json:"enable_time_decay"`     // Enable time decay

    TopK int `json:"top_k"` // Maximum results per source
}
```

**Retrieval Result:**
```go
type SearchResult struct {
    ID        string                 `json:"id"`
    Content   string                 `json:"content"`
    Score     float64                `json:"score"`
    Source    string                 `json:"source"`   // knowledge, experience, tool, task_result
    Type      string                 `json:"type"`     // Result type
    Metadata  map[string]interface{} `json:"metadata"` // Additional metadata
    CreatedAt time.Time              `json:"created_at"`
}
```

**Default Retrieval Strategy:**
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

### 4. Embedding Client

Provides embedding vector generation service supporting multiple embedding models.

**Core Features:**
- Embedding vector generation
- Batch processing
- Cache support
- Timeout protection
- Retry mechanism

**Interface:**
```go
type EmbeddingClient struct {
    serviceURL string
    model      string
    httpClient *http.Client
    cache      *cache.Cache
    enabled    bool
}

// Create embedding client
func NewEmbeddingClient(serviceURL, model string) *EmbeddingClient

// Generate embedding vector
func (c *EmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error)

// Batch generate embedding vectors
func (c *EmbeddingClient) EmbedBatch(ctx context.Context, texts []string) ([][]float64, error)

// Check if enabled
func (c *EmbeddingClient) IsEnabled() bool

// Enable client
func (c *EmbeddingClient) Enable()

// Disable client
func (c *EmbeddingClient) Disable()
```

## Usage Guide

### Basic Usage

#### 1. Initialize Connection Pool

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

#### 2. Create Repository

```go
repo := postgres.NewRepository(pool)
defer repo.Close()
```

#### 3. Use Knowledge Base

```go
import "goagent/internal/storage/postgres/repositories"
import "goagent/internal/storage/postgres/models"

// Create knowledge base repository
kbRepo := repositories.NewKnowledgeRepository(pool.DB(), pool.DB())

// Create knowledge chunk
chunk := &models.KnowledgeChunk{
    TenantID:        "tenant-001",
    Content:         "This is knowledge content",
    Embedding:       []float64{0.1, 0.2, ...}, // 1024-dimensional vector
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

#### 4. Vector Retrieval

```go
// Query vector
queryEmbedding := []float64{0.1, 0.2, ...}

// Vector retrieval
results, err := kbRepo.SearchByVector(ctx, queryEmbedding, "tenant-001", 10)
if err != nil {
    log.Fatal(err)
}

for _, result := range results {
    fmt.Printf("ID: %s, Content: %s\n", result.ID, result.Content)
}
```

#### 5. Use Retrieval Service

```go
import "goagent/internal/storage/postgres/services"
import "goagent/internal/storage/postgres/embedding"

// Create embedding client
embeddingClient := embedding.NewEmbeddingClient(
    "http://localhost:11434",
    "nomic-embed-text",
)

// Create tenant guard
tenantGuard := postgres.NewTenantGuard(pool)

// Create retrieval guard
retrievalGuard := postgres.NewRetrievalGuard()

// Create retrieval service
retrievalService := services.NewRetrievalService(
    pool,
    embeddingClient,
    tenantGuard,
    retrievalGuard,
)

// Execute retrieval
req := &services.SearchRequest{
    Query:    "What is RAG?",
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

### Advanced Usage

#### 1. Transaction Operations

```go
err := repo.Transaction(ctx, func(txRepo *postgres.Repository) error {
    // Create session
    session := &models.Session{
        ID:       "session-001",
        TenantID: "tenant-001",
        Status:   "active",
    }
    if err := txRepo.Session.Create(ctx, session); err != nil {
        return err
    }

    // Create recommendation result
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

#### 2. Batch Import

```go
chunks := []*models.KnowledgeChunk{
    {Content: "Content 1", ...},
    {Content: "Content 2", ...},
    {Content: "Content 3", ...},
}

err := kbRepo.CreateBatch(ctx, chunks)
```

#### 3. Key Management

```go
import "goagent/internal/storage/postgres/adapters"

// Create key repository
secretRepo := repositories.NewSecretRepository(pool.DB(), pool.DB())

// Import keys (supports JSON/YAML/CSV)
data := []byte(`
openai_api_key: sk-xxx123
anthropic_api_key: sk-yyy456
`)

adapter := &adapters.SecretAdapter{}
items, err := adapter.ParseFrom(data)
if err != nil {
    log.Fatal(err)
}

// Batch import
err = secretRepo.Import(ctx, "tenant-001", items)
```

#### 4. Custom Retrieval Strategy

```go
// Custom retrieval plan
customPlan := &services.RetrievalPlan{
    SearchKnowledge:     true,
    SearchExperience:    false,  // Don't search experience
    SearchTools:         false,  // Don't search tools
    KnowledgeWeight:     1.0,    // Only consider knowledge base
    EnableKeywordSearch: true,
    EnableTimeDecay:     false,  // Don't use time decay
    TopK:                3,
}

req := &services.SearchRequest{
    Query:    "Query content",
    TenantID: "tenant-001",
    Plan:     customPlan,
}

results, err := retrievalService.Search(ctx, req)
```

#### 5. Retrieval Tracing

```go
req := &services.SearchRequest{
    Query:       "Query content",
    TenantID:    "tenant-001",
    EnableTrace: true,  // Enable tracing
}

results, err := retrievalService.Search(ctx, req)

// View trace information
if req.Trace != nil {
    log.Printf("Original query: %s", req.Trace.OriginalQuery)
    log.Printf("Vector results: %d", req.Trace.VectorResults)
    log.Printf("Keyword results: %d", req.Trace.KeywordResults)
    log.Printf("Final results: %d", req.Trace.FinalResults)
    log.Printf("Execution time: %v", req.Trace.ExecutionTime)
}
```

## Best Practices

### 1. Connection Pool Configuration

```go
config := &postgres.Config{
    MaxOpenConns:    25,  // Adjust based on concurrency needs
    MaxIdleConns:    10,  // Maintain some idle connections
    ConnMaxLifetime: 5 * time.Minute,  // Refresh connections periodically
    QueryTimeout:    30 * time.Second,  // Set reasonable timeout
}
```

### 2. Error Handling

```go
results, err := kbRepo.SearchByVector(ctx, embedding, tenantID, limit)
if err != nil {
    if errors.Is(err, errors.ErrRecordNotFound) {
        // Record does not exist
        return nil, nil
    }
    // Other errors
    return nil, fmt.Errorf("search failed: %w", err)
}
```

### 3. Context Management

```go
// Set timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

results, err := kbRepo.SearchByVector(ctx, embedding, tenantID, limit)
```

### 4. Multi-Tenant Isolation

```go
// Create tenant guard
tenantGuard := postgres.NewTenantGuard(pool)

// Set tenant context at the start of each request
ctx = context.WithValue(ctx, "tenant_id", "tenant-001")
if err := tenantGuard.SetTenantContext(ctx, "tenant-001"); err != nil {
    return err
}

// Subsequent operations automatically apply tenant isolation
results, err := kbRepo.SearchByVector(ctx, embedding, "tenant-001", limit)
```

### 5. Batch Operations

```go
// Batch import is better than individual imports
chunks := make([]*models.KnowledgeChunk, 0, 100)
for _, doc := range documents {
    chunks = append(chunks, doc.Chunks...)
}
err := kbRepo.CreateBatch(ctx, chunks)
```

## Performance Optimization

### 1. Vector Retrieval Optimization

- Use appropriate TopK values (5-10)
- Set reasonable minimum similarity (0.6-0.8)
- Use time decay to avoid interference from outdated data

### 2. Caching Strategy

- Enable embedding vector caching
- Cache common query results
- Set reasonable cache TTL

### 3. Batch Processing

- Use batch import interfaces
- Batch generate embedding vectors
- Batch update operations

### 4. Index Optimization

- Ensure tenant_id has index
- Ensure embedding_status has index
- Ensure content_hash has index

## Error Handling

### Error Types

```go
// Core errors
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

### Error Handling Examples

```go
// Check error types
if errors.Is(err, errors.ErrRecordNotFound) {
    // Record does not exist
    return nil, nil
}

if errors.Is(err, errors.ErrRateLimitExceeded) {
    // Rate limit error, wait and retry
    time.Sleep(time.Second)
    return nil, err
}

if errors.Is(err, errors.ErrCircuitBreakerOpen) {
    // Circuit breaker open, fallback processing
    return fallbackSearch(ctx, query)
}
```

## Security Considerations

### 1. Data Encryption

- Sensitive data encrypted with AES-256-GCM
- Key management using dedicated Secret Repository
- Regular rotation of encryption keys

### 2. Multi-Tenant Isolation

- Use RLS policies for automatic isolation
- Tenant Guard dual protection
- Validate all cross-tenant access

### 3. SQL Injection Protection

- Use parameterized queries
- Avoid string concatenation for SQL
- Validate all input parameters

### 4. Access Control

- Implement principle of least privilege
- Regularly audit access logs
- Monitor abnormal access patterns

## Monitoring and Logging

### 1. Performance Monitoring

```go
// Enable retrieval tracing
req.EnableTrace = true
results, err := retrievalService.Search(ctx, req)

// Record performance metrics
log.Printf("Execution time: %v", req.Trace.ExecutionTime)
log.Printf("Vector results: %d", req.Trace.VectorResults)
log.Printf("Keyword results: %d", req.Trace.KeywordResults)
```

### 2. Error Logging

```go
// Record detailed error information
slog.Error("Failed to search knowledge base",
    "error", err,
    "tenant_id", tenantID,
    "query", query,
    "top_k", topK,
)
```

### 3. Metrics Collection

```go
// Record retrieval count
metrics.RecordSearch("knowledge", tenantID)

// Record retrieval duration
metrics.RecordSearchDuration("knowledge", time.Since(start))

// Record retrieval result count
metrics.RecordSearchResults("knowledge", len(results))
```

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/16/)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [RLS Policies](https://www.postgresql.org/docs/current/ddl-rowsecurity.html)
- [Vector Retrieval Best Practices](https://github.com/pgvector/pgvector#best-practices)