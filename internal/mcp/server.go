// MCP server lifecycle and request routing.
//
// Responsibilities:
// - Initialize server with store and job manager
// - Handle JSON-RPC 2.0 requests over stdio
// - Route initialize, tools/list, and tools/call methods
//
// Does NOT handle:
// - Tool execution logic (delegated to handlers.go)
// - Job execution or worker pool management
//
// Invariants:
// - Server context is independent of Serve context (for graceful shutdown)
// - Requests are line-delimited JSON over stdin/stdout
// - Manager is started immediately and stopped on Close
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	appRuntime "github.com/fitchmultz/spartan-scraper/internal/runtime"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func NewServer(cfg config.Config) (*Server, error) {
	st, err := store.Open(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	mgr, err := appRuntime.InitJobManager(ctx, cfg, st)
	if err != nil {
		_ = st.Close()
		cancel()
		return nil, err
	}
	aiExtractor, err := extract.NewAIExtractor(cfg.AI)
	if err != nil {
		slog.Warn("failed to initialize AI extractor for MCP authoring tools", "error", err)
	}
	return &Server{
		store:       st,
		manager:     mgr,
		cfg:         cfg,
		aiAuthoring: aiauthoring.NewService(cfg, aiExtractor, true),
		ctx:         ctx,
		cancel:      cancel,
	}, nil
}

func (s *Server) Close() error {
	if s.cancel != nil {
		s.cancel()
	}

	if s.manager != nil {
		s.manager.Wait()
	}

	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Server) Serve(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var base map[string]json.RawMessage
		if err := json.Unmarshal(line, &base); err != nil {
			_ = encoder.Encode(response{ID: nil, Error: &rpcError{Code: -32700, Message: "parse error"}})
			continue
		}

		var method string
		if raw, ok := base["method"]; ok {
			_ = json.Unmarshal(raw, &method)
		}

		var id interface{}
		if raw, ok := base["id"]; ok {
			_ = json.Unmarshal(raw, &id)
		}

		switch method {
		case "initialize":
			_ = encoder.Encode(response{ID: id, Result: map[string]interface{}{
				"name":    "spartan-scraper-mcp",
				"version": buildinfo.Version,
			}})
		case "tools/list":
			_ = encoder.Encode(response{ID: id, Result: map[string]interface{}{"tools": s.toolsList()}})
		case "tools/call":
			result, err := s.handleToolCall(ctx, base)
			if err != nil {
				_ = encoder.Encode(response{ID: id, Error: &rpcError{Code: -32000, Message: apperrors.SafeMessage(err)}})
				continue
			}
			_ = encoder.Encode(response{ID: id, Result: result})
		default:
			_ = encoder.Encode(response{ID: id, Error: &rpcError{Code: -32601, Message: "method not found"}})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}
