package resources

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// IDGenerator generates unique identifiers.
type IDGenerator struct {
	*BaseTool
}

// NewIDGenerator creates a new IDGenerator tool.
func NewIDGenerator() *IDGenerator {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"operation": {
				Type:        "string",
				Description: "Operation to perform (generate_uuid, generate_short_id)",
				Enum:        []interface{}{"generate_uuid", "generate_short_id"},
			},
			"count": {
				Type:        "integer",
				Description: "Number of IDs to generate (default: 1)",
				Default:     1,
			},
		},
		Required: []string{"operation"},
	}

	return &IDGenerator{
		BaseTool: NewBaseToolWithCategory("id_generator", "Generate unique identifiers (UUID or short ID)", CategorySystem, params),
	}
}

// Execute performs the ID generation operation.
func (t *IDGenerator) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return NewErrorResult("operation is required"), nil
	}

	count := getInt(params, "count", 1)
	if count <= 0 {
		count = 1
	}
	if count > 100 {
		return NewErrorResult("count cannot exceed 100"), nil
	}

	switch operation {
	case "generate_uuid":
		return t.generateUUID(ctx, count)
	case "generate_short_id":
		return t.generateShortID(ctx, count)
	default:
		return NewErrorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// generateUUID generates one or more UUIDs.
func (t *IDGenerator) generateUUID(ctx context.Context, count int) (Result, error) {
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		id := uuid.New()
		ids[i] = id.String()
	}

	result := map[string]interface{}{
		"operation": "generate_uuid",
		"count":     count,
		"ids":       ids,
	}

	if count == 1 {
		result["id"] = ids[0]
	}

	return NewResult(true, result), nil
}

// generateShortID generates one or more short IDs.
func (t *IDGenerator) generateShortID(ctx context.Context, count int) (Result, error) {
	ids := make([]string, count)
	for i := 0; i < count; i++ {
		// Generate a short ID from first 8 characters of UUID
		id := uuid.New()
		shortID := id.String()[:8]
		ids[i] = shortID
	}

	result := map[string]interface{}{
		"operation": "generate_short_id",
		"count":     count,
		"ids":       ids,
	}

	if count == 1 {
		result["id"] = ids[0]
	}

	return NewResult(true, result), nil
}
