package engine

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCompareRejectsPathsOutsideTheWorkerDirectory(t *testing.T) {
	_, err := Runner{Binary: "upgraderail", Timeout: time.Second}.Compare(context.Background(), "job", "current.wasm", "candidate.wasm")
	if err == nil {
		t.Fatal("expected relative paths to fail")
	}
}

func TestDecodeReportPreservesBlockedStatus(t *testing.T) {
	report, err := decodeReport([]byte(`{"status":"BLOCKED","findings":[],"storage_compatibility":"NOT PROVEN BY STATIC ANALYSIS","authorization_behavior":"NOT TESTED"}`))
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "BLOCKED" {
		t.Fatalf("got %q", report.Status)
	}
}

func TestDecodeReportRequiresEvidence(t *testing.T) {
	_, err := decodeReport([]byte(`{"status":"READY","findings":[]}`))
	if err == nil {
		t.Fatal("expected missing evidence to fail")
	}
}

func TestSafeEnvironmentDoesNotForwardSecrets(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://secret")
	t.Setenv("SESSION_SECRET", "secret")
	t.Setenv("PATH", os.Getenv("PATH"))
	for _, value := range safeEnvironment() {
		if strings.Contains(value, "DATABASE_URL=") || strings.Contains(value, "SESSION_SECRET=") {
			t.Fatalf("secret environment variable was forwarded: %q", value)
		}
	}
}
