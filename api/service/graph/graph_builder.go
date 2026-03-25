// Package graph provides graph building functionality from YAML configuration.

package graph

import (
	"context"
	"fmt"
	"time"

	"goagent/internal/agents/base"
	"goagent/internal/observability"
	wfgraph "goagent/internal/workflow/graph"
)

// GraphBuilder builds graph instances from configuration.
type GraphBuilder struct {
	agentRegistry map[string]base.Agent
	toolRegistry  map[string]interface{}
}

// NewGraphBuilder creates a new graph builder.
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		agentRegistry: make(map[string]base.Agent),
		toolRegistry:  make(map[string]interface{}),
	}
}

// RegisterAgent registers an agent for use in graph configuration.
func (b *GraphBuilder) RegisterAgent(agent base.Agent) {
	if b == nil || agent == nil {
		return
	}
	b.agentRegistry[agent.ID()] = agent
}

// RegisterTool registers a tool for use in graph configuration.
func (b *GraphBuilder) RegisterTool(id string, tool interface{}) {
	if b == nil || id == "" || tool == nil {
		return
	}
	b.toolRegistry[id] = tool
}

// Build builds a graph from configuration.
func (b *GraphBuilder) Build(config *GraphConfig) (*wfgraph.Graph, error) {
	if b == nil {
		return nil, fmt.Errorf("builder is nil")
	}
	if config == nil {
		return nil, fmt.Errorf("config is nil")
	}

	gdef := &config.Graph

	// Create new graph
	g := wfgraph.NewGraph(gdef.ID)

	// Build nodes
	for _, nodeConfig := range gdef.Nodes {
		node, err := b.buildNode(nodeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build node '%s': %w", nodeConfig.ID, err)
		}
		g.Node(nodeConfig.ID, node)
	}

	// Build edges
	for _, edgeConfig := range gdef.Edges {
		if edgeConfig.Condition != "" {
			// TODO: implement condition parsing from string (expected by 2026-04-01)
			g.Edge(edgeConfig.From, edgeConfig.To)
		} else {
			g.Edge(edgeConfig.From, edgeConfig.To)
		}
	}

	// Set start node
	g.Start(gdef.StartNode)

	return g, nil
}

// buildNode builds a single node from configuration.
func (b *GraphBuilder) buildNode(config Node) (wfgraph.Node, error) {
	switch config.Type {
	case "function":
		return b.buildFuncNode(config)
	case "agent":
		return b.buildAgentNode(config)
	case "tool":
		return b.buildToolNode(config)
	default:
		return nil, fmt.Errorf("unsupported node type: %s", config.Type)
	}
}

// buildFuncNode builds a function node.
func (b *GraphBuilder) buildFuncNode(config Node) (wfgraph.Node, error) {
	if config.ID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	// Create a simple function that logs the node execution.
	fn := func(ctx context.Context, state *wfgraph.State) error {
		if ctx == nil {
			return fmt.Errorf("context is nil")
		}
		if state == nil {
			return fmt.Errorf("state is nil")
		}

		fmt.Printf("[Node %s] Executing...\n", config.ID)
		if config.Description != "" {
			fmt.Printf("  Description: %s\n", config.Description)
		}

		// Store node execution timestamp.
		state.Set(fmt.Sprintf("node.%s.timestamp", config.ID), "executed")
		state.Set(fmt.Sprintf("node.%s.status", config.ID), "success")

		return nil
	}

	return wfgraph.NewFuncNode(config.ID, fn), nil
}

// buildAgentNode builds an agent node.
func (b *GraphBuilder) buildAgentNode(config Node) (wfgraph.Node, error) {
	if b == nil {
		return nil, fmt.Errorf("builder is nil")
	}

	if config.ID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	// Get agent ID from config.
	agentID, ok := config.Config["agent_id"].(string)
	if !ok {
		// Use node ID as agent ID.
		agentID = config.ID
	}

	agent, exists := b.agentRegistry[agentID]
	if !exists {
		return nil, fmt.Errorf("agent '%s' not registered", agentID)
	}

	return wfgraph.NewAgentNode(agent), nil
}

// buildToolNode builds a tool node.
func (b *GraphBuilder) buildToolNode(config Node) (wfgraph.Node, error) {
	if b == nil {
		return nil, fmt.Errorf("builder is nil")
	}

	if config.ID == "" {
		return nil, fmt.Errorf("node ID is required")
	}

	// Get tool ID from config.
	toolID, ok := config.Config["tool_id"].(string)
	if !ok {
		toolID = config.ID
	}

	tool, exists := b.toolRegistry[toolID]
	if !exists {
		return nil, fmt.Errorf("tool '%s' not registered", toolID)
	}

	// TODO: implement ToolNode once we have the Tool interface (expected by 2026-04-01)
	_ = tool
	return nil, fmt.Errorf("tool nodes not yet implemented")
}

// BuildWithService creates a complete graph service from configuration.
func BuildWithService(configYAML []byte, builder *GraphBuilder) (*Service, *wfgraph.Graph, error) {
	if len(configYAML) == 0 {
		return nil, nil, fmt.Errorf("config YAML is empty")
	}
	if builder == nil {
		return nil, nil, fmt.Errorf("builder is nil")
	}

	// Parse configuration.
	graphConfig, err := ParseGraphConfig(configYAML)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Build graph.
	g, err := builder.Build(graphConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build graph: %w", err)
	}

	// Create service.
	serviceConfig := &Config{
		RequestTimeout: 30 * time.Second,
		Tracer:         observability.NewLogTracer(nil),
	}

	service, err := NewService(serviceConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create service: %w", err)
	}

	return service, g, nil
}

// BuildSimple creates a simple graph from YAML without agent registration.
func BuildSimple(configYAML []byte) (*wfgraph.Graph, error) {
	if len(configYAML) == 0 {
		return nil, fmt.Errorf("config YAML is empty")
	}

	graphConfig, err := ParseGraphConfig(configYAML)
	if err != nil {
		return nil, err
	}

	builder := NewGraphBuilder()
	return builder.Build(graphConfig)
}
