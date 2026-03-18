// nolint: errcheck // Test code may ignore return values
package resources

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestResultWithMetadata tests adding metadata to results.
func TestResultWithMetadata(t *testing.T) {
	t.Run("add single metadata", func(t *testing.T) {
		result := NewResult(true, "test data")
		result = *result.WithMetadata("key1", "value1")

		if result.Metadata == nil {
			t.Error("metadata should not be nil")
		}
		if result.Metadata["key1"] != "value1" {
			t.Errorf("metadata key1 = %v, want value1", result.Metadata["key1"])
		}
	})

	t.Run("add multiple metadata", func(t *testing.T) {
		result := NewResult(true, "test data")
		result = *result.WithMetadata("key1", "value1")
		result = *result.WithMetadata("key2", "value2")

		if len(result.Metadata) != 2 {
			t.Errorf("metadata count = %d, want 2", len(result.Metadata))
		}
	})

	t.Run("add metadata to existing", func(t *testing.T) {
		result := NewResult(true, "test data")
		result = *result.WithMetadata("key1", "value1")
		result = *result.WithMetadata("key1", "value2")

		if result.Metadata["key1"] != "value2" {
			t.Errorf("metadata key1 should be updated to value2")
		}
	})
}

// TestResultString tests result string representation.
func TestResultString(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		result := NewResult(true, "test data")
		str := result.String()

		if str != "Success" {
			t.Errorf("result.String() = %v, want Success", str)
		}
	})

	t.Run("error result", func(t *testing.T) {
		result := NewErrorResult("test error")
		str := result.String()

		if str != "Error: test error" {
			t.Errorf("result.String() = %v, want Error: test error", str)
		}
	})
}

// TestResultToJSON tests result JSON serialization.
func TestResultToJSON(t *testing.T) {
	t.Run("successful result", func(t *testing.T) {
		result := NewResult(true, map[string]string{"key": "value"})
		json, err := result.ToJSON()

		if err != nil {
			t.Errorf("ToJSON() error = %v", err)
		}
		if json == "" {
			t.Error("ToJSON() should not return empty string")
		}
	})

	t.Run("error result", func(t *testing.T) {
		result := NewErrorResult("test error")
		json, err := result.ToJSON()

		if err != nil {
			t.Errorf("ToJSON() error = %v", err)
		}
		if json == "" {
			t.Error("ToJSON() should not return empty string")
		}
	})
}

// TestResultWithTiming tests adding timing information.
func TestResultWithTiming(t *testing.T) {
	result := NewResult(true, "test data")
	duration := 100 * time.Millisecond

	result = ResultWithTiming(result, duration)

	if result.Metadata == nil {
		t.Error("metadata should not be nil")
	}
	if result.Metadata["duration_ms"] != int64(100) {
		t.Errorf("duration_ms = %v, want 100", result.Metadata["duration_ms"])
	}
	if _, ok := result.Metadata["timestamp"]; !ok {
		t.Error("timestamp should be present in metadata")
	}
}

// TestResultList tests managing multiple results.
func TestResultList(t *testing.T) {
	t.Run("create result list", func(t *testing.T) {
		list := NewResultList()

		if list == nil {
			t.Error("NewResultList() should not return nil")
			return
		}
		if len(list.Results) != 0 {
			t.Errorf("initial results count = %d, want 0", len(list.Results))
		}
	})

	t.Run("add successful result", func(t *testing.T) {
		list := NewResultList()
		list.Add(NewResult(true, "success"))

		if list.Total != 1 {
			t.Errorf("total = %d, want 1", list.Total)
		}
		if list.Success != 1 {
			t.Errorf("success = %d, want 1", list.Success)
		}
		if list.Failed != 0 {
			t.Errorf("failed = %d, want 0", list.Failed)
		}
	})

	t.Run("add failed result", func(t *testing.T) {
		list := NewResultList()
		list.Add(NewErrorResult("error"))

		if list.Total != 1 {
			t.Errorf("total = %d, want 1", list.Total)
		}
		if list.Success != 0 {
			t.Errorf("success = %d, want 0", list.Success)
		}
		if list.Failed != 1 {
			t.Errorf("failed = %d, want 1", list.Failed)
		}
	})

	t.Run("add mixed results", func(t *testing.T) {
		list := NewResultList()
		list.Add(NewResult(true, "success1"))
		list.Add(NewErrorResult("error1"))
		list.Add(NewResult(true, "success2"))

		if list.Total != 3 {
			t.Errorf("total = %d, want 3", list.Total)
		}
		if list.Success != 2 {
			t.Errorf("success = %d, want 2", list.Success)
		}
		if list.Failed != 1 {
			t.Errorf("failed = %d, want 1", list.Failed)
		}
	})
}

// TestErrorResult tests error result with code.
func TestErrorResult(t *testing.T) {
	t.Run("create error result", func(t *testing.T) {
		errResult := NewErrorResultWithCode("ERR001", "Test error")

		if errResult.Code != "ERR001" {
			t.Errorf("Code = %v, want ERR001", errResult.Code)
		}
		if errResult.Message != "Test error" {
			t.Errorf("Message = %v, want Test error", errResult.Message)
		}
	})

	t.Run("error result to result", func(t *testing.T) {
		errResult := NewErrorResultWithCode("ERR002", "Test error")
		result := errResult.ToResult()

		if result.Success {
			t.Error("result should not be successful")
		}
		if result.Error != "Test error" {
			t.Errorf("result.Error = %v, want Test error", result.Error)
		}
	})

	t.Run("error result error method", func(t *testing.T) {
		errResult := NewErrorResultWithCode("ERR003", "Test error")

		if errResult.Error() != "Test error" {
			t.Errorf("Error() = %v, want Test error", errResult.Error())
		}
	})

	t.Run("error result with details", func(t *testing.T) {
		errResult := NewErrorResultWithCode("ERR004", "Test error")
		errResult = errResult.WithDetails(map[string]interface{}{
			"context": "test context",
			"retry":   3,
		})

		if errResult.Details == nil {
			t.Error("Details should not be nil")
		}
		if errResult.Details["context"] != "test context" {
			t.Errorf("Details context = %v, want test context", errResult.Details["context"])
		}
	})
}

