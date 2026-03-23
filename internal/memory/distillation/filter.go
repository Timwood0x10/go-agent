// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"strings"
)

const (
	minMessageLength    = 10
	conflictThreshold   = 0.85
	maxSolutionsPerUser = 5000
)

// NoiseFilter provides filtering capabilities to remove low-value and noisy messages.
type NoiseFilter struct {
	enableCodeFilter          bool
	enableStacktraceFilter    bool
	enableLogFilter           bool
	enableMarkdownTableFilter bool
}

// NewNoiseFilter creates a new NoiseFilter instance with default settings.
func NewNoiseFilter() *NoiseFilter {
	return &NoiseFilter{
		enableCodeFilter:          true,
		enableStacktraceFilter:    true,
		enableLogFilter:           true,
		enableMarkdownTableFilter: true,
	}
}

// NoiseFilterConfig holds configuration for noise filtering.
type NoiseFilterConfig struct {
	EnableCodeFilter          bool
	EnableStacktraceFilter    bool
	EnableLogFilter           bool
	EnableMarkdownTableFilter bool
}

// NewNoiseFilterWithConfig creates a new NoiseFilter instance with custom configuration.
func NewNoiseFilterWithConfig(config *NoiseFilterConfig) *NoiseFilter {
	return &NoiseFilter{
		enableCodeFilter:          config.EnableCodeFilter,
		enableStacktraceFilter:    config.EnableStacktraceFilter,
		enableLogFilter:           config.EnableLogFilter,
		enableMarkdownTableFilter: config.EnableMarkdownTableFilter,
	}
}

// IsNoise determines if a message is noise and should be filtered out.
//
// Args:
//
//	text - the message text to analyze.
//
// Returns:
//
//	true if the message is noise, false otherwise.
func (f *NoiseFilter) IsNoise(text string) bool {
	if text == "" {
		return true
	}

	// Check minimum length
	if len(text) < minMessageLength {
		return true
	}

	lower := strings.ToLower(text)

	// Filter out casual acknowledgments
	casualAcknowledgments := []string{
		"ok", "okay", "thanks", "thank you", "got it",
		"sure", "alright", "yes", "yeah", "yep",
		"no", "nope", "cool", "great", "awesome",
		"perfect", "fine", "good", "noted", "understood",
	}
	for _, ack := range casualAcknowledgments {
		if lower == ack || strings.HasPrefix(lower, ack+" ") || strings.HasSuffix(lower, " "+ack) {
			return true
		}
	}

	// Filter code blocks if enabled
	if f.enableCodeFilter && f.CodeBlockFilter(text) {
		return true
	}

	// Filter stacktrace if enabled
	if f.enableStacktraceFilter && f.StacktraceFilter(text) {
		return true
	}

	// Filter logs if enabled
	if f.enableLogFilter && f.LogFilter(text) {
		return true
	}

	// Filter markdown tables if enabled
	if f.enableMarkdownTableFilter && f.MarkdownTableFilter(text) {
		return true
	}

	return false
}

// CodeBlockFilter detects code blocks and code-related content.
//
// Args:
//
//	text - the text to analyze.
//
// Returns:
//
//	true if the text appears to contain code blocks, false otherwise.
func (f *NoiseFilter) CodeBlockFilter(text string) bool {
	lower := strings.ToLower(text)

	// Check for markdown code blocks
	if strings.Contains(text, "```") {
		return true
	}

	// Check for Go language keywords
	codeKeywords := []string{"func ", "package ", "import ", "struct ", "interface "}
	for _, keyword := range codeKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}

// StacktraceFilter detects stack traces and error messages.
//
// Args:
//
//	text - the text to analyze.
//
// Returns:
//
//	true if the text appears to be a stack trace, false otherwise.
func (f *NoiseFilter) StacktraceFilter(text string) bool {
	lower := strings.ToLower(text)

	// Check for common stack trace indicators
	stacktraceIndicators := []string{
		"exception", "traceback", "panic:", "fatal error",
		"runtime error", "segmentation fault", "stack trace",
	}
	for _, indicator := range stacktraceIndicators {
		if strings.Contains(lower, indicator) {
			return true
		}
	}

	// Check for file:line patterns (common in stack traces)
	if strings.Contains(text, ".go:") || strings.Contains(text, ".py:") {
		return true
	}

	return false
}

// LogFilter detects log messages.
//
// Args:
//
//	text - the text to analyze.
//
// Returns:
//
//	true if the text appears to be a log message, false otherwise.
func (f *NoiseFilter) LogFilter(text string) bool {
	lower := strings.ToLower(text)

	// Check for log prefixes
	logPrefixes := []string{"log.", "log::", "[info]", "[error]", "[warn]", "[debug]", "[trace]"}
	for _, prefix := range logPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}

	return false
}

// MarkdownTableFilter detects markdown tables.
//
// Args:
//
//	text - the text to analyze.
//
// Returns:
//
//	true if the text appears to be a markdown table, false otherwise.
func (f *NoiseFilter) MarkdownTableFilter(text string) bool {
	// Check for markdown table format: | ... | ... |
	if strings.Contains(text, "|") && strings.Contains(text, "---") {
		return true
	}
	return false
}

// SecurityFilter detects sensitive information that should not be stored in memory.
//
// Args:
//
//	text - the text to analyze.
//
// Returns:
//
//	false if the text contains sensitive information, true otherwise.
func SecurityFilter(text string) bool {
	if text == "" {
		return false
	}

	lower := strings.ToLower(text)

	// Sensitive keywords that should never be stored
	sensitiveKeywords := []string{
		"password", "api key", "apikey", "secret", "token",
		"credential", "private key", "auth token", "bearer token",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(lower, keyword) {
			return false
		}
	}

	return true
}
