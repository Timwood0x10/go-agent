// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"testing"
)

func TestMemoryClassifier_ClassifyMemory(t *testing.T) {
	classifier := NewMemoryClassifier()

	tests := []struct {
		name         string
		problem      string
		solution     string
		expectedType MemoryType
	}{
		{
			name:         "solution type",
			problem:      "I have an error in my code",
			solution:     "Fix the syntax error on line 10",
			expectedType: MemorySolution,
		},
		{
			name:         "preference type",
			problem:      "I prefer Go over Python",
			solution:     "Use Go for performance",
			expectedType: MemoryPreference,
		},
		{
			name:         "fact type",
			problem:      "What platform am I using?",
			solution:     "You are using macOS",
			expectedType: MemoryFact,
		},
		{
			name:         "rule type",
			problem:      "What are the coding standards?",
			solution:     "Follow the Google Go style guide",
			expectedType: MemoryRule,
		},
		{
			name:         "default to fact",
			problem:      "Tell me something",
			solution:     "This is a generic response",
			expectedType: MemoryFact,
		},
		{
			name:         "debug keyword",
			problem:      "Help me debug this",
			solution:     "Check the logs for errors",
			expectedType: MemorySolution,
		},
		{
			name:         "troubleshoot keyword",
			problem:      "I need to troubleshoot",
			solution:     "Restart the service",
			expectedType: MemorySolution,
		},
		{
			name:         "like keyword",
			problem:      "I like verbose responses",
			solution:     "Will provide detailed answers",
			expectedType: MemoryPreference,
		},
		{
			name:         "usually keyword",
			problem:      "I usually prefer Go",
			solution:     "Noted",
			expectedType: MemoryPreference,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := &Experience{
				Problem:  tt.problem,
				Solution: tt.solution,
			}

			result := classifier.ClassifyMemory(exp)
			if result != tt.expectedType {
				t.Errorf("ClassifyMemory() = %v, want %v", result, tt.expectedType)
			}
		})
	}
}

func TestGetMemoryTypeFromString(t *testing.T) {
	tests := []struct {
		name     string
		typeStr  string
		expected MemoryType
	}{
		{
			name:     "fact",
			typeStr:  "fact",
			expected: MemoryFact,
		},
		{
			name:     "preference",
			typeStr:  "preference",
			expected: MemoryPreference,
		},
		{
			name:     "solution",
			typeStr:  "solution",
			expected: MemorySolution,
		},
		{
			name:     "rule",
			typeStr:  "rule",
			expected: MemoryRule,
		},
		{
			name:     "case insensitive",
			typeStr:  "FACT",
			expected: MemoryFact,
		},
		{
			name:     "invalid defaults to fact",
			typeStr:  "invalid",
			expected: MemoryFact,
		},
		{
			name:     "empty defaults to fact",
			typeStr:  "",
			expected: MemoryFact,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMemoryTypeFromString(tt.typeStr)
			if result != tt.expected {
				t.Errorf("GetMemoryTypeFromString(%q) = %v, want %v", tt.typeStr, result, tt.expected)
			}
		})
	}
}

func TestMemoryType_String(t *testing.T) {
	tests := []struct {
		name     string
		memType  MemoryType
		expected string
	}{
		{
			name:     "fact",
			memType:  MemoryFact,
			expected: "fact",
		},
		{
			name:     "preference",
			memType:  MemoryPreference,
			expected: "preference",
		},
		{
			name:     "solution",
			memType:  MemorySolution,
			expected: "solution",
		},
		{
			name:     "rule",
			memType:  MemoryRule,
			expected: "rule",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.memType.String()
			if result != tt.expected {
				t.Errorf("MemoryType.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}
