// Package graph provides tests for YAML configuration parsing.

package graph

import (
	"testing"
)

func TestParseGraphConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{
			name: "valid simple config",
			yaml: `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
    - id: "node2"
      type: "function"
  edges:
    - from: "node1"
      to: "node2"
`,
			wantErr: false,
		},
		{
			name: "missing graph id",
			yaml: `
graph:
  id: ""
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
  edges: []
`,
			wantErr: true,
		},
		{
			name: "missing start node",
			yaml: `
graph:
  id: "test-graph"
  start_node: ""
  nodes:
    - id: "node1"
      type: "function"
  edges: []
`,
			wantErr: true,
		},
		{
			name: "invalid node type",
			yaml: `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "invalid_type"
  edges: []
`,
			wantErr: true,
		},
		{
			name: "duplicate node ids",
			yaml: `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
    - id: "node1"
      type: "function"
  edges: []
`,
			wantErr: true,
		},
		{
			name: "invalid start node",
			yaml: `
graph:
  id: "test-graph"
  start_node: "nonexistent"
  nodes:
    - id: "node1"
      type: "function"
  edges: []
`,
			wantErr: true,
		},
		{
			name: "edge with invalid source",
			yaml: `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
  edges:
    - from: "invalid"
      to: "node1"
`,
			wantErr: true,
		},
		{
			name: "edge with invalid target",
			yaml: `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
  edges:
    - from: "node1"
      to: "invalid"
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := ParseGraphConfig([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseGraphConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && config == nil {
				t.Error("ParseGraphConfig() returned nil config")
			}
		})
	}
}

func TestGraphDefinitionGetNodeByID(t *testing.T) {
	gdef := &GraphDefinition{
		Nodes: []Node{
			{ID: "node1", Type: "function"},
			{ID: "node2", Type: "function"},
		},
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"existing node", "node1", true},
		{"existing node 2", "node2", true},
		{"non-existing node", "node3", false},
		{"empty id", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := gdef.GetNodeByID(tt.id)
			if got != tt.want {
				t.Errorf("GetNodeByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGraphDefinitionGetAgentByID(t *testing.T) {
	gdef := &GraphDefinition{
		Agents: []Agent{
			{ID: "agent1", Type: "leader"},
			{ID: "agent2", Type: "processor"},
		},
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"existing agent", "agent1", true},
		{"existing agent 2", "agent2", true},
		{"non-existing agent", "agent3", false},
		{"empty id", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := gdef.GetAgentByID(tt.id)
			if got != tt.want {
				t.Errorf("GetAgentByID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildSimpleBasic(t *testing.T) {
	yaml := `
graph:
  id: "test-graph"
  start_node: "node1"
  nodes:
    - id: "node1"
      type: "function"
    - id: "node2"
      type: "function"
  edges:
    - from: "node1"
      to: "node2"
`

	g, err := BuildSimple([]byte(yaml))
	if err != nil {
		t.Fatalf("BuildSimple() error = %v", err)
	}

	if g == nil {
		t.Fatal("BuildSimple() returned nil graph")
	}

	if g.ID() != "test-graph" {
		t.Errorf("Graph ID = %s, want test-graph", g.ID())
	}
}

func TestBuildSimpleWithInvalidYAML(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{"empty yaml", "", true},
		{"invalid yaml syntax", "graph: {", true},
		{"missing required fields", "graph: {}", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildSimple([]byte(tt.yaml))
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildSimple() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