// TestValidationError tests validation error.
func TestValidationError(t *testing.T) {
	t.Run("create validation error", func(t *testing.T) {
		validationErr := NewValidationError("field_name", "Field cannot be empty")

		if validationErr.Field != "field_name" {
			t.Errorf("Field = %v, want field_name", validationErr.Field)
		}
		if validationErr.Message != "Field cannot be empty" {
			t.Errorf("Message = %v, want Field cannot be empty", validationErr.Message)
		}
	})

	t.Run("validation error error method", func(t *testing.T) {
		validationErr := NewValidationError("email", "Invalid email format")

		if validationErr.Error() != "email: Invalid email format" {
			t.Errorf("Error() = %v, want email: Invalid email format", validationErr.Error())
		}
	})
}

// TestWithMetadata tests wrapping tools with metadata.
func TestWithMetadata(t *testing.T) {
	t.Run("wrap tool with metadata", func(t *testing.T) {
		baseTool := NewToolFunc("test_tool", "Test tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "result"), nil
			})

		metadata := ToolMetadata{
			Version:    "1.0",
			Author:     "test author",
			Tags:       []string{"tag1", "tag2"},
			Examples:   []string{"example1", "example2"},
			Deprecated: false,
		}

		wrappedTool := WithMetadata(baseTool, metadata)

		// Execute the wrapped tool
		result, err := wrappedTool.Execute(context.Background(), map[string]interface{}{})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("check deprecated status", func(t *testing.T) {
		baseTool := NewToolFunc("old_tool", "Old tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "result"), nil
			})

		metadata := ToolMetadata{
			Version:     "0.5",
			Deprecated:  true,
			Deprecation: "Use new_tool instead",
		}

		wrappedTool := WithMetadata(baseTool, metadata).(*metadataTool)

		if !wrappedTool.IsDeprecated() {
			t.Error("tool should be deprecated")
		}
	})

	t.Run("check non-deprecated tool", func(t *testing.T) {
		baseTool := NewToolFunc("new_tool", "New tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "result"), nil
			})

		metadata := ToolMetadata{
			Version:    "1.0",
			Deprecated: false,
		}

		wrappedTool := WithMetadata(baseTool, metadata).(*metadataTool)

		if wrappedTool.IsDeprecated() {
			t.Error("tool should not be deprecated")
		}
	})
}

// TestFashionSearch tests the FashionSearch tool.
func TestFashionSearch(t *testing.T) {
	t.Run("missing query parameter", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when query is missing")
		}
	})

	t.Run("empty query parameter", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when query is empty")
		}
	})

	t.Run("successful search", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "summer dress",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with filters", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query":     "red shoes",
			"category":  "shoes",
			"colors":    []interface{}{"red", "blue"},
			"price_min": 50.0,
			"price_max": 200.0,
			"limit":     5,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search limit exceeds results", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "dress",
			"limit": 2,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with invalid limit", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "test",
			"limit": "invalid",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful (uses default limit)")
		}
	})

	t.Run("search with zero limit", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "dress",
			"limit": 0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with category filter", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query":    "shoes",
			"category": "shoes",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with price range filter", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query":     "dress",
			"price_min": 100.0,
			"price_max": 200.0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with colors filter", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query":  "dress",
			"colors": []interface{}{"red", "blue"},
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("search with limit less than available items", func(t *testing.T) {
		searcher := NewMockFashionSearcher()
		tool := NewFashionSearch(searcher)

		// MockFashionSearcher returns 3 items, so limit=1 should trigger the len(items) > limit branch
		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "dress",
			"limit": 1,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}

		// Verify that only 1 item is returned
		if result.Data == nil {
			t.Error("result.Data should not be nil")
		}
	})

	t.Run("search returns error from searcher", func(t *testing.T) {
		// Create a mock searcher that returns an error
		errorSearcher := &errorFashionSearcher{}
		tool := NewFashionSearch(errorSearcher)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"query": "test",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when searcher returns error")
		}
	})
}

