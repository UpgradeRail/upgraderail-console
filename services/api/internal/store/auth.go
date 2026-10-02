package store

import (
	"context"
	"fmt"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/api/internal/auth"
	"github.com/jackc/pgx/v5"
)

func (s *Store) CreateChallenge(ctx context.Context, value auth.Challenge) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO auth_challenges (id, domain, network_id, public_address, nonce_hash, purpose, issued_at, expires_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, value.ID, value.Domain, value.Network, value.Address, value.NonceHash(), value.Purpose, value.IssuedAt, value.ExpiresAt)
	if err != nil {
		return fmt.Errorf("create challenge: %w", err)
	}
	return nil
}

func (s *Store) GetChallenge(ctx context.Context, id string) (auth.StoredChallenge, error) {
	var value auth.StoredChallenge
	err := s.pool.QueryRow(ctx, `SELECT id, domain, network_id, public_address, nonce_hash, purpose, issued_at, expires_at, used_at FROM auth_challenges WHERE id = $1`, id).Scan(&value.ID, &value.Domain, &value.Network, &value.Address, &value.NonceHash, &value.Purpose, &value.IssuedAt, &value.ExpiresAt, &value.UsedAt)
	if err != nil {
		return auth.StoredChallenge{}, fmt.Errorf("get challenge: %w", err)
	}
	return value, nil
}

func (s *Store) ConsumeChallenge(ctx context.Context, id, nonceHash string, now time.Time) (bool, error) {
	result, err := s.pool.Exec(ctx, `UPDATE auth_challenges SET used_at = $1 WHERE id = $2 AND nonce_hash = $3 AND used_at IS NULL AND expires_at > $1`, now, id, nonceHash)
	if err != nil {
		return false, fmt.Errorf("consume challenge: %w", err)
	}
	return result.RowsAffected() == 1, nil
}

func (s *Store) CreateSession(ctx context.Context, id, address, network, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO sessions (id, public_address, network_id, token_hash, expires_at) VALUES ($1,$2,$3,$4,$5)`, id, address, network, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE sessions SET revoked_at = $1 WHERE token_hash = $2 AND revoked_at IS NULL`, now, tokenHash)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}
