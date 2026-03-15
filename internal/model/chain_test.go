// Package model provides tests for chain-related types and functions.
package model

import (
	"testing"
)

func TestValidateChainDefinition(t *testing.T) {
	tests := []struct {
		name    string
		def     ChainDefinition
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid simple chain",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/2"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid diamond chain",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/2"}`)},
					{ID: "c", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/3"}`)},
					{ID: "d", Kind: KindCrawl, Request: []byte(`{"url":"http://example.com/4"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
					{From: "a", To: "c"},
					{From: "b", To: "d"},
					{From: "c", To: "d"},
				},
			},
			wantErr: false,
		},
		{
			name: "empty node ID",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
				},
				Edges: []ChainEdge{},
			},
			wantErr: true,
			errMsg:  "node ID cannot be empty",
		},
		{
			name: "duplicate node ID",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
					{ID: "a", Kind: KindCrawl, Request: []byte(`{"url":"http://example.com/2"}`)},
				},
				Edges: []ChainEdge{},
			},
			wantErr: true,
			errMsg:  "duplicate node ID: a",
		},
		{
			name: "edge references unknown node (from)",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
				},
				Edges: []ChainEdge{
					{From: "x", To: "a"},
				},
			},
			wantErr: true,
			errMsg:  "edge references unknown node: x",
		},
		{
			name: "edge references unknown node (to)",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "y"},
				},
			},
			wantErr: true,
			errMsg:  "edge references unknown node: y",
		},
		{
			name: "self-referencing edge",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "a"},
				},
			},
			wantErr: true,
			errMsg:  "self-referencing edge: a",
		},
		{
			name: "simple cycle",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/2"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
					{From: "b", To: "a"},
				},
			},
			wantErr: true,
			errMsg:  "chain contains a cycle",
		},
		{
			name: "indirect cycle",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{"url":"http://example.com"}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/2"}`)},
					{ID: "c", Kind: KindScrape, Request: []byte(`{"url":"http://example.com/3"}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
					{From: "b", To: "c"},
					{From: "c", To: "a"},
				},
			},
			wantErr: true,
			errMsg:  "chain contains a cycle",
		},
		{
			name: "empty chain (no nodes)",
			def: ChainDefinition{
				Nodes: []ChainNode{},
				Edges: []ChainEdge{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChainDefinition(tt.def)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateChainDefinition() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err.Error() != "chain validation: "+tt.errMsg {
					t.Errorf("ValidateChainDefinition() error = %v, want %v", err.Error(), "chain validation: "+tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateChainDefinition() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestChainDefinitionGetRootNodes(t *testing.T) {
	tests := []struct {
		name     string
		def      ChainDefinition
		expected []string
	}{
		{
			name: "linear chain",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "c", Kind: KindScrape, Request: []byte(`{}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
					{From: "b", To: "c"},
				},
			},
			expected: []string{"a"},
		},
		{
			name: "diamond chain",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "c", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "d", Kind: KindCrawl, Request: []byte(`{}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "b"},
					{From: "a", To: "c"},
					{From: "b", To: "d"},
					{From: "c", To: "d"},
				},
			},
			expected: []string{"a"},
		},
		{
			name: "multiple roots",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "c", Kind: KindCrawl, Request: []byte(`{}`)},
				},
				Edges: []ChainEdge{
					{From: "a", To: "c"},
					{From: "b", To: "c"},
				},
			},
			expected: []string{"a", "b"},
		},
		{
			name: "all isolated nodes",
			def: ChainDefinition{
				Nodes: []ChainNode{
					{ID: "a", Kind: KindScrape, Request: []byte(`{}`)},
					{ID: "b", Kind: KindScrape, Request: []byte(`{}`)},
				},
				Edges: []ChainEdge{},
			},
			expected: []string{"a", "b"},
		},
		{
			name: "empty chain",
			def: ChainDefinition{
				Nodes: []ChainNode{},
				Edges: []ChainEdge{},
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roots := tt.def.GetRootNodes()
			if len(roots) != len(tt.expected) {
				t.Errorf("GetRootNodes() returned %d roots, expected %d", len(roots), len(tt.expected))
				return
			}
			rootIDs := make([]string, len(roots))
			for i, r := range roots {
				rootIDs[i] = r.ID
			}
			// Check that all expected IDs are present
			for _, expected := range tt.expected {
				found := false
				for _, id := range rootIDs {
					if id == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("GetRootNodes() missing expected root: %s", expected)
				}
			}
		})
	}
}

func TestChainDefinitionGetDependencies(t *testing.T) {
	def := ChainDefinition{
		Nodes: []ChainNode{
			{ID: "a", Kind: KindScrape, Request: []byte(`{}`)},
			{ID: "b", Kind: KindScrape, Request: []byte(`{}`)},
			{ID: "c", Kind: KindCrawl, Request: []byte(`{}`)},
		},
		Edges: []ChainEdge{
			{From: "a", To: "c"},
			{From: "b", To: "c"},
		},
	}

	// Test getting dependencies for node c
	deps := def.GetDependencies("c")
	if len(deps) != 2 {
		t.Errorf("GetDependencies('c') returned %d deps, expected 2", len(deps))
	}

	// Test getting dependencies for node a (no dependencies)
	depsA := def.GetDependencies("a")
	if len(depsA) != 0 {
		t.Errorf("GetDependencies('a') returned %d deps, expected 0", len(depsA))
	}

	// Test getting dependencies for non-existent node
	depsX := def.GetDependencies("x")
	if len(depsX) != 0 {
		t.Errorf("GetDependencies('x') returned %d deps, expected 0", len(depsX))
	}
}

func TestDependencyStatusIsValid(t *testing.T) {
	tests := []struct {
		status DependencyStatus
		valid  bool
	}{
		{DependencyStatusPending, true},
		{DependencyStatusReady, true},
		{DependencyStatusFailed, true},
		{DependencyStatus("invalid"), false},
		{DependencyStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if got := tt.status.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}