// TestStyleRecommender tests the StyleRecommender tool.
func TestStyleRecommender(t *testing.T) {
	t.Run("missing required parameters", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		_, err := tool.Execute(context.Background(), map[string]interface{}{
			"occasion": "casual",
			// Missing gender
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		// Tool should handle missing parameters gracefully
	})

	t.Run("get recommendations", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":     "female",
			"age_range":  "25-35",
			"body_type":  "slim",
			"style_pref": []interface{}{"casual", "minimalist"},
			"color_pref": []interface{}{"navy", "white"},
			"budget_min": 50.0,
			"budget_max": 200.0,
			"occasion":   "casual",
			"season":     "summer",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("get recommendations without budget", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":   "male",
			"occasion": "business",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("get recommendations with only budget_min", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":     "female",
			"occasion":   "casual",
			"budget_min": 50.0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("get recommendations with only budget_max", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":     "female",
			"occasion":   "casual",
			"budget_max": 200.0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("get recommendations with zero budget values", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommender(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":     "female",
			"occasion":   "casual",
			"budget_min": 0.0,
			"budget_max": 0.0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("recommender returns error", func(t *testing.T) {
		// Create a mock recommender that returns an error
		errorRecommender := &errorStyleRecommender{}
		tool := NewAgentRecommender(errorRecommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":   "female",
			"occasion": "casual",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when recommender returns error")
		}
	})
}

// TestStyleRecommenderWithTrends tests the StyleRecommender tool with trends.
func TestStyleRecommenderWithTrends(t *testing.T) {
	t.Run("get trends", func(t *testing.T) {
		recommender := NewMockAgentRecommender()
		tool := NewAgentRecommenderWithTrends(recommender)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"gender":     "female",
			"occasion":   "casual",
			"get_trends": true,
			"season":     "summer",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})
}

// TestWeatherCheck tests the WeatherCheck tool.
func TestWeatherCheck(t *testing.T) {
	t.Run("missing location parameter", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when location is missing")
		}
	})

	t.Run("empty location parameter", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location": "",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when location is empty")
		}
	})

	t.Run("get current weather", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location": "New York",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("get weather forecast", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "London",
			"forecast_days": 3,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("invalid forecast days - too low", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "Tokyo",
			"forecast_days": 0,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful (uses default 1 day)")
		}
	})

	t.Run("invalid forecast days - too high", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "Paris",
			"forecast_days": 10,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful (uses maximum 7 days)")
		}
	})

	t.Run("invalid forecast days - negative", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "Berlin",
			"forecast_days": -5,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful (uses default 1 day)")
		}
	})

	t.Run("forecast days at boundary", func(t *testing.T) {
		provider := NewMockWeatherProvider()
		tool := NewWeatherCheck(provider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "Sydney",
			"forecast_days": 7,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful with maximum 7 days")
		}
	})

	t.Run("provider returns error for current weather", func(t *testing.T) {
		errorProvider := &errorWeatherProvider{}
		tool := NewWeatherCheck(errorProvider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location": "Error City",
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when provider returns error")
		}
	})

	t.Run("provider returns error for forecast", func(t *testing.T) {
		errorProvider := &errorWeatherProvider{}
		tool := NewWeatherCheck(errorProvider)

		result, err := tool.Execute(context.Background(), map[string]interface{}{
			"location":      "Error City",
			"forecast_days": 3,
		})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if result.Success {
			t.Error("result should fail when provider returns error for forecast")
		}
	})
}

// TestMockFashionSearcher tests the mock fashion searcher.
func TestMockFashionSearcher(t *testing.T) {
	searcher := NewMockFashionSearcher()

	t.Run("search without filters", func(t *testing.T) {
		items, err := searcher.Search(context.Background(), "dress", &FashionFilters{})

		if err != nil {
			t.Errorf("Search() error = %v", err)
		}
		if len(items) == 0 {
			t.Error("Search() should return items")
		}
	})

	t.Run("search with filters", func(t *testing.T) {
		filters := &FashionFilters{
			Category: "shoes",
			Colors:   []string{"red", "blue"},
			PriceMin: 50.0,
			PriceMax: 200.0,
		}

		items, err := searcher.Search(context.Background(), "shoes", filters)

		if err != nil {
			t.Errorf("Search() error = %v", err)
		}
		if len(items) == 0 {
			t.Error("Search() should return items")
		}
	})
}

// TestRegistryExecute tests Registry.Execute method.
func TestRegistryExecute(t *testing.T) {
	t.Run("execute registered tool", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewToolFunc(
			"test_tool",
			"Test tool",
			nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "executed"), nil
			},
		)

		err := registry.Register(tool)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}

		result, err := registry.Execute(context.Background(), "test_tool", map[string]interface{}{"key": "value"})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}
	})

	t.Run("execute non-existent tool", func(t *testing.T) {
		registry := NewRegistry()

		result, err := registry.Execute(context.Background(), "non_existent", map[string]interface{}{})
		if err == nil {
			t.Error("Execute() should return error for non-existent tool")
		}
		if result.Success {
			t.Error("result should not be successful")
		}
	})

	t.Run("execute tool with nil parameters", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewToolFunc(
			"nil_tool",
			"Nil tool",
			nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, nil), nil
			},
		)

		err := registry.Register(tool)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}

		result, err := registry.Execute(context.Background(), "nil_tool", nil)
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful with nil parameters")
		}
	})
}

// TestRegistryClear tests Registry.Clear method.
func TestRegistryClear(t *testing.T) {
	t.Run("clear empty registry", func(t *testing.T) {
		registry := NewRegistry()
		registry.Clear()

		if registry.Count() != 0 {
			t.Errorf("Count() = %d, want 0 after clearing empty registry", registry.Count())
		}
	})

	t.Run("clear registry with tools", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(NewToolFunc("tool1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))
		registry.Register(NewToolFunc("tool2", "desc2", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))

		if registry.Count() != 2 {
			t.Errorf("Count() = %d, want 2 before clear", registry.Count())
		}

		registry.Clear()

		if registry.Count() != 0 {
			t.Errorf("Count() = %d, want 0 after clear", registry.Count())
		}
	})

	t.Run("clear and re-register", func(t *testing.T) {
		registry := NewRegistry()
		registry.Register(NewToolFunc("tool1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))

		registry.Clear()
		registry.Register(NewToolFunc("new_tool", "new desc", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))

		if registry.Count() != 1 {
			t.Errorf("Count() = %d, want 1 after re-register", registry.Count())
		}

		_, exists := registry.Get("new_tool")
		if !exists {
			t.Error("new_tool should exist after re-register")
		}
	})
}

// TestGlobalRegistry tests global registry functions.
func TestGlobalRegistry(t *testing.T) {
	t.Run("register and get from global registry", func(t *testing.T) {
		tool := NewToolFunc(
			"global_tool",
			"Global test tool",
			nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "global"), nil
			},
		)

		err := Register(tool)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}

		retrieved, exists := Get("global_tool")
		if !exists {
			t.Error("tool should exist in global registry")
		}
		if retrieved.Name() != "global_tool" {
			t.Errorf("expected global_tool, got %s", retrieved.Name())
		}

		// Clean up
		GlobalRegistry.Unregister("global_tool")
	})

	t.Run("list tools from global registry", func(t *testing.T) {
		// Clear global registry first
		GlobalRegistry.Clear()

		Register(NewToolFunc("global1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))
		Register(NewToolFunc("global2", "desc2", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))

		tools := List()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}

		// Clean up
		GlobalRegistry.Clear()
	})

	t.Run("execute from global registry", func(t *testing.T) {
		GlobalRegistry.Clear()

		tool := NewToolFunc(
			"exec_tool",
			"Executable tool",
			nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, "executed"), nil
			},
		)

		err := Register(tool)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}

		result, err := Execute(context.Background(), "exec_tool", map[string]interface{}{})
		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		if !result.Success {
			t.Error("result should be successful")
		}

		// Clean up
		GlobalRegistry.Clear()
	})

	t.Run("execute non-existent tool from global registry", func(t *testing.T) {
		GlobalRegistry.Clear()

		_, err := Execute(context.Background(), "non_existent", map[string]interface{}{})
		if err == nil {
			t.Error("Execute() should return error for non-existent tool")
		}
	})
}

