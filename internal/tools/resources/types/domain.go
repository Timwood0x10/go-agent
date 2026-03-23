package types

import "time"

// FashionFilters holds search filters for fashion items.
type FashionFilters struct {
	Category         string
	AgentPreferences []string
	PriceMin         float64
	PriceMax         float64
	Colors           []string
	Brands           []string
	Occasion         string
	Season           string
}

// FashionItem represents a fashion item.
type FashionItem struct {
	ItemID           string                 `json:"item_id"`
	Name             string                 `json:"name"`
	Brand            string                 `json:"brand"`
	Category         string                 `json:"category"`
	Price            float64                `json:"price"`
	URL              string                 `json:"url"`
	ImageURL         string                 `json:"image_url"`
	AgentPreferences []string               `json:"agent_preferences"`
	Colors           []string               `json:"colors"`
	Occasion         string                 `json:"occasion"`
	Season           string                 `json:"season"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// AgentProfile holds user style preferences.
type AgentProfile struct {
	Gender           string       `json:"gender"`
	AgeRange         string       `json:"age_range"`
	BodyType         string       `json:"body_type"`
	StylePreferences []string     `json:"style_preferences"`
	ColorPreferences []string     `json:"color_preferences"`
	BudgetRange      *BudgetRange `json:"budget_range"`
	Occasion         string       `json:"occasion"`
	Season           string       `json:"season"`
	Location         string       `json:"location"`
}

// BudgetRange represents a budget range.
type BudgetRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// AgentRecommendation holds style recommendations.
type AgentRecommendation struct {
	PrimaryStyle    string                 `json:"primary_style"`
	SecondaryStyles []string               `json:"secondary_styles"`
	ColorPalette    []string               `json:"color_palette"`
	Outfits         []OutfitSuggestion     `json:"outfits"`
	Tips            []string               `json:"tips"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// OutfitSuggestion represents an outfit suggestion.
type OutfitSuggestion struct {
	Name        string   `json:"name"`
	Items       []string `json:"items"`
	Occasion    string   `json:"occasion"`
	MatchScore  float64  `json:"match_score"`
	Description string   `json:"description"`
}

// AgentTrend represents a style trend.
type AgentTrend struct {
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
