package output

import (
	"testing"
)

func TestFactory(t *testing.T) {
	t.Run("create factory", func(t *testing.T) {
		factory := NewFactory()

		if factory == nil {
			t.Errorf("factory should not be nil")
		}
	})

	t.Run("list providers", func(t *testing.T) {
		factory := NewFactory()
		providers := factory.ListProviders()

		if len(providers) == 0 {
			t.Errorf("expected providers")
		}
	})

	t.Run("create openai adapter", func(t *testing.T) {
		factory := NewFactory()
		adapter, err := factory.Create("openai", &Config{Model: "gpt-4"})

		if err != nil {
			t.Errorf("create error: %v", err)
		}
		if adapter == nil {
			t.Errorf("adapter should not be nil")
		}
	})

	t.Run("create ollama adapter", func(t *testing.T) {
		factory := NewFactory()
		adapter, err := factory.Create("ollama", &Config{Model: "llama2"})

		if err != nil {
			t.Errorf("create error: %v", err)
		}
		if adapter == nil {
			t.Errorf("adapter should not be nil")
		}
	})

	t.Run("create unknown adapter", func(t *testing.T) {
		factory := NewFactory()
		_, err := factory.Create("unknown", &Config{})

		if err == nil {
			t.Errorf("expected error for unknown provider")
		}
	})
}

func TestConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		config := DefaultConfig()

		if config.Model != "gpt-3.5-turbo" {
			t.Errorf("expected gpt-3.5-turbo")
		}
		if config.MaxTokens != 2048 {
			t.Errorf("expected 2048 tokens")
		}
		if config.Temperature != 0.7 {
			t.Errorf("expected 0.7 temperature")
		}
	})
}

func TestParser(t *testing.T) {
	t.Run("create parser", func(t *testing.T) {
		parser := NewParser()

		if parser == nil {
			t.Errorf("parser should not be nil")
		}
	})

	t.Run("extract json from markdown", func(t *testing.T) {
		parser := NewParser()
		input := "```json\n{\"key\": \"value\"}\n```"

		result := parser.extractJSON(input)
		if result == "" {
			t.Errorf("should extract json")
		}
	})

	t.Run("extract json from plain text", func(t *testing.T) {
		parser := NewParser()
		input := "{\"key\": \"value\"}"

		result := parser.extractJSON(input)
		if result == "" {
			t.Errorf("should extract json")
		}
	})

	t.Run("parse recommend result", func(t *testing.T) {
		parser := NewParser()
		input := `{"items": [{"item_id": "item1", "category": "top", "name": "T-Shirt", "price": 199.00}]}`

		result, err := parser.ParseRecommendResult(input)
		if err != nil {
			t.Errorf("ParseRecommendResult error: %v", err)
		}
		if result != nil && len(result.Items) > 0 {
			if result.Items[0].ItemID != "item1" {
				t.Errorf("expected item1")
			}
		}
	})

	t.Run("parse recommend result invalid", func(t *testing.T) {
		parser := NewParser()
		input := "not valid json"

		result, err := parser.ParseRecommendResult(input)
		if err == nil {
			t.Errorf("expected error for invalid json")
		}
		_ = result
	})

	t.Run("parse generic", func(t *testing.T) {
		parser := NewParser()
		input := `{"key": "value", "number": 123}`

		var result interface{}
		err := parser.ParseGeneric(input, &result)
		if err != nil {
			t.Errorf("ParseGeneric error: %v", err)
		}
		if result == nil {
			t.Errorf("expected result")
		}
	})

	t.Run("parse array", func(t *testing.T) {
		parser := NewParser()
		// ParseArray expects the JSON to be extracted, so we need to wrap it
		input := "```json\n[{\"id\": 1}, {\"id\": 2}]\n```"

		result, err := parser.ParseArray(input)
		if err != nil {
			t.Errorf("ParseArray error: %v", err)
		}
		if result == nil || len(result) != 2 {
			t.Errorf("expected 2 items")
		}
	})
}