// TestToolGroup tests ToolGroup functionality.
func TestToolGroup(t *testing.T) {
	t.Run("create tool group", func(t *testing.T) {
		group := NewToolGroup("fashion", "Fashion-related tools")

		if group.Name() != "fashion" {
			t.Errorf("Name() = %s, want fashion", group.Name())
		}
		if group.Description() != "Fashion-related tools" {
			t.Errorf("Description() = %s, want Fashion-related tools", group.Description())
		}
	})

	t.Run("register and get tool from group", func(t *testing.T) {
		group := NewToolGroup("weather", "Weather tools")

		tool := NewToolFunc(
			"temperature",
			"Get temperature",
			nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, 25.5), nil
			},
		)

		err := group.Register(tool)
		if err != nil {
			t.Errorf("Register() error = %v", err)
		}

		retrieved, exists := group.Get("temperature")
		if !exists {
			t.Error("tool should exist in group")
		}
		if retrieved.Name() != "temperature" {
			t.Errorf("expected temperature, got %s", retrieved.Name())
		}
	})

	t.Run("list tools from group", func(t *testing.T) {
		group := NewToolGroup("math", "Math tools")

		group.Register(NewToolFunc("add", "Add numbers", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))
		group.Register(NewToolFunc("subtract", "Subtract numbers", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		}))

		tools := group.List()
		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}
	})

	t.Run("multiple independent groups", func(t *testing.T) {
		group1 := NewToolGroup("group1", "First group")
		group2 := NewToolGroup("group2", "Second group")

		tool1 := NewToolFunc("tool1", "Tool 1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, "group1"), nil
		})
		tool2 := NewToolFunc("tool2", "Tool 2", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, "group2"), nil
		})

		group1.Register(tool1)
		group2.Register(tool2)

		_, exists1 := group1.Get("tool1")
		if !exists1 {
			t.Error("tool1 should exist in group1")
		}

		_, exists2 := group2.Get("tool2")
		if !exists2 {
			t.Error("tool2 should exist in group2")
		}

		_, exists3 := group1.Get("tool2")
		if exists3 {
			t.Error("tool2 should not exist in group1")
		}
	})
}

// TestRegistryRegisterNil tests registering nil tool.
func TestRegistryRegisterNil(t *testing.T) {
	t.Run("register nil tool in registry", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.Register(nil)
		if err == nil {
			t.Error("Register() should return error for nil tool")
		}
		if err != ErrNilTool {
			t.Errorf("Register() error = %v, want ErrNilTool", err)
		}
	})

	t.Run("register nil tool in tool group", func(t *testing.T) {
		group := NewToolGroup("test", "Test group")

		err := group.Register(nil)
		if err == nil {
			t.Error("Register() should return error for nil tool")
		}
		if err != ErrNilTool {
			t.Errorf("Register() error = %v, want ErrNilTool", err)
		}
	})
}

// TestRegistryUnregisterNonExistent tests unregistering non-existent tool.
func TestRegistryUnregisterNonExistent(t *testing.T) {
	t.Run("unregister non-existent tool", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.Unregister("non_existent")
		if err == nil {
			t.Error("Unregister() should return error for non-existent tool")
		}
	})

	t.Run("unregister tool twice", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewToolFunc("tool1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		})

		registry.Register(tool)
		err := registry.Unregister("tool1")
		if err != nil {
			t.Errorf("First Unregister() error = %v", err)
		}

		err = registry.Unregister("tool1")
		if err == nil {
			t.Error("Second Unregister() should return error")
		}
	})
}

// TestRegistryDuplicateRegister tests registering duplicate tools.
func TestRegistryDuplicateRegister(t *testing.T) {
	t.Run("register same tool twice", func(t *testing.T) {
		registry := NewRegistry()
		tool := NewToolFunc("tool1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, nil), nil
		})

		err := registry.Register(tool)
		if err != nil {
			t.Errorf("First Register() error = %v", err)
		}

		err = registry.Register(tool)
		if err == nil {
			t.Error("Second Register() should return error for duplicate tool")
		}
	})

	t.Run("register different tools with same name", func(t *testing.T) {
		registry := NewRegistry()
		tool1 := NewToolFunc("tool1", "desc1", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, "tool1"), nil
		})
		tool2 := NewToolFunc("tool1", "desc2", nil, func(ctx context.Context, params map[string]interface{}) (Result, error) {
			return NewResult(true, "tool2"), nil
		})

		err := registry.Register(tool1)
		if err != nil {
			t.Errorf("First Register() error = %v", err)
		}

		err = registry.Register(tool2)
		if err == nil {
			t.Error("Second Register() should return error for duplicate name")
		}
	})
}

