package types

import (
	"testing"
	"time"
)

// TestResourceFilters tests ResourceFilters structure.
func TestResourceFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters ResourceFilters
	}{
		{
			name: "fully populated filters",
			filters: ResourceFilters{
				Category:         "technology",
				AgentPreferences: []string{"performance", "reliable"},
				PriceMin:         50.0,
				PriceMax:         200.0,
				Tags:             []string{"cloud", "ai"},
				Labels:           []string{"enterprise", "open-source"},
				Context:          "production",
				Season:           "Q1",
			},
		},
		{
			name: "minimal filters",
			filters: ResourceFilters{
				Category: "software",
			},
		},
		{
			name:    "empty filters",
			filters: ResourceFilters{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify that filters can be created without errors.
			_ = tt.filters
		})
	}
}

// TestResourceItem tests ResourceItem structure.
func TestResourceItem(t *testing.T) {
	tests := []struct {
		name string
		item ResourceItem
	}{
		{
			name: "fully populated item",
			item: ResourceItem{
				ItemID:           "item-123",
				Name:             "Cloud Service",
				Brand:            "TechCorp",
				Category:         "compute",
				Price:            120.50,
				URL:              "https://example.com/item/123",
				ImageURL:         "https://example.com/images/123.jpg",
				AgentPreferences: []string{"scalable", "secure"},
				Tags:             []string{"cloud", "enterprise"},
				Context:          "production",
				Season:           "all",
				Metadata: map[string]interface{}{
					"region": "us-east-1",
					"tier":   "premium",
				},
			},
		},
		{
			name: "minimal item",
			item: ResourceItem{
				ItemID:   "item-456",
				Name:     "API Gateway",
				Category: "networking",
			},
		},
		{
			name: "item with nil metadata",
			item: ResourceItem{
				ItemID:   "item-789",
				Name:     "Load Balancer",
				Category: "networking",
				Metadata: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.item.ItemID == "" {
				t.Error("ItemID should not be empty")
			}
			if tt.item.Name == "" {
				t.Error("Name should not be empty")
			}
		})
	}
}

// TestAgentUserProfile tests AgentUserProfile structure.
func TestAgentUserProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile AgentUserProfile
	}{
		{
			name: "fully populated profile",
			profile: AgentUserProfile{
				Gender:           "unspecified",
				AgeRange:         "25-35",
				BodyType:         "",
				StylePreferences: []string{"minimalist", "efficient"},
				ColorPreferences: []string{"dark", "light"},
				BudgetRange: &BudgetRange{
					Min: 50.0,
					Max: 500.0,
				},
				Context:  "work",
				Season:   "spring",
				Location: "New York",
			},
		},
		{
			name: "profile with nil budget range",
			profile: AgentUserProfile{
				Gender:           "unspecified",
				AgeRange:         "20-30",
				StylePreferences: []string{"casual"},
				BudgetRange:      nil,
			},
		},
		{
			name: "minimal profile",
			profile: AgentUserProfile{
				Gender: "unspecified",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.profile.Gender == "" {
				t.Error("Gender should not be empty")
			}
		})
	}
}

// TestBudgetRange tests BudgetRange structure.
func TestBudgetRange(t *testing.T) {
	tests := []struct {
		name string
		b    BudgetRange
	}{
		{
			name: "valid budget range",
			b: BudgetRange{
				Min: 100.0,
				Max: 1000.0,
			},
		},
		{
			name: "zero budget",
			b: BudgetRange{
				Min: 0.0,
				Max: 0.0,
			},
		},
		{
			name: "min equals max",
			b: BudgetRange{
				Min: 500.0,
				Max: 500.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.b.Min < 0 {
				t.Error("Min budget should not be negative")
			}
			if tt.b.Max < 0 {
				t.Error("Max budget should not be negative")
			}
		})
	}
}

// TestTaskRecommendation tests TaskRecommendation structure.
func TestTaskRecommendation(t *testing.T) {
	tests := []struct {
		name string
		rec  TaskRecommendation
	}{
		{
			name: "fully populated recommendation",
			rec: TaskRecommendation{
				PrimaryCategory:     "infrastructure",
				SecondaryCategories: []string{"compute", "storage"},
				Tags:                []string{"cloud", "scalable"},
				Suggestions: []Suggestion{
					{
						Name:        "Auto Scaling Setup",
						Items:       []string{"EC2", "ALB", "CloudWatch"},
						Context:     "production",
						MatchScore:  0.95,
						Description: "Configure auto scaling for production workloads",
					},
				},
				Tips: []string{
					"Monitor CPU utilization",
					"Set appropriate cooldown periods",
				},
				Metadata: map[string]interface{}{
					"generated_at": "2024-01-01",
					"version":      "1.0",
				},
			},
		},
		{
			name: "minimal recommendation",
			rec: TaskRecommendation{
				PrimaryCategory: "networking",
				Suggestions:     []Suggestion{},
				Tips:            []string{},
			},
		},
		{
			name: "recommendation with nil metadata",
			rec: TaskRecommendation{
				PrimaryCategory: "security",
				Metadata:        nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rec.PrimaryCategory == "" {
				t.Error("PrimaryCategory should not be empty")
			}
		})
	}
}

