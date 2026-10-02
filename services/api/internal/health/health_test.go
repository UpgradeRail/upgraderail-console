package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type failingProbe struct{}

func (failingProbe) Ping(context.Context) error { return errors.New("database unavailable") }

func TestLivenessDoesNotRequireDependencies(t *testing.T) {
	response := httptest.NewRecorder()
	New(failingProbe{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if response.Code != http.StatusOK { t.Fatalf("got %d", response.Code) }
}

func TestReadinessReportsUnavailableDependency(t *testing.T) {
	response := httptest.NewRecorder()
	New(failingProbe{}).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable { t.Fatalf("got %d", response.Code) }
}
