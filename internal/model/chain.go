// Package model defines shared domain types for job chains and workflow definitions.
//
// This file is responsible for:
// - Defining JobChain for named, reusable workflow definitions
// - Defining ChainDefinition with DAG structure (nodes and edges)
// - Providing chain node and edge types
//
// This file does NOT handle:
// - Chain persistence (see store package)
// - Chain execution (see jobs package)
//
// Invariants:
// - ChainDefinition must form a valid DAG (no cycles) - validated at submission time
// - Node IDs within a chain must be unique
// - Edge references must point to valid nodes
package model

import (
	"encoding/json"
	"time"
)

// JobChain represents a named, reusable workflow definition.
// Chains are templates that can be instantiated multiple times to create jobs.
type JobChain struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Definition  ChainDefinition `json:"definition"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

// ChainDefinition defines the workflow structure as a DAG.
type ChainDefinition struct {
	Nodes []ChainNode `json:"nodes"` // Job templates
	Edges []ChainEdge `json:"edges"` // Dependency relationships
}

// ChainNode represents a job template within a chain.
type ChainNode struct {
	ID       string          `json:"id"`      // Node identifier within chain
	Kind     Kind            `json:"kind"`    // Job kind (scrape, crawl, research)
	Request  json.RawMessage `json:"request"` // Operator-facing job request template
	Metadata ChainMetadata   `json:"metadata,omitempty"`
}

// ChainEdge represents a dependency relationship between nodes.
// The "From" node must complete successfully before the "To" node can start.
type ChainEdge struct {
	From string `json:"from"` // Source node ID
	To   string `json:"to"`   // Target node ID
}

// ChainMetadata provides human-readable information about a node.
type ChainMetadata struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// ValidateChainDefinition validates that a chain definition forms a valid DAG.
// Returns an error if there are cycles or invalid node references.
func ValidateChainDefinition(def ChainDefinition) error {
	// Build node lookup
	nodeIDs := make(map[string]bool)
	for _, node := range def.Nodes {
		if node.ID == "" {
			return &ChainValidationError{Message: "node ID cannot be empty"}
		}
		if nodeIDs[node.ID] {
			return &ChainValidationError{Message: "duplicate node ID: " + node.ID}
		}
		nodeIDs[node.ID] = true
	}

	// Validate edges reference valid nodes
	for _, edge := range def.Edges {
		if !nodeIDs[edge.From] {
			return &ChainValidationError{Message: "edge references unknown node: " + edge.From}
		}
		if !nodeIDs[edge.To] {
			return &ChainValidationError{Message: "edge references unknown node: " + edge.To}
		}
		if edge.From == edge.To {
			return &ChainValidationError{Message: "self-referencing edge: " + edge.From}
		}
	}

	// Check for cycles using DFS
	if hasCycle(def) {
		return &ChainValidationError{Message: "chain contains a cycle"}
	}

	return nil
}

// ChainValidationError represents a validation error for chain definitions.
type ChainValidationError struct {
	Message string
}

func (e *ChainValidationError) Error() string {
	return "chain validation: " + e.Message
}

// hasCycle detects cycles in the chain definition using DFS.
func hasCycle(def ChainDefinition) bool {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, edge := range def.Edges {
		graph[edge.From] = append(graph[edge.From], edge.To)
	}

	// Track visited nodes and nodes in current recursion stack
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(node string) bool
	dfs = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if dfs(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	// Check all nodes (in case of disconnected components)
	for _, node := range def.Nodes {
		if !visited[node.ID] {
			if dfs(node.ID) {
				return true
			}
		}
	}

	return false
}

// GetRootNodes returns nodes with no incoming edges (can start immediately).
func (def ChainDefinition) GetRootNodes() []ChainNode {
	// Find all nodes with incoming edges
	hasIncoming := make(map[string]bool)
	for _, edge := range def.Edges {
		hasIncoming[edge.To] = true
	}

	// Return nodes without incoming edges
	var roots []ChainNode
	for _, node := range def.Nodes {
		if !hasIncoming[node.ID] {
			roots = append(roots, node)
		}
	}
	return roots
}

// GetDependencies returns the node IDs that a given node depends on.
func (def ChainDefinition) GetDependencies(nodeID string) []string {
	var deps []string
	for _, edge := range def.Edges {
		if edge.To == nodeID {
			deps = append(deps, edge.From)
		}
	}
	return deps
}
