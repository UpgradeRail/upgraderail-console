package auth

import "testing"

func TestMatchesDomainRejectsCrossSiteOrigin(t *testing.T) {
	if !matchesDomain("https://console.example", "console.example") {
		t.Fatal("expected matching origin")
	}
	if matchesDomain("https://attacker.example", "console.example") {
		t.Fatal("expected cross-site origin to fail")
	}
}
