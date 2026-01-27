// Package server contains long-running CLI services (server/mcp/health/tui).
//
// It does NOT define API routes or scheduler logic; internal/api and internal/scheduler do.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func RunServer(ctx context.Context, cfg config.Config, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h" || args[0] == "help") {
		// preserve behavior: server help returns 0
		_, _ = os.Stderr.WriteString(`Usage:
  spartan server

Notes:
  Starts API server, job workers, and scheduler.
`)
		return 0
	}

	serverCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		slog.Error("failed to open store", "error", err)
		return 1
	}
	defer st.Close()

	manager := common.InitJobManager(serverCtx, cfg, st)
	go func() {
		if err := scheduler.Run(serverCtx, cfg.DataDir, manager); err != nil {
			slog.Error("scheduler error", "error", err)
		}
	}()

	apiServer := api.NewServer(manager, st, cfg)
	httpServer := newHTTPServer(cfg, apiServer.Routes())

	go func() {
		slog.Info("Spartan server listening", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
			stop()
		}
	}()

	<-serverCtx.Done()
	slog.Info("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown error", "error", err)
	}

	slog.Info("waiting for job workers to finish...")
	waitCh := make(chan struct{})
	go func() {
		manager.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		slog.Info("all workers finished")
	case <-shutdownCtx.Done():
		slog.Warn("timed out waiting for workers to finish")
	}
	slog.Info("shutdown complete")
	return 0
}

func newHTTPServer(cfg config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    cfg.BindAddr + ":" + cfg.Port,
		Handler: handler,

		// Timeouts mitigate slowloris/resource-exhaustion attacks.
		ReadHeaderTimeout: time.Duration(cfg.ServerReadHeaderTimeoutSecs) * time.Second,
		ReadTimeout:       time.Duration(cfg.ServerReadTimeoutSecs) * time.Second,
		WriteTimeout:      time.Duration(cfg.ServerWriteTimeoutSecs) * time.Second,
		IdleTimeout:       time.Duration(cfg.ServerIdleTimeoutSecs) * time.Second,
	}
}
