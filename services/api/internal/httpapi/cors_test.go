package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowsCredentialedConsolePreflight(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/challenge", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	request.Header.Set("Access-Control-Request-Method", "POST")
	request.Header.Set("Access-Control-Request-Headers", "content-type")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("unexpected allowed origin %q", got)
	}
	if response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credentialed requests were not enabled")
	}
}

func TestCORSRejectsForeignPreflight(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "http://localhost:3000")
	request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/challenge", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected foreign preflight to be rejected, got %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("foreign origin received CORS permission")
	}
}

func TestCORSRejectsDifferentSchemeOrPort(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), "https://console.example")
	for _, origin := range []string{"http://console.example", "https://console.example:8443"} {
		request := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/challenge", nil)
		request.Header.Set("Origin", origin)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("expected %q to be rejected, got %d", origin, response.Code)
		}
	}
}
