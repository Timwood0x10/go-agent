package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"goagent/internal/core/models"
)

// Pre-compiled regular expressions for better performance.
var (
	markdownPattern   = regexp.MustCompile("```(?:json)?\\s*([\\s\\S]*?)\\s*```")
	trailingComma     = regexp.MustCompile(",\\s*([\\}\\]])")
	singleLineComment = regexp.MustCompile("//.*$")
	multiLineComment  = regexp.MustCompile("/\\*[\\s\\S]*?\\*/")
	unquotedKey       = regexp.MustCompile("([{,])\\s*([a-zA-Z_][a-zA-Z0-9_]*)\\s*:")
	singleQuote       = regexp.MustCompile("'([^']*)'")
)

// Parser parses LLM output into structured types.
type Parser struct {
	fixJSON        bool
	inputValidator *InputValidator
}

// NewParser creates a new Parser.
func NewParser() *Parser {
	return &Parser{
		fixJSON:        true,
		inputValidator: NewInputValidator(),
	}
}

// NewParserWithValidator creates a new Parser with custom input validation.
func NewParserWithValidator(validator *InputValidator) *Parser {
	return &Parser{
		fixJSON:        true,
		inputValidator: validator,
	}
}

// ParseRecommendResult parses LLM output into RecommendResult.
func (p *Parser) ParseRecommendResult(output string) (*models.RecommendResult, error) {
	// Validate input length before processing
	if err := p.inputValidator.ValidateInput(output); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return nil, ErrInvalidJSON
	}

	// Validate JSON content length
	if err := p.inputValidator.ValidateJSONLength(jsonStr); err != nil {
		return nil, fmt.Errorf("JSON validation failed: %w", err)
	}

	// Try to detect if it's an array or object
	jsonStr = strings.TrimSpace(jsonStr)

	// If it's an array, wrap it in an object
	if strings.HasPrefix(jsonStr, "[") {
		return p.parseArrayFormat(jsonStr)
	}

	// Try parsing as object first
	var result models.RecommendResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err == nil {
		return &result, nil
	}

	// If object parsing fails, try to fix JSON
	if p.fixJSON {
		fixed, fixErr := p.fixJSONString(jsonStr)
		if fixErr == nil {
			if err := json.Unmarshal([]byte(fixed), &result); err == nil {
				return &result, nil
			}
		}
	}

	// Try array format as fallback
	return p.parseArrayFormat(jsonStr)
}

// parseArrayFormat handles JSON array format.
func (p *Parser) parseArrayFormat(jsonStr string) (*models.RecommendResult, error) {
	// Ensure it's a valid array
	if !strings.HasPrefix(jsonStr, "[") {
		jsonStr = "[" + jsonStr + "]"
	}

	var items []*models.RecommendItem
	if err := json.Unmarshal([]byte(jsonStr), &items); err != nil {
		// Try to fix JSON
		if p.fixJSON {
			fixed, fixErr := p.fixJSONString(jsonStr)
			if fixErr == nil {
				if err := json.Unmarshal([]byte(fixed), &items); err == nil {
					return &models.RecommendResult{
						Items: items,
					}, nil
				}
			}
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidJSON, err)
	}

	return &models.RecommendResult{
		Items: items,
	}, nil
}

// extractJSON extracts JSON from output.
func (p *Parser) extractJSON(output string) string {
	output = strings.TrimSpace(output)

	// Try to find JSON in markdown code blocks
	matches := markdownPattern.FindStringSubmatch(output)
	if len(matches) > 1 {
		result := strings.TrimSpace(matches[1])
		// Check if it's a valid JSON (object or array)
		if strings.HasPrefix(result, "{") || strings.HasPrefix(result, "[") {
			return result
		}
	}

	// Try to find JSON object directly
	start := strings.Index(output, "{")
	end := -1

	// If no object found, try array
	if start == -1 {
		start = strings.Index(output, "[")
		if start == -1 {
			return ""
		}
		// Find matching closing bracket
		depth := 0
		for i := start; i < len(output); i++ {
			if output[i] == '[' {
				depth++
			} else if output[i] == ']' {
				depth--
				if depth == 0 {
					end = i + 1
					break
				}
			}
		}
	} else {
		// Find matching closing brace
		depth := 0
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
	fixed = trailingComma.ReplaceAllString(fixed, "$1")

	// Remove comments
	fixed = singleLineComment.ReplaceAllString(fixed, "")
	fixed = multiLineComment.ReplaceAllString(fixed, "")

	// Fix unquoted keys
	fixed = unquotedKey.ReplaceAllString(fixed, "$1\"$2\":")

	// Fix single-quoted strings
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

// ParseJSON parses LLM output into a generic map.
func (p *Parser) ParseJSON(output string) (map[string]interface{}, error) {
	// Validate input length before processing
	if err := p.inputValidator.ValidateInput(output); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return nil, ErrInvalidJSON
	}

	// Validate JSON content length
	if err := p.inputValidator.ValidateJSONLength(jsonStr); err != nil {
		return nil, fmt.Errorf("JSON validation failed: %w", err)
	}

	var result map[string]interface{}
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

	return result, nil
}

// ParseArray parses JSON array output.
func (p *Parser) ParseArray(output string) ([]interface{}, error) {
	// Validate input length before processing
	if err := p.inputValidator.ValidateInput(output); err != nil {
		return nil, fmt.Errorf("input validation failed: %w", err)
	}

	jsonStr := p.extractJSON(output)
	if jsonStr == "" {
		return nil, ErrInvalidJSON
	}

	// Validate JSON content length
	if err := p.inputValidator.ValidateJSONLength(jsonStr); err != nil {
		return nil, fmt.Errorf("JSON validation failed: %w", err)
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

	// Validate array length
	if len(result) > p.inputValidator.GetMaxArrayLength() {
		return nil, fmt.Errorf("%w: %d elements (max %d)", ErrArrayTooLarge, len(result), p.inputValidator.GetMaxArrayLength())
	}

	return result, nil
}

// Parser errors.
var (
	ErrInvalidJSON = errors.New("invalid JSON")
)