// TestSuggestion tests Suggestion structure.
func TestSuggestion(t *testing.T) {
	tests := []struct {
		name string
		s    Suggestion
	}{
		{
			name: "fully populated suggestion",
			s: Suggestion{
				Name:        "Database Migration",
				Items:       []string{"schema", "data", "validation"},
				Context:     "production",
				MatchScore:  0.88,
				Description: "Migrate database to new schema version",
			},
		},
		{
			name: "minimal suggestion",
			s: Suggestion{
				Name:  "Simple Task",
				Items: []string{"step1"},
			},
		},
		{
			name: "suggestion with empty items",
			s: Suggestion{
				Name:  "Template",
				Items: []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.s.Name == "" {
				t.Error("Name should not be empty")
			}
			if tt.s.MatchScore < 0 || tt.s.MatchScore > 1 {
				t.Errorf("MatchScore should be between 0 and 1, got %f", tt.s.MatchScore)
			}
		})
	}
}

// TestTrend tests Trend structure.
func TestTrend(t *testing.T) {
	tests := []struct {
		name  string
		trend Trend
	}{
		{
			name: "fully populated trend",
			trend: Trend{
				TrendID:     "trend-2024-001",
				Name:        "Edge Computing",
				Category:    "infrastructure",
				Popularity:  0.85,
				Season:      "Q1",
				KeyElements: []string{"low-latency", "distributed"},
				Description: "Processing data closer to the source",
			},
		},
		{
			name: "minimal trend",
			trend: Trend{
				TrendID: "trend-001",
				Name:    "Test Trend",
			},
		},
		{
			name: "trend with popularity 0",
			trend: Trend{
				TrendID:    "trend-002",
				Name:       "New Trend",
				Popularity: 0.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.trend.TrendID == "" {
				t.Error("TrendID should not be empty")
			}
			if tt.trend.Name == "" {
				t.Error("Name should not be empty")
			}
			if tt.trend.Popularity < 0 || tt.trend.Popularity > 1 {
				t.Errorf("Popularity should be between 0 and 1, got %f", tt.trend.Popularity)
			}
		})
	}
}

// TestWeatherData tests WeatherData structure.
func TestWeatherData(t *testing.T) {
	tests := []struct {
		name    string
		weather WeatherData
	}{
		{
			name: "fully populated weather data",
			weather: WeatherData{
				Location:      "New York",
				Temperature:   22.5,
				Condition:     "sunny",
				Humidity:      65,
				WindSpeed:     12.3,
				UVIndex:       6,
				Precipitation: 0.0,
				Timestamp:     time.Now(),
				Metadata: map[string]interface{}{
					"source": "weather-api",
					"unit":   "celsius",
				},
			},
		},
		{
			name: "minimal weather data",
			weather: WeatherData{
				Location:    "London",
				Temperature: 15.0,
				Condition:   "cloudy",
			},
		},
		{
			name: "weather data with nil metadata",
			weather: WeatherData{
				Location:  "Tokyo",
				Timestamp: time.Now(),
				Condition: "clear",
				Metadata:  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.weather.Location == "" {
				t.Error("Location should not be empty")
			}
			if tt.weather.Condition == "" {
				t.Error("Condition should not be empty")
			}
			if tt.weather.Humidity < 0 || tt.weather.Humidity > 100 {
				t.Errorf("Humidity should be between 0 and 100, got %d", tt.weather.Humidity)
			}
			if tt.weather.UVIndex < 0 {
				t.Errorf("UVIndex should not be negative, got %d", tt.weather.UVIndex)
			}
		})
	}
}

// TestResourceFiltersEdgeCases tests edge cases for ResourceFilters.
func TestResourceFiltersEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		filters ResourceFilters
	}{
		{
			name: "negative price range",
			filters: ResourceFilters{
				PriceMin: -10.0,
				PriceMax: 100.0,
			},
		},
		{
			name: "max less than min",
			filters: ResourceFilters{
				PriceMin: 200.0,
				PriceMax: 100.0,
			},
		},
		{
			name: "empty arrays",
			filters: ResourceFilters{
				AgentPreferences: []string{},
				Tags:             []string{},
				Labels:           []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filters.PriceMin > tt.filters.PriceMax {
				t.Logf("Warning: PriceMin (%f) is greater than PriceMax (%f)",
					tt.filters.PriceMin, tt.filters.PriceMax)
			}
		})
	}
}
