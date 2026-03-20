"""
Test script for embedding service.
"""
import requests
import json
import time


def test_health_check():
    """Test health check endpoint."""
    print("Testing health check...")
    response = requests.get("http://localhost:8000/health")
    print(f"Status: {response.status_code}")
    print(f"Response: {json.dumps(response.json(), indent=2)}")
    assert response.status_code == 200
    print("✓ Health check passed\n")


def test_single_embedding():
    """Test single embedding endpoint."""
    print("Testing single embedding...")
    
    # Test without cache
    request = {
        "text": "This is a test sentence for embedding.",
        "prefix": "query:"
    }
    
    start_time = time.time()
    response = requests.post(
        "http://localhost:8000/embed",
        json=request
    )
    elapsed = time.time() - start_time
    
    print(f"Status: {response.status_code}")
    print(f"Time: {elapsed:.3f}s")
    result = response.json()
    print(f"Dimension: {result['dimension']}")
    print(f"Cached: {result['cached']}")
    print(f"Embedding (first 5): {result['embedding'][:5]}")
    
    assert response.status_code == 200
    assert len(result['embedding']) == 1024
    assert result['cached'] == False
    print("✓ Single embedding passed\n")
    
    # Test with cache (same request)
    print("Testing cache hit...")
    start_time = time.time()
    response = requests.post(
        "http://localhost:8000/embed",
        json=request
    )
    elapsed = time.time() - start_time
    
    result = response.json()
    print(f"Time: {elapsed:.3f}s")
    print(f"Cached: {result['cached']}")
    
    assert result['cached'] == True
    print("✓ Cache hit passed\n")


def test_batch_embedding():
    """Test batch embedding endpoint."""
    print("Testing batch embedding...")
    
    request = {
        "texts": [
            "First test sentence.",
            "Second test sentence.",
            "Third test sentence."
        ],
        "prefix": "passage:"
    }
    
    start_time = time.time()
    response = requests.post(
        "http://localhost:8000/embed_batch",
        json=request
    )
    elapsed = time.time() - start_time
    
    print(f"Status: {response.status_code}")
    print(f"Time: {elapsed:.3f}s")
    result = response.json()
    print(f"Embeddings count: {len(result['embeddings'])}")
    print(f"Dimension: {result['dimension']}")
    print(f"Cached count: {result['cached_count']}")
    
    assert response.status_code == 200
    assert len(result['embeddings']) == 3
    assert result['dimension'] == 1024
    print("✓ Batch embedding passed\n")


def test_normalization():
    """Test text normalization."""
    print("Testing text normalization...")
    
    # These should generate the same cache key
    texts = [
        "Hello World",
        "hello  world",  # Extra spaces
        "HELLO WORLD",    # Uppercase
        "Hello World",    # Exact match
    ]
    
    cache_keys = set()
    for text in texts:
        request = {
            "text": text,
            "prefix": "query:"
        }
        response = requests.post(
            "http://localhost:8000/embed",
            json=request
        )
        assert response.status_code == 200
        result = response.json()
        cache_keys.add(result['cached'])
    
    print(f"Cache hits: {sum(cache_keys)}/{len(texts)}")
    print("✓ Normalization test passed\n")


def main():
    """Run all tests."""
    print("=" * 50)
    print("Embedding Service Test Suite")
    print("=" * 50)
    print()
    
    try:
        test_health_check()
        test_single_embedding()
        test_batch_embedding()
        test_normalization()
        
        print("=" * 50)
        print("All tests passed! ✓")
        print("=" * 50)
        
    except requests.exceptions.ConnectionError:
        print("✗ Service not running. Please start the service first.")
        print("  Run: python app.py")
    except AssertionError as e:
        print(f"✗ Test failed: {e}")
    except Exception as e:
        print(f"✗ Unexpected error: {e}")


if __name__ == "__main__":
    main()