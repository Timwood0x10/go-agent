"""
Configuration module for embedding service.
"""
import os
from typing import Optional
from dotenv import load_dotenv

load_dotenv()


class Config:
    """Configuration for embedding service."""
    
    # Model configuration
    MODEL_NAME: str = os.getenv("MODEL_NAME", "qwen3-embedding:0.6b")
    EMBEDDING_DIM: int = int(os.getenv("EMBEDDING_DIM", "1024"))
    BATCH_SIZE: int = int(os.getenv("BATCH_SIZE", "32"))
    MAX_LENGTH: int = int(os.getenv("MAX_LENGTH", "512"))
    
    # Redis configuration
    REDIS_URL: str = os.getenv("REDIS_URL", "redis://localhost:6379")
    REDIS_PASSWORD: Optional[str] = os.getenv("REDIS_PASSWORD")
    CACHE_TTL: int = int(os.getenv("CACHE_TTL", "86400"))  # 24 hours
    
    # Server configuration
    HOST: str = os.getenv("HOST", "0.0.0.0")
    PORT: int = int(os.getenv("PORT", "8000"))
    LOG_LEVEL: str = os.getenv("LOG_LEVEL", "INFO")
    
    # Timeout configuration
    REQUEST_TIMEOUT: int = int(os.getenv("REQUEST_TIMEOUT", "30"))
    
    @classmethod
    def validate(cls) -> None:
        """Validate configuration values."""
        if cls.EMBEDDING_DIM <= 0:
            raise ValueError("EMBEDDING_DIM must be positive")
        
        if cls.BATCH_SIZE <= 0:
            raise ValueError("BATCH_SIZE must be positive")
        
        if cls.MAX_LENGTH <= 0:
            raise ValueError("MAX_LENGTH must be positive")
        
        if cls.CACHE_TTL <= 0:
            raise ValueError("CACHE_TTL must be positive")


# Validate configuration on import
Config.validate()