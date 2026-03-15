// Package manage contains chain management CLI commands.
//
// It does NOT implement chain execution; internal/jobs and internal/api do.
package manage

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func RunChains(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printChainsHelp()
		return 1
	}
	if isHelpToken(args[0]) {
		printChainsHelp()
		return 0
	}

	switch args[0] {
	case "list":
		return runChainsList(ctx, cfg, args[1:])
	case "get":
		return runChainsGet(ctx, cfg, args[1:])
	case "create":
		return runChainsCreate(ctx, cfg, args[1:])
	case "submit":
		return runChainsSubmit(ctx, cfg, args[1:])
	case "delete":
		return runChainsDelete(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printChainsHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown chains subcommand: %s\n", args[0])
		printChainsHelp()
		return 1
	}
}

func runChainsList(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("chains list", flag.ExitOnError)
	_ = fs.Parse(args)

	// Check if server is running
	if !isServerRunning(ctx, cfg.Port) {
		fmt.Fprintln(os.Stderr, "server is not running. Start it with: spartan server")
		return 1
	}

	url := fmt.Sprintf("http://localhost:%s/v1/chains", cfg.Port)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to list chains: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d\n", resp.StatusCode)
		}
		return 1
	}

	var result struct {
		Chains []struct {
			ID          string    `json:"id"`
			Name        string    `json:"name"`
			Description string    `json:"description"`
			CreatedAt   time.Time `json:"createdAt"`
		} `json:"chains"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode response: %v\n", err)
		return 1
	}

	if len(result.Chains) == 0 {
		fmt.Println("No chains found.")
		return 0
	}

	// Tabular output: ID, Name, Description, CreatedAt
	fmt.Printf("%-12s  %-20s  %-30s  %s\n", "ID", "NAME", "DESCRIPTION", "CREATED")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, chain := range result.Chains {
		desc := chain.Description
		if len(desc) > 28 {
			desc = desc[:25] + "..."
		}
		name := chain.Name
		if len(name) > 18 {
			name = name[:15] + "..."
		}
		fmt.Printf("%-12s  %-20s  %-30s  %s\n",
			truncateString(chain.ID, 12),
			name,
			desc,
			chain.CreatedAt.Format(time.RFC3339))
	}
	return 0
}

func truncateString(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

func runChainsGet(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "chain id is required")
		return 1
	}
	id := args[0]

	// Check if server is running
	if !isServerRunning(ctx, cfg.Port) {
		fmt.Fprintln(os.Stderr, "server is not running. Start it with: spartan server")
		return 1
	}

	url := fmt.Sprintf("http://localhost:%s/v1/chains/%s", cfg.Port, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get chain: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "chain not found: %s\n", id)
		return 1
	}
	if resp.StatusCode != http.StatusOK {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d\n", resp.StatusCode)
		}
		return 1
	}

	// Output JSON
	var chain interface{}
	if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode response: %v\n", err)
		return 1
	}

	payload, _ := json.MarshalIndent(chain, "", "  ")
	fmt.Println(string(payload))
	return 0
}

func runChainsCreate(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("chains create", flag.ExitOnError)
	filePath := fs.String("file", "", "Path to JSON file containing chain definition (if not provided, reads from stdin)")
	_ = fs.Parse(args)

	// Check if server is running
	if !isServerRunning(ctx, cfg.Port) {
		fmt.Fprintln(os.Stderr, "server is not running. Start it with: spartan server")
		return 1
	}

	// Read chain definition
	var reader io.Reader = os.Stdin
	if *filePath != "" {
		file, err := os.Open(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to open file: %v\n", err)
			return 1
		}
		defer file.Close()
		reader = file
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read input: %v\n", err)
		return 1
	}

	// Validate JSON
	var req api.ChainCreateRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fmt.Fprintf(os.Stderr, "invalid JSON: %v\n", err)
		return 1
	}

	if req.Name == "" {
		fmt.Fprintln(os.Stderr, "name is required in chain definition")
		return 1
	}

	url := fmt.Sprintf("http://localhost:%s/v1/chains", cfg.Port)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create chain: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "validation error: %s\n", errResp.Error)
		} else {
			fmt.Fprintln(os.Stderr, "validation error")
		}
		return 1
	}
	if resp.StatusCode != http.StatusCreated {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d\n", resp.StatusCode)
		}
		return 1
	}

	var chain struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chain); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode response: %v\n", err)
		return 1
	}

	fmt.Println("created", chain.ID)
	return 0
}

func runChainsSubmit(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "chain id is required")
		return 1
	}
	id := args[0]

	fs := flag.NewFlagSet("chains submit", flag.ExitOnError)
	overridesPath := fs.String("overrides", "", "Path to JSON file containing parameter overrides by node ID")
	_ = fs.Parse(args[1:])

	// Check if server is running
	if !isServerRunning(ctx, cfg.Port) {
		fmt.Fprintln(os.Stderr, "server is not running. Start it with: spartan server")
		return 1
	}

	// Build submit request
	submitReq := api.ChainSubmitRequest{}
	if *overridesPath != "" {
		data, err := os.ReadFile(*overridesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read overrides file: %v\n", err)
			return 1
		}
		if err := json.Unmarshal(data, &submitReq.Overrides); err != nil {
			fmt.Fprintf(os.Stderr, "invalid overrides JSON: %v\n", err)
			return 1
		}
	}

	reqBody, err := json.Marshal(submitReq)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to encode request: %v\n", err)
		return 1
	}

	url := fmt.Sprintf("http://localhost:%s/v1/chains/%s/submit", cfg.Port, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to submit chain: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "chain not found: %s\n", id)
		return 1
	}
	if resp.StatusCode == http.StatusBadRequest {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "validation error: %s\n", errResp.Error)
		} else {
			fmt.Fprintln(os.Stderr, "validation error")
		}
		return 1
	}
	if resp.StatusCode != http.StatusCreated {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d\n", resp.StatusCode)
		}
		return 1
	}

	var result struct {
		Jobs []struct {
			ID string `json:"id"`
		} `json:"jobs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "failed to decode response: %v\n", err)
		return 1
	}

	jobIDs := make([]string, len(result.Jobs))
	for i, job := range result.Jobs {
		jobIDs[i] = job.ID
	}
	fmt.Printf("submitted %s, created %d job(s): %v\n", id, len(jobIDs), jobIDs)
	return 0
}

