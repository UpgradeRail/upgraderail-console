package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthHandlerReportsLiveness(t *testing.T) {
	server := httptest.NewServer(healthHandler())
	defer server.Close()
	response, err := http.Get(server.URL + "/health/live")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	body, _ := io.ReadAll(response.Body)
	if !strings.Contains(string(body), `"service":"indexer"`) {
		t.Fatalf("unexpected body %q", body)
	}
	other, err := http.Get(server.URL + "/anything-else")
	if err != nil {
		t.Fatal(err)
	}
	defer other.Body.Close()
	if other.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for other paths, got %d", other.StatusCode)
	}
}