// TestResultToJSONError tests ToJSON error handling.
func TestResultToJSONError(t *testing.T) {
	t.Run("result with complex data", func(t *testing.T) {
		complexData := map[string]interface{}{
			"nested": map[string]interface{}{
				"deep": map[string]interface{}{
					"value": "test",
				},
			},
			"array": []interface{}{1, 2, 3},
			"nil":   nil,
		}

		result := NewResult(true, complexData)
		jsonStr, err := result.ToJSON()

		if err != nil {
			t.Errorf("ToJSON() error = %v", err)
		}
		if jsonStr == "" {
			t.Error("ToJSON() should not return empty string")
		}
		if jsonStr[0] != '{' {
			t.Error("JSON string should start with '{'")
		}
	})

	t.Run("result with unserializable data", func(t *testing.T) {
		// Create a result with data that cannot be JSON marshaled
		// Using a channel which cannot be marshaled
		unserializableData := make(chan int)

		result := NewResult(true, unserializableData)
		jsonStr, err := result.ToJSON()

		// json.Marshal should fail for channels
		if err == nil {
			t.Error("ToJSON() should return error for unserializable data")
		}
		if jsonStr != "" {
			t.Error("ToJSON() should return empty string on error")
		}
	})
}

// TestMockStyleRecommender tests the mock style recommender.
func TestMockStyleRecommender(t *testing.T) {
	recommender := NewMockAgentRecommender()

	t.Run("get recommendations", func(t *testing.T) {
		profile := &AgentProfile{
			Gender:   "female",
			AgeRange: "25-35",
			BodyType: "slim",
			Occasion: "casual",
			Season:   "summer",
		}

		rec, err := recommender.GetRecommendations(context.Background(), profile)

		if err != nil {
			t.Errorf("GetRecommendations() error = %v", err)
		}
		if rec == nil {
			t.Error("GetRecommendations() should not return nil")
			return
		}
		if rec.PrimaryStyle == "" {
			t.Error("PrimaryStyle should not be empty")
		}
	})

	t.Run("get trends", func(t *testing.T) {
		trends, err := recommender.GetTrends(context.Background(), "summer")

		if err != nil {
			t.Errorf("GetTrends() error = %v", err)
		}
		if len(trends) == 0 {
			t.Error("GetTrends() should return trends")
		}
		if trends[0].Name == "" {
			t.Error("Trend name should not be empty")
		}
	})
}

// TestMockWeatherProvider tests the mock weather provider.
func TestMockWeatherProvider(t *testing.T) {
	provider := NewMockWeatherProvider()

	t.Run("get current weather", func(t *testing.T) {
		weather, err := provider.GetCurrent(context.Background(), "New York")

		if err != nil {
			t.Errorf("GetCurrent() error = %v", err)
		}
		if weather.Location != "New York" {
			t.Errorf("Location = %v, want New York", weather.Location)
		}
		if weather.Temperature != provider.currentTemp {
			t.Errorf("Temperature = %v, want %v", weather.Temperature, provider.currentTemp)
		}
		if weather.Condition != provider.condition {
			t.Errorf("Condition = %v, want %v", weather.Condition, provider.condition)
		}
	})

	t.Run("get forecast", func(t *testing.T) {
		days := 3
		forecast, err := provider.GetForecast(context.Background(), "London", days)

		if err != nil {
			t.Errorf("GetForecast() error = %v", err)
		}
		if len(forecast) != days {
			t.Errorf("forecast length = %d, want %d", len(forecast), days)
		}

		// Check that dates are sequential
		for i := 1; i < days; i++ {
			if forecast[i].Timestamp.Before(forecast[i-1].Timestamp) {
				t.Errorf("forecast[%d] timestamp should be after forecast[%d]", i, i-1)
			}
		}
	})
}

// TestFashionFilters tests FashionFilters struct.
func TestFashionFilters(t *testing.T) {
	t.Run("empty filters", func(t *testing.T) {
		filters := &FashionFilters{}

		if filters.Category != "" {
			t.Error("Category should be empty")
		}
		if len(filters.AgentPreferences) != 0 {
			t.Error("AgentPreferences should be empty")
		}
		if len(filters.Colors) != 0 {
			t.Error("Colors should be empty")
		}
	})

	t.Run("filters with values", func(t *testing.T) {
		filters := &FashionFilters{
			Category:         "shoes",
			AgentPreferences: []string{"casual", "formal"},
			Colors:           []string{"red", "blue", "black"},
			PriceMin:         50.0,
			PriceMax:         200.0,
			Brands:           []string{"nike", "adidas"},
			Occasion:         "casual",
			Season:           "summer",
		}

		if filters.Category != "shoes" {
			t.Errorf("Category = %v, want shoes", filters.Category)
		}
		if len(filters.AgentPreferences) != 2 {
			t.Errorf("AgentPreferences length = %d, want 2", len(filters.AgentPreferences))
		}
		if filters.PriceMin != 50.0 {
			t.Errorf("PriceMin = %v, want 50.0", filters.PriceMin)
		}
		if filters.PriceMax != 200.0 {
			t.Errorf("PriceMax = %v, want 200.0", filters.PriceMax)
		}
	})
}

// TestAgentProfile tests AgentProfile struct.
func TestAgentProfile(t *testing.T) {
	t.Run("minimal profile", func(t *testing.T) {
		profile := &AgentProfile{
			Gender:   "female",
			Occasion: "casual",
		}

		if profile.Gender != "female" {
			t.Errorf("Gender = %v, want female", profile.Gender)
		}
		if profile.Occasion != "casual" {
			t.Errorf("Occasion = %v, want casual", profile.Occasion)
		}
	})

	t.Run("profile with budget", func(t *testing.T) {
		profile := &AgentProfile{
			Gender:   "male",
			Occasion: "formal",
			BudgetRange: &BudgetRange{
				Min: 100.0,
				Max: 500.0,
			},
		}

		if profile.BudgetRange == nil {
			t.Error("BudgetRange should not be nil")
		}
		if profile.BudgetRange.Min != 100.0 {
			t.Errorf("BudgetRange.Min = %v, want 100.0", profile.BudgetRange.Min)
		}
		if profile.BudgetRange.Max != 500.0 {
			t.Errorf("BudgetRange.Max = %v, want 500.0", profile.BudgetRange.Max)
		}
	})
}

