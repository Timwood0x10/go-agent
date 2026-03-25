// package graph - provides dynamic agent orchestration with pluggable scheduling.

package graph

import (
	"context"
	"fmt"

	"goagent/internal/agents/base"
	"goagent/internal/tools/resources/core"
)

// Node represents an executable unit in the graph.
type Node interface {
	// Execute runs the node with the given state.
	Execute(ctx context.Context, state *State) error
	// ID returns the unique identifier of the node.
	ID() string
}

// AgentNode wraps an existing agent to be used as a node.
type AgentNode struct {
	agent base.Agent
}

// NewAgentNode creates a new agent node.
func NewAgentNode(agent base.Agent) *AgentNode {
	if agent == nil {
		panic("agent cannot be nil")
	}
	return &AgentNode{agent: agent}
}

// Execute runs the agent node.
func (n *AgentNode) Execute(ctx context.Context, state *State) error {
	if n == nil || n.agent == nil {
		return fmt.Errorf("agent node is not initialized")
	}

	input, _ := state.Get("input")
	result, err := n.agent.Process(ctx, input)
	if err != nil {
		return fmt.Errorf("agent %s execution failed: %w", n.ID(), err)
	}

	state.Set("node."+n.ID(), result)
	return nil
}

// ID returns the agent ID.
func (n *AgentNode) ID() string {
	if n == nil || n.agent == nil {
		return ""
	}
	return n.agent.ID()
}

// ToolNode wraps an existing tool to be used as a node.
type ToolNode struct {
	tool core.Tool
}

// NewToolNode creates a new tool node.
func NewToolNode(tool core.Tool) *ToolNode {
	if tool == nil {
		panic("tool cannot be nil")
	}
	return &ToolNode{tool: tool}
}

// Execute runs the tool node.
func (n *ToolNode) Execute(ctx context.Context, state *State) error {
	if n == nil || n.tool == nil {
		return fmt.Errorf("tool node is not initialized")
	}

	params := state.ToParams()
	result, err := n.tool.Execute(ctx, params)
	if err != nil {
		return fmt.Errorf("tool %s execution failed: %w", n.ID(), err)
	}

	if result.Success {
		state.Set("node."+n.ID(), result.Data)
	} else {
		state.Set("node."+n.ID(), result.Error)
	}
	return nil
}

// ID returns the tool name.
func (n *ToolNode) ID() string {
	if n == nil || n.tool == nil {
		return ""
	}
	return n.tool.Name()
}

// FuncNode wraps a simple function to be used as a node.
type FuncNode struct {
	id string
	fn func(context.Context, *State) error
}

// NewFuncNode creates a new function node.
func NewFuncNode(id string, fn func(context.Context, *State) error) *FuncNode {
	if id == "" {
		panic("node id cannot be empty")
	}
	if fn == nil {
		panic("function cannot be nil")
	}
	return &FuncNode{id: id, fn: fn}
}

// Execute runs the function node.
func (n *FuncNode) Execute(ctx context.Context, state *State) error {
	if n == nil || n.fn == nil {
		return fmt.Errorf("function node is not initialized")
	}

	err := n.fn(ctx, state)
	if err != nil {
		return fmt.Errorf("function %s execution failed: %w", n.ID(), err)
	}

	return nil
}

// ID returns the function node ID.
func (n *FuncNode) ID() string {
	if n == nil {
		return ""
	}
	return n.id
}
