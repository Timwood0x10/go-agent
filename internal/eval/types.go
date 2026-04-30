package eval

import (
	"encoding/json"
	"time"
)

// TestCase represents a single evaluation test case.
type TestCase struct {
	// ID is the unique test case identifier.
	ID string `json:"id" yaml:"id"`
	// Name is a human-readable test case name.
	Name string `json:"name" yaml:"name"`
	// Input is the input text for the agent.
	Input string `json:"input" yaml:"input"`
	// ExpectedOutput is the expected output (for exact match evaluation).
	ExpectedOutput string `json:"expected_output,omitempty" yaml:"expected_output,omitempty"`
	// ExpectedTools is the list of tools expected to be used.
	ExpectedTools []string `json:"expected_tools,omitempty" yaml:"expected_tools,omitempty"`
	// Timeout is the maximum duration for this test case.
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	// Metadata contains additional test case metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	// Tags for selective test execution.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// TestResult represents the result of executing a test case.
type TestResult struct {
	// TestCaseID is the ID of the executed test case.
	TestCaseID string `json:"test_case_id"`
	// ActualOutput is the actual output from the agent.
	ActualOutput string `json:"actual_output"`
	// ToolsUsed is the list of tools that were actually used.
	ToolsUsed []string `json:"tools_used"`
	// Duration is the execution duration.
	Duration time.Duration `json:"duration"`
	// TokensUsed is the total number of tokens consumed.
	TokensUsed int `json:"tokens_used"`
	// Error contains any error message.
	Error string `json:"error,omitempty"`
	// Metrics contains computed evaluation metrics.
	Metrics map[string]float64 `json:"metrics"`
	// Timestamp is when the test was executed.
	Timestamp time.Time `json:"timestamp"`
}

// EvalScore represents a single evaluation metric score.
type EvalScore struct {
	// Metric is the name of the metric.
	Metric string `json:"metric"`
	// Score is the metric value (typically 0.0 to 1.0).
	Score float64 `json:"score"`
	// Details contains additional information about the score.
	Details string `json:"details,omitempty"`
}

// TestSuite represents a collection of test cases.
type TestSuite struct {
	// Name is the test suite name.
	Name string `json:"name" yaml:"name"`
	// Description is the test suite description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// TestCases is the list of test cases in this suite.
	TestCases []TestCase `json:"test_cases" yaml:"test_cases"`
	// Tags for selective suite execution.
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// ToJSON serializes a TestCase to JSON.
func (tc *TestCase) ToJSON() (string, error) {
	data, err := json.MarshalIndent(tc, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSON serializes a TestResult to JSON.
func (tr *TestResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(tr, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ToJSON serializes a TestSuite to JSON.
func (ts *TestSuite) ToJSON() (string, error) {
	data, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