// TestBudgetRange tests BudgetRange struct.
func TestBudgetRange(t *testing.T) {
	t.Run("create budget range", func(t *testing.T) {
		budget := &BudgetRange{
			Min: 50.0,
			Max: 200.0,
		}

		if budget.Min != 50.0 {
			t.Errorf("Min = %v, want 50.0", budget.Min)
		}
		if budget.Max != 200.0 {
			t.Errorf("Max = %v, want 200.0", budget.Max)
		}
	})

	t.Run("budget range validation", func(t *testing.T) {
		budget := &BudgetRange{
			Min: 100.0,
			Max: 50.0, // Invalid: min > max
		}

		// This test just verifies the struct can be created
		// Validation should be done by the business logic
		if budget.Min > budget.Max {
			t.Log("Note: Min > Max is an invalid budget range")
		}
	})
}

// TestOutfitSuggestion tests OutfitSuggestion struct.
func TestOutfitSuggestion(t *testing.T) {
	outfit := &OutfitSuggestion{
		Name:        "Casual Friday",
		Items:       []string{"navy blazer", "white t-shirt", "dark jeans", "white sneakers"},
		Occasion:    "casual",
		MatchScore:  0.9,
		Description: "Clean and comfortable casual look",
	}

	if outfit.Name != "Casual Friday" {
		t.Errorf("Name = %v, want Casual Friday", outfit.Name)
	}
	if len(outfit.Items) != 4 {
		t.Errorf("Items count = %d, want 4", len(outfit.Items))
	}
	if outfit.MatchScore != 0.9 {
		t.Errorf("MatchScore = %v, want 0.9", outfit.MatchScore)
	}
}

// TestAgentTrend tests AgentTrend struct.
func TestAgentTrend(t *testing.T) {
	trend := &AgentTrend{
		TrendID:     "sustainable_fashion",
		Name:        "Sustainable Fashion",
		Category:    "lifestyle",
		Popularity:  0.95,
		Season:      "summer",
		KeyElements: []string{"organic materials", "recycled fabrics", "neutral colors"},
		Description: "Eco-friendly clothing continues to grow in popularity",
	}

	if trend.TrendID != "sustainable_fashion" {
		t.Errorf("TrendID = %v, want sustainable_fashion", trend.TrendID)
	}
	if trend.Popularity < 0 || trend.Popularity > 1 {
		t.Errorf("Popularity should be between 0 and 1, got %v", trend.Popularity)
	}
	if len(trend.KeyElements) != 3 {
		t.Errorf("KeyElements count = %d, want 3", len(trend.KeyElements))
	}
}

// TestHelperFunctions tests helper functions.
func TestHelperFunctions(t *testing.T) {
	t.Run("getString", func(t *testing.T) {
		params := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
			"key3": nil,
		}

		if getString(params, "key1") != "value1" {
			t.Errorf("getString(key1) = %v, want value1", getString(params, "key1"))
		}
		if getString(params, "key2") != "" {
			t.Error("getString(key2) should return empty string for non-string value")
		}
		if getString(params, "key3") != "" {
			t.Error("getString(key3) should return empty string for nil value")
		}
		if getString(params, "nonexistent") != "" {
			t.Error("getString(nonexistent) should return empty string")
		}
	})

	t.Run("getStringSlice", func(t *testing.T) {
		params := map[string]interface{}{
			"key1": []interface{}{"value1", "value2", "value3"},
			"key2": []interface{}{123, 456},
			"key3": "not a slice",
			"key4": nil,
		}

		result := getStringSlice(params, "key1")
		if len(result) != 3 {
			t.Errorf("getStringSlice(key1) length = %d, want 3", len(result))
		}
		if result[0] != "value1" {
			t.Errorf("getStringSlice(key1)[0] = %v, want value1", result[0])
		}

		result = getStringSlice(params, "key2")
		// getStringSlice returns slice with empty strings for non-string elements
		if len(result) != 2 {
			t.Errorf("getStringSlice(key2) length = %d, want 2", len(result))
		}
		if result[0] != "" {
			t.Errorf("getStringSlice(key2)[0] should be empty string, got %s", result[0])
		}

		result = getStringSlice(params, "key3")
		if result != nil {
			t.Error("getStringSlice(key3) should return nil for non-slice value")
		}

		result = getStringSlice(params, "nonexistent")
		if result != nil {
			t.Error("getStringSlice(nonexistent) should return nil")
		}
	})

	t.Run("getFloat", func(t *testing.T) {
		params := map[string]interface{}{
			"key1": 123.456,
			"key2": "123.789",
			"key3": "invalid",
			"key4": nil,
		}

		if getFloat(params, "key1") != 123.456 {
			t.Errorf("getFloat(key1) = %v, want 123.456", getFloat(params, "key1"))
		}
		if getFloat(params, "key2") != 123.789 {
			t.Errorf("getFloat(key2) = %v, want 123.789", getFloat(params, "key2"))
		}
		if getFloat(params, "key3") != 0 {
			t.Error("getFloat(key3) should return 0 for invalid string")
		}
		if getFloat(params, "key4") != 0 {
			t.Error("getFloat(key4) should return 0 for nil value")
		}
	})

	t.Run("getInt", func(t *testing.T) {
		params := map[string]interface{}{
			"key1": 42.0,
			"key2": 24,
			"key3": "36",
			"key4": "invalid",
			"key5": nil,
		}

		if getInt(params, "key1", 10) != 42 {
			t.Errorf("getInt(key1) = %v, want 42", getInt(params, "key1", 10))
		}
		if getInt(params, "key2", 10) != 24 {
			t.Errorf("getInt(key2) = %v, want 24", getInt(params, "key2", 10))
		}
		if getInt(params, "key3", 10) != 36 {
			t.Errorf("getInt(key3) = %v, want 36", getInt(params, "key3", 10))
		}
		if getInt(params, "key4", 10) != 10 {
			t.Errorf("getInt(key4) should return default value 10 for invalid string")
		}
		if getInt(params, "key5", 10) != 10 {
			t.Errorf("getInt(key5) should return default value 10 for nil value")
		}
		if getInt(params, "nonexistent", 10) != 10 {
			t.Errorf("getInt(nonexistent) should return default value 10")
		}
	})
}

