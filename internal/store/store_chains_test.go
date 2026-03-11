// Package store provides tests for chain-related storage operations.
package store

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestCreateAndGetChain(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	chain := model.JobChain{
		ID:          "test-chain-1",
		Name:        "Test Chain",
		Description: "A test chain",
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{
				{ID: "node1", Kind: model.KindScrape, Spec: []byte(`{"url":"http://example.com"}`)},
				{ID: "node2", Kind: model.KindCrawl, Spec: []byte(`{"url":"http://example.com/2"}`)},
			},
			Edges: []model.ChainEdge{
				{From: "node1", To: "node2"},
			},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Create chain
	err = s.CreateChain(ctx, chain)
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	// Get chain by ID
	got, err := s.GetChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("GetChain failed: %v", err)
	}

	if got.ID != chain.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, chain.ID)
	}
	if got.Name != chain.Name {
		t.Errorf("Name mismatch: got %s, want %s", got.Name, chain.Name)
	}
	if got.Description != chain.Description {
		t.Errorf("Description mismatch: got %s, want %s", got.Description, chain.Description)
	}
	if len(got.Definition.Nodes) != len(chain.Definition.Nodes) {
		t.Errorf("Nodes length mismatch: got %d, want %d", len(got.Definition.Nodes), len(chain.Definition.Nodes))
	}
	if len(got.Definition.Edges) != len(chain.Definition.Edges) {
		t.Errorf("Edges length mismatch: got %d, want %d", len(got.Definition.Edges), len(chain.Definition.Edges))
	}
}

func TestGetChainByName(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	chain := model.JobChain{
		ID:   "test-chain-2",
		Name: "Unique Chain Name",
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{
				{ID: "node1", Kind: model.KindScrape, Spec: []byte(`{}`)},
			},
			Edges: []model.ChainEdge{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.CreateChain(ctx, chain)
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	// Get by name
	got, err := s.GetChainByName(ctx, chain.Name)
	if err != nil {
		t.Fatalf("GetChainByName failed: %v", err)
	}

	if got.ID != chain.ID {
		t.Errorf("ID mismatch: got %s, want %s", got.ID, chain.ID)
	}

	// Try to get non-existent name
	_, err = s.GetChainByName(ctx, "non-existent-chain")
	if err == nil {
		t.Error("Expected error for non-existent chain name")
	}
}

func TestGetChainNotFound(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	_, err = s.GetChain(ctx, "non-existent-id")
	if err == nil {
		t.Error("Expected error for non-existent chain")
	}
}

func TestUpdateChain(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	chain := model.JobChain{
		ID:          "test-chain-3",
		Name:        "Original Name",
		Description: "Original description",
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{
				{ID: "node1", Kind: model.KindScrape, Spec: []byte(`{}`)},
			},
			Edges: []model.ChainEdge{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.CreateChain(ctx, chain)
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	// Update chain
	chain.Name = "Updated Name"
	chain.Description = "Updated description"
	chain.Definition.Nodes = append(chain.Definition.Nodes, model.ChainNode{
		ID: "node2", Kind: model.KindCrawl, Spec: []byte(`{}`),
	})

	err = s.UpdateChain(ctx, chain)
	if err != nil {
		t.Fatalf("UpdateChain failed: %v", err)
	}

	// Verify update
	got, err := s.GetChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("GetChain failed: %v", err)
	}

	if got.Name != "Updated Name" {
		t.Errorf("Name not updated: got %s, want %s", got.Name, "Updated Name")
	}
	if got.Description != "Updated description" {
		t.Errorf("Description not updated: got %s, want %s", got.Description, "Updated description")
	}
	if len(got.Definition.Nodes) != 2 {
		t.Errorf("Nodes not updated: got %d, want 2", len(got.Definition.Nodes))
	}
}

func TestDeleteChain(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	chain := model.JobChain{
		ID:   "test-chain-4",
		Name: "Chain To Delete",
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{
				{ID: "node1", Kind: model.KindScrape, Spec: []byte(`{}`)},
			},
			Edges: []model.ChainEdge{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.CreateChain(ctx, chain)
	if err != nil {
		t.Fatalf("CreateChain failed: %v", err)
	}

	// Delete chain
	err = s.DeleteChain(ctx, chain.ID)
	if err != nil {
		t.Fatalf("DeleteChain failed: %v", err)
	}

	// Verify deletion
	_, err = s.GetChain(ctx, chain.ID)
	if err == nil {
		t.Error("Expected error after deleting chain")
	}
}

func TestListChains(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	// Create multiple chains
	chains := []model.JobChain{
		{
			ID:   "chain-a",
			Name: "Alpha Chain",
			Definition: model.ChainDefinition{
				Nodes: []model.ChainNode{{ID: "n1", Kind: model.KindScrape, Spec: []byte(`{}`)}},
				Edges: []model.ChainEdge{},
			},
			CreatedAt: time.Now().Add(-2 * time.Hour),
			UpdatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:   "chain-b",
			Name: "Beta Chain",
			Definition: model.ChainDefinition{
				Nodes: []model.ChainNode{{ID: "n2", Kind: model.KindCrawl, Spec: []byte(`{}`)}},
				Edges: []model.ChainEdge{},
			},
			CreatedAt: time.Now().Add(-1 * time.Hour),
			UpdatedAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:   "chain-c",
			Name: "Gamma Chain",
			Definition: model.ChainDefinition{
				Nodes: []model.ChainNode{{ID: "n3", Kind: model.KindResearch, Spec: []byte(`{}`)}},
				Edges: []model.ChainEdge{},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, chain := range chains {
		err := s.CreateChain(ctx, chain)
		if err != nil {
			t.Fatalf("CreateChain failed: %v", err)
		}
	}

	// List chains
	got, err := s.ListChains(ctx)
	if err != nil {
		t.Fatalf("ListChains failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("Expected 3 chains, got %d", len(got))
	}

	// Verify ordering (newest first)
	if len(got) >= 3 {
		if got[0].ID != "chain-c" {
			t.Errorf("Expected first chain to be chain-c, got %s", got[0].ID)
		}
		if got[2].ID != "chain-a" {
			t.Errorf("Expected last chain to be chain-a, got %s", got[2].ID)
		}
	}
}

func TestCreateDuplicateChainName(t *testing.T) {
	dataDir := t.TempDir()
	s, err := Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer s.Close()

	ctx := context.Background()

	chain1 := model.JobChain{
		ID:   "chain-1",
		Name: "Duplicate Name",
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{{ID: "n1", Kind: model.KindScrape, Spec: []byte(`{}`)}},
			Edges: []model.ChainEdge{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	chain2 := model.JobChain{
		ID:   "chain-2",
		Name: "Duplicate Name", // Same name
		Definition: model.ChainDefinition{
			Nodes: []model.ChainNode{{ID: "n2", Kind: model.KindCrawl, Spec: []byte(`{}`)}},
			Edges: []model.ChainEdge{},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.CreateChain(ctx, chain1)
	if err != nil {
		t.Fatalf("CreateChain first chain failed: %v", err)
	}

	// Second chain with same name should fail
	err = s.CreateChain(ctx, chain2)
	if err == nil {
		t.Error("Expected error for duplicate chain name")
	}
}
