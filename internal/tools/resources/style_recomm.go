package resources

import (
	"context"
)

// StyleRecommender provides style recommendations.
type StyleRecommender struct {
	*BaseTool
	recommender StyleRecommenderEngine
}

// StyleRecommenderEngine defines the interface for style recommendations.
type StyleRecommenderEngine interface {
	GetRecommendations(ctx context.Context, profile *StyleProfile) (*StyleRecommendation, error)
	GetTrends(ctx context.Context, season string) ([]*StyleTrend, error)
}

// StyleProfile holds user style preferences.
type StyleProfile struct {
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

// StyleRecommendation holds style recommendations.
type StyleRecommendation struct {
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

// StyleTrend represents a style trend.
type StyleTrend struct {
	TrendID     string   `json:"trend_id"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	Popularity  float64  `json:"popularity"`
	Season      string   `json:"season"`
	KeyElements []string `json:"key_elements"`
	Description string   `json:"description"`
}

// NewStyleRecommender creates a new StyleRecommender tool.
func NewStyleRecommender(recommender StyleRecommenderEngine) *StyleRecommender {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"gender": {
				Type:        "string",
				Description: "User gender",
				Enum:        []interface{}{"male", "female", "other"},
			},
			"age_range": {
				Type:        "string",
				Description: "Age range (e.g., 18-25, 26-35)",
			},
			"body_type": {
				Type:        "string",
				Description: "Body type",
			},
			"style_preferences": {
				Type:        "array",
				Description: "List of style preferences",
			},
			"color_preferences": {
				Type:        "array",
				Description: "List of preferred colors",
			},
			"budget_min": {
				Type:        "number",
				Description: "Minimum budget",
			},
			"budget_max": {
				Type:        "number",
				Description: "Maximum budget",
			},
			"occasion": {
				Type:        "string",
				Description: "Occasion (casual, business, formal, party)",
			},
			"season": {
				Type:        "string",
				Description: "Season (spring, summer, autumn, winter)",
			},
		},
		Required: []string{"gender", "occasion"},
	}

	sr := &StyleRecommender{
		recommender: recommender,
	}
	sr.BaseTool = NewBaseTool("style_recommend", "Get personalized style recommendations", params)

	return sr
}

// Execute provides style recommendations.
func (t *StyleRecommender) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	profile := &StyleProfile{
		Gender:           getString(params, "gender"),
		AgeRange:         getString(params, "age_range"),
		BodyType:         getString(params, "body_type"),
		StylePreferences: getStringSlice(params, "style_preferences"),
		ColorPreferences: getStringSlice(params, "color_preferences"),
		Occasion:         getString(params, "occasion"),
		Season:           getString(params, "season"),
		Location:         getString(params, "location"),
	}

	budgetMin := getFloat(params, "budget_min")
	budgetMax := getFloat(params, "budget_max")
	if budgetMin > 0 || budgetMax > 0 {
		profile.BudgetRange = &BudgetRange{
			Min: budgetMin,
			Max: budgetMax,
		}
	}

	rec, err := t.recommender.GetRecommendations(ctx, profile)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	return NewResult(true, rec), nil
}

// NewStyleRecommenderWithTrends creates a tool that also supports trend queries.
func NewStyleRecommenderWithTrends(recommender StyleRecommenderEngine) *StyleRecommender {
	tool := NewStyleRecommender(recommender)

	// Add trend parameter
	tool.parameters.Properties["get_trends"] = &Parameter{
		Type:        "boolean",
		Description: "Get current trends instead of recommendations",
	}
	tool.parameters.Properties["season"] = &Parameter{
		Type:        "string",
		Description: "Season for trends",
	}

	return tool
}

// MockStyleRecommender provides mock recommendations.
type MockStyleRecommender struct{}

// NewMockStyleRecommender creates a MockStyleRecommender.
func NewMockStyleRecommender() *MockStyleRecommender {
	return &MockStyleRecommender{}
}

// GetRecommendations returns mock recommendations.
func (m *MockStyleRecommender) GetRecommendations(ctx context.Context, profile *StyleProfile) (*StyleRecommendation, error) {
	return &StyleRecommendation{
		PrimaryStyle:    "casual",
		SecondaryStyles: []string{"minimalist", "streetwear"},
		ColorPalette:    []string{"navy", "white", "gray"},
		Outfits: []OutfitSuggestion{
			{
				Name:        "Casual Friday",
				Items:       []string{"navy blazer", "white t-shirt", "dark jeans", "white sneakers"},
				Occasion:    "casual",
				MatchScore:  0.9,
				Description: "Clean and comfortable casual look",
			},
		},
		Tips: []string{
			"Layer with a light jacket for cooler evenings",
			"Accessorize with a simple watch",
		},
	}, nil
}

// GetTrends returns mock trends.
func (m *MockStyleRecommender) GetTrends(ctx context.Context, season string) ([]*StyleTrend, error) {
	return []*StyleTrend{
		{
			TrendID:     "sustainable_fashion",
			Name:        "Sustainable Fashion",
			Category:    "lifestyle",
			Popularity:  0.95,
			Season:      season,
			KeyElements: []string{"organic materials", "recycled fabrics", "neutral colors"},
			Description: "Eco-friendly clothing continues to grow in popularity",
		},
	}, nil
}
