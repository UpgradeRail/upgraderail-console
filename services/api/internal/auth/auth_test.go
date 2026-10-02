package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestChallengeHasRandomNonceAndCanonicalMessage(t *testing.T) {
	first, err := NewChallenge("one", "console.local", "testnet", "GABC", "analysis", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewChallenge("two", "console.local", "testnet", "GABC", "analysis", time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if first.Nonce == second.Nonce || first.NonceHash() == first.Nonce || !strings.Contains(first.Message(), "purpose: analysis") {
		t.Fatal("challenge content is invalid")
	}
}

func TestVerifyChecksStellarPublicKeySignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	address := stellarAddress(public)
	message := "challenge"
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, []byte(message)))
	if err := Verify(address, message, signature); err != nil {
		t.Fatal(err)
	}
	if err := Verify(address, message+"x", signature); err == nil {
		t.Fatal("expected invalid signature")
	}
}

func stellarAddress(key ed25519.PublicKey) string {
	payload := append([]byte{6 << 3}, key...)
	checksum := crc16(payload)
	payload = append(payload, byte(checksum), byte(checksum>>8))
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
}
