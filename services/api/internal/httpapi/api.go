// Package httpapi exposes versioned Console read endpoints.
package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/auth"
	"github.com/UpgradeRail/upgraderail-console/services/api/internal/store"
	"github.com/UpgradeRail/upgraderail-console/services/shared/artifactstore"
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
	ListAllFleetUpgrades(context.Context, store.Page) ([]store.FleetUpgrade, error)
	ListEvents(context.Context, store.Page) ([]store.ControllerEvent, error)
	CreateChallenge(context.Context, auth.Challenge) error
	GetChallenge(context.Context, string) (auth.StoredChallenge, error)
	ConsumeChallenge(context.Context, string, string, time.Time) (bool, error)
	CreateSession(context.Context, string, string, string, string, time.Time) error
	RevokeSession(context.Context, string, time.Time) error
	GetSession(context.Context, string) (store.Session, error)
	CreateAnalysis(context.Context, store.AnalysisInput) error
	ListAnalyses(context.Context, store.Page) ([]store.AnalysisJob, error)
	GetAnalysis(context.Context, string) (store.AnalysisJob, error)
	GetAnalysisReport(context.Context, string) (store.AnalysisReport, error)
	GetManifest(context.Context, string) (store.Manifest, error)
	CreateArtifact(context.Context, store.ArtifactInput) (store.Artifact, error)
	GetArtifact(context.Context, string) (store.Artifact, error)
}

func New(repository Repository, domain, origin string, artifacts artifactstore.Filesystem) http.Handler {
	mux := http.NewServeMux()
	auth.Register(mux, repository, domain, origin)
	registerArtifactRoutes(mux, repository, artifacts)
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
	mux.HandleFunc("GET /api/v1/upgrades", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListAllFleetUpgrades(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListEvents(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("POST /api/v1/analyses", func(w http.ResponseWriter, request *http.Request) {
		session, ok := requireSession(w, request, repository)
		if !ok {
			return
		}
		var input struct {
			Network             string `json:"network"`
			CurrentArtifactID   string `json:"current_artifact_id"`
			CandidateArtifactID string `json:"candidate_artifact_id"`
		}
		if !decode(request, &input) {
			errorResponse(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
			return
		}
		if input.Network == "" || input.CurrentArtifactID == "" || input.CandidateArtifactID == "" {
			errorResponse(w, http.StatusBadRequest, "invalid_analysis", "Network, current artifact, and candidate artifact are required.")
			return
		}
		if input.Network != session.Network {
			errorResponse(w, http.StatusForbidden, "network_mismatch", "Session does not authorize this network.")
			return
		}
		if _, err := verifyArtifactExists(request.Context(), repository, artifacts, input.CurrentArtifactID); err != nil {
			errorResponse(w, http.StatusBadRequest, "artifact_not_found", "The current artifact does not exist or its stored bytes failed verification.")
			return
		}
		if _, err := verifyArtifactExists(request.Context(), repository, artifacts, input.CandidateArtifactID); err != nil {
			errorResponse(w, http.StatusBadRequest, "artifact_not_found", "The candidate artifact does not exist or its stored bytes failed verification.")
			return
		}
		id := randomID()
		err := repository.CreateAnalysis(request.Context(), store.AnalysisInput{ID: id, Network: input.Network, CurrentArtifactID: input.CurrentArtifactID, CandidateArtifactID: input.CandidateArtifactID, CreatedBy: session.Address})
		if err != nil {
			errorResponse(w, http.StatusInternalServerError, "analysis_unavailable", "Analysis job could not be created.")
			return
		}
		write(w, http.StatusAccepted, map[string]string{"id": id, "status": "queued"})
	})
	mux.HandleFunc("GET /api/v1/analyses", func(w http.ResponseWriter, r *http.Request) {
		page, ok := parsePage(w, r)
		if !ok {
			return
		}
		values, err := repository.ListAnalyses(r.Context(), page)
		respond(w, values, err)
	})
	mux.HandleFunc("GET /api/v1/analyses/{id}", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetAnalysis(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/v1/analyses/{id}/report", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetAnalysisReport(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	mux.HandleFunc("GET /api/v1/analyses/{id}/manifest", func(w http.ResponseWriter, r *http.Request) {
		value, err := repository.GetManifest(r.Context(), r.PathValue("id"))
		respond(w, value, err)
	})
	return withCORS(mux, origin)
}

func requireSession(w http.ResponseWriter, request *http.Request, repository Repository) (store.Session, bool) {
	cookie, err := request.Cookie("upgraderail_session")
	if err != nil {
		errorResponse(w, http.StatusUnauthorized, "session_required", "Sign in with a wallet before creating an analysis.")
		return store.Session{}, false
	}
	session, err := repository.GetSession(request.Context(), auth.Hash(cookie.Value))
	if errors.Is(err, pgx.ErrNoRows) {
		errorResponse(w, http.StatusUnauthorized, "session_expired", "Your session is no longer active.")
		return store.Session{}, false
	}
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "session_unavailable", "Session could not be verified.")
		return store.Session{}, false
	}
	return session, true
}
func randomID() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		panic("cryptographic randomness unavailable")
	}
	return hex.EncodeToString(bytes)
}
func decode(request *http.Request, output any) bool {
	return json.NewDecoder(io.LimitReader(request.Body, 16<<10)).Decode(output) == nil
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
