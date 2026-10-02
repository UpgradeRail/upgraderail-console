package engine

import (
	"context"
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
