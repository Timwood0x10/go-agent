# pgvector Type Conversion in Go

## Problem Statement

When using PostgreSQL with pgvector extension in Go applications, we encountered several data type conversion issues due to limitations in the `github.com/lib/pq` driver:

1. **Vector Conversion**: pq driver cannot directly scan `[]float64` to/from pgvector's VECTOR type
2. **JSONB Conversion**: pq driver cannot directly scan `map[string]interface{}` to/from PostgreSQL JSONB type  
3. **UUID NULL Handling**: Empty string UUIDs cause "invalid input syntax" errors

## Root Cause Analysis

### pq Driver Limitations

The pq driver (Go's PostgreSQL driver) has inherent limitations:
- No native support for custom PostgreSQL types like VECTOR
- Direct mapping only for basic SQL types (TEXT, INT, FLOAT, etc.)
- Binary format decoding limited to standard types

### pgvector Binary Format

pgvector stores vectors in binary format that pq cannot parse:
```go
// ❌ This fails: pq stores as []uint8, cannot scan to []float64
db.Scan(&chunk.Embedding) // []float64
```

## Solution Implementation

### 1. Vector Type Conversion

**Encoding (Go → PostgreSQL)**:
```go
func float64ToVectorString(vec []float64) string {
    if len(vec) == 0 {
        return "[]"
    }
    
    strs := make([]string, len(vec))
    for i, v := range vec {
        strs[i] = fmt.Sprintf("%f", v)
    }
    return "[" + strings.Join(strs, ",") + "]"
}

// Usage in INSERT/UPDATE
embeddingStr := float64ToVectorString(chunk.Embedding)
query := `INSERT INTO table (embedding) VALUES ($1::vector)`
db.Exec(query, embeddingStr)
```

**Decoding (PostgreSQL → Go)**:
```go
func parseVectorString(vecStr string) ([]float64, error) {
    vecStr = strings.Trim(vecStr, "[]")
    if vecStr == "" {
        return []float64{}, nil
    }
    
    parts := strings.Split(vecStr, ",")
    result := make([]float64, len(parts))
    for i, part := range parts {
        _, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &result[i])
        if err != nil {
            return nil, err
        }
    }
    return result, nil
}

// Usage in SELECT
query := `SELECT embedding::text FROM table`
var embeddingStr string
db.Scan(&embeddingStr)
chunk.Embedding, _ = parseVectorString(embeddingStr)
```

### 2. JSONB Type Conversion

**Encoding**:
```go
metadataJSON, err := json.Marshal(chunk.Metadata)
query := `INSERT INTO table (metadata) VALUES ($1)`
db.Exec(query, metadataJSON)
```

**Decoding**:
```go
query := `SELECT metadata::text FROM table`
var metadataStr string
db.Scan(&metadataStr)
json.Unmarshal([]byte(metadataStr), &chunk.Metadata)
```

### 3. NULL Handling for Optional Fields

```go
// Document ID (optional, can be NULL)
var documentID interface{}
if chunk.DocumentID != "" {
    documentID = chunk.DocumentID
} else {
    documentID = nil
}

// Embedding (optional, can be NULL)
var embeddingStr interface{}
if len(chunk.Embedding) == 0 {
    embeddingStr = nil
} else {
    embeddingStr = float64ToVectorString(chunk.Embedding)
}

// Reading nullable fields
var documentID sql.NullString
db.Scan(&documentID)
if documentID.Valid {
    chunk.DocumentID = documentID.String
}
```

## Architecture Decision

**Repository Layer Responsibility**:
- Type conversion happens in Repository layer (not Model layer)
- Models remain pure Go types (`[]float64`, `map[string]interface{}`)
- Repository acts as translation layer between Go types and SQL types

**Benefits**:
- Clean separation of concerns
- Models stay database-agnostic
- Easy to switch database backends (only change Repository)

## Performance Considerations

### Text Format vs Binary Format

**Trade-off Analysis**:
- **Text Format** (current solution): 
  - ✅ Compatible with pq driver
  - ✅ Human-readable in database
  - ❌ Slightly slower than binary
  - ❌ More storage space

- **Binary Format** (hypothetical):
  - ✅ Faster and more compact
  - ❌ Requires custom pq driver or pgx driver
  - ❌ More complex implementation

**Decision**: Use text format for simplicity and compatibility.

### Impact on Performance

- **Encoding**: O(n) where n is vector dimension (1024)
- **Decoding**: O(n) string parsing
- **Overall**: Negligible impact compared to network latency and embedding computation

## Alternative Solutions Considered

### 1. Use pgx Driver
- ✅ Better type support
- ✅ Binary format support
- ❌ Requires significant refactoring
- ❌ Different API from pq

### 2. Custom pq Type Scanner
- ✅ Native support in pq
- ❌ Complex to implement
- ❌ Limited documentation

### 3. Store as Arrays Instead of pgvector
- ✅ Native PostgreSQL array support
- ❌ No vector operations (cosine similarity, etc.)
- ❌ No vector indexing

**Decision**: Current text conversion approach provides best balance of simplicity, compatibility, and functionality.

## Best Practices

### 1. Always Use Text Cast in SELECT
```sql
-- ✅ Correct: cast to text for pq compatibility
SELECT embedding::text FROM knowledge_chunks

-- ❌ Wrong: pq cannot parse binary format
SELECT embedding FROM knowledge_chunks
```

### 2. Use Type Cast in INSERT/UPDATE
```sql
-- ✅ Correct: explicitly cast to vector type
INSERT INTO table (embedding) VALUES ($1::vector)

-- ❌ Wrong: implicit conversion may fail
INSERT INTO table (embedding) VALUES ($1)
```

### 3. Handle Empty Vectors Gracefully
```go
// Empty vectors should be stored as NULL, not "[]"
if len(chunk.Embedding) == 0 {
    embeddingStr = nil
} else {
    embeddingStr = float64ToVectorString(chunk.Embedding)
}
```

## Migration Notes

### When Implementing Similar Patterns

For other repositories (Experience, Tool, TaskResult):

1. **Add Helper Functions**:
```go
// In each repository file
func float64ToVectorString(vec []float64) string { ... }
func parseVectorString(vecStr string) ([]float64, error) { ... }
```

2. **Update Create Methods**:
- Convert embedding to string before INSERT
- Use `::vector` type cast in SQL

3. **Update GetByID/Select Methods**:
- Select embedding as `::text`
- Parse string back to `[]float64`

4. **Update Update Methods**:
- Same as Create methods

## Testing Strategy

### Unit Tests
- Test conversion functions with various inputs
- Test edge cases (empty vectors, nil values)
- Test error handling

### Integration Tests
- Test full CRUD operations with real database
- Test vector similarity search
- Test concurrent operations

### Performance Tests
- Benchmark conversion functions
- Measure impact on insert/select operations
- Compare with binary format (if implemented)

## Lessons Learned

1. **Database Drivers Have Limitations**: Don't assume perfect type support
2. **Text Conversion is Reliable**: When in doubt, use text format
3. **Repository Layer is Right Place**: Keep models pure, handle conversion in Repository
4. **Test Edge Cases**: NULL values, empty arrays, special characters
5. **Document Decisions**: Record why specific solutions were chosen

## References

- [pgvector Documentation](https://github.com/pgvector/pgvector)
- [lib/pq Driver](https://github.com/lib/pq)
- [pgx Driver](https://github.com/jackc/pgx) - Alternative with better type support
- [PostgreSQL JSONB](https://www.postgresql.org/docs/current/datatype-json.html)

---
