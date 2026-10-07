package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/engine"
	"github.com/UpgradeRail/upgraderail-console/services/worker/internal/jobs"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required", "service", "worker")
		os.Exit(1)
	}
	artifacts, err := artifactstore.FromEnv(1 << 62)
	if err != nil {
		logger.Error("artifact storage configuration invalid", "service", "worker", "error", err)
		os.Exit(1)
	}
	workspaceRoot := absoluteEnv("UPGRADERAIL_WORK_DIR", "./tmp")
	if err := os.MkdirAll(workspaceRoot, 0700); err != nil {
		logger.Error("create workspace root", "service", "worker", "error", err)
		os.Exit(1)
	}
	store, err := jobs.Open(context.Background(), databaseURL)
	if err != nil {
		logger.Error("database startup failed", "service", "worker", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	serveHealth(ctx)
	runner := engine.Runner{Binary: env("UPGRADERAIL_ENGINE_BIN", "upgraderail"), Timeout: 2 * time.Minute}
	for ctx.Err() == nil {
		ran, err := jobs.RunOne(ctx, store, runner, artifacts, workspaceRoot)
		if err != nil {
			logger.Error("analysis job failed", "service", "worker", "error", err)
		}
		if !ran {
			_ = store.Wait(ctx, time.Second)
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func absoluteEnv(key, fallback string) string {
	value := env(key, fallback)
	absolute, err := filepath.Abs(value)
	if err != nil {
		panic(err)
	}
	return absolute
}
