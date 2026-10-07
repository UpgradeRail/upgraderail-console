package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// serveHealth exposes a minimal liveness endpoint when PORT is set, so the
// process can run on hosts that require a bound port (for example a free
// web-service tier) and be probed by an external monitor. It carries no
// data beyond the service name. Without PORT nothing is listened on.
func serveHealth(ctx context.Context) {
	port := os.Getenv("PORT")
	if port == "" {
		return
	}
	server := &http.Server{Addr: ":" + port, Handler: healthHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("health listening", "service", "worker", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health listener stopped", "service", "worker", "error", err)
		}
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
}

func healthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "worker"})
	})
	return mux
}