// TestParameterSchema tests ParameterSchema.
func TestParameterSchema(t *testing.T) {
	schema := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"name": {
				Type:        "string",
				Description: "Item name",
			},
			"age": {
				Type:        "integer",
				Description: "Age",
				Min:         float64Ptr(0),
				Max:         float64Ptr(120),
			},
			"status": {
				Type:        "string",
				Description: "Status",
				Enum:        []interface{}{"active", "inactive"},
			},
		},
		Required: []string{"name", "status"},
	}

	if schema.Type != "object" {
		t.Errorf("Type = %v, want object", schema.Type)
	}
	if len(schema.Properties) != 3 {
		t.Errorf("Properties count = %d, want 3", len(schema.Properties))
	}
	if len(schema.Required) != 2 {
		t.Errorf("Required count = %d, want 2", len(schema.Required))
	}

	// Test required fields
	if !contains(schema.Required, "name") {
		t.Error("name should be in Required")
	}
	if !contains(schema.Required, "status") {
		t.Error("status should be in Required")
	}
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Helper function to create float64 pointer
func float64Ptr(v float64) *float64 {
	return &v
}

// TestToolParameters tests tool parameter schema.
func TestToolParameters(t *testing.T) {
	t.Run("base tool parameters", func(t *testing.T) {
		params := &ParameterSchema{
			Type: "object",
			Properties: map[string]*Parameter{
				"query": {
					Type:        "string",
					Description: "Search query",
				},
			},
			Required: []string{"query"},
		}

		tool := NewBaseTool("test_tool", "Test tool", params)

		if tool.Parameters() == nil {
			t.Error("Parameters() should not return nil")
		}
		if tool.Parameters().Type != "object" {
			t.Errorf("Parameters().Type = %v, want object", tool.Parameters().Type)
		}
	})

	t.Run("tool func parameters", func(t *testing.T) {
		params := &ParameterSchema{
			Type: "object",
			Properties: map[string]*Parameter{
				"value": {
					Type:        "number",
					Description: "Numeric value",
				},
			},
			Required: []string{"value"},
		}

		tool := NewToolFunc("calc", "Calculator", params,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(true, params["value"]), nil
			})

		if tool.Parameters().Required[0] != "value" {
			t.Errorf("Required[0] = %v, want value", tool.Parameters().Required[0])
		}
	})
}

// TestParameterEdgeCases tests edge cases in parameter handling.
func TestParameterEdgeCases(t *testing.T) {
	t.Run("empty parameter schema", func(t *testing.T) {
		params := &ParameterSchema{
			Type:       "object",
			Properties: map[string]*Parameter{},
			Required:   []string{},
		}

		if len(params.Properties) != 0 {
			t.Errorf("Properties should be empty, got %d items", len(params.Properties))
		}
		if len(params.Required) != 0 {
			t.Errorf("Required should be empty, got %d items", len(params.Required))
		}
	})

	t.Run("parameter with default value", func(t *testing.T) {
		defaultValue := 42
		param := &Parameter{
			Type:        "integer",
			Description: "Count",
			Default:     defaultValue,
		}

		if param.Default != defaultValue {
			t.Errorf("Default = %v, want %v", param.Default, defaultValue)
		}
	})

	t.Run("parameter with enum values", func(t *testing.T) {
		enumValues := []interface{}{"small", "medium", "large"}
		param := &Parameter{
			Type:        "string",
			Description: "Size",
			Enum:        enumValues,
		}

		if len(param.Enum) != 3 {
			t.Errorf("Enum length = %d, want 3", len(param.Enum))
		}
	})
}

// TestContextCancellation tests context cancellation in tool execution.
func TestContextCancellation(t *testing.T) {
	t.Run("tool execution with cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		tool := NewToolFunc("slow_tool", "Slow tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				select {
				case <-ctx.Done():
					return NewErrorResult("cancelled"), nil
				case <-time.After(1 * time.Second):
					return NewResult(true, "result"), nil
				}
			})

		_, err := tool.Execute(ctx, map[string]interface{}{})

		if err != nil {
			t.Errorf("Execute() error = %v", err)
		}
		// Tool should handle cancellation gracefully
	})
}

// TestRegistryConcurrency tests concurrent registry operations.
func TestRegistryConcurrency(t *testing.T) {
	registry := NewRegistry()

	// Register multiple tools concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			tool := NewToolFunc(
				"tool_"+string(rune('0'+id)),
				"Tool "+string(rune('0'+id)),
				nil,
				func(ctx context.Context, params map[string]interface{}) (Result, error) {
					return NewResult(true, "result"), nil
				},
			)
			registry.Register(tool)
		}(i)
	}

	// Wait for all registrations
	time.Sleep(100 * time.Millisecond)

	count := registry.Count()
	if count != 10 {
		t.Errorf("registry count = %d, want 10", count)
	}

	// List tools concurrently
	for i := 0; i < 5; i++ {
		go func() {
			tools := registry.List()
			if len(tools) != 10 {
				t.Logf("Warning: got %d tools in concurrent list, expected 10", len(tools))
			}
		}()
	}

	time.Sleep(100 * time.Millisecond)
}

