// nolint: errcheck // Test code may ignore return values
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

func TestSanitizeMultipleSensitiveFields(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "api_key=sk-1234567890abcdef&password=secret123&token=abc123xyz"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "1234567890") {
		t.Error("Expected API key to be masked")
	}
	if strings.Contains(result, "secret123") {
		t.Error("Expected password to be masked")
	}
	if strings.Contains(result, "abc123xyz") {
		t.Error("Expected token to be masked")
	}
}

func TestSanitizeEmail(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "email=user@example.com"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "user@example.com") {
		t.Error("Expected email to be masked")
	}
}

func TestSanitizePhone(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "phone=+1-555-123-4567"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "555-123-4567") {
		t.Error("Expected phone number to be masked")
	}
}

func TestSanitizeSSN(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "ssn=123-45-6789"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "123-45-6789") {
		t.Error("Expected SSN to be masked")
	}
}

func TestSanitizeCreditCard(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "card=4111111111111111"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "4111111111111111") {
		t.Error("Expected credit card number to be masked")
	}
}

func TestSanitizeWithKeepLength(t *testing.T) {
	options := SanitizeOptions{
		KeepLength: true,
		MaskChar:   '*',
	}
	sanitizer := NewSanitizerWithOptions(options)

	input := "api_key=sk-1234567890abcdef"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "1234567890") {
		t.Error("Expected API key to be masked")
	}

	// Check that length is preserved
	inputLength := len(input)
	resultLength := len(result)
	if resultLength != inputLength {
		t.Errorf("Expected length to be preserved, got %d vs %d", resultLength, inputLength)
	}
}

func TestSanitizeEmptyInput(t *testing.T) {
	sanitizer := NewSanitizer()

	result := sanitizer.Sanitize("")
	if result != "" {
		t.Error("Expected empty string to remain empty")
	}
}

func TestSanitizeNoSensitiveData(t *testing.T) {
	sanitizer := NewSanitizer()

	input := "username=johndoe&age=30"
	result := sanitizer.Sanitize(input)

	if result != input {
		t.Errorf("Expected unchanged output for non-sensitive data, got %s", result)
	}
}

func TestSanitizeWithOptions(t *testing.T) {
	options := SanitizeOptions{
		KeepLength: true,
		MaskChar:   '#',
		PreserveLengthFor: map[SensitiveFieldType]int{
			SensitiveFieldTypeAPIKey: 4,
		},
	}
	sanitizer := NewSanitizerWithOptions(options)

	input := "api_key=sk-1234567890abcdef"
	result := sanitizer.Sanitize(input)

	if strings.Contains(result, "1234567890") {
		t.Error("Expected API key to be masked")
	}
	if !strings.Contains(result, "sk-") {
		t.Error("Expected prefix to be preserved")
	}
}

// nolint: errcheck // Test code may ignore return values