func TestSchema(t *testing.T) {
	t.Run("recommend result schema", func(t *testing.T) {
		schema := GetRecommendResultSchema()

		if schema.Type != "object" {
			t.Errorf("expected object type")
		}
		if schema.Properties == nil {
			t.Errorf("properties should not be nil")
		}
	})

	t.Run("recommend item schema", func(t *testing.T) {
		schema := GetRecommendItemSchema()

		if schema.Type != "object" {
			t.Errorf("expected object type")
		}
	})

	t.Run("user profile schema", func(t *testing.T) {
		schema := GetUserProfileSchema()

		if schema.Type != "object" {
			t.Errorf("expected object type")
		}
	})

	t.Run("to JSON", func(t *testing.T) {
		schema := &Schema{Type: "string"}
		jsonStr, err := schema.ToJSON()

		if err != nil {
			t.Errorf("to JSON error: %v", err)
		}
		if jsonStr == "" {
			t.Errorf("should have JSON output")
		}
	})

	t.Run("to JSON string", func(t *testing.T) {
		schema := &Schema{Type: "number"}
		jsonStr, err := schema.ToJSONString()

		if err != nil {
			t.Errorf("to JSONString error: %v", err)
		}
		if jsonStr == "" {
			t.Errorf("should have JSON output")
		}
	})
}

func TestTemplateEngine(t *testing.T) {
	t.Run("create template engine", func(t *testing.T) {
		engine := NewTemplateEngine()

		if engine == nil {
			t.Errorf("engine should not be nil")
		}
	})

	t.Run("render recommendation", func(t *testing.T) {
		engine := NewTemplateEngine()
		data := map[string]interface{}{
			"user_id": "user1",
			"style":   "casual",
		}

		result, err := engine.RenderRecommendation(data)
		if err != nil {
			t.Errorf("RenderRecommendation error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render profile extraction", func(t *testing.T) {
		engine := NewTemplateEngine()

		result, err := engine.RenderProfileExtraction("I want casual style")
		if err != nil {
			t.Errorf("RenderProfileExtraction error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render style analysis", func(t *testing.T) {
		engine := NewTemplateEngine()

		result, err := engine.RenderStyleAnalysis("casual top and jeans")
		if err != nil {
			t.Errorf("RenderStyleAnalysis error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render with defaults", func(t *testing.T) {
		result, err := RenderWithDefault("recommendation", nil)
		if err != nil {
			t.Errorf("RenderWithDefault error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render recommendation with default", func(t *testing.T) {
		data := map[string]interface{}{
			"user_id": "user1",
		}
		result, err := RenderRecommendationWithDefault(data)
		if err != nil {
			t.Errorf("RenderRecommendationWithDefault error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render profile extraction with default", func(t *testing.T) {
		result, err := RenderProfileExtractionWithDefault("I want casual style")
		if err != nil {
			t.Errorf("RenderProfileExtractionWithDefault error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})

	t.Run("render style analysis with default", func(t *testing.T) {
		result, err := RenderStyleAnalysisWithDefault("casual style")
		if err != nil {
			t.Errorf("RenderStyleAnalysisWithDefault error: %v", err)
		}
		if result == "" {
			t.Errorf("expected result")
		}
	})
}

func TestOllamaAdapter(t *testing.T) {
	t.Run("create ollama adapter", func(t *testing.T) {
		adapter := NewOllamaAdapter(&Config{
			Model: "llama2",
		})

		if adapter == nil {
			t.Errorf("adapter should not be nil")
		}
		if adapter.GetModel() != "llama2" {
			t.Errorf("expected llama2")
		}
	})
}

func TestOpenAIAdapter(t *testing.T) {
	t.Run("create openai adapter", func(t *testing.T) {
		adapter := NewOpenAIAdapter(&Config{
			Model: "gpt-4",
		})

		if adapter == nil {
			t.Errorf("adapter should not be nil")
		}
		if adapter.GetModel() != "gpt-4" {
			t.Errorf("expected gpt-4")
		}
	})
}