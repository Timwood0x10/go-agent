package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the server.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	LLM      LLMConfig      `yaml:"llm"`
	Agents   AgentsConfig   `yaml:"agents"`
	Prompts  PromptsConfig  `yaml:"prompts"`
	Workflow WorkflowConfig `yaml:"workflow"`
}

// ServerConfig holds server configuration.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// LLMConfig holds LLM provider configuration.
type LLMConfig struct {
	Provider string            `yaml:"provider"` // "openai", "ollama"
	APIKey   string            `yaml:"api_key"`
	BaseURL  string            `yaml:"base_url"`
	Model    string            `yaml:"model"`
	Timeout  int               `yaml:"timeout"` // seconds
	Extra    map[string]string `yaml:"extra"`
}

// AgentsConfig holds agent configuration.
type AgentsConfig struct {
	Leader LeaderConfig     `yaml:"leader"`
	Sub    []SubAgentConfig `yaml:"sub"`
}

// LeaderConfig holds Leader Agent configuration.
type LeaderConfig struct {
	ID                 string `yaml:"id"`
	MaxSteps           int    `yaml:"max_steps"`
	MaxParallelTasks   int    `yaml:"max_parallel_tasks"`
	MaxValidationRetry int    `yaml:"max_validation_retry"`
	EnableCache        bool   `yaml:"enable_cache"`
}

// SubAgentConfig holds Sub Agent configuration.
type SubAgentConfig struct {
	ID         string `yaml:"id"`
	Type       string `yaml:"type"` // "top", "bottom", "shoes", "head", "accessory"
	Category   string `yaml:"category"`
	MaxRetries int    `yaml:"max_retries"`
	Timeout    int    `yaml:"timeout"` // seconds
}

// PromptsConfig holds prompt templates.
type PromptsConfig struct {
	ProfileExtraction string `yaml:"profile_extraction"`
	Recommendation    string `yaml:"recommendation"`
	StyleAnalysis     string `yaml:"style_analysis"`
}

// WorkflowConfig holds workflow configuration.
type WorkflowConfig struct {
	DefinitionPath string `yaml:"definition_path"` // path to workflow YAML
	AutoReload     bool   `yaml:"auto_reload"`
	ReloadInterval int    `yaml:"reload_interval"` // seconds
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Set defaults
	cfg.setDefaults()

	return &cfg, nil
}

// LoadFromEnv loads configuration from environment variables.
// Environment variables override YAML config.
func LoadFromEnv(cfg *Config) error {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		var port int
		if _, err := fmt.Sscanf(v, "%d", &port); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("LLM_API_KEY"); v != "" {
		cfg.LLM.APIKey = v
	}
	if v := os.Getenv("LLM_PROVIDER"); v != "" {
		cfg.LLM.Provider = v
	}
	if v := os.Getenv("LLM_BASE_URL"); v != "" {
		cfg.LLM.BaseURL = v
	}
	if v := os.Getenv("LLM_MODEL"); v != "" {
		cfg.LLM.Model = v
	}

	return nil
}

func (c *Config) setDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "localhost"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.LLM.Provider == "" {
		c.LLM.Provider = "ollama"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "llama3"
	}
	if c.LLM.Timeout == 0 {
		c.LLM.Timeout = 60
	}
	if c.Agents.Leader.MaxSteps == 0 {
		c.Agents.Leader.MaxSteps = 10
	}
	if c.Agents.Leader.MaxParallelTasks == 0 {
		c.Agents.Leader.MaxParallelTasks = 5
	}
	if c.Agents.Leader.MaxValidationRetry == 0 {
		c.Agents.Leader.MaxValidationRetry = 3
	}
}
