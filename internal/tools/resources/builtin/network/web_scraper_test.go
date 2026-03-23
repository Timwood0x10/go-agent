package builtin

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// MockHTTPGetter is a mock implementation of HTTPGetter for testing.
type MockHTTPGetter struct {
	content []byte
	err     error
}

func (m *MockHTTPGetter) Get(ctx context.Context, url string) ([]byte, error) {
	return m.content, m.err
}

func (m *MockHTTPGetter) SetUserAgent(userAgent string) {
	// No-op for mock
}

func TestWebScraper_ExtractTitle(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "Simple title",
			html:     `<html><head><title>Test Page</title></head><body></body></html>`,
			expected: "Test Page",
		},
		{
			name:     "Title with spaces",
			html:     `<html><head><title>  My Test Page  </title></head></html>`,
			expected: "My Test Page",
		},
		{
			name:     "No title",
			html:     `<html><body><p>No title here</p></body></html>`,
			expected: "",
		},
		{
			name:     "Title with special characters",
			html:     `<html><head><title>Test & Page © 2024</title></head></html>`,
			expected: "Test & Page © 2024",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTitle(tt.html)
			if result != tt.expected {
				t.Errorf("Expected title '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestWebScraper_ExtractBody(t *testing.T) {
	tests := []struct {
		name      string
		html      string
		removeNav bool
		expected  string
	}{
		{
			name:      "Simple body",
			html:      `<html><body><p>Hello World</p></body></html>`,
			removeNav: false,
			expected:  "Hello World",
		},
		{
			name:      "Body with script",
			html:      `<html><body><p>Content</p><script>alert('test');</script></body></html>`,
			removeNav: false,
			expected:  "Content",
		},
		{
			name:      "Body with style",
			html:      `<html><body><style>body { color: red; }</style><p>Text</p></body></html>`,
			removeNav: false,
			expected:  "Text",
		},
		{
			name:      "Remove navigation",
			html:      `<html><body><nav>Menu</nav><p>Content</p></body></html>`,
			removeNav: true,
			expected:  "Content",
		},
		{
			name:      "Multiple paragraphs",
			html:      `<html><body><p>First</p><p>Second</p></body></html>`,
			removeNav: false,
			expected:  "First Second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBody(tt.html, tt.removeNav)
			if result != tt.expected {
				t.Errorf("Expected body '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestWebScraper_ExtractLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected int
	}{
		{
			name:     "Single link",
			html:     `<a href="https://example.com">Example</a>`,
			expected: 1,
		},
		{
			name:     "Multiple links",
			html:     `<a href="https://example1.com">Link 1</a><a href="https://example2.com">Link 2</a>`,
			expected: 2,
		},
		{
			name:     "No links",
			html:     `<p>No links here</p>`,
			expected: 0,
		},
		{
			name:     "Link with quotes",
			html:     `<a href="https://example.com/path?arg=value">Link</a>`,
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			links := extractLinks(tt.html)
			if len(links) != tt.expected {
				t.Errorf("Expected %d links, got %d", tt.expected, len(links))
			}
		})
	}
}

func TestWebScraper_Execute(t *testing.T) {
	tests := []struct {
		name           string
		html           string
		params         map[string]interface{}
		expectError    bool
		checkTitle     string
		checkContent   string
		checkLinkCount int
	}{
		{
			name: "Successful extraction",
			html: `<html><head><title>Test Page</title></head><body><p>Test content</p></body></html>`,
			params: map[string]interface{}{
				"url":           "https://example.com",
				"extract_title": true,
				"extract_body":  true,
			},
			expectError:    false,
			checkTitle:     "Test Page",
			checkContent:   "Test content",
			checkLinkCount: -1, // Don't check link count
		},
		{
			name: "With links",
			html: `<html><head><title>Links Page</title></head><body><a href="https://example.com">Link</a></body></html>`,
			params: map[string]interface{}{
				"url":           "https://example.com",
				"extract_title": true,
				"extract_links": true,
			},
			expectError:    false,
			checkTitle:     "Links Page",
			checkLinkCount: 1,
		},
		{
			name: "Missing URL",
			params: map[string]interface{}{
				"extract_title": true,
			},
			expectError: true,
		},
		{
			name: "Invalid URL",
			params: map[string]interface{}{
				"url": "not-a-url",
			},
			expectError: true,
		},
		{
			name: "HTTP error",
			html: "",
			params: map[string]interface{}{
				"url": "https://example.com",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockGetter := &MockHTTPGetter{
				content: []byte(tt.html),
				err:     nil,
			}

			if tt.name == "HTTP error" {
				mockGetter.err = errors.New("network error")
			}

			scraper := NewWebScraper(mockGetter)
			result, err := scraper.Execute(context.Background(), tt.params)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !result.Success {
				t.Error("Expected success, got failure")
				return
			}

			data, ok := result.Data.(map[string]interface{})
			if !ok {
				t.Error("Result data is not a map")
				return
			}

			if tt.checkTitle != "" {
				if title, ok := data["title"].(string); !ok || title != tt.checkTitle {
					t.Errorf("Expected title '%s', got '%s'", tt.checkTitle, title)
				}
			}

			if tt.checkContent != "" {
				if content, ok := data["content"].(string); !ok || content != tt.checkContent {
					t.Errorf("Expected content '%s', got '%s'", tt.checkContent, content)
				}
			}

			if tt.checkLinkCount >= 0 {
				if links, ok := data["link_count"].(int); !ok || links != tt.checkLinkCount {
					t.Errorf("Expected %d links, got %d", tt.checkLinkCount, links)
				}
			}
		})
	}
}

func TestWebScraper_IsValidURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com", true},
		{"http://example.com", true},
		{"https://example.com/path", true},
		{"ftp://example.com", false},
		{"example.com", false},
		{"", false},
		{"not-a-url", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := isValidURL(tt.url)
			if result != tt.expected {
				t.Errorf("Expected %v for URL '%s', got %v", tt.expected, tt.url, result)
			}
		})
	}
}

func TestWebScraper_Capabilities(t *testing.T) {
	scraper := NewWebScraper(&MockHTTPGetter{})

	caps := scraper.Capabilities()
	if len(caps) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(caps))
	}

	hasNetwork := false
	hasText := false
	for _, cap := range caps {
		if cap == "network" {
			hasNetwork = true
		}
		if cap == "text" {
			hasText = true
		}
	}

	if !hasNetwork {
		t.Error("Missing network capability")
	}
	if !hasText {
		t.Error("Missing text capability")
	}
}

func TestWebScraper_MaxLength(t *testing.T) {
	longHTML := `<html><head><title>Long Content</title></head><body><p>` + strings.Repeat("a", 20000) + `</p></body></html>`

	mockGetter := &MockHTTPGetter{
		content: []byte(longHTML),
	}

	scraper := NewWebScraper(mockGetter)
	result, err := scraper.Execute(context.Background(), map[string]interface{}{
		"url":        "https://example.com",
		"max_length": 1000,
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Result data is not a map")
	}

	content, ok := data["content"].(string)
	if !ok {
		t.Fatal("Content is not a string")
	}

	if len(content) > 1100 { // 1000 + "..."
		t.Errorf("Content too long: %d characters", len(content))
	}

	if !strings.HasSuffix(content, "...") {
		t.Error("Truncated content should end with '...'")
	}
}