func runChainsDelete(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "chain id is required")
		return 1
	}
	id := args[0]

	// Check if server is running
	if !isServerRunning(ctx, cfg.Port) {
		fmt.Fprintln(os.Stderr, "server is not running. Start it with: spartan server")
		return 1
	}

	url := fmt.Sprintf("http://localhost:%s/v1/chains/%s", cfg.Port, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to delete chain: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		fmt.Fprintf(os.Stderr, "chain not found: %s\n", id)
		return 1
	}
	if resp.StatusCode == http.StatusBadRequest {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "cannot delete: %s\n", errResp.Error)
		} else {
			fmt.Fprintln(os.Stderr, "cannot delete chain (may have active jobs)")
		}
		return 1
	}
	if resp.StatusCode != http.StatusNoContent {
		var errResp api.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			fmt.Fprintf(os.Stderr, "server returned %d: %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "server returned %d\n", resp.StatusCode)
		}
		return 1
	}

	fmt.Println("deleted", id)
	return 0
}

func printChainsHelp() {
	fmt.Print(`Usage:
  spartan chains <subcommand> [options]

Subcommands:
  list    List all chains
  get     Get chain details by ID
  create  Create a new chain from JSON file or stdin
  submit  Submit/instantiate a chain (creates jobs)
  delete  Delete a chain

Examples:
  spartan chains list
  spartan chains get <chain-id>
  spartan chains create --file ./my-chain.json
  cat chain.json | spartan chains create
  spartan chains submit <chain-id>
  spartan chains submit <chain-id> --overrides ./overrides.json
  spartan chains delete <chain-id>

Chain Definition JSON Format:
  {
    "name": "My Chain",
    "description": "Example chain",
    "definition": {
      "nodes": [
        {
          "id": "step1",
          "kind": "scrape",
          "request": { "url": "https://example.com" },
          "metadata": { "name": "Homepage" }
        }
      ],
      "edges": [
        { "from": "step1", "to": "step2" }
      ]
    }
  }
`)
}
