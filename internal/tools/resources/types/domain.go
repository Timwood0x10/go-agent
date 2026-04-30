package types

import "time"

// ResourceFilters holds search filters for resource items.
type ResourceFilters struct {
	Category         string
	AgentPreferences []string
	PriceMin         float64
	PriceMax         float64
	Tags             []string
	Labels           []string
	Context          string
	Season           string
}

// ResourceItem represents a generic resource item.
type ResourceItem struct {
	ItemID           string                 `json:"item_id"`
	Name             string                 `json:"name"`
	Brand            string                 `json:"brand"`
	Category         string                 `json:"category"`
	Price            float64                `json:"price"`
	URL              string                 `json:"url"`
	ImageURL         string                 `json:"image_url"`
	AgentPreferences []string               `json:"agent_preferences"`
	Tags             []string               `json:"tags"`
	Context          string                 `json:"context"`
	Season           string                 `json:"season"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// AgentUserProfile holds user preferences for agent processing.
type AgentUserProfile struct {
	Gender           string       `json:"gender"`
	AgeRange         string       `json:"age_range"`
	BodyType         string       `json:"body_type"`
	StylePreferences []string     `json:"style_preferences"`
	ColorPreferences []string     `json:"color_preferences"`
	BudgetRange      *BudgetRange `json:"budget_range"`
	Context          string       `json:"context"`
	Season           string       `json:"season"`
	Location         string       `json:"location"`
}

// BudgetRange represents a budget range.
type BudgetRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// TaskRecommendation holds agent-generated recommendations.
type TaskRecommendation struct {
	PrimaryCategory     string                 `json:"primary_category"`
	SecondaryCategories []string               `json:"secondary_categories"`
	Tags                []string               `json:"tags"`
	Suggestions         []Suggestion           `json:"suggestions"`
	Tips                []string               `json:"tips"`
	Metadata            map[string]interface{} `json:"metadata"`
}

// Suggestion represents a single suggestion from an agent.
type Suggestion struct {
	Name        string   `json:"name"`
	Items       []string `json:"items"`
	Context     string   `json:"context"`
	MatchScore  float64  `json:"match_score"`
	Description string   `json:"description"`
}

// Trend represents a domain trend.
type Trend struct {
	TrendID     string   `json:"trend_id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Popularity  float64  `json:"popularity"`
	Season      string   `json:"season"`
	KeyElements []string `json:"key_elements"`
	Description string   `json:"description"`
}

// WeatherData holds weather information.
type WeatherData struct {
	Location      string                 `json:"location"`
	Temperature   float64                `json:"temperature"`
	Condition     string                 `json:"condition"`
	Humidity      int                    `json:"humidity"`
	WindSpeed     float64                `json:"wind_speed"`
	UVIndex       int                    `json:"uv_index"`
	Precipitation float64                `json:"precipitation"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata"`
}
