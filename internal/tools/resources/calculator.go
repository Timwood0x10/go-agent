package resources

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// Calculator performs mathematical calculations.
type Calculator struct {
	*BaseTool
}

// NewCalculator creates a new Calculator tool.
func NewCalculator() *Calculator {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"operation": {
				Type:        "string",
				Description: "Operation to perform (add, subtract, multiply, divide, power, sqrt, abs, max, min)",
				Enum:        []interface{}{"add", "subtract", "multiply", "divide", "power", "sqrt", "abs", "max", "min"},
			},
			"operands": {
				Type:        "array",
				Description: "List of operands (numbers)",
			},
		},
		Required: []string{"operation", "operands"},
	}

	return &Calculator{
		BaseTool: NewBaseToolWithCategory("calculator", "Perform mathematical calculations", CategoryCore, params),
	}
}

// Execute performs the calculation.
func (t *Calculator) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return NewErrorResult("operation is required"), nil
	}

	operandsParam, ok := params["operands"].([]interface{})
	if !ok || len(operandsParam) == 0 {
		return NewErrorResult("operands is required and must be a non-empty array"), nil
	}

	// Parse operands
	operands := make([]float64, len(operandsParam))
	for i, op := range operandsParam {
		switch v := op.(type) {
		case float64:
			operands[i] = v
		case int:
			operands[i] = float64(v)
		case string:
			num, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return NewErrorResult(fmt.Sprintf("invalid operand at index %d: %v", i, err)), nil
			}
			operands[i] = num
		default:
			return NewErrorResult(fmt.Sprintf("invalid operand type at index %d", i)), nil
		}
	}

	var result float64
	var err error

	switch operation {
	case "add":
		result = 0
		for _, op := range operands {
			result += op
		}
	case "subtract":
		if len(operands) < 2 {
			return NewErrorResult("subtract requires at least 2 operands"), nil
		}
		result = operands[0]
		for i := 1; i < len(operands); i++ {
			result -= operands[i]
		}
	case "multiply":
		result = 1
		for _, op := range operands {
			result *= op
		}
	case "divide":
		if len(operands) < 2 {
			return NewErrorResult("divide requires at least 2 operands"), nil
		}
		result = operands[0]
		for i := 1; i < len(operands); i++ {
			if operands[i] == 0 {
				return NewErrorResult("division by zero"), nil
			}
			result /= operands[i]
		}
	case "power":
		if len(operands) != 2 {
			return NewErrorResult("power requires exactly 2 operands (base, exponent)"), nil
		}
		result = math.Pow(operands[0], operands[1])
	case "sqrt":
		if len(operands) != 1 {
			return NewErrorResult("sqrt requires exactly 1 operand"), nil
		}
		if operands[0] < 0 {
			return NewErrorResult("sqrt requires non-negative operand"), nil
		}
		result = math.Sqrt(operands[0])
	case "abs":
		if len(operands) != 1 {
			return NewErrorResult("abs requires exactly 1 operand"), nil
		}
		result = math.Abs(operands[0])
	case "max":
		result = operands[0]
		for i := 1; i < len(operands); i++ {
			if operands[i] > result {
				result = operands[i]
			}
		}
	case "min":
		result = operands[0]
		for i := 1; i < len(operands); i++ {
			if operands[i] < result {
				result = operands[i]
			}
		}
	default:
		return NewErrorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}

	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	return NewResult(true, map[string]interface{}{
		"operation": operation,
		"operands":  operands,
		"result":    result,
	}), nil
}

// DateTime provides date and time operations.
type DateTime struct {
	*BaseTool
}

// NewDateTime creates a new DateTime tool.
func NewDateTime() *DateTime {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"operation": {
				Type:        "string",
				Description: "Operation to perform (now, format, parse, add, diff)",
				Enum:        []interface{}{"now", "format", "parse", "add", "diff"},
			},
			"time_string": {
				Type:        "string",
				Description: "Time string for parse/format operations",
			},
			"format": {
				Type:        "string",
				Description: "Format string (e.g., '2006-01-02 15:04:05')",
			},
			"duration": {
				Type:        "string",
				Description: "Duration to add (e.g., '1h', '30m', '2d')",
			},
		},
		Required: []string{"operation"},
	}

	return &DateTime{
		BaseTool: NewBaseToolWithCategory("datetime", "Get current time and perform date/time operations", CategoryCore, params),
	}
}

