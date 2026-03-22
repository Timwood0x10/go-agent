package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// HTTPRequest performs HTTP requests to external APIs.
type HTTPRequest struct {
	*BaseTool
	client *http.Client
}

// NewHTTPRequest creates a new HTTPRequest tool.
func NewHTTPRequest() *HTTPRequest {
	params := &ParameterSchema{
		Type: "object",
		Properties: map[string]*Parameter{
			"url": {
				Type:        "string",
				Description: "Target URL",
			},
			"method": {
				Type:        "string",
				Description: "HTTP method (GET, POST, PUT, DELETE, PATCH)",
				Default:     "GET",
				Enum:        []interface{}{"GET", "POST", "PUT", "DELETE", "PATCH"},
			},
			"headers": {
				Type:        "object",
				Description: "Request headers as key-value pairs",
			},
			"body": {
				Type:        "string",
				Description: "Request body (for POST, PUT, PATCH)",
			},
			"timeout": {
				Type:        "integer",
				Description: "Request timeout in seconds",
				Default:     30,
			},
		},
		Required: []string{"url"},
	}

	hr := &HTTPRequest{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	hr.BaseTool = NewBaseTool("http_request", "Perform HTTP requests to external APIs", params)

	return hr
}

// Execute performs the HTTP request.
func (t *HTTPRequest) Execute(ctx context.Context, params map[string]interface{}) (Result, error) {
	url, ok := params["url"].(string)
	if !ok || url == "" {
		return NewErrorResult("url is required"), nil
	}

	method := getString(params, "method")
	if method == "" {
		method = "GET"
	}

	// Parse headers
	headers := make(map[string]string)
	if headersParam, ok := params["headers"].(map[string]interface{}); ok {
		for k, v := range headersParam {
			if val, ok := v.(string); ok {
				headers[k] = val
			}
		}
	}

	// Set timeout
	timeout := getInt(params, "timeout", 30)
	if timeout > 0 {
		t.client.Timeout = time.Duration(timeout) * time.Second
	}

	// Prepare request body
	var bodyReader io.Reader
	if body, ok := params["body"].(string); ok && body != "" {
		// Check if body is JSON
		if strings.Contains(body, "{") || strings.Contains(body, "[") {
			// Validate JSON
			var js interface{}
			if err := json.Unmarshal([]byte(body), &js); err != nil {
				return NewErrorResult(fmt.Sprintf("invalid JSON body: %v", err)), nil
			}
		}
		bodyReader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// Set default Content-Type for POST/PUT/PATCH
	if method != "GET" && method != "DELETE" && bodyReader != nil {
		if headers["Content-Type"] == "" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// Execute request
	startTime := time.Now()
	resp, err := t.client.Do(req)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("request failed: %v", err)), nil
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body: ", "error", err)
		}
	}()

	duration := time.Since(startTime)

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// Try to parse as JSON
	var jsonBody interface{}
	if err := json.Unmarshal(respBody, &jsonBody); err != nil {
		// If not JSON, return as string
		jsonBody = string(respBody)
	}

	// Collect response headers
	respHeaders := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			respHeaders[k] = v[0]
		}
	}

	return NewResult(true, map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     respHeaders,
		"body":        jsonBody,
		"size_bytes":  len(respBody),
		"duration_ms": duration.Milliseconds(),
	}), nil
}

// SetClient sets a custom HTTP client.
func (t *HTTPRequest) SetClient(client *http.Client) {
	t.client = client
}
