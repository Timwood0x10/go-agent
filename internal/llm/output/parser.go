package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"goagent/internal/core/models"
)

// Parser parses LLM output into structured types.
type Parser struct {
	fixJSON bool
}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{
		fixJSON: true,
	}
}

// ParseRecommendResult parses LLM output into RecommendResult.
func (p *Parser) ParseRecommendResult(output string) (*models.RecommendResult, error) {
	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return nil, ErrInvalidJSON
	}

	var result models.RecommendResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		if !p.fixJSON {
			return nil, fmt.Errorf("%w: %w", ErrInvalidJSON, err)
		}

		fixed, fixErr := p.fixJSONString(jsonStr)
		if fixErr != nil {
			return nil, fmt.Errorf("%w: %w (tried fix: %v)", ErrInvalidJSON, err, fixErr)
		}

		if err := json.Unmarshal([]byte(fixed), &result); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalidJSON, err)
		}
	}

	return &result, nil
}

// extractJSON extracts JSON from output.
func (p *Parser) extractJSON(output string) string {
	output = strings.TrimSpace(output)

	// Try to find JSON in markdown code blocks
	markdownPattern := regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
	matches := markdownPattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to find JSON object directly
	start := strings.Index(output, "{")
	if start == -1 {
		return ""
	}

	// Find matching closing brace
	depth := 0
	end := -1
	for i := start; i < len(output); i++ {
		if output[i] == '{' {
			depth++
		} else if output[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
				break
			}
		}
	}

	if end == -1 {
		return ""
	}

	return output[start:end]
}

// fixJSONString attempts to fix common JSON errors.
func (p *Parser) fixJSONString(jsonStr string) (string, error) {
	fixed := jsonStr

	// Remove trailing commas
	trailingComma := regexp.MustCompile(",\\s*([\\}\\]])")
	fixed = trailingComma.ReplaceAllString(fixed, "$1")

	// Remove comments
	singleLineComment := regexp.MustCompile("//.*$")
	fixed = singleLineComment.ReplaceAllString(fixed, "")

	multiLineComment := regexp.MustCompile("/\\*[\\s\\S]*?\\*/")
	fixed = multiLineComment.ReplaceAllString(fixed, "")

	// Fix unquoted keys
	unquotedKey := regexp.MustCompile("([{,])\\s*([a-zA-Z_][a-zA-Z0-9_]*)\\s*:")
	fixed = unquotedKey.ReplaceAllString(fixed, "$1\"$2\":")

	// Fix single-quoted strings
	singleQuote := regexp.MustCompile("'([^']*)'")
	fixed = singleQuote.ReplaceAllString(fixed, "\"$1\"")

	// Check if it's valid JSON
	if !json.Valid([]byte(fixed)) {
		return "", errors.New("failed to fix JSON")
	}

	return fixed, nil
}

// ParseGeneric parses generic JSON output.
func (p *Parser) ParseGeneric(output string, target interface{}) error {
	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return ErrInvalidJSON
	}

	if err := json.Unmarshal([]byte(jsonStr), target); err != nil {
		if !p.fixJSON {
			return fmt.Errorf("%w: %w", ErrInvalidJSON, err)
		}

		fixed, fixErr := p.fixJSONString(jsonStr)
		if fixErr != nil {
			return fmt.Errorf("%w: %w (tried fix: %v)", ErrInvalidJSON, err, fixErr)
		}

		if err := json.Unmarshal([]byte(fixed), target); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidJSON, err)
		}
	}

	return nil
}

// ParseArray parses JSON array output.
func (p *Parser) ParseArray(output string) ([]interface{}, error) {
	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return nil, ErrInvalidJSON
	}

	// Check if it's an array
	jsonStr = strings.TrimSpace(jsonStr)
	if !strings.HasPrefix(jsonStr, "[") {
		return nil, ErrInvalidJSON
	}

	var result []interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidJSON, err)
	}

	return result, nil
}

// Parser errors.
var (
	ErrInvalidJSON = errors.New("invalid JSON")
)
