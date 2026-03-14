package leader

import (
	"context"
	"fmt"

	apperrors "goagent/internal/core/errors"
	"goagent/internal/core/models"
	"goagent/internal/llm/output"
)

// profileParser parses user profile from natural language input.
type profileParser struct {
	llmAdapter output.LLMAdapter
	template   *output.TemplateEngine
	promptTpl  string
	validator  *output.Validator
	maxRetries int
}

// NewProfileParser creates a new ProfileParser with LLM support.
func NewProfileParser(
	llmAdapter output.LLMAdapter,
	template *output.TemplateEngine,
	promptTpl string,
	validator *output.Validator,
	maxRetries int,
) ProfileParser {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &profileParser{
		llmAdapter: llmAdapter,
		template:   template,
		promptTpl:  promptTpl,
		validator:  validator,
		maxRetries: maxRetries,
	}
}

// Parse parses user input into UserProfile using LLM.
func (p *profileParser) Parse(ctx context.Context, input string) (*models.UserProfile, error) {
	// If no LLM adapter, return default profile
	if p.llmAdapter == nil {
		return p.getDefaultProfile(), nil
	}

	fmt.Printf("[DEBUG] Parsing profile with LLM for input: %s\n", input)

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		profile, err := p.parseOnce(ctx, input)
		if err != nil {
			fmt.Printf("[DEBUG] Parse attempt %d failed: %v\n", attempt+1, err)
			continue
		}

		// Validate result
		if err := p.validateProfile(profile); err != nil {
			fmt.Printf("[DEBUG] Validate attempt %d failed: %v\n", attempt+1, err)
			continue
		}

		fmt.Printf("[DEBUG] Profile parsed: %+v\n", profile)
		return profile, nil
	}

	// Fallback to default profile
	return p.getDefaultProfile(), nil
}

func (p *profileParser) getDefaultProfile() *models.UserProfile {
	return &models.UserProfile{
		Style:     []models.StyleTag{models.StyleCasual},
		Occasions: []models.Occasion{models.OccasionDaily},
		Budget:    models.NewPriceRange(0, 1000),
	}
}

func (p *profileParser) parseOnce(ctx context.Context, input string) (*models.UserProfile, error) {
	// Render prompt
	prompt, err := p.template.Render(p.promptTpl, map[string]string{
		"input": input,
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrPromptRenderFailed, err)
	}

	// Call LLM
	response, err := p.llmAdapter.Generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrLLMGenerateFailed, err)
	}

	// Parse response
	profile, err := p.parseResponse(response)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrProfileParsingFailed, err)
	}

	return profile, nil
}

func (p *profileParser) parseResponse(response string) (*models.UserProfile, error) {
	// Debug: print raw response
	fmt.Printf("[DEBUG ProfileParser] Raw LLM response: %s\n", response[:min(500, len(response))])

	// Try to parse as JSON
	parser := output.NewParser()
	data, err := parser.ParseJSON(response)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", apperrors.ErrLLMParserFailed, err)
	}

	// Debug: print parsed data
	fmt.Printf("[DEBUG ProfileParser] Parsed data keys: ")
	for k := range data {
		fmt.Printf("%s ", k)
	}
	fmt.Println()

	// Extract fields
	profile := &models.UserProfile{}

	// Parse style
	if style, ok := data["style"]; ok {
		if styles, ok := style.([]interface{}); ok {
			for _, s := range styles {
				if str, ok := s.(string); ok {
					profile.Style = append(profile.Style, models.StyleTag(str))
				}
			}
		}
	}

	// Parse occasions
	if occasions, ok := data["occasions"]; ok {
		if occs, ok := occasions.([]interface{}); ok {
			for _, o := range occs {
				if str, ok := o.(string); ok {
					profile.Occasions = append(profile.Occasions, models.Occasion(str))
				}
			}
		}
	}

	// Parse budget - support both number (e.g., 10000) and object (e.g., {"min": 5000, "max": 10000})
	if budget, ok := data["budget"]; ok && budget != nil {
		switch b := budget.(type) {
		case float64:
			// Budget is a number like 10000
			profile.Budget = models.NewPriceRange(0, b)
		case map[string]interface{}:
			// Budget is an object like {"min": 5000, "max": 10000}
			min := 0.0
			max := 10000.0
			if v, ok := b["min"]; ok {
				if f, ok := toFloat64(v); ok {
					min = f
				}
			}
			if v, ok := b["max"]; ok {
				if f, ok := toFloat64(v); ok {
					max = f
				}
			}
			profile.Budget = models.NewPriceRange(min, max)
		}
	}

	// Set defaults if empty
	if len(profile.Style) == 0 {
		profile.Style = []models.StyleTag{models.StyleCasual}
	}
	if len(profile.Occasions) == 0 {
		profile.Occasions = []models.Occasion{models.OccasionDaily}
	}
	if profile.Budget == nil {
		profile.Budget = models.NewPriceRange(0, 10000)
	}

	// Initialize Preferences map if nil
	if profile.Preferences == nil {
		profile.Preferences = make(map[string]any)
	}

	// Dynamically extract ALL fields from JSON response into Preferences
	// This makes the parser flexible for any scenario (travel, fashion, etc.)
	// The TaskPlanner then decides which agents to call based on triggers
	for key, value := range data {
		// Skip fields already parsed into dedicated fields
		if key == "style" || key == "occasions" || key == "budget" {
			continue
		}
		// Store all other fields in Preferences for trigger-based matching
		if value != nil {
			profile.Preferences[key] = value
		}
	}

	return profile, nil
}

func (p *profileParser) validateProfile(profile *models.UserProfile) error {
	if profile == nil {
		return apperrors.ErrNilPointer
	}
	if len(profile.Style) == 0 {
		return apperrors.ErrProfileValidationFailed
	}
	return nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		var f float64
		_, err := fmt.Sscanf(val, "%f", &f)
		return f, err == nil
	}
	return 0, false
}
