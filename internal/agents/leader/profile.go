package leader

import (
	"context"

	"styleagent/internal/core/models"
)

// profileParser parses user profile from natural language input.
type profileParser struct {
	// Add LLM client or parser dependencies here
}

// NewProfileParser creates a new ProfileParser.
func NewProfileParser() ProfileParser {
	return &profileParser{}
}

// Parse parses user input into UserProfile.
func (p *profileParser) Parse(ctx context.Context, input string) (*models.UserProfile, error) {
	// TODO: Implement LLM-based parsing
	// This is a placeholder implementation
	profile := &models.UserProfile{
		Style:     []models.StyleTag{models.StyleCasual},
		Occasions: []models.Occasion{models.OccasionDaily},
		Budget:    models.NewPriceRange(0, 1000),
	}

	return profile, nil
}
