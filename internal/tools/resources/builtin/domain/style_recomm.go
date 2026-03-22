package builtin

import (
	"context"

	"goagent/internal/tools/resources/base"
	"goagent/internal/tools/resources/core"
	"goagent/internal/tools/resources/types"
)

// AgentRecommender provides style recommendations.
type AgentRecommender struct {
	*base.BaseTool
	recommender AgentRecommenderEngine
}

// AgentRecommenderEngine defines the interface for style recommendations.
type AgentRecommenderEngine interface {
	GetRecommendations(ctx context.Context, profile *types.AgentProfile) (*types.AgentRecommendation, error)
	GetTrends(ctx context.Context, season string) ([]*types.AgentTrend, error)
}

// NewAgentRecommender creates a new AgentRecommender tool.
func NewAgentRecommender(recommender AgentRecommenderEngine) *AgentRecommender {
	params := &core.ParameterSchema{
		Type: "object",
		Properties: map[string]*core.Parameter{
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

	sr := &AgentRecommender{
		recommender: recommender,
	}
	sr.BaseTool = base.NewBaseTool("style_recommend", "Get personalized style recommendations", params)

	return sr
}

// Execute provides style recommendations.
func (t *AgentRecommender) Execute(ctx context.Context, params map[string]interface{}) (core.Result, error) {
	profile := &types.AgentProfile{
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
		profile.BudgetRange = &types.BudgetRange{
			Min: budgetMin,
			Max: budgetMax,
		}
	}

	rec, err := t.recommender.GetRecommendations(ctx, profile)
	if err != nil {
		return core.NewErrorResult(err.Error()), nil
	}

	return core.NewResult(true, rec), nil
}

// NewAgentRecommenderWithTrends creates a tool that also supports trend queries.
func NewAgentRecommenderWithTrends(recommender AgentRecommenderEngine) *AgentRecommender {
	tool := NewAgentRecommender(recommender)

	// Add trend parameter
	tool.Parameters().Properties["get_trends"] = &core.Parameter{
		Type:        "boolean",
		Description: "Get current trends instead of recommendations",
	}
	tool.Parameters().Properties["season"] = &core.Parameter{
		Type:        "string",
		Description: "Season for trends",
	}

	return tool
}

// MockAgentRecommender provides mock recommendations.
type MockAgentRecommender struct{}

// NewMockAgentRecommender creates a MockAgentRecommender.
func NewMockAgentRecommender() *MockAgentRecommender {
	return &MockAgentRecommender{}
}

// GetRecommendations returns mock recommendations.
func (m *MockAgentRecommender) GetRecommendations(ctx context.Context, profile *types.AgentProfile) (*types.AgentRecommendation, error) {
	return &types.AgentRecommendation{
		PrimaryStyle:    "casual",
		SecondaryStyles: []string{"minimalist", "streetwear"},
		ColorPalette:    []string{"navy", "white", "gray"},
		Outfits: []types.OutfitSuggestion{
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
func (m *MockAgentRecommender) GetTrends(ctx context.Context, season string) ([]*types.AgentTrend, error) {
	return []*types.AgentTrend{
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