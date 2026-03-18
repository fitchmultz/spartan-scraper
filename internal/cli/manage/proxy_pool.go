// Package manage contains proxy-pool inspection CLI wiring.
//
// Purpose:
// - Present proxy-pool status with capability-aware guidance in the CLI.
//
// Responsibilities:
// - Fetch proxy-pool status from the local server when available, or inspect local runtime state as a fallback.
// - Explain whether proxy pooling is disabled, degraded, or ready before dumping low-level pool details.
// - Keep operator-facing recovery actions aligned with shared health diagnostics.
//
// Scope:
// - Proxy-pool CLI status rendering only.
//
// Usage:
// - Called by `spartan proxy-pool status`.
//
// Invariants/Assumptions:
// - Missing proxy-pool configuration is a valid optional state.
// - Detailed proxy stats are helpful only after the capability state is clear.
package manage

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

const proxyPoolCommandHelpText = `Inspect proxy-pool configuration and runtime status.

Usage: spartan proxy-pool <command> [options]

Commands:
  status              Show proxy-pool status from the local server when available,
                      otherwise validate and inspect the local configured pool.

Examples:
  spartan proxy-pool status
`

func RunProxyPool(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) < 1 {
		printProxyPoolHelp()
		return 1
	}

	switch args[0] {
	case "status":
		return runProxyPoolStatus(ctx, cfg, args[1:])
	case "help", "--help", "-h":
		printProxyPoolHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown proxy-pool subcommand: %s\n", args[0])
		printProxyPoolHelp()
		return 1
	}
}

func runProxyPoolStatus(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("proxy-pool status", flag.ExitOnError)
	_ = fs.Parse(args)

	if status, ok := fetchProxyPoolStatusFromServer(ctx, cfg); ok {
		printProxyPoolStatus(status, true, cfg.ProxyPoolFile)
		return 0
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer st.Close()

	managerCtx, cancel := context.WithCancel(ctx)
	manager, err := common.InitJobManager(managerCtx, cfg, st)
	if err != nil {
		cancel()
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer func() {
		cancel()
		manager.Wait()
	}()

	status := api.BuildProxyPoolStatusResponse(manager.GetProxyPool())
	printProxyPoolStatus(status, false, cfg.ProxyPoolFile)
	return 0
}

func fetchProxyPoolStatusFromServer(ctx context.Context, cfg config.Config) (api.ProxyPoolStatusResponse, bool) {
	url := fmt.Sprintf("http://localhost:%s/v1/proxy-pool/status", cfg.Port)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return api.ProxyPoolStatusResponse{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return api.ProxyPoolStatusResponse{}, false
	}
	var status api.ProxyPoolStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return api.ProxyPoolStatusResponse{}, false
	}
	return status, true
}

func printProxyPoolStatus(status api.ProxyPoolStatusResponse, fromServer bool, configPath string) {
	source := "local configuration"
	if fromServer {
		source = "running server"
	}

	component := api.BuildProxyPoolComponentStatus(
		config.Config{ProxyPoolFile: configPath},
		proxyPoolRuntimeStateFromStatus(status),
	)

	fmt.Printf("Proxy Pool Status (%s)\n", source)
	fmt.Printf("Capability: %s\n", strings.ToUpper(strings.ReplaceAll(component.Status, "_", " ")))
	if message := strings.TrimSpace(component.Message); message != "" {
		fmt.Printf("%s\n", message)
	}
	renderRecommendedActions(component.Actions, "spartan")
	fmt.Println()
	fmt.Printf("  Config Path:      %s\n", displayOr(configPath, "disabled"))
	fmt.Printf("  Strategy:         %s\n", status.Strategy)
	fmt.Printf("  Total Proxies:    %d\n", status.TotalProxies)
	fmt.Printf("  Healthy Proxies:  %d\n", status.HealthyProxies)
	fmt.Printf("  Regions:          %s\n", displaySlice(status.Regions))
	fmt.Printf("  Tags:             %s\n", displaySlice(status.Tags))
	if len(status.Proxies) == 0 {
		fmt.Println()
		fmt.Println("No proxy pool is currently loaded.")
		return
	}
	fmt.Println()
	fmt.Println("Proxies:")
	for _, proxy := range status.Proxies {
		fmt.Printf("  - %s\n", proxy.ID)
		fmt.Printf("      Region:            %s\n", displayOr(proxy.Region, "n/a"))
		fmt.Printf("      Tags:              %s\n", displaySlice(proxy.Tags))
		fmt.Printf("      Healthy:           %t\n", proxy.IsHealthy)
		fmt.Printf("      Requests:          %d\n", proxy.RequestCount)
		fmt.Printf("      Successes:         %d\n", proxy.SuccessCount)
		fmt.Printf("      Failures:          %d\n", proxy.FailureCount)
		fmt.Printf("      Success Rate:      %.2f%%\n", proxy.SuccessRate)
		fmt.Printf("      Avg Latency (ms):  %d\n", proxy.AvgLatencyMs)
		fmt.Printf("      Consecutive Fails: %d\n", proxy.ConsecutiveFails)
	}
}

func proxyPoolRuntimeStateFromStatus(status api.ProxyPoolStatusResponse) api.ProxyPoolRuntimeState {
	if status.TotalProxies > 0 {
		return api.ProxyPoolRuntimeLoaded
	}
	return api.ProxyPoolRuntimeUnloaded
}

func renderRecommendedActions(actions []api.RecommendedAction, commandName string) {
	translated := api.CLIRecommendedActions(actions, commandName)
	for _, action := range translated {
		label := strings.TrimSpace(action.Label)
		value := strings.TrimSpace(action.Value)
		switch {
		case label == "" && value == "":
			continue
		case value == "":
			fmt.Printf("Next step: %s\n", label)
		case label == "":
			fmt.Printf("Next step: %s\n", value)
		default:
			fmt.Printf("Next step: %s: %s\n", label, value)
		}
	}
}

func displayOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func displaySlice(values []string) string {
	if len(values) == 0 {
		return "n/a"
	}
	return strings.Join(values, ", ")
}

func printProxyPoolHelp() {
	fmt.Fprint(os.Stderr, proxyPoolCommandHelpText)
}
