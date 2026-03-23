// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"strings"
)

// MemoryClassifier classifies experiences into memory types.
type MemoryClassifier struct{}

// NewMemoryClassifier creates a new MemoryClassifier instance.
func NewMemoryClassifier() *MemoryClassifier {
	return &MemoryClassifier{}
}

// ClassifyMemory determines the memory type based on the experience content.
// It uses keyword-based classification to categorize memories into one of four types.
//
// Args:
//   experience - the experience to classify.
//
// Returns:
//   MemoryType - the classified memory type.
func (c *MemoryClassifier) ClassifyMemory(experience *Experience) MemoryType {
	if experience == nil {
		return MemoryFact // Default fallback
	}

	content := strings.ToLower(experience.Problem + " " + experience.Solution)

	// Check for solution type first (highest priority)
	if c.isSolution(content) {
		return MemorySolution
	}

	// Check for preference type
	if c.isPreference(content) {
		return MemoryPreference
	}

	// Check for rule type
	if c.isRule(content) {
		return MemoryRule
	}

	// Default to fact type
	return MemoryFact
}

// isSolution determines if the content represents a solution.
func (c *MemoryClassifier) isSolution(content string) bool {
	solutionKeywords := []string{
		"fix", "solution", "error", "issue", "problem", "resolve",
		"debug", "troubleshoot", "resolve", "workaround", "patch",
		"error:", "exception", "fail", "failure", "bug",
		"restart", "update", "change", "modify", "adjust",
		"correct", "repair", "heal", "recover",
	}

	for _, keyword := range solutionKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// isPreference determines if the content represents a user preference.
func (c *MemoryClassifier) isPreference(content string) bool {
	preferenceKeywords := []string{
		"prefer", "like", "usually", "want", "would like",
		"favorite", "choose", "opt for", "rather", "instead",
		"enjoy", "love", "dislike", "hate", "avoid",
	}

	for _, keyword := range preferenceKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// isRule determines if the content represents a system rule or constraint.
func (c *MemoryClassifier) isRule(content string) bool {
	ruleKeywords := []string{
		"configuration", "config", "setting", "policy", "rule",
		"constraint", "requirement", "must", "should", "system",
		"framework", "architecture", "pattern", "convention",
		"standard", "style guide", "guideline", "specification",
		"protocol", "format", "structure", "schema",
		"best practice", "convention", "coding standard",
	}

	for _, keyword := range ruleKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}

	return false
}

// String returns the string representation of MemoryType.
func (mt MemoryType) String() string {
	return string(mt)
}

// GetMemoryTypeFromString converts a string to MemoryType.
// Returns MemoryFact as default for invalid input.
func GetMemoryTypeFromString(s string) MemoryType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "fact":
		return MemoryFact
	case "preference":
		return MemoryPreference
	case "solution":
		return MemorySolution
	case "rule":
		return MemoryRule
	default:
		return MemoryFact
	}
}