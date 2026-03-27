package output

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"goagent/internal/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TemplateEngine handles prompt templates.
type TemplateEngine struct {
	funcs template.FuncMap
}

// NewTemplateEngine creates a new TemplateEngine.
func NewTemplateEngine() *TemplateEngine {
	titleCase := cases.Title(language.English)
	return &TemplateEngine{
		funcs: template.FuncMap{
			"upper":   strings.ToUpper,
			"lower":   strings.ToLower,
			"title":   func(s string) string { return titleCase.String(s) },
			"trim":    strings.TrimSpace,
			"join":    strings.Join,
			"json":    toJSON,
			"to_yaml": toYAML,
		},
	}
}

// Render renders a template string with given data.
func (e *TemplateEngine) Render(tmpl string, data interface{}) (string, error) {
	t, err := template.New("prompt").Funcs(e.funcs).Parse(tmpl)
	if err != nil {
		return "", errors.Wrap(err, "parse template")
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", errors.Wrap(err, "execute template")
	}

	return buf.String(), nil
}

// RenderFile renders a template file with given data.
func (e *TemplateEngine) RenderFile(path string, data interface{}) (string, error) {
	tmpl, err := template.ParseFiles(path)
	if err != nil {
		return "", errors.Wrap(err, "parse template file")
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", errors.Wrap(err, "execute template")
	}

	return buf.String(), nil
}

// RegisterFunc registers a custom function.
func (e *TemplateEngine) RegisterFunc(name string, fn interface{}) {
	e.funcs[name] = fn
}

// Common templates.
var (
	// RecommendationPrompt is the prompt for style recommendations.
	RecommendationPrompt = `You are a fashion consultant. Analyze the user's preferences and provide personalized style recommendations.

User Profile:
- Gender: {{.Gender}}
- Age: {{.Age}}
- Style Preferences: {{.StylePreferences}}
- Budget Range: ${{.BudgetMin}} - ${{.BudgetMax}}
- Favorite Colors: {{.FavoriteColors}}
- Favorite Brands: {{.FavoriteBrands}}
- Occasion: {{.Occasion}}
- Season: {{.Season}}

Provide a JSON response with the following schema:
{{.Schema}}

Focus on:
1. Coordinating colors and styles
2. Matching the occasion
3. Staying within budget
4. Expressing personal style`

	// ProfileExtractionPrompt extracts user profile from natural language.
	ProfileExtractionPrompt = `Extract user profile information from the following text:

{{.Input}}

Extract and return a JSON object with these fields:
- user_id (optional)
- gender (male/female/other)
- age (number)
- style_preferences (array of strings)
- budget_min (number)
- budget_max (number)
- favorite_colors (array)
- favorite_brands (array)
- body_type (string)
- occupation (string)
- location (string)

If a field is not mentioned, omit it from the JSON.`

	// StyleAnalysisPrompt analyzes style preferences.
	StyleAnalysisPrompt = `Analyze the following style description and identify key characteristics:

{{.Description}}

Return a JSON object with:
- primary_style (e.g., casual, formal, sporty, bohemian)
- secondary_styles (array)
- color_palette (array of hex colors)
- silhouette_preference (e.g., slim, loose, fitted)
- key_elements (array of strings)`
)

// RenderRecommendation renders the recommendation prompt.
func (e *TemplateEngine) RenderRecommendation(data map[string]interface{}) (string, error) {
	schema, _ := GetRecommendResultSchema().ToJSONString()
	data["Schema"] = schema
	return e.Render(RecommendationPrompt, data)
}

// RenderProfileExtraction renders the profile extraction prompt.
func (e *TemplateEngine) RenderProfileExtraction(input string) (string, error) {
	return e.Render(ProfileExtractionPrompt, map[string]string{
		"Input": input,
	})
}

// RenderStyleAnalysis renders the style analysis prompt.
func (e *TemplateEngine) RenderStyleAnalysis(description string) (string, error) {
	return e.Render(StyleAnalysisPrompt, map[string]string{
		"Description": description,
	})
}

// Helper functions.
func toJSON(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func toYAML(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

// Global default template engine.
var defaultEngine = NewTemplateEngine()

// RenderWithDefault renders using the default engine.
func RenderWithDefault(tmpl string, data interface{}) (string, error) {
	return defaultEngine.Render(tmpl, data)
}

// RenderRecommendationWithDefault renders recommendation prompt.
func RenderRecommendationWithDefault(data map[string]interface{}) (string, error) {
	return defaultEngine.RenderRecommendation(data)
}

// RenderProfileExtractionWithDefault renders profile extraction prompt.
func RenderProfileExtractionWithDefault(input string) (string, error) {
	return defaultEngine.RenderProfileExtraction(input)
}

// RenderStyleAnalysisWithDefault renders style analysis prompt.
func RenderStyleAnalysisWithDefault(description string) (string, error) {
	return defaultEngine.RenderStyleAnalysis(description)
}
