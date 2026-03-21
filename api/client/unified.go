// Package client provides unified client interface for GoAgent API.
package client

import (
	"context"
	"fmt"
	"time"

	"goagent/api/core"
	agentSvc "goagent/api/service/agent"
	llmSvc "goagent/api/service/llm"
	memorySvc "goagent/api/service/memory"
	retrievalSvc "goagent/api/service/retrieval"
)

// Client provides unified client interface for all GoAgent modules.
type Client struct {
	agentService     *agentSvc.Service
	memoryService    *memorySvc.Service
	retrievalService *retrievalSvc.Service
	llmService       *llmSvc.Service
	config           *Config
	configFile       *ConfigFile
}

// Config configuration for GoAgent client.
type Config struct {
	// BaseConfig is the base configuration.
	BaseConfig *core.BaseConfig
	// Agent is the agent service configuration.
	Agent *agentSvc.Config
	// Memory is the memory service configuration.
	Memory *memorySvc.Config
	// Retrieval is the retrieval service configuration.
	Retrieval *retrievalSvc.Config
	// LLM is the LLM service configuration.
	LLM *llmSvc.Config
}

// NewClient creates a new GoAgent client instance.
// Args:
// config - client configuration.
// Returns new client instance or error.
func NewClient(config *Config) (*Client, error) {
	if config == nil {
		return nil, ErrInvalidConfig
	}

	if config.BaseConfig == nil {
		config.BaseConfig = &core.BaseConfig{
			RequestTimeout: 30 * time.Second,
			MaxRetries:     3,
			RetryDelay:     1 * time.Second,
		}
	}

	client := &Client{
		config: config,
	}

	// Initialize services if configurations are provided
	if config.Agent != nil {
		agentService, err := agentSvc.NewService(config.Agent)
		if err != nil {
			return nil, fmt.Errorf("create agent service: %w", err)
		}
		client.agentService = agentService
	}

	if config.Memory != nil {
		memoryService, err := memorySvc.NewService(config.Memory)
		if err != nil {
			return nil, fmt.Errorf("create memory service: %w", err)
		}
		client.memoryService = memoryService
	}

	if config.Retrieval != nil {
		retrievalService, err := retrievalSvc.NewService(config.Retrieval)
		if err != nil {
			return nil, fmt.Errorf("create retrieval service: %w", err)
		}
		client.retrievalService = retrievalService
	}

	if config.LLM != nil {
		llmService, err := llmSvc.NewService(config.LLM)
		if err != nil {
			return nil, fmt.Errorf("create LLM service: %w", err)
		}
		client.llmService = llmService
	}

	return client, nil
}

// Agent returns the agent service.
// Returns the agent service or error if not configured.
func (c *Client) Agent() (*agentSvc.Service, error) {
	if c.agentService == nil {
		return nil, ErrAgentNotConfigured
	}
	return c.agentService, nil
}

// Memory returns the memory service.
// Returns the memory service or error if not configured.
func (c *Client) Memory() (*memorySvc.Service, error) {
	if c.memoryService == nil {
		return nil, ErrMemoryNotConfigured
	}
	return c.memoryService, nil
}

// Retrieval returns the retrieval service.
// Returns the retrieval service or error if not configured.
func (c *Client) Retrieval() (*retrievalSvc.Service, error) {
	if c.retrievalService == nil {
		return nil, ErrRetrievalNotConfigured
	}
	return c.retrievalService, nil
}

// LLM returns the LLM service.
// Returns the LLM service or error if not configured.
func (c *Client) LLM() (*llmSvc.Service, error) {
	if c.llmService == nil {
		return nil, ErrLLMNotConfigured
	}
	return c.llmService, nil
}

// Close closes the client and cleans up resources.
// Args:
// ctx - operation context.
// Returns error if cleanup fails.
func (c *Client) Close(ctx context.Context) error {
	return nil
}

// GetConfig returns the loaded configuration file.
// Returns configuration file structure or nil if not available.
func (c *Client) GetConfig() *ConfigFile {
	return c.configFile
}

// NewClientWithConfigFile creates a new GoAgent client with both config and config file.
// Args:
// config - client configuration.
// configFile - configuration file structure.
// Returns new client instance or error.
func NewClientWithConfigFile(config *Config, configFile *ConfigFile) (*Client, error) {
	client, err := NewClient(config)
	if err != nil {
		return nil, err
	}
	client.configFile = configFile
	return client, nil
}

// Ping checks if all configured services are available.
// Args:
// ctx - operation context.
// Returns true if all services are available, false otherwise.
func (c *Client) Ping(ctx context.Context) bool {
	if c.agentService != nil {
		// TODO: Ping agent service
	}
	if c.memoryService != nil {
		// TODO: Ping memory service
	}
	if c.retrievalService != nil {
		// TODO: Ping retrieval service
	}
	if c.llmService != nil {
		return c.llmService.IsEnabled()
	}
	return true
}
