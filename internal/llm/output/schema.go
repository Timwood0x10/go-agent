package output

import "encoding/json"

// Schema represents a JSON Schema.
type Schema struct {
	Type         string          `json:"type,omitempty"`
	Properties   map[string]*Schema `json:"properties,omitempty"`
	Items        *Schema          `json:"items,omitempty"`
	Required     []string        `json:"required,omitempty"`
	Minimum      *float64        `json:"minimum,omitempty"`
	Maximum      *float64        `json:"maximum,omitempty"`
	MinLength    *int            `json:"minLength,omitempty"`
	MaxLength    *int            `json:"maxLength,omitempty"`
	Pattern      string          `json:"pattern,omitempty"`
	Enum         []interface{}   `json:"enum,omitempty"`
	Nullable     bool            `json:"nullable,omitempty"`
	MinItems     *int            `json:"minItems,omitempty"`
	MaxItems     *int            `json:"maxItems,omitempty"`
	Description  string          `json:"description,omitempty"`
	Format       string          `json:"format,omitempty"`
	Ref          string          `json:"$ref,omitempty"`
}

// GetRecommendResultSchema returns the schema for RecommendResult.
func GetRecommendResultSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"session_id": {
				Type:      "string",
				MinLength: pointerToInt(1),
			},
			"user_id": {
				Type:      "string",
				MinLength: pointerToInt(1),
			},
			"items": {
				Type:     "array",
				MinItems: pointerToInt(1),
				Items:    GetRecommendItemSchema(),
			},
			"reason": {
				Type: "string",
			},
			"total_price": {
				Type:     "number",
				Minimum:  pointerToFloat64(0),
			},
			"match_score": {
				Type:     "number",
				Minimum:  pointerToFloat64(0),
				Maximum:  pointerToFloat64(1),
			},
			"occasion": {
				Type: "string",
				Enum: []interface{}{
					"casual", "business", "formal", "party", "date", "sports", "outdoor",
				},
			},
			"season": {
				Type: "string",
				Enum: []interface{}{
					"spring", "summer", "autumn", "winter", "all_season",
				},
			},
			"metadata": {
				Type: "object",
			},
		},
		Required: []string{"session_id", "user_id", "items"},
	}
}

// GetRecommendItemSchema returns the schema for RecommendItem.
func GetRecommendItemSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"item_id": {
				Type:      "string",
				MinLength: pointerToInt(1),
			},
			"category": {
				Type: "string",
				Enum: []interface{}{
					"top", "bottom", "dress", "outerwear", "shoes", "accessory", "bag", "hat",
				},
			},
			"name": {
				Type:      "string",
				MinLength: pointerToInt(1),
			},
			"brand": {
				Type: "string",
			},
			"price": {
				Type:     "number",
				Minimum:  pointerToFloat64(0),
			},
			"url": {
				Type:   "string",
				Format: "uri",
			},
			"image_url": {
				Type:   "string",
				Format: "uri",
			},
			"style": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
			"colors": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
			"description": {
				Type: "string",
			},
			"match_reason": {
				Type: "string",
			},
			"metadata": {
				Type: "object",
			},
		},
		Required: []string{"item_id", "category", "name", "price"},
	}
}

// GetUserProfileSchema returns the schema for UserProfile.
func GetUserProfileSchema() *Schema {
	return &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"user_id": {
				Type:      "string",
				MinLength: pointerToInt(1),
			},
			"gender": {
				Type: "string",
				Enum: []interface{}{"male", "female", "other"},
			},
			"age": {
				Type:     "integer",
				Minimum:  pointerToFloat64(0),
				Maximum:  pointerToFloat64(150),
			},
			"style_preferences": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
			"budget_range": {
				Type: "object",
				Properties: map[string]*Schema{
					"min": {
						Type:     "number",
						Minimum:  pointerToFloat64(0),
					},
					"max": {
						Type:     "number",
						Minimum:  pointerToFloat64(0),
					},
				},
			},
			"favorite_colors": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
			"favorite_brands": {
				Type: "array",
				Items: &Schema{
					Type: "string",
				},
			},
			"body_type": {
				Type: "string",
			},
			"occupation": {
				Type: "string",
			},
			"location": {
				Type: "string",
			},
		},
		Required: []string{"user_id"},
	}
}

// ToJSON returns JSON representation of the schema.
func (s *Schema) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSONString returns compact JSON representation.
func (s *Schema) ToJSONString() (string, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Helper functions.
func pointerToInt(v int) *int {
	return &v
}

func pointerToFloat64(v float64) *float64 {
	return &v
}
