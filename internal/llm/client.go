// Package llm provides LLM client functionality for various providers.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"goagent/internal/core/errors"
)

// ProviderType represents the LLM provider type.
type ProviderType string

const (
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderOllama     ProviderType = "ollama"
)

// Config holds LLM client configuration.
type Config struct {
	Provider string            `yaml:"provider"`
	APIKey   string            `yaml:"api_key"`
	BaseURL  string            `yaml:"base_url"`
	Model    string            `yaml:"model"`
	Timeout  int               `yaml:"timeout"`
	Extra    map[string]string `yaml:"extra"`
}

// Client represents an LLM client that supports multiple providers.
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient creates a new LLM client.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, errors.ErrInvalidArgument
	}

	if config.Timeout <= 0 {
		config.Timeout = 60
	}

	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}, nil
}

// Generate sends a text generation request to the LLM.
// Args:
// ctx - operation context.
// prompt - the prompt text.
// Returns generated text or error.
func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", errors.ErrInvalidArgument
	}

	switch ProviderType(c.config.Provider) {
	case ProviderOpenRouter:
		return c.generateOpenRouter(ctx, prompt)
	case ProviderOllama:
		return c.generateOllama(ctx, prompt)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.config.Provider)
	}
}

// generateOpenRouter generates text using OpenRouter API.
func (c *Client) generateOpenRouter(ctx context.Context, prompt string) (string, error) {
	if c.config.APIKey == "" {
		return "", fmt.Errorf("API key is required for OpenRouter")
	}

	requestBody := map[string]interface{}{
		"model": c.config.Model,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"max_tokens":  256,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/your-repo")
	req.Header.Set("X-Title", "GoAgent Framework")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return response.Choices[0].Message.Content, nil
}

// generateOllama generates text using Ollama API.
func (c *Client) generateOllama(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  c.config.Model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.7,
			"num_predict": 256,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	baseURL := c.config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	return response.Response, nil
}

// IsEnabled checks if the LLM client is properly configured.
func (c *Client) IsEnabled() bool {
	if c == nil || c.config == nil {
		return false
	}

	switch ProviderType(c.config.Provider) {
	case ProviderOpenRouter:
		return c.config.APIKey != ""
	case ProviderOllama:
		return true // Ollama doesn't require API key
	default:
		return false
	}
}

// GetProvider returns the current provider type.
func (c *Client) GetProvider() string {
	if c.config != nil {
		return c.config.Provider
	}
	return ""
}

// GetModel returns the current model name.
func (c *Client) GetModel() string {
	if c.config != nil {
		return c.config.Model
	}
	return ""
}

// NewClientFromEnv creates an LLM client from environment variables.
func NewClientFromEnv() (*Client, error) {
	config := &Config{
		Provider: os.Getenv("LLM_PROVIDER"),
		APIKey:   os.Getenv("LLM_API_KEY"),
		BaseURL:  os.Getenv("LLM_BASE_URL"),
		Model:    os.Getenv("LLM_MODEL"),
	}

	// Set defaults
	if config.Provider == "" {
		config.Provider = "ollama"
	}
	if config.BaseURL == "" {
		if config.Provider == "openrouter" {
			config.BaseURL = "https://openrouter.ai/api/v1"
		} else {
			config.BaseURL = "http://localhost:11434"
		}
	}
	if config.Model == "" {
		if config.Provider == "ollama" {
			config.Model = "llama3"
		} else {
			config.Model = "minimax/minimax-m2-her"
		}
	}

	return NewClient(config)
}