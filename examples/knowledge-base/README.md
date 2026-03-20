# Local Knowledge Base Example

This is a local knowledge base example based on the goagent storage module. It demonstrates how to quickly build a fully functional document retrieval and Q&A system using the high-level APIs of the storage module.

## Features

- 📄 **Document Import**: Import text documents with automatic chunking, vectorization, and storage
- 🔍 **Intelligent Retrieval**: Hybrid search combining vector retrieval and BM25 full-text search
- 💬 **Interactive Q&A**: Command-line interactive knowledge Q&A
- 📊 **Document Management**: List and delete imported documents
- 🏢 **Multi-Tenant Isolation**: Support for multiple independent tenant spaces
- ⚡ **High Performance**: Efficient vector retrieval based on pgvector

## System Requirements

### Required Components

1. **PostgreSQL 16 + pgvector extension**
   ```bash
   # Start PostgreSQL with Docker
   docker run -d \
     --name postgres-pgvector \
     -p 5433:5432 \
     -e POSTGRES_PASSWORD=postgres \
     -e POSTGRES_DB=goagent \
     pgvector/pgvector:pg16
   ```

2. **Ollama embedding service**
   ```bash
   # Install Ollama
   curl -fsSL https://ollama.com/install.sh | sh

   # Pull embedding model
   ollama pull nomic-embed-text

   # Start Ollama service
   ollama serve
   ```

### Verify Installation

```bash
# Check PostgreSQL
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT * FROM pg_extension WHERE extname='vector';"

# Check Ollama
curl http://localhost:11434/api/tags
```

## Quick Start

### One-Click Startup

```shell
./services/embedding/start.sh  # Start embedding service

go run main.go --save ./example.md

go run main.go --chat
```

### Detailed Setup

#### 1. Configure Database

Ensure PostgreSQL is running and properly configured:

```bash
# Check database connection
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT version();"
```

### 2. Configure Application

Edit `config.yaml` file to confirm database and embedding service configuration:

```yaml
database:
  host: localhost
  port: 5433
  user: postgres
  password: postgres
  database: goagent

embedding_service_url: http://localhost:11434
embedding_model: nomic-embed-text
```

### 3. Import Documents

```bash
# Import a document
go run main.go --save README.md

# Import other documents
go run main.go --save ../../docs/storage/api.md
go run main.go --save ../../README.md
```

### 4. Start Q&A

```bash
# Start interactive Q&A
go run main.go --chat
```

Then enter your questions:

```
You: What is RAG?

Found 3 relevant results:

[1] Score: 0.892
Content: RAG (Retrieval-Augmented Generation) is an AI system architecture combining retrieval and generation...
Source: README.md

[2] Score: 0.856
Content: Storage module supports hybrid search combining vector retrieval and BM25 full-text search...
Source: api.md

[3] Score: 0.743
Content: Vector retrieval uses pgvector for efficient similarity search...
Source: api.md

You: What are the features of the storage module?

Found 2 relevant results:

[1] Score: 0.934
Content: Storage module provides vector storage, retrieval, multi-tenant isolation, hybrid retrieval...
Source: api.md

[2] Score: 0.887
Content: Core capabilities include: vector storage and retrieval, multi-tenant isolation, hybrid retrieval, intelligent caching...
Source: README.md

You: exit
Goodbye!
```

### 5. Manage Documents

```bash
# List all documents
go run main.go --list

# Output example:
# Documents:
#   - ID: 1234567890abcdef, Source: README.md, Chunks: 12
#   - ID: abcdef1234567890, Source: api.md, Chunks: 45

# Delete specified document
go run main.go --delete 1234567890abcdef
```

## Usage Guide

### Command Line Options

```bash
go run main.go [options]

Options:
  --save <path>     Import document to knowledge base
  --chat            Start interactive Q&A mode
  --list            List all imported documents
  --delete <id>     Delete specified document
  --tenant <id>     Specify tenant ID (default: default)
  --config <path>   Config file path (default: config.yaml)
```

### Multi-Tenant Usage

