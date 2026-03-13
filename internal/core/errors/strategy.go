package errors

import "time"

// ErrorStrategy defines retry and handling strategy for error codes.
type ErrorStrategy struct {
	Backoff      time.Duration
	MaxRetries   int
	DLQEnabled   bool
	AlertEnabled bool
	AlertMessage string
}

// StrategyMap contains the error handling strategies.
var StrategyMap = map[string]ErrorStrategy{
	// Agent module strategies
	"01-002": {Backoff: 5 * time.Second, MaxRetries: 3, DLQEnabled: true, AlertEnabled: false},
	"01-003": {Backoff: 10 * time.Second, MaxRetries: 2, DLQEnabled: true, AlertEnabled: true, AlertMessage: "Agent panic detected"},
	"01-004": {Backoff: 1 * time.Second, MaxRetries: 5, DLQEnabled: false, AlertEnabled: true, AlertMessage: "Task queue full"},

	// Protocol module strategies
	"02-002": {Backoff: 3 * time.Second, MaxRetries: 3, DLQEnabled: false, AlertEnabled: false},
	"02-003": {Backoff: 5 * time.Second, MaxRetries: 5, DLQEnabled: true, AlertEnabled: true, AlertMessage: "Heartbeat missed"},

	// Storage module strategies
	"03-001": {Backoff: 2 * time.Second, MaxRetries: 3, DLQEnabled: false, AlertEnabled: true, AlertMessage: "DB connection failed"},
	"03-002": {Backoff: 1 * time.Second, MaxRetries: 2, DLQEnabled: false, AlertEnabled: false},
	"03-003": {Backoff: 2 * time.Second, MaxRetries: 2, DLQEnabled: false, AlertEnabled: false},

	// LLM module strategies
	"04-001": {Backoff: 3 * time.Second, MaxRetries: 3, DLQEnabled: false, AlertEnabled: true, AlertMessage: "LLM request failed"},
	"04-002": {Backoff: 5 * time.Second, MaxRetries: 2, DLQEnabled: false, AlertEnabled: false},
	"04-003": {Backoff: 0, MaxRetries: 0, DLQEnabled: false, AlertEnabled: true, AlertMessage: "LLM quota exceeded"},
	"04-005": {Backoff: 1 * time.Second, MaxRetries: 3, DLQEnabled: false, AlertEnabled: false},
	"04-006": {Backoff: 0, MaxRetries: 0, DLQEnabled: false, AlertEnabled: true, AlertMessage: "LLM validation failed"},
}

// GetStrategy returns the error strategy for the given error code.
func GetStrategy(code string) ErrorStrategy {
	if strategy, ok := StrategyMap[code]; ok {
		return strategy
	}
	// Default strategy
	return ErrorStrategy{
		Backoff:      1 * time.Second,
		MaxRetries:   1,
		DLQEnabled:   false,
		AlertEnabled: false,
	}
}

// ShouldDLQ checks if the error should be sent to DLQ.
func ShouldDLQ(code string) bool {
	return GetStrategy(code).DLQEnabled
}

// ShouldAlert checks if the error should trigger an alert.
func ShouldAlert(code string) bool {
	return GetStrategy(code).AlertEnabled
}

// GetAlertMessage returns the alert message for the error code.
func GetAlertMessage(code string) string {
	return GetStrategy(code).AlertMessage
}
