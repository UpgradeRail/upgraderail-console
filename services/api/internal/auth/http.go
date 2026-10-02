package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type Repository interface {
	CreateChallenge(context.Context, Challenge) error
	GetChallenge(context.Context, string) (StoredChallenge, error)
	ConsumeChallenge(context.Context, string, string, time.Time) (bool, error)
	CreateSession(context.Context, string, string, string, string, time.Time) error
	RevokeSession(context.Context, string, time.Time) error
}

func Register(mux *http.ServeMux, repository Repository, domain string) {
	mux.HandleFunc("POST /api/v1/auth/challenge", func(w http.ResponseWriter, request *http.Request) {
		var input struct{ Network, Address, Purpose string }
		if !decodeJSON(w, request, &input) {
			return
		}
		challenge, err := NewChallenge(randomID(), domain, input.Network, input.Address, input.Purpose, time.Now())
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_challenge", err.Error())
			return
		}
		if err := repository.CreateChallenge(request.Context(), challenge); err != nil {
			writeError(w, http.StatusInternalServerError, "challenge_unavailable", "Could not create challenge.")
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"id": challenge.ID, "message": challenge.Message(), "nonce": challenge.Nonce, "issued_at": challenge.IssuedAt, "expires_at": challenge.ExpiresAt})
	})
	mux.HandleFunc("POST /api/v1/auth/verify", func(w http.ResponseWriter, request *http.Request) {
		var input struct{ ChallengeID, Nonce, Signature string }
		if !decodeJSON(w, request, &input) {
			return
		}
		stored, err := repository.GetChallenge(request.Context(), input.ChallengeID)
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "challenge_not_found", "Challenge was not found.")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "challenge_unavailable", "Could not load challenge.")
			return
		}
		if stored.UsedAt != nil || !stored.ExpiresAt.After(time.Now()) || Hash(input.Nonce) != stored.NonceHash {
			writeError(w, http.StatusUnauthorized, "challenge_invalid", "Challenge is expired or has already been used.")
			return
		}
		challenge := Challenge{ID: stored.ID, Domain: stored.Domain, Network: stored.Network, Address: stored.Address, Nonce: input.Nonce, Purpose: stored.Purpose, IssuedAt: stored.IssuedAt, ExpiresAt: stored.ExpiresAt}
		if err := Verify(challenge.Address, challenge.Message(), input.Signature); err != nil {
			writeError(w, http.StatusUnauthorized, "signature_invalid", "Wallet signature could not be verified.")
			return
		}
		now := time.Now()
		used, err := repository.ConsumeChallenge(request.Context(), stored.ID, stored.NonceHash, now)
		if err != nil || !used {
			writeError(w, http.StatusUnauthorized, "challenge_used", "Challenge could not be used.")
			return
		}
		token := randomID()
		expires := now.Add(12 * time.Hour)
		if err := repository.CreateSession(request.Context(), randomID(), stored.Address, stored.Network, Hash(token), expires); err != nil {
			writeError(w, http.StatusInternalServerError, "session_unavailable", "Could not create session.")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "upgraderail_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, Expires: expires})
		writeJSON(w, http.StatusOK, map[string]any{"address": stored.Address, "network": stored.Network, "expires_at": expires})
	})
	mux.HandleFunc("POST /api/v1/auth/logout", func(w http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("upgraderail_session")
		if err == nil {
			_ = repository.RevokeSession(request.Context(), Hash(cookie.Value), time.Now())
		}
		http.SetCookie(w, &http.Cookie{Name: "upgraderail_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
		writeJSON(w, http.StatusNoContent, nil)
	})
}

func randomID() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		panic("cryptographic randomness unavailable")
	}
	return hex.EncodeToString(bytes)
}
func decodeJSON(w http.ResponseWriter, request *http.Request, output any) bool {
	request.Body = http.MaxBytesReader(w, request.Body, 16<<10)
	if err := json.NewDecoder(request.Body).Decode(output); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return false
	}
	return true
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if value != nil {
		_ = json.NewEncoder(w).Encode(value)
	}
}
