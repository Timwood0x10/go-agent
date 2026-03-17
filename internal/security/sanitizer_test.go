package security

import (
	"strings"
	"testing"
)

func TestSanitize(t *testing.T) {
	sanitizer := NewSanitizer()
	
	result := sanitizer.Sanitize("api_key=sk-1234567890abcdef")
	if result == "api_key=sk-1234567890abcdef" {
		t.Error("Expected API key to be sanitized")
	}
	
	if strings.Contains(result, "1234567890") {
		t.Error("Expected partial key to be masked")
	}
}

func TestSanitizeLog(t *testing.T) {
	result := SanitizeLog("password: secret123")
	if strings.Contains(result, "secret123") {
		t.Error("Expected password to be masked")
	}
}

func TestSafeLogger(t *testing.T) {
	var loggedMessages []string
	logger := NewSafeLogger(func(msg string) {
		loggedMessages = append(loggedMessages, msg)
	})
	
	logger.Log("User logged in with password secret123")
	if strings.Contains(loggedMessages[0], "secret123") {
		t.Error("Expected password to be masked")
	}
}