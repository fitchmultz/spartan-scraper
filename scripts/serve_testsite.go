// Command serve_testsite runs the deterministic local fixture server used by stress and e2e validation.
//
// Purpose:
//   - Expose the shared testsite handler on a real loopback port for shell-driven test flows.
//
// Responsibilities:
//   - Parse CLI flags.
//   - Start the fixture HTTP server.
//   - Print a useful help menu with examples.
//
// Scope:
//   - Local test infrastructure only.
//
// Usage:
//   - go run ./scripts/serve_testsite.go --addr 127.0.0.1:8765
//
// Invariants/Assumptions:
//   - Uses only loopback bindings unless the caller overrides --addr.
//   - Exits non-zero on listen failures.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "127.0.0.1:8765", "listen address for the local fixture server")
	flag.Usage = func() {
		_, _ = fmt.Fprint(flag.CommandLine.Output(), `Usage: go run ./scripts/serve_testsite.go [options]

Run the deterministic local fixture server used by Spartan Scraper stress and end-to-end checks.

Options:
  --addr <host:port>   Listen address (default: 127.0.0.1:8765)
  -h, --help           Show this help and exit

Examples:
  go run ./scripts/serve_testsite.go
  go run ./scripts/serve_testsite.go --addr 127.0.0.1:8899

Exit codes:
  0  Clean shutdown
  1  Runtime failure while starting or serving
  2  Usage error from flag parsing
`)
	}
	flag.Parse()

	server := &http.Server{
		Addr:    addr,
		Handler: testsite.NewHandler(),
	}

	log.Printf("starting deterministic test fixture on http://%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		_, _ = fmt.Fprintf(os.Stderr, "serve_testsite: %v\n", err)
		os.Exit(1)
	}
}