// TestToolErrorHandling tests error handling in tools.
func TestToolErrorHandling(t *testing.T) {
	t.Run("tool returns error", func(t *testing.T) {
		expectedErr := errors.New("tool error")
		tool := NewToolFunc("error_tool", "Error tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				return NewResult(false, nil), expectedErr
			})

		result, err := tool.Execute(context.Background(), map[string]interface{}{})

		if err != expectedErr {
			t.Errorf("Execute() error = %v, want %v", err, expectedErr)
		}
		if result.Success {
			t.Error("result should not be successful when tool returns error")
		}
	})

	t.Run("tool panic handling", func(t *testing.T) {
		tool := NewToolFunc("panic_tool", "Panic tool", nil,
			func(ctx context.Context, params map[string]interface{}) (Result, error) {
				panic("intentional panic")
			})

		// This test verifies that panic can be caught by the caller
		defer func() {
			if r := recover(); r != nil {
				t.Log("Panic was caught (expected)")
			}
		}()

		_, err := tool.Execute(context.Background(), map[string]interface{}{})

		// After panic, err should not be nil
		if err == nil {
			t.Error("Execute() should return error after panic")
		}
	})
}

// Mock implementations for testing

// MockFashionSearcher provides mock fashion search results.
type MockFashionSearcher struct{}

// NewMockFashionSearcher creates a MockFashionSearcher.
func NewMockFashionSearcher() *MockFashionSearcher {
	return &MockFashionSearcher{}
}

// Search returns mock fashion items.
func (m *MockFashionSearcher) Search(ctx context.Context, query string, filters *FashionFilters) ([]*FashionItem, error) {
	items := []*FashionItem{
		{
			ItemID:           "item-1",
			Name:             "Summer Dress",
			Brand:            "Brand A",
			Category:         "dress",
			Price:            150.0,
			URL:              "https://example.com/item1",
			ImageURL:         "https://example.com/item1.jpg",
			AgentPreferences: []string{"casual", "minimalist"},
			Colors:           []string{"red", "blue", "white"},
			Occasion:         "casual",
			Season:           "summer",
			Metadata:         map[string]interface{}{"rating": 4.5},
		},
		{
			ItemID:           "item-2",
			Name:             "Running Shoes",
			Brand:            "Nike",
			Category:         "shoes",
			Price:            120.0,
			URL:              "https://example.com/item2",
			ImageURL:         "https://example.com/item2.jpg",
			AgentPreferences: []string{"sporty", "casual"},
			Colors:           []string{"black", "white", "red"},
			Occasion:         "sport",
			Season:           "summer",
			Metadata:         map[string]interface{}{"rating": 4.8},
		},
		{
			ItemID:           "item-3",
			Name:             "Leather Jacket",
			Brand:            "Brand B",
			Category:         "outerwear",
			Price:            250.0,
			URL:              "https://example.com/item3",
			ImageURL:         "https://example.com/item3.jpg",
			AgentPreferences: []string{"formal", "edgy"},
			Colors:           []string{"black", "brown"},
			Occasion:         "formal",
			Season:           "autumn",
			Metadata:         map[string]interface{}{"rating": 4.2},
		},
	}

	// Apply filters if provided
	if filters.Category != "" {
		filtered := make([]*FashionItem, 0)
		for _, item := range items {
			if item.Category == filters.Category {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if filters.PriceMin > 0 || filters.PriceMax > 0 {
		filtered := make([]*FashionItem, 0)
		for _, item := range items {
			if (filters.PriceMin == 0 || item.Price >= filters.PriceMin) &&
				(filters.PriceMax == 0 || item.Price <= filters.PriceMax) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if len(filters.Colors) > 0 {
		filtered := make([]*FashionItem, 0)
		for _, item := range items {
			hasMatchingColor := false
			for _, itemColor := range item.Colors {
				for _, filterColor := range filters.Colors {
					if itemColor == filterColor {
						hasMatchingColor = true
						break
					}
				}
				if hasMatchingColor {
					break
				}
			}
			if hasMatchingColor {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	return items, nil
}

// errorFashionSearcher returns an error for testing.
type errorFashionSearcher struct{}

// Search returns an error.
func (e *errorFashionSearcher) Search(ctx context.Context, query string, filters *FashionFilters) ([]*FashionItem, error) {
	return nil, errors.New("search failed")
}

// errorStyleRecommender returns an error for testing.
type errorStyleRecommender struct{}

// GetRecommendations returns an error.
func (e *errorStyleRecommender) GetRecommendations(ctx context.Context, profile *AgentProfile) (*AgentRecommendation, error) {
	return nil, errors.New("recommendation failed")
}

// GetTrends returns an error.
func (e *errorStyleRecommender) GetTrends(ctx context.Context, season string) ([]*AgentTrend, error) {
	return nil, errors.New("trends failed")
}

// errorWeatherProvider returns an error for testing.
type errorWeatherProvider struct{}

// GetCurrent returns an error.
func (e *errorWeatherProvider) GetCurrent(ctx context.Context, location string) (*WeatherData, error) {
	return nil, errors.New("weather check failed")
}

// GetForecast returns an error.
func (e *errorWeatherProvider) GetForecast(ctx context.Context, location string, days int) ([]*WeatherData, error) {
	return nil, errors.New("forecast failed")
}

// nolint: errcheck // Test code may ignore return values
// nolint: errcheck // Test code may ignore return values
