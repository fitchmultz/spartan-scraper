// Package server contains long-running CLI services (server/mcp/health/tui).
//
// Purpose:
// - Start and stop the local HTTP server entrypoint for Spartan.
//
// Responsibilities:
// - Perform startup preflight checks.
// - Boot either the normal API runtime or guided setup mode.
// - Own HTTP listener lifecycle and graceful shutdown behavior.
//
// Scope:
// - Server process orchestration only; route handling and scheduler internals live in sibling packages.
//
// Usage:
// - Invoked from `spartan server` after configuration is loaded.
//
// Invariants/Assumptions:
// - Setup-mode startup should still bind the server so diagnostics stay visible in-product.
// - Graceful shutdown should stop HTTP handling before waiting on background workers.
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

	setupStatus, err := inspectStartupPreflight(cfg, currentCommandName())
	if err != nil {
		slog.Error("failed to inspect data directory", "error", err)
		return 1
	}
	if setupStatus != nil {
		slog.Warn("startup requires operator action; serving setup mode",
			"code", setupStatus.Code,
			"dataDir", setupStatus.DataDir,
		)
		return serveAPIServer(serverCtx, stop, cfg, api.NewSetupServer(cfg, *setupStatus), "Spartan setup mode listening")
	}

	st, err := store.Open(cfg.DataDir)
	if err != nil {
		slog.Error("failed to open store", "error", err)
		return 1
	}
	defer st.Close()

	manager, err := common.InitJobManager(serverCtx, cfg, st)
	if err != nil {
		slog.Error("failed to initialize job manager", "error", err)
		return 1
	}

	apiServer := api.NewServer(manager, st, cfg)

	// Initialize export trigger for automated export scheduling.
	exportStorage := scheduler.NewExportStorage(cfg.DataDir)
	exportHistoryStore := scheduler.NewExportHistoryStore(cfg.DataDir)
	exportTrigger := scheduler.NewExportTrigger(cfg.DataDir, exportStorage, exportHistoryStore, manager, apiServer.WebhookDispatcher())
	apiServer.SetExportScheduleRuntime(exportTrigger)
	if err := exportTrigger.Start(); err != nil {
		slog.Error("failed to start export trigger", "error", err)
		apiServer.Stop()
		return 1
	}
	defer exportTrigger.Stop()

	manager.SetExportTrigger(exportTrigger)

	go func() {
		if err := scheduler.Run(serverCtx, cfg.DataDir, manager, cfg); err != nil {
			slog.Error("scheduler error", "error", err)
		}
	}()

	return serveAPIServer(serverCtx, stop, cfg, apiServer, "Spartan server listening")
}

func serveAPIServer(serverCtx context.Context, stop context.CancelFunc, cfg config.Config, apiServer *api.Server, listenMessage string) int {
	httpServer := newHTTPServer(cfg, apiServer.Routes())

	go func() {
		slog.Info(listenMessage, "addr", httpServer.Addr)
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

	slog.Info("stopping analytics collector...")
	apiServer.Stop()

	if apiServer.Manager() == nil {
		slog.Info("shutdown complete")
		return 0
	}

	slog.Info("waiting for job workers to finish...")
	waitCh := make(chan struct{})
	go func() {
		apiServer.Manager().Wait()
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
