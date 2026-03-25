// package graph - provides dynamic agent orchestration with pluggable scheduling.

package graph

import (
	"time"
)

// Edge represents a connection between two nodes with optional condition.
type Edge struct {
	from string
	to   string
	cond Condition
}

// Condition defines a predicate function for edge traversal.
type Condition func(state *State) bool

// IfFunc creates a condition from a function.
func IfFunc(fn func(state *State) bool) Condition {
	return fn
}

// Graph represents a DAG of nodes with conditional edges.
type Graph struct {
	id        string
	nodes     map[string]Node
	edges     map[string][]*Edge
	start     string
	scheduler Scheduler
}

// NewGraph creates a new graph with the given ID.
func NewGraph(id string) *Graph {
	if id == "" {
		panic("graph ID cannot be empty")
	}
	return &Graph{
		id:        id,
		nodes:     make(map[string]Node),
		edges:     make(map[string][]*Edge),
		scheduler: NewDefaultScheduler(),
	}
}

// Node adds a node to the graph.
func (g *Graph) Node(id string, node Node) *Graph {
	if g == nil {
		panic("graph is nil")
	}
	if id == "" {
		panic("node ID cannot be empty")
	}
	if node == nil {
		panic("node cannot be nil")
	}
	g.nodes[id] = node
	return g
}

// Edge adds an edge from one node to another with optional condition.
func (g *Graph) Edge(from, to string, cond ...Condition) *Graph {
	if g == nil {
		panic("graph is nil")
	}
	if from == "" {
		panic("from node ID cannot be empty")
	}
	if to == "" {
		panic("to node ID cannot be empty")
	}

	edge := &Edge{from: from, to: to}
	if len(cond) > 0 {
		edge.cond = cond[0]
	}

	g.edges[from] = append(g.edges[from], edge)
	return g
}

// Start sets the starting node for the graph.
func (g *Graph) Start(id string) *Graph {
	if g == nil {
		panic("graph is nil")
	}
	if id == "" {
		panic("start node ID cannot be empty")
	}
	g.start = id
	return g
}

// SetScheduler sets a custom scheduler for the graph.
func (g *Graph) SetScheduler(scheduler Scheduler) *Graph {
	if g == nil {
		panic("graph is nil")
	}
	if scheduler == nil {
		panic("scheduler cannot be nil")
	}
	g.scheduler = scheduler
	return g
}

// ID returns the graph ID.
func (g *Graph) ID() string {
	if g == nil {
		return ""
	}
	return g.id
}

// Result represents the result of graph execution.
type Result struct {
	GraphID  string
	State    *State
	Duration time.Duration
	Error    error
}
