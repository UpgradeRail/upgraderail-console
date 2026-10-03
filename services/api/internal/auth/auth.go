// Package auth implements replay-safe wallet authentication challenges.
package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const defaultTTL = 5 * time.Minute
const sep53MessagePrefix = "Stellar Signed Message:\n"

type Challenge struct {
	ID, Domain, Network, Address, Nonce, Purpose string
	IssuedAt, ExpiresAt                          time.Time
}

type StoredChallenge struct {
	ID, Domain, Network, Address, NonceHash, Purpose string
	IssuedAt, ExpiresAt                              time.Time
	UsedAt                                           *time.Time
}

type Session struct {
	Address string `json:"address"`
	Network string `json:"network"`
}

func NewChallenge(id, domain, network, address, purpose string, now time.Time) (Challenge, error) {
	if id == "" || domain == "" || network == "" || address == "" || purpose == "" {
		return Challenge{}, errors.New("challenge fields are required")
	}
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return Challenge{}, fmt.Errorf("generate nonce: %w", err)
	}
	return Challenge{ID: id, Domain: domain, Network: network, Address: address, Nonce: base64.RawURLEncoding.EncodeToString(nonce), Purpose: purpose, IssuedAt: now.UTC(), ExpiresAt: now.UTC().Add(defaultTTL)}, nil
}

func (c Challenge) Message() string {
	return strings.Join([]string{"UpgradeRail authentication", "domain: " + c.Domain, "network: " + c.Network, "address: " + c.Address, "nonce: " + c.Nonce, "issued_at: " + c.IssuedAt.UTC().Format(time.RFC3339), "expires_at: " + c.ExpiresAt.UTC().Format(time.RFC3339), "purpose: " + c.Purpose}, "\n")
}
func (c Challenge) NonceHash() string { return hash(c.Nonce) }
func Hash(value string) string        { return hash(value) }

func Verify(address, message, signature string) error {
	publicKey, err := stellarPublicKey(address)
	if err != nil {
		return err
	}
	bytes, err := decodeSignature(signature)
	if err != nil {
		return err
	}
	messageHash := sha256.Sum256([]byte(sep53MessagePrefix + message))
	if !ed25519.Verify(publicKey, messageHash[:], bytes) {
		return errors.New("signature verification failed")
	}
	return nil
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func decodeSignature(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(value)
	}
	if err != nil || len(decoded) != ed25519.SignatureSize {
		return nil, errors.New("signature must be base64 Ed25519 bytes")
	}
	return decoded, nil
}

func stellarPublicKey(address string) (ed25519.PublicKey, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(address)
	if err != nil || len(decoded) != 35 || decoded[0] != 6<<3 {
		return nil, errors.New("address must be a Stellar Ed25519 public key")
	}
	checksum := uint16(decoded[33]) | uint16(decoded[34])<<8
	if crc16(decoded[:33]) != checksum {
		return nil, errors.New("address checksum is invalid")
	}
	return ed25519.PublicKey(decoded[1:33]), nil
}

func crc16(value []byte) uint16 {
	var crc uint16
	for _, b := range value {
		crc ^= uint16(b) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
