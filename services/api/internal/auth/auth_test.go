package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
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

func TestChallengeMessageIsStableAfterTimezoneConversion(t *testing.T) {
	issuedAt := time.Date(2026, 10, 3, 6, 15, 21, 0, time.UTC)
	challenge := Challenge{Domain: "127.0.0.1", Network: "testnet", Address: "GTEST", Nonce: "nonce", IssuedAt: issuedAt, ExpiresAt: issuedAt.Add(5 * time.Minute), Purpose: "console_session"}
	stored := challenge
	stored.IssuedAt = stored.IssuedAt.In(time.FixedZone("WAT", 60*60))
	stored.ExpiresAt = stored.ExpiresAt.In(time.FixedZone("WAT", 60*60))
	if challenge.Message() != stored.Message() {
		t.Fatal("challenge message changed when PostgreSQL returned timestamps in local time")
	}
}

func TestVerifyChecksStellarPublicKeySignature(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	address := stellarAddress(public)
	message := "challenge"
	messageHash := sha256.Sum256([]byte(sep53MessagePrefix + message))
	signature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, messageHash[:]))
	if err := Verify(address, message, signature); err != nil {
		t.Fatal(err)
	}
	legacySignature := base64.StdEncoding.EncodeToString(ed25519.Sign(private, []byte(message)))
	if err := Verify(address, message, legacySignature); err == nil {
		t.Fatal("expected raw-message signature to fail SEP-53 verification")
	}
	if err := Verify(address, message+"x", signature); err == nil {
		t.Fatal("expected invalid signature")
	}
}

func TestVerifySEP53SpecificationVector(t *testing.T) {
	const address = "GBXFXNDLV4LSWA4VB7YIL5GBD7BVNR22SGBTDKMO2SBZZHDXSKZYCP7L"
	const message = "Hello, World!"
	const signature = "fO5dbYhXUhBMhe6kId/cuVq/AfEnHRHEvsP8vXh03M1uLpi5e46yO2Q8rEBzu3feXQewcQE5GArp88u6ePK6BA=="
	if err := Verify(address, message, signature); err != nil {
		t.Fatalf("SEP-53 specification signature rejected: %v", err)
	}
}

func stellarAddress(key ed25519.PublicKey) string {
	payload := append([]byte{6 << 3}, key...)
	checksum := crc16(payload)
	payload = append(payload, byte(checksum), byte(checksum>>8))
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(payload)
}