```bash
# Create independent knowledge base spaces for different users/projects
go run main.go --save user1_doc.pdf --tenant user1
go run main.go --save user2_doc.pdf --tenant user2

# Each tenant can only see their own documents
go run main.go --list --tenant user1
go run main.go --chat --tenant user2
```

### Configuration Tuning

Edit `config.yaml` to optimize retrieval performance:

```yaml
knowledge:
  chunk_size: 500          # Smaller chunks improve precision
  chunk_overlap: 50        # Maintain context continuity
  top_k: 5                 # Return more candidate results
  min_score: 0.6           # Raise similarity threshold
```

**Parameter Description:**

- `chunk_size`: Document chunk size
  - Small values (200-300): More precise, but may lose context
  - Medium values (500-700): Balance precision and context (recommended)
  - Large values (1000+): More context, but lower precision

- `chunk_overlap`: Chunk overlap size
  - Usually set to 10-20% of chunk_size
  - Helps maintain semantic continuity

- `top_k`: Number of retrieval results
  - 3-5: Precise answers
  - 5-10: Comprehensive answers (recommended)
  - 10+: Broad exploration

- `min_score`: Minimum similarity threshold
  - 0.7-0.8: High relevance, fewer results
  - 0.6-0.7: Balance relevance and result count (recommended)
  - 0.5-0.6: More results, may include irrelevant content

## How It Works

### Import Flow

```
Document Read → Intelligent Chunking → Generate Embedding Vectors → Store in PostgreSQL + pgvector
```

1. **Document Read**: Read document content
2. **Intelligent Chunking**: Split into chunks based on configured size and overlap
3. **Generate Embedding**: Generate 1024-dimensional vectors for each chunk using Ollama service
4. **Vector Storage**: Store in PostgreSQL pgvector table

### Retrieval Flow

```
User Question → Vectorization → Hybrid Retrieval → Result Ranking → Return Relevant Content
```

1. **Question Vectorization**: Convert user question to vector
2. **Hybrid Retrieval**: Simultaneously perform vector retrieval and BM25 retrieval
3. **Result Ranking**: Merge and rank results using RRF algorithm
4. **Return Results**: Return Top-K most relevant knowledge chunks

## Advanced Usage

### Batch Import

```bash
# Batch import multiple documents
for file in docs/*.md; do
  go run main.go --save "$file" --tenant default
done
```

### Custom Chunking

Modify the `chunkDocument` method in `main.go` to implement custom chunking logic:

```go
func (kb *KnowledgeBase) chunkDocument(content string, chunkSize, chunkOverlap int) []*Chunk {
    // Implement custom chunking logic
    // - Chunk by paragraph
    // - Chunk by semantics
    // - Chunk by chapter
}
```

### LLM Integration

Extend the `StartChat` method to integrate LLM service for answer generation:

```go
func (kb *KnowledgeBase) StartChat(ctx context.Context, tenantID string) {
    // ... retrieval logic ...

    // Call LLM to generate answer
    answer := callLLM(question, results)
    fmt.Printf("\nAI: %s\n", answer)
}
```

## Architecture

### Core Components

```
KnowledgeBase (High-level API)
    ├── Pool (Database connection pool)
    ├── KnowledgeRepository (Knowledge base data access)
    ├── RetrievalService (Intelligent retrieval)
    ├── EmbeddingClient (Embedding service)
    ├── TenantGuard (Tenant isolation)
    └── RetrievalGuard (Rate limiting circuit breaker)
```

### Data Flow

**Document Import:**
```
Document → Chunking → Embedding Vector → PostgreSQL + pgvector
```

**Knowledge Q&A:**
```
Question → Retrieval Request → Hybrid Retrieval → Result Ranking → Return Relevant Content
```

## Performance Optimization

### 1. Database Optimization

```sql
-- Create indexes
CREATE INDEX idx_knowledge_tenant_id ON knowledge_chunks_1024(tenant_id);
CREATE INDEX idx_knowledge_document_id ON knowledge_chunks_1024(document_id);
CREATE INDEX idx_knowledge_embedding_status ON knowledge_chunks_1024(embedding_status);
```

