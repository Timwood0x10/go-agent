package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goagent/internal/agents/base"
	"goagent/internal/agents/leader"
	"goagent/internal/agents/sub"
	"goagent/internal/config"
	"goagent/internal/core/models"
	"goagent/internal/llm/output"
	"goagent/internal/observability"
	"goagent/internal/protocol/ahp"
)

// This is an example demonstrating how to use the framework.
// Users can configure:
// - Number and types of Agents
// - Prompt templates
// - Max retries, max steps, etc.
// All through configuration files.

func main() {
	log.Println("Starting Style Agent Example...")

	// Load configuration from file
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./examples/simple/config/server.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override with environment variables if present
	if err := config.LoadFromEnv(cfg); err != nil {
		log.Fatalf("Failed to load env config: %v", err)
	}

	// Initialize components
	components, err := initializeComponents(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize components: %v", err)
	}

	// Create Leader Agent with user configuration
	leaderAgent := createLeaderAgent(cfg, components)

	// Create Sub Agents based on user configuration
	subAgents := createSubAgents(cfg, components)

	log.Printf("Initialized %d Sub Agents", len(subAgents))

	// Start all agents
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := leaderAgent.Start(ctx); err != nil {
		log.Fatalf("Failed to start leader agent: %v", err)
	}

	for _, agent := range subAgents {
		if err := agent.Start(ctx); err != nil {
			log.Printf("Warning: failed to start sub agent %s: %v", agent.ID(), err)
		}
	}

	// Setup graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		time.Sleep(time.Second)
		os.Exit(0)
	}()

	// Process a sample request
	processSampleRequest(leaderAgent)

	log.Println("Example completed successfully")
}

type components struct {
	llmAdapter   output.LLMAdapter
	tracer       observability.Tracer
	messageQueue *ahp.MessageQueue
	validator    *output.Validator
	template     *output.TemplateEngine
}

func initializeComponents(cfg *config.Config) (*components, error) {
	// Create LLM adapter based on user configuration
	llmFactory := output.NewFactory()
	llmAdapter, err := llmFactory.Create(cfg.LLM.Provider, &output.Config{
		APIKey:  cfg.LLM.APIKey,
		BaseURL: cfg.LLM.BaseURL,
		Model:   cfg.LLM.Model,
		Timeout: cfg.LLM.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create LLM adapter: %w", err)
	}

	// Create tracer (NoopTracer for production, LogTracer for development)
	tracer := observability.NewNoopTracer()

	// Create message queue
	messageQueue := ahp.NewMessageQueue("main", &ahp.QueueOptions{MaxSize: 1000})

	// Create validator
	validator := output.NewValidator()

	// Create template engine
	template := output.NewTemplateEngine()

	return &components{
		llmAdapter:   llmAdapter,
		tracer:       tracer,
		messageQueue: messageQueue,
		validator:    validator,
		template:     template,
	}, nil
}

func createLeaderAgent(cfg *config.Config, comps *components) leader.Agent {
	// Create ProfileParser with user-configured prompts
	profileParser := leader.NewProfileParser(
		comps.llmAdapter,
		comps.template,
		cfg.Prompts.ProfileExtraction,
		comps.validator,
		cfg.Agents.Leader.MaxValidationRetry,
	)

	// Create TaskPlanner
	taskPlanner := leader.NewTaskPlanner(len(cfg.Agents.Sub))

	// Create TaskDispatcher
	agentRegistry := make(map[models.AgentType]string)
	for _, subCfg := range cfg.Agents.Sub {
		agentRegistry[models.AgentType(subCfg.Type)] = subCfg.ID
	}

	taskDispatcher := leader.NewTaskDispatcher(
		agentRegistry,
		cfg.Agents.Leader.MaxParallelTasks,
		cfg.Agents.Leader.MaxSteps,
	)

	// Register executor functions for each sub-agent type
	for _, subCfg := range cfg.Agents.Sub {
		agentType := models.AgentType(subCfg.Type)
		executor := sub.NewTaskExecutor(
			nil, // Tool binder (not needed for simple example)
			comps.llmAdapter,
			comps.template,
			cfg.Prompts.Recommendation,
			comps.validator,
			subCfg.MaxRetries,
		)
		// Register the executor
		taskDispatcher.RegisterExecutor(agentType, executor.Execute)
	}

	// Create ResultAggregator
	resultAggregator := leader.NewResultAggregator(true, 10)

	// Create LeaderAgent config from user configuration
	leaderCfg := &leader.LeaderAgentConfig{
		Config: base.Config{
			ID:   cfg.Agents.Leader.ID,
			Type: models.AgentTypeLeader,
		},
		MaxParallelTasks: cfg.Agents.Leader.MaxParallelTasks,
		MaxSteps:         cfg.Agents.Leader.MaxSteps,
		EnableCache:      cfg.Agents.Leader.EnableCache,
	}

	// Create heartbeat monitor
	hbMon := ahp.NewHeartbeatMonitor(ahp.DefaultHeartbeatConfig())

	return leader.New(
		cfg.Agents.Leader.ID,
		profileParser,
		taskPlanner,
		taskDispatcher,
		resultAggregator,
		comps.messageQueue,
		hbMon,
		leaderCfg,
	)
}

func createSubAgents(cfg *config.Config, comps *components) []sub.Agent {
	agents := make([]sub.Agent, 0, len(cfg.Agents.Sub))

	for _, subCfg := range cfg.Agents.Sub {
		// Create ToolBinder (for future tool integration)
		toolBinder := sub.NewToolBinder()

		// Create TaskExecutor with user-configured prompts
		executor := sub.NewTaskExecutor(
			toolBinder,
			comps.llmAdapter,
			comps.template,
			cfg.Prompts.Recommendation,
			comps.validator,
			subCfg.MaxRetries,
		)

		// Create heartbeat monitor
		hbMon := ahp.NewHeartbeatMonitor(ahp.DefaultHeartbeatConfig())

		// Create SubAgent config
		subCfgModel := &sub.SubAgentConfig{
			Config: base.Config{
				ID:   subCfg.ID,
				Type: models.AgentType(subCfg.Type),
			},
			EnableTools: false,
		}

		// Create message handler
		handler := sub.NewMessageHandler(subCfg.ID)

		agent := sub.New(
			subCfg.ID,
			models.AgentType(subCfg.Type),
			executor,
			handler,
			comps.messageQueue,
			hbMon,
			subCfgModel,
		)

		agents = append(agents, agent)
	}

	return agents
}

func processSampleRequest(agent leader.Agent) {
	ctx := context.Background()

	// Sample user input
	input := "我想找一些适合日常通勤的衣服，休闲风格，预算500-1000元"

	log.Printf("Processing request: %s", input)

	result, err := agent.Process(ctx, input)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}

	if recommendResult, ok := result.(*models.RecommendResult); ok {
		log.Printf("Got %d recommendations", len(recommendResult.Items))
		for _, item := range recommendResult.Items {
			log.Printf("  - %s: %s (%.2f)", item.ItemID, item.Name, item.Price)
		}
	} else {
		log.Printf("Result: %+v", result)
	}
}
