package store

import "testing"

func TestNewPageBoundsUntrustedValues(t *testing.T) {
	page, err := NewPage(0, 0)
	if err != nil || page.Limit != 50 {
		t.Fatalf("got %#v, %v", page, err)
	}
	if _, err := NewPage(101, 0); err == nil {
		t.Fatal("expected limit failure")
	}
	if _, err := NewPage(1, -1); err == nil {
		t.Fatal("expected offset failure")
	}
}