### 2. Cache Configuration

```yaml
embedding_service_url: http://localhost:11434
embedding_model: nomic-embed-text
```

### 3. Batch Processing

When importing a large number of documents, process in batches:

```bash
# Import 10 documents per batch
find docs/ -name "*.md" | head -n 10 | xargs -I {} go run main.go --save "{}"
```

## Troubleshooting

### Issue 1: Database Connection Failed

```
Error: create database pool: connection refused
```

**Solution:**
- Check if PostgreSQL is running: `docker ps | grep postgres`
- Check if port is correct: `netstat -an | grep 5433`
- Check database configuration in config file

### Issue 2: Embedding Service Unavailable

```
Error: Failed to embed chunk: connection refused
```

**Solution:**
- Check if Ollama is running: `ps aux | grep ollama`
- Check if model is downloaded: `ollama list`
- Restart Ollama service: `ollama serve`

### Issue 3: Import Timeout

```
Import timeout (5 minutes exceeded)
```

**Solution:**
- Check embedding service response speed
- Reduce document size or increase chunk count
- Check network connection
- Check which chunk timed out (logs show details)

### Issue 4: Search Timeout

```
Search timeout. Please try again.
```

**Solution:**
- Check database connection status
- Check if embedding service is normal
- Reduce search result count (lower top_k value)
- Check for high concurrent requests

### Issue 5: Program Freezes

**Symptoms**: Program unresponsive, cannot exit

**Prevention Measures:**
- All operations have timeout protection (import 5 minutes, search 30 seconds)
- Each chunk has independent timeout (60 seconds)
- Use Ctrl+C to interrupt program
- Non-blocking input (bufio.Scanner)

**Solution:**
- Press Ctrl+C to interrupt program
- Check if Ollama service is stuck: `curl http://localhost:11434/api/tags`
- Check if database is stuck: `docker exec -it postgres-pgvector psql -U postgres -d goagent -c "SELECT 1;"`
- Restart related services

### Issue 6: Poor Retrieval Results

**Solution:**
- Adjust `chunk_size` and `chunk_overlap`
- Lower `min_score` threshold
- Increase `top_k` value
- Try different embedding models

### Issue 7: pgvector Not Installed

```
Error: type "vector" does not exist
```

**Solution:**
```bash
# Install pgvector extension in PostgreSQL
docker exec -it postgres-pgvector psql -U postgres -d goagent -c "CREATE EXTENSION vector;"
```

## Extension Features

### 1. Support More Document Formats

Integrate PDF, Word and other document parsing libraries:

```go
import "github.com/unidoc/unipdf/v3/extractor"

func loadPDF(path string) (string, error) {
    // PDF parsing logic
}
```

### 2. Add Document Metadata

Add more metadata to documents:

```go
type DocumentMetadata struct {
    Title       string    `json:"title"`
    Author      string    `json:"author"`
    CreatedAt   time.Time `json:"created_at"`
    Tags        []string  `json:"tags"`
    Category    string    `json:"category"`
}
```

### 3. Implement Document Version Management

Support version control and updates for documents:

```go
func (kb *KnowledgeBase) UpdateDocument(ctx context.Context, tenantID, docID string) error {
    // Delete old version
    kb.DeleteDocument(ctx, tenantID, docID)
    // Import new version
    kb.ImportDocuments(ctx, tenantID, docPath)
}
```

## Tech Stack

- **Language**: Go 1.21+
- **Database**: PostgreSQL 16 + pgvector
- **Embedding Service**: Ollama (nomic-embed-text)
- **Configuration**: YAML
- **Retrieval**: Vector similarity + BM25 + RRF

## References

- [Storage API Documentation](../../docs/storage/api.md)
- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [Ollama Documentation](https://github.com/ollama/ollama)
- [RAG Best Practices](https://docs.anthropic.com/claude/docs/retrieval-augmented-generation)

## License

MIT License

## Contributing

Issues and Pull Requests are welcome!