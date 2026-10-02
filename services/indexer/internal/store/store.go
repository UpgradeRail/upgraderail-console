// Package store persists normalized indexer events and checkpoints.
package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) ApplyControllerBatch(ctx context.Context, state projection.State, controllerID string, events []projection.Event, nextCursor string) (projection.State, error) {
	next, err := projection.ApplyBatch(state, events, nextCursor)
	if err != nil {
		return state, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return state, err
	}
	defer tx.Rollback(ctx)
	for _, event := range events {
		topics, err := json.Marshal([]string{event.Type})
		if err != nil {
			return state, err
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO controller_events
				(id, controller_id, network_id, ledger_sequence, transaction_hash, event_index, event_type, topics, data)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (network_id, transaction_hash, event_index) DO NOTHING
		`, event.Key(), controllerID, event.Network, event.Ledger, event.TransactionHash, event.Index, event.Type, topics, event.Data)
		if err != nil {
			return state, fmt.Errorf("insert controller event: %w", err)
		}
	}
	ledger := int64(0)
	if len(events) > 0 {
		ledger = int64(events[len(events)-1].Ledger)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO indexer_checkpoints (controller_id, cursor, ledger_sequence)
		VALUES ($1, $2, $3)
		ON CONFLICT (controller_id) DO UPDATE
		SET cursor = EXCLUDED.cursor,
		    ledger_sequence = EXCLUDED.ledger_sequence,
		    updated_at = now()
	`, controllerID, nextCursor, ledger)
	if err != nil {
		return state, fmt.Errorf("upsert checkpoint: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return state, err
	}
	return next, nil
}
