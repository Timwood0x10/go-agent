package main

import (
	"testing"
)

func TestExtractUserID_Consistency(t *testing.T) {
	// Create a mock KnowledgeBase for testing
	kb := &KnowledgeBase{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "English self-introduction with job title",
			input:    "hello I'm Ken font-end programmer , like JS,TS,VUE,nice to met you",
			expected: "ken",
		},
		{
			name:     "Simple English self-introduction",
			input:    "I'm Ken",
			expected: "ken",
		},
		{
			name:     "English self-introduction with comma",
			input:    "I'm Ken, are you remember me?",
			expected: "ken",
		},
		{
			name:     "English self-introduction with 'a'",
			input:    "I'm Ken a frontend programmer",
			expected: "ken",
		},
		{
			name:     "Chinese self-introduction with job title",
			input:    "你好，我叫小刚，我是前端开发工程师技术栈是JS,TS,VUE，不喜欢后端开发",
			expected: "小刚",
		},
		{
			name:     "Simple Chinese self-introduction",
			input:    "我叫小刚",
			expected: "小刚",
		},
		{
			name:     "Chinese self-introduction with job",
			input:    "我是张三，是一名程序员",
			expected: "张三",
		},
		{
			name:     "No self-introduction",
			input:    "what's the go-agent",
			expected: "",
		},
		{
			name:     "My name is pattern",
			input:    "my name is John Doe",
			expected: "john",
		},
		{
			name:     "I am pattern",
			input:    "I am Alice",
			expected: "alice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := kb.extractUserID(tt.input)
			if result != tt.expected {
				t.Errorf("extractUserID() = %q, expected %q", result, tt.expected)
			}
			t.Logf("Input: %q -> Output: %q", tt.input, result)
		})
	}
}

func TestExtractUserID_SameUserDifferentContexts(t *testing.T) {
	// Test that the same user is extracted consistently across different contexts
	kb := &KnowledgeBase{}

	inputs := []string{
		"hello I'm Ken font-end programmer , like JS,TS,VUE,nice to met you",
		"I'm Ken, are you remember me?",
		"what's up, I'm Ken here",
	}

	expected := "ken"
	for _, input := range inputs {
		result := kb.extractUserID(input)
		if result != expected {
			t.Errorf("extractUserID(%q) = %q, expected %q (inconsistent extraction)", input, result, expected)
		}
		t.Logf("Input: %q -> Output: %q", input, result)
	}
}