package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/httpapi"
	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required", "service", "api")
		os.Exit(1)
	}
	store, err := store.Open(context.Background(), databaseURL)
	if err != nil {
		logger.Error("database startup failed", "service", "api", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	server := &http.Server{
		Addr:              address(),
		Handler:           httpapi.New(store),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("api listening", "service", "api", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("api stopped", "service", "api", "error", err)
			os.Exit(1)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("api shutdown failed", "service", "api", "error", err)
	}
}

func address() string {
	if value := os.Getenv("API_ADDR"); value != "" {
		return value
	}
	return ":8080"
}
