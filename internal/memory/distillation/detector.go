// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"strings"
)

// IsProblem detects if a user message represents a genuine problem or question.
// It checks for problem-related keywords and question marks to avoid extracting
// non-problematic conversations like "thanks", "ok", "got it".
//
// Args:
//
//	text - the user message to analyze.
//
// Returns:
//
//	true if the text appears to be a problem or question, false otherwise.
func IsProblem(text string) bool {
	if text == "" {
		return false
	}

	lower := strings.TrimSpace(strings.ToLower(text))

	// Negative keywords - these should NOT be treated as problems
	negativeKeywords := []string{
		"thanks",
		"thank you",
		"ok",
		"okay",
		"got it",
		"understood",
		"alright",
		"sure",
		"fine",
		"good",
		"great",
		"perfect",
		"awesome",
		"excellent",
		"yes",
		"no",
		"maybe",
		"right",
		"correct",
		"agree",
		"cool",
		"nice",
		"sounds good",
		"that works",
		"makes sense",
		"got it, thanks",
		"thanks for the",
		"you're welcome",
		"glad i could",
		"appreciate",
		"welcome",
		"show me",
		"tell me",
		"what's happening",
		"what is this",
	}

	// Check negative keywords - return false immediately if matched
	for _, keyword := range negativeKeywords {
		if lower == keyword || strings.HasPrefix(lower, keyword+" ") || strings.HasSuffix(lower, " "+keyword) {
			return false
		}
	}

	// Problem-related keywords (must be more specific than casual terms)
	problemKeywords := []string{
		"error",
		"issue",
		"problem",
		"fix",
		"help",
		"unable",
		"cannot",
		"can't",
		"fail",
		"failed",
		"broken",
		"wrong",
		"not working",
		"doesn't work",
		"won't work",
		"won't start",
		"won't",
		"bug",
		"crash",
		"exception",
		"panic",
		"stack trace",
		"leak",
		"timeout",
		"refused",
		"denied",
		"missing",
		"undefined",
		"null",
		"invalid",
	}

	for _, keyword := range problemKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	// Check for question mark (but filter out rhetorical questions)
	if strings.Contains(text, "?") {
		// Filter out questions that are just acknowledgments
		questionExclusions := []string{
			"can you?",
			"could you?",
			"would you?",
			"right?",
			"ok?",
			"sure?",
		}
		for _, exclusion := range questionExclusions {
			if strings.HasSuffix(lower, exclusion) {
				return false
			}
		}
		return true
	}

	return false
}

// QuestionDetector detects questions in conversations.
type QuestionDetector struct {
	// Configurable sensitivity (0.0 to 1.0)
	sensitivity float64
}

// NewQuestionDetector creates a new QuestionDetector with default sensitivity.
func NewQuestionDetector() *QuestionDetector {
	return &QuestionDetector{
		sensitivity: 0.7,
	}
}

// Detect checks if a message is a question.
//
// Args:
//
//	text - the message to analyze.
//
// Returns:
//
//	true if the message is likely a question.
func (d *QuestionDetector) Detect(text string) bool {
	return IsProblem(text)
}
