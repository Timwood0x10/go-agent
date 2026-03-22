package resources

import (
	"context"
	"fmt"
	"time"

	"goagent/internal/storage/postgres/repositories"
)

// CorrectKnowledge corrects knowledge base content.
type CorrectKnowledge struct {
	*BaseTool
	repo *repositories.KnowledgeRepository
}

// NewCorrectKnowledge creates a new CorrectKnowledge tool.
func NewCorrectKnowledge(repo *repositories.KnowledgeRepository) *CorrectKnowledge {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"chunk_id": {
				Type:        "string",
				Description: "Knowledge chunk ID to correct",
			},
			"corrected_content": {
				Type:        "string",
				Description: "Corrected content",
			},
		},
		Required: []string{"chunk_id", "corrected_content"},
	}

	ck := &CorrectKnowledge{
		repo: repo,
	}
	ck.BaseTool = NewBaseTool("correct_knowledge", "Correct knowledge base content", params)

	return ck
}

// Execute corrects knowledge content.
func (t *CorrectKnowledge) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	chunkID, ok := params["chunk_id"].(string)
	if !ok || chunkID == "" {
		return NewErrorResult("chunk_id is required"), nil
	}

	correctedContent, ok := params["corrected_content"].(string)
	if !ok || correctedContent == "" {
		return NewErrorResult("corrected_content is required"), nil
	}

	// Get existing chunk
	chunk, err := t.repo.GetByID(ctx, chunkID)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to get chunk: %v", err)), nil
	}

	if chunk == nil {
		return NewErrorResult("chunk not found"), nil
	}

	// Update content
	chunk.Content = correctedContent
	chunk.UpdatedAt = time.Now()

	// Add metadata
	if chunk.Metadata == nil {
		chunk.Metadata = make(map[string]interface{})
	}
	chunk.Metadata["corrected_at"] = time.Now()
	chunk.Metadata["correction"] = true

	// Perform update
	if err := t.repo.Update(ctx, chunk); err != nil {
		return NewErrorResult(fmt.Sprintf("update failed: %v", err)), nil
	}

	return NewResult(true, map[string]interface{}{
		"chunk_id":         chunk.ID,
		"corrected":        true,
		"corrected_at":     chunk.UpdatedAt,
		"original_content": "updated",
	}), nil
}
