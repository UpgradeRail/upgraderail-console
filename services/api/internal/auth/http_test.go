package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type sessionRepository struct {
	session Session
}

func (sessionRepository) CreateChallenge(context.Context, Challenge) error { return nil }
func (sessionRepository) GetChallenge(context.Context, string) (StoredChallenge, error) {
	return StoredChallenge{}, pgx.ErrNoRows
}
func (sessionRepository) ConsumeChallenge(context.Context, string, string, time.Time) (bool, error) {
	return false, nil
}
func (sessionRepository) CreateSession(context.Context, string, string, string, string, time.Time) error {
	return nil
}
func (sessionRepository) RevokeSession(context.Context, string, time.Time) error { return nil }
func (repository sessionRepository) GetSession(_ context.Context, tokenHash string) (Session, error) {
	if tokenHash != Hash("valid-token") {
		return Session{}, pgx.ErrNoRows
	}
	return repository.session, nil
}

func TestSessionEndpointRequiresAndReturnsAuthenticatedCookie(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, sessionRepository{session: Session{Address: "GTEST", Network: "testnet"}}, "localhost", "http://localhost:3000")

	unauthenticated := httptest.NewRecorder()
	mux.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("expected missing session to return 401, got %d", unauthenticated.Code)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: "upgraderail_session", Value: "valid-token"})
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("expected valid session to return 200, got %d", response.Code)
	}
	if got := response.Body.String(); got != "{\"address\":\"GTEST\",\"network\":\"testnet\"}\n" {
		t.Fatalf("unexpected session response: %s", got)
	}

	request = httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	request.AddCookie(&http.Cookie{Name: "upgraderail_session", Value: "expired-token"})
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired session to return 401, got %d", response.Code)
	}
}
