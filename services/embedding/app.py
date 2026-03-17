"""
Embedding Service - Provides vector embeddings for AI agent framework.
This service uses sentence-transformers with e5-large model for semantic search.
"""
import os
import json
import hashlib
import logging
from typing import List, Optional

import redis
from fastapi import FastAPI, HTTPException, BackgroundTasks
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field
from sentence_transformers import SentenceTransformer
from dotenv import load_dotenv

# Load environment variables
load_dotenv()

# Configure logging
logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO"),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Embedding Service",
    description="Vector embedding service for AI agent framework",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Global variables
model: Optional[SentenceTransformer] = None
redis_client: Optional[redis.Redis] = None
MODEL_NAME = os.getenv("MODEL_NAME", "intfloat/e5-large")
EMBEDDING_DIM = int(os.getenv("EMBEDDING_DIM", "1024"))
REDIS_URL = os.getenv("REDIS_URL", "redis://localhost:6379")
CACHE_TTL = int(os.getenv("CACHE_TTL", "86400"))  # 24 hours
BATCH_SIZE = int(os.getenv("BATCH_SIZE", "32"))


# Data models
class EmbedRequest(BaseModel):
    text: str = Field(..., description="Text to embed")
    prefix: str = Field("", description="Prefix for the text (e.g., 'query:', 'passage:')")


class EmbedResponse(BaseModel):
    embedding: List[float] = Field(..., description="Vector embedding")
    dimension: int = Field(..., description="Embedding dimension")
    cached: bool = Field(default=False, description="Whether result was cached")


class BatchEmbedRequest(BaseModel):
    texts: List[str] = Field(..., description="List of texts to embed")
    prefix: str = Field("", description="Prefix for all texts")


class BatchEmbedResponse(BaseModel):
    embeddings: List[List[float]] = Field(..., description="List of vector embeddings")
    dimension: int = Text(..., description="Embedding dimension")
    cached_count: int = Field(default=0, description("Number of cached results")


class HealthResponse(BaseModel):
    status: str = Field(..., description="Service health status")
    model: str = Field(..., description="Loaded model name")
    cache_enabled: bool = Field(..., description="Whether cache is enabled")


# Startup event
@app.on_event("startup")
async def startup_event():
    """Initialize model and connections on startup."""
    global model, redis_client
    
    logger.info("Starting embedding service...")
    
    try:
        # Load embedding model
        logger.info(f"Loading model: {MODEL_NAME}")
        model = SentenceTransformer(MODEL_NAME)
        logger.info(f"Model loaded successfully, dimension: {model.get_sentence_embedding_dimension()}")
        
        # Initialize Redis cache
        try:
            redis_client = redis.from_url(REDIS_URL, decode_responses=True)
            redis_client.ping()
            logger.info("Redis cache connected successfully")
        except Exception as e:
            logger.warning(f"Redis connection failed, running without cache: {e}")
        
        logger.info("Embedding service started successfully")
        
    except Exception as e:
        logger.error(f"Failed to start embedding service: {e}")
        raise


# Shutdown event
@app.on_event("shutdown")
async def shutdown_event():
    """Clean up resources on shutdown."""
    global redis_client
    
    logger.info("Shutting down embedding service...")
    
    if redis_client:
        redis_client.close()
        logger.info("Redis connection closed")
    
    logger.info("Embedding service stopped")


# API endpoints
@app.get("/health", response_model=HealthResponse)
async def health_check():
    """Health check endpoint."""
    return HealthResponse(
        status="healthy",
        model=MODEL_NAME,
        cache_enabled=redis_client is not None
    )


@app.post("/embed", response_model=EmbedResponse)
async def embed(request: EmbedRequest):
    """
    Generate vector embedding for a single text.
    
    This endpoint supports model-specific prefixes (e.g., 'query:', 'passage:') for e5 models.
    """
    if not model:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    # Check cache first
    cache_key = generate_cache_key(request.text, request.prefix)
    if redis_client:
        try:
            cached_data = redis_client.get(cache_key)
            if cached_data:
                embedding = json.loads(cached_data)
                return EmbedResponse(
                    embedding=embedding,
                    dimension=len(embedding),
                    cached=True
                )
        except Exception as e:
            logger.warning(f"Cache lookup failed: {e}")
    
    # Add prefix for e5 models
    text_with_prefix = request.prefix + request.text if request.prefix else request.text
    
    # Generate embedding
    embedding = model.encode(text_with_prefix).tolist()
    
    # Cache the result
    if redis_client:
        try:
            redis_client.setex(
                cache_key,
                CACHE_TTL,
                json.dumps(embedding)
            )
        except Exception as e:
            logger.warning(f"Cache write failed: {e}")
    
    return EmbedResponse(
        embedding=embedding,
        dimension=len(embedding),
        cached=False
    )


@app.post("/embed_batch", response_model=BatchEmbedResponse)
async def embed_batch(request: BatchEmbedRequest):
    """
    Generate vector embeddings for multiple texts.
    
    This is more efficient than multiple single embed calls.
    """
    if not model:
        raise HTTPException(status_code=503, detail="Model not loaded")
    
    texts_with_prefix = [
        request.prefix + text if request.prefix else text
        for text in request.texts
    ]
    
    # Generate embeddings
    embeddings = model.encode(texts_with_prefix).tolist()
    
    # Try to cache results
    cached_count = 0
    if redis_client:
        for i, text in enumerate(request.texts):
            cache_key = generate_cache_key(text, request.prefix)
            try:
                cached_data = redis_client.get(cache_key)
                if cached_data:
                    cached_count += 1
                else:
                    # Cache non-cached results
                    redis_client.setex(
                        cache_key,
                        CACHE_TTL,
                        json.dumps(embeddings[i])
                    )
            except Exception as e:
                logger.warning(f"Cache operation failed for text {i}: {e}")
    
    return BatchEmbedResponse(
        embeddings=embeddings,
        dimension=len(embeddings[0]) if embeddings else 0,
        cached_count=cached_count
    )


def generate_cache_key(text: str, prefix: str) -> str:
    """
    Generate cache key for storing embeddings.
    
    Args:
        text: Text content to embed.
        prefix: Model-specific prefix.
    
    Returns:
        Cache key as hex string.
    """
    key_data = f"{prefix}|{text}|{MODEL_NAME}"
    hash_obj = hashlib.sha256(key_data.encode())
    return f"embed:{hash_obj.hexdigest()[:16]}"


if __name__ == "__main__":
    import uvicorn
    
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", "8000"))
    
    logger.info(f"Starting embedding service on {host}:{port}")
    uvicorn.run(app, host=host, port=port)