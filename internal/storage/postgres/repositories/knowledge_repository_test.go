// Package repositories provides comprehensive tests for KnowledgeRepository.
package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storage_models "goagent/internal/storage/postgres/models"
)

// createTestEmbedding creates a 1024-dimensional test embedding vector.
func createTestEmbedding() []float64 {
	embedding := make([]float64, 1024)
	for i := range embedding {
		embedding[i] = float64(i) / 1024.0
	}
	return embedding
}

// TestKnowledgeRepository_Create tests creating a single knowledge chunk.
func TestKnowledgeRepository_Create(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		ID:               "", // Let database generate UUID
		TenantID:         "tenant-1",
		Content:          "test content for knowledge",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		Source:           "test-source",
		Metadata:         map[string]interface{}{"key": "value"},
		DocumentID:       "", // Remove this field, let it be NULL
		ChunkIndex:       0,
		ContentHash:      "hash-123",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.ID)
}

// TestKnowledgeRepository_Create_Duplicate tests handling of duplicate content.
func TestKnowledgeRepository_Create_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "duplicate content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-duplicate",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// First create
	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	firstID := chunk.ID

	// Second create with same content hash (should update access count)
	err = repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.Equal(t, firstID, chunk.ID) // ID should remain the same
}

// TestKnowledgeRepository_CreateBatch tests creating multiple knowledge chunks.
func TestKnowledgeRepository_CreateBatch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunks := []*storage_models.KnowledgeChunk{
		{
			TenantID:         "tenant-1",
			Content:          "batch content 1",
			Embedding:        createTestEmbedding(),
			EmbeddingModel:   "e5-large",
			EmbeddingVersion: 1,
			EmbeddingStatus:  storage_models.EmbeddingStatusPending,
			SourceType:       "document",
			ContentHash:      "hash-batch-1",
			AccessCount:      0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			TenantID:         "tenant-1",
			Content:          "batch content 2",
			Embedding:        createTestEmbedding(),
			EmbeddingModel:   "e5-large",
			EmbeddingVersion: 1,
			EmbeddingStatus:  storage_models.EmbeddingStatusPending,
			SourceType:       "document",
			ContentHash:      "hash-batch-2",
			AccessCount:      0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			TenantID:         "tenant-1",
			Content:          "batch content 3",
			Embedding:        createTestEmbedding(),
			EmbeddingModel:   "e5-large",
			EmbeddingVersion: 1,
			EmbeddingStatus:  storage_models.EmbeddingStatusPending,
			SourceType:       "document",
			ContentHash:      "hash-batch-3",
			AccessCount:      0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	err := repo.CreateBatch(ctx, chunks)
	require.NoError(t, err)

	for _, chunk := range chunks {
		assert.NotEmpty(t, chunk.ID)
	}
}

// TestKnowledgeRepository_CreateBatch_Empty tests creating empty batch.
func TestKnowledgeRepository_CreateBatch_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	err := repo.CreateBatch(ctx, []*storage_models.KnowledgeChunk{})
	require.NoError(t, err)
}

// TestKnowledgeRepository_CreateBatch_NoDBPool tests creating batch without DB pool.
func TestKnowledgeRepository_CreateBatch_NoDBPool(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)

	repo := NewKnowledgeRepository(db, nil) // No DB pool
	ctx := context.Background()

	chunks := []*storage_models.KnowledgeChunk{
		{
			TenantID:    "tenant-1",
			Content:     "test content",
			ContentHash: "hash-123",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}

	err := repo.CreateBatch(ctx, chunks)
	assert.Error(t, err)
}

// TestKnowledgeRepository_GetByID tests retrieving a knowledge chunk by ID.
func TestKnowledgeRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	// Create a chunk
	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "get by id test content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-getbyid",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)

	// Retrieve by ID
	retrieved, err := repo.GetByID(ctx, chunk.ID)
	require.NoError(t, err)
	assert.Equal(t, chunk.ID, retrieved.ID)
	assert.Equal(t, chunk.TenantID, retrieved.TenantID)
	assert.Equal(t, chunk.Content, retrieved.Content)
	assert.Equal(t, chunk.ContentHash, retrieved.ContentHash)
}

// TestKnowledgeRepository_GetByID_NotFound tests retrieving non-existent chunk.
func TestKnowledgeRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, "non-existent-id")
	assert.Error(t, err)
}

// TestKnowledgeRepository_Update tests updating a knowledge chunk.
func TestKnowledgeRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	// Create a chunk
	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "original content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-update",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)

	// Update the chunk
	chunk.Content = "updated content"
	chunk.EmbeddingStatus = storage_models.EmbeddingStatusCompleted
	chunk.AccessCount = 5

	err = repo.Update(ctx, chunk)
	require.NoError(t, err)

	// Verify update
	retrieved, err := repo.GetByID(ctx, chunk.ID)
	require.NoError(t, err)
	assert.Equal(t, "updated content", retrieved.Content)
	assert.Equal(t, storage_models.EmbeddingStatusCompleted, retrieved.EmbeddingStatus)
	assert.Equal(t, 5, retrieved.AccessCount)
}

// TestKnowledgeRepository_Delete tests deleting a knowledge chunk.
func TestKnowledgeRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	// Create a chunk
	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "delete test content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-delete",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	id := chunk.ID

	// Delete the chunk
	err = repo.Delete(ctx, id)
	require.NoError(t, err)

	// Verify deletion
	_, err = repo.GetByID(ctx, id)
	assert.Error(t, err)
}

// TestKnowledgeRepository_Delete_NotFound tests deleting non-existent chunk.
func TestKnowledgeRepository_Delete_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	err := repo.Delete(ctx, "non-existent-id")
	assert.Error(t, err)
}

// TestKnowledgeRepository_Search tests vector search functionality.
func TestKnowledgeRepository_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	// Create multiple chunks
	chunks := []*storage_models.KnowledgeChunk{
		{
			TenantID:         "tenant-1",
			Content:          "machine learning algorithms",
			Embedding:        createTestEmbedding(),
			EmbeddingModel:   "e5-large",
			EmbeddingVersion: 1,
			EmbeddingStatus:  storage_models.EmbeddingStatusCompleted,
			SourceType:       "document",
			ContentHash:      "hash-ml",
			AccessCount:      0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			TenantID:         "tenant-1",
			Content:          "deep learning neural networks",
			Embedding:        createTestEmbedding(),
			EmbeddingModel:   "e5-large",
			EmbeddingVersion: 1,
			EmbeddingStatus:  storage_models.EmbeddingStatusCompleted,
			SourceType:       "document",
			ContentHash:      "hash-dl",
			AccessCount:      0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}

	for _, chunk := range chunks {
		err := repo.Create(ctx, chunk)
		require.NoError(t, err)
	}

	// Note: Search functionality requires actual embedding values and vector operations
	// This is a placeholder test to demonstrate the test structure
	// In real implementation, you would test with actual search queries
}

// TestKnowledgeRepository_Create_WithNilFields tests creating chunk with nil fields.
func TestKnowledgeRepository_Create_WithNilFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "test content",
		Embedding:        nil,
		EmbeddingModel:   "",
		EmbeddingVersion: 0,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-nil",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.ID)
}

// TestKnowledgeRepository_Create_WithEmptyFields tests creating chunk with empty fields.
func TestKnowledgeRepository_Create_WithEmptyFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "test content",
		Embedding:        []float64{},
		EmbeddingModel:   "",
		EmbeddingVersion: 0,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-empty",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.ID)
}

// TestKnowledgeRepository_Create_WithComplexMetadata tests creating chunk with complex metadata.
func TestKnowledgeRepository_Create_WithComplexMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          "test content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		Metadata: map[string]interface{}{
			"string":  "value",
			"number":  123,
			"bool":    true,
			"array":   []string{"a", "b", "c"},
			"object":  map[string]interface{}{"nested": "value"},
		},
		ContentHash: "hash-complex",
		AccessCount: 0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.ID)

	// Retrieve and verify metadata
	retrieved, err := repo.GetByID(ctx, chunk.ID)
	require.NoError(t, err)
	assert.NotNil(t, retrieved.Metadata)
}

// TestKnowledgeRepository_Create_WithLargeContent tests creating chunk with large content.
func TestKnowledgeRepository_Create_WithLargeContent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	largeContent := make([]byte, 10000)
	for i := range largeContent {
		largeContent[i] = 'a'
	}

	chunk := &storage_models.KnowledgeChunk{
		TenantID:         "tenant-1",
		Content:          string(largeContent),
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-large",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Create(ctx, chunk)
	require.NoError(t, err)
	assert.NotEmpty(t, chunk.ID)
}

// TestKnowledgeRepository_Update_NonExistent tests updating non-existent chunk.
func TestKnowledgeRepository_Update_NonExistent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	chunk := &storage_models.KnowledgeChunk{
		ID:               "non-existent-id",
		TenantID:         "tenant-1",
		Content:          "update test content",
		Embedding:        createTestEmbedding(),
		EmbeddingModel:   "e5-large",
		EmbeddingVersion: 1,
		EmbeddingStatus:  storage_models.EmbeddingStatusPending,
		SourceType:       "document",
		ContentHash:      "hash-update-nonexistent",
		AccessCount:      0,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	err := repo.Update(ctx, chunk)
	assert.Error(t, err)
}

// TestKnowledgeRepository_ConcurrentOperations tests concurrent repository operations.
func TestKnowledgeRepository_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := getTestDB(t)
	defer closeTestDB(t, db)
	defer cleanupTestDB(t, db)

	repo := NewKnowledgeRepository(db, db)
	ctx := context.Background()

	// Create chunks concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			chunk := &storage_models.KnowledgeChunk{
				TenantID:         "tenant-1",
				Content:          fmt.Sprintf("concurrent content %d", index),
				Embedding:        createTestEmbedding(),
				EmbeddingModel:   "e5-large",
				EmbeddingVersion: 1,
				EmbeddingStatus:  storage_models.EmbeddingStatusPending,
				SourceType:       "document",
				ContentHash:      fmt.Sprintf("hash-concurrent-%d", index),
				AccessCount:      0,
				CreatedAt:        time.Now(),
				UpdatedAt:        time.Now(),
			}

			err := repo.Create(ctx, chunk)
			assert.NoError(t, err, "concurrent create failed for chunk %d", index)
			done <- true
		}(i)
	}

	// Wait for all operations to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}