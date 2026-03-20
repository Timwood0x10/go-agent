# Embedding Service

Production-grade vector embedding service for AI agent framework using sentence-transformers.

## Features

- **High Performance**: Uses sentence-transformers with e5-large model (1024 dimensions)
- **Caching**: Redis-based caching with automatic normalization to avoid cache miss explosion
- **Batch Processing**: Efficient batch embedding support
- **Health Checks**: Built-in health check endpoint
- **Error Handling**: Comprehensive error handling and logging
- **Docker Ready**: Fully containerized with health checks

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP
       ↓
┌─────────────────────────────┐
│      FastAPI Service        │
│  ┌───────────────────────┐  │
│  │  Text Normalization   │  │
│  └───────────┬───────────┘  │
│              ↓               │
│  ┌───────────────────────┐  │
│  │  Cache Lookup (Redis) │  │
│  └───────────┬───────────┘  │
│         ↓ (miss)           │
│  ┌───────────────────────┐  │
│  │  SentenceTransformer  │  │
│  │  (e5-large model)     │  │
│  └───────────┬───────────┘  │
│              ↓               │
│  ┌───────────────────────┐  │
│  │  Cache Store (Redis)  │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

## Installation

### Prerequisites

- Python 3.11+
- uv (fast Python package manager)
- Ollama (for local embedding model)
- Redis (optional, for caching)
- Docker (optional, for containerized deployment)

### Quick Start with Setup Scripts

We provide automated setup scripts using uv and Ollama:

1. **Run the setup script** (installs uv, creates virtual environment, downloads model):
```bash
./setup.sh
```

2. **Start the service**:
```bash
./start.sh
```

3. **Stop the service**:
```bash
./stop.sh
```

### Manual Installation

1. **Install uv** (if not already installed):
```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

2. **Install Ollama** (if not already installed):
```bash
curl -fsSL https://ollama.com/install.sh | sh
```

3. **Create virtual environment with uv**:
```bash
uv venv
source .venv/bin/activate
```

4. **Install dependencies**:
```bash
uv pip install -r requirements.txt
```

5. **Set up environment variables**:
```bash
cp .env.example .env
# Edit .env with your configuration
```

6. **Download embedding model**:
```bash
ollama pull hf.co/ChristianAzinn/e5-large-v2-gguf:Q8_0
```

7. **Start Ollama** (in a separate terminal):
```bash
ollama serve
```

8. **Run the service**:
```bash
python app.py
```

### Docker Deployment

1. Build the image:
```bash
docker build -t embedding-service .
```

2. Run the container:
```bash
docker run -p 8000:8000 \
  -e MODEL_NAME=intfloat/e5-large \
  -e REDIS_URL=redis://host.docker.internal:6379 \
  embedding-service
```

## Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `MODEL_NAME` | HuggingFace model name | `intfloat/e5-large` |
| `EMBEDDING_DIM` | Embedding dimension | `1024` |
| `BATCH_SIZE` | Batch size for processing | `32` |
| `MAX_LENGTH` | Maximum text length | `512` |
| `REDIS_URL` | Redis connection URL | `redis://localhost:6379` |
| `CACHE_TTL` | Cache TTL in seconds | `86400` |
| `HOST` | Server host | `0.0.0.0` |
| `PORT` | Server port | `8000` |
| `LOG_LEVEL` | Logging level | `INFO` |

## API Endpoints

### Health Check

```bash
GET /health
```

Response:
```json
{
  "status": "healthy",
  "model": "intfloat/e5-large",
  "cache_enabled": true
}
```

### Single Embedding

```bash
POST /embed
Content-Type: application/json

{
  "text": "Your text here",
  "prefix": "query:"
}
```

Response:
```json
{
  "embedding": [0.123, 0.456, ...],
  "dimension": 1024,
  "cached": false
}
```

### Batch Embedding

```bash
POST /embed_batch
Content-Type: application/json

{
  "texts": ["Text 1", "Text 2", "Text 3"],
  "prefix": "passage:"
}
```

Response:
```json
{
  "embeddings": [
    [0.123, 0.456, ...],
    [0.789, 0.012, ...],
    [0.345, 0.678, ...]
  ],
  "dimension": 1024,
  "cached_count": 0
}
```

## Text Normalization

The service automatically normalizes text to avoid cache miss explosion:

1. Unicode normalization (NFKC)
2. Lowercase conversion
3. Trim whitespace
4. Remove extra spaces (including unicode spaces)
5. Remove control characters

This ensures that similar texts (e.g., "Hello World", "hello  world", "HELLO WORLD") 
generate the same cache key.

## Performance

- **Single embedding**: ~50-100ms (first call), ~5-10ms (cached)
- **Batch embedding**: ~100-200ms for 32 texts
- **Cache hit rate**: >80% with normalization

## Testing

Run the health check:
```bash
curl http://localhost:8000/health
```

Test single embedding:
```bash
curl -X POST http://localhost:8000/embed \
  -H "Content-Type: application/json" \
  -d '{"text": "Hello, world!", "prefix": "query:"}'
```

Test batch embedding:
```bash
curl -X POST http://localhost:8000/embed_batch \
  -H "Content-Type: application/json" \
  -d '{"texts": ["Text 1", "Text 2"], "prefix": "passage:"}'
```

## Monitoring

The service provides structured logging with the following information:

- Request/response times
- Cache hit/miss rates
- Error messages
- Model loading status

Logs are available in JSON format when running in production.

## Troubleshooting

### Model Not Loading

If the model fails to load, ensure you have:
- Sufficient disk space (models are ~2GB)
- Stable internet connection (for first download)
- Correct model name in configuration

### Redis Connection Failed

If Redis connection fails:
- Check Redis is running: `redis-cli ping`
- Verify REDIS_URL in configuration
- Check network connectivity

### Memory Issues

If you encounter memory issues:
- Reduce BATCH_SIZE
- Use smaller models (e.g., `intfloat/e5-small`)
- Increase available RAM

## License

This service is part of the StyleAgent project.