// Execute performs the date/time operation.
func (t *DateTime) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return NewErrorResult("operation is ·required"), nil
	}

	now := time.Now()

	switch operation {
	case "now":
		format := getString(params, "format")
		if format == "" {
			format = "2006-01-02 15:04:05"
		}
		return NewResult(true, map[string]interface{}{
			"formatted": now.Format(format),
			"unix":      now.Unix(),
			"unix_nano": now.UnixNano(),
		}), nil

	case "format":
		timeStr := getString(params, "time_string")
		if timeStr == "" {
			return NewErrorResult("time_string is required for format operation"), nil
		}
		format := getString(params, "format")
		if format == "" {
			format = "2006-01-02 15:04:05"
		}
		parsedTime, err := time.Parse(format, timeStr)
		if err != nil {
			return NewErrorResult(fmt.Sprintf("failed to parse time: %v", err)), nil
		}
		return NewResult(true, map[string]interface{}{
			"parsed":    parsedTime,
			"unix":      parsedTime.Unix(),
			"formatted": parsedTime.Format(format),
		}), nil

	case "parse":
		timeStr := getString(params, "time_string")
		if timeStr == "" {
			return NewErrorResult("time_string is required for parse operation"), nil
		}
		// Try common formats
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
			"2006/01/02 15:04:05",
			"2006/01/02",
		}
		var parsedTime time.Time
		var err error
		for _, format := range formats {
			parsedTime, err = time.Parse(format, timeStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			return NewErrorResult(fmt.Sprintf("failed to parse time with common formats: %v", err)), nil
		}
		return NewResult(true, map[string]interface{}{
			"parsed": parsedTime,
			"unix":   parsedTime.Unix(),
		}), nil

	case "add":
		durationStr := getString(params, "duration")
		if durationStr == "" {
			return NewErrorResult("duration is required for add operation"), nil
		}
		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return NewErrorResult(fmt.Sprintf("failed to parse duration: %v", err)), nil
		}
		result := now.Add(duration)
		return NewResult(true, map[string]interface{}{
			"original": now,
			"duration": durationStr,
			"result":   result,
			"unix":     result.Unix(),
		}), nil

	case "diff":
		timeStr := getString(params, "time_string")
		if timeStr == "" {
			return NewErrorResult("time_string is required for diff operation"), nil
		}
		// Parse time
		var parsedTime time.Time
		var err error
		formats := []string{
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, format := range formats {
			parsedTime, err = time.Parse(format, timeStr)
			if err == nil {
				break
			}
		}
		if err != nil {
			return NewErrorResult(fmt.Sprintf("failed to parse time: %v", err)), nil
		}
		diff := now.Sub(parsedTime)
		return NewResult(true, map[string]interface{}{
			"now":      now,
			"target":   parsedTime,
			"duration": diff,
			"seconds":  diff.Seconds(),
			"minutes":  diff.Minutes(),
			"hours":    diff.Hours(),
			"days":     diff.Hours() / 24,
		}), nil

	default:
		return NewErrorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}

// TextProcessor provides text processing operations.
type TextProcessor struct {
	*BaseTool
}

// NewTextProcessor creates a new TextProcessor tool.
func NewTextProcessor() *TextProcessor {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"operation": {
				Type:        "string",
				Description: "Operation to perform (count, split, replace, uppercase, lowercase, trim, contains)",
				Enum:        []interface{}{"count", "split", "replace", "uppercase", "lowercase", "trim", "contains"},
			},
			"text": {
				Type:        "string",
				Description: "Text to process",
			},
			"separator": {
				Type:        "string",
				Description: "Separator for split operation",
			},
			"old": {
				Type:        "string",
				Description: "Old substring to replace",
			},
			"new": {
				Type:        "string",
				Description: "New substring",
			},
			"substring": {
				Type:        "string",
				Description: "Substring to check for contains operation",
			},
		},
		Required: []string{"operation", "text"},
	}

	return &TextProcessor{
		BaseTool: NewBaseToolWithCategory("text_processor", "Perform text processing operations", CategoryCore, params),
	}
}

// Execute performs the text processing operation.
func (t *TextProcessor) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	operation, ok := params["operation"].(string)
	if !ok || operation == "" {
		return NewErrorResult("operation is required"), nil
	}

	text, ok := params["text"].(string)
	if !ok {
		return NewErrorResult("text is required"), nil
	}

	switch operation {
	case "count":
		return NewResult(true, map[string]interface{}{
			"length": len(text),
			"chars":  len([]rune(text)),
			"words":  len(strings.Fields(text)),
			"lines":  len(strings.Split(text, "\n")),
		}), nil

	case "split":
		separator := getString(params, "separator")
		if separator == "" {
			separator = " "
		}
		parts := strings.Split(text, separator)
		return NewResult(true, map[string]interface{}{
			"parts": parts,
			"count": len(parts),
		}), nil

	case "replace":
		oldStr := getString(params, "old")
		newStr := getString(params, "new")
		if oldStr == "" {
			return NewErrorResult("old substring is required for replace operation"), nil
		}
		result := strings.ReplaceAll(text, oldStr, newStr)
		return NewResult(true, map[string]interface{}{
			"original": text,
			"result":   result,
		}), nil

	case "uppercase":
		return NewResult(true, map[string]interface{}{
			"original": text,
			"result":   strings.ToUpper(text),
		}), nil

	case "lowercase":
		return NewResult(true, map[string]interface{}{
			"original": text,
			"result":   strings.ToLower(text),
		}), nil

	case "trim":
		return NewResult(true, map[string]interface{}{
			"original": text,
			"result":   strings.TrimSpace(text),
		}), nil

	case "contains":
		substring := getString(params, "substring")
		if substring == "" {
			return NewErrorResult("substring is required for contains operation"), nil
		}
		contains := strings.Contains(text, substring)
		return NewResult(true, map[string]interface{}{
			"contains": contains,
		}), nil

	default:
		return NewErrorResult(fmt.Sprintf("unsupported operation: %s", operation)), nil
	}
}
