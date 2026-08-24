package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/local/printforge/apps/backend/internal/config"
	"github.com/local/printforge/apps/backend/internal/database"
	httpapi "github.com/local/printforge/apps/backend/internal/http"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := database.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
	retryDeadline := time.Now().Add(60 * time.Second)
	for err != nil && time.Now().Before(retryDeadline) {
		slog.Warn("database is not ready; retrying", "error", err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
			pool, err = database.Open(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConns)
		}
	}
	if err != nil {
		slog.Error("database connection failed after retries", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	api := httpapi.New(cfg, pool)
	if err := api.BackfillModelAnalysis(ctx); err != nil {
		slog.Warn("some stored models could not be analysed", "error", err)
	}
	if err := api.BackfillPrinterCatalog(ctx); err != nil {
		slog.Warn("some printers could not be matched to the catalog", "error", err)
	}
	api.StartTelegramBot(ctx)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("PrintForge API started", "port", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server stopped unexpectedly", "error", err)
			cancel()
		}
	}()

	<-ctx.Done()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}
}
