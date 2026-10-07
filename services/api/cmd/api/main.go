package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/httpapi"
	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
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
	origin := os.Getenv("WEB_ORIGIN")
	parsedOrigin, err := url.Parse(origin)
	if err != nil || (parsedOrigin.Scheme != "https" && parsedOrigin.Scheme != "http") || parsedOrigin.Host == "" || parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		logger.Error("WEB_ORIGIN must be an absolute web origin", "service", "api")
		os.Exit(1)
	}
	domain := os.Getenv("AUTH_DOMAIN")
	if domain == "" || domain != parsedOrigin.Hostname() {
		logger.Error("AUTH_DOMAIN must match WEB_ORIGIN hostname", "service", "api")
		os.Exit(1)
	}
	artifactRoot := absoluteEnv("ARTIFACT_LOCAL_DIR", "./artifacts")
	artifacts := artifactstore.Filesystem{Directory: artifactRoot, MaxBytes: maxArtifactBytes()}

	server := &http.Server{
		Addr:              address(),
		Handler:           httpapi.New(store, domain, origin, artifacts),
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

func absoluteEnv(key, fallback string) string {
	value := fallback
	if raw := os.Getenv(key); raw != "" {
		value = raw
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		panic(err)
	}
	return absolute
}

// maxArtifactBytes is the storage-layer size limit. It defaults to the
// same limit the HTTP handler enforces at the request layer; ARTIFACT_MAX_BYTES
// can lower it further but never needs to raise it past what the handler
// already rejects.
func maxArtifactBytes() int64 {
	raw := os.Getenv("ARTIFACT_MAX_BYTES")
	if raw == "" {
		return httpapi.MaxArtifactUploadBytes
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return httpapi.MaxArtifactUploadBytes
	}
	return value
}
