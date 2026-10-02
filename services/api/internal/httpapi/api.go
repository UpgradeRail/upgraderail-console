// Package httpapi exposes versioned Console read endpoints.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/auth"
	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/jackc/pgx/v5"
)

type Repository interface {
	Ping(context.Context) error
	ListControllers(context.Context, store.Page) ([]store.Controller, error)
	GetController(context.Context, string) (store.Controller, error)
	ListFleets(context.Context, store.Page) ([]store.Fleet, error)
	GetFleet(context.Context, string) (store.Fleet, error)
	ListFleetUpgrades(context.Context, string, store.Page) ([]store.FleetUpgrade, error)
	ListProposals(context.Context, store.Page) ([]store.Proposal, error)
	GetProposal(context.Context, string) (store.Proposal, error)
	ListApprovals(context.Context, string, store.Page) ([]store.Approval, error)
	CreateChallenge(context.Context, auth.Challenge) error
	GetChallenge(context.Context, string) (auth.StoredChallenge, error)
	ConsumeChallenge(context.Context, string, string, time.Time) (bool, error)
	CreateSession(context.Context, string, string, string, string, time.Time) error
	RevokeSession(context.Context, string, time.Time) error
}

func New(repository Repository, domain string) http.Handler {
	mux := http.NewServeMux()
	auth.Register(mux, repository, domain)
	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]string{"status": "live"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, request *http.Request) {
		if err := repository.Ping(request.Context()); err != nil {
			errorResponse(w, http.StatusServiceUnavailable, "dependency_unavailable", "database is unavailable")
			return
		}
		write(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1/controllers", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListControllers(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/controllers/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetController(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/v1/fleets", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListFleets(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/fleets/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetFleet(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/v1/fleets/{id}/upgrades", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListFleetUpgrades(r.Context(), r.PathValue("id"), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/proposals", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListProposals(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/proposals/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetProposal(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/v1/proposals/{id}/approvals", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListApprovals(r.Context(), r.PathValue("id"), page)
		respond(w, values, err)
	})
	return mux
}

func parsePage(w http.ResponseWriter, request *http.Request) (store.Page, bool) {
	limit, err := integer(request.URL.Query().Get("limit"))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid_limit", "limit must be an integer")
		return store.Page{}, false
	}
	offset, err := integer(request.URL.Query().Get("offset"))
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid_offset", "offset must be an integer")
		return store.Page{}, false
	}
	page, err := store.NewPage(limit, offset)
	if err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid_pagination", err.Error())
		return store.Page{}, false
	}
	return page, true
}

func integer(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func respond(w http.ResponseWriter, value any, err error) {
	if err == nil {
		write(w, http.StatusOK, value)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		errorResponse(w, http.StatusNotFound, "not_found", "resource was not found")
		return
	}
	errorResponse(w, http.StatusInternalServerError, "internal_error", "request could not be completed")
}

func errorResponse(w http.ResponseWriter, status int, code, message string) {
	write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
