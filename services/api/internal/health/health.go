// Package health provides separate liveness and readiness handlers.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type Probe interface {
	Ping(context.Context) error
}

func New(probe Probe) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, "live")
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, request *http.Request) {
		if probe != nil {
			ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
			defer cancel()
			if err := probe.Ping(ctx); err != nil {
				write(w, http.StatusServiceUnavailable, "not ready")
				return
			}
		}
		write(w, http.StatusOK, "ready")
	})
	return mux
}

func write(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": value})
}
