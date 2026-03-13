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
}