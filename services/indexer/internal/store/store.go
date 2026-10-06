// Package store persists normalized indexer events and checkpoints.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	"github.com/jackc/pgx/v5"
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

func (s *Store) ControllerID(ctx context.Context, networkID, contractID string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM controllers WHERE network_id = $1 AND contract_id = $2`, networkID, contractID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("controller %s is not configured in network %s; seed the migrated database first", contractID, networkID)
	}
	if err != nil {
		return "", fmt.Errorf("lookup controller: %w", err)
	}
	return id, nil
}

func (s *Store) LoadControllerState(ctx context.Context, controllerID string) (projection.State, error) {
	state := projection.NewState()
	rows, err := s.pool.Query(ctx, `
		SELECT network_id, transaction_hash, ledger_sequence, event_index, event_type, data, rpc_event_id
		FROM controller_events WHERE controller_id = $1
		ORDER BY rpc_event_id ASC NULLS FIRST
	`, controllerID)
	if err != nil {
		return state, fmt.Errorf("load controller journal: %w", err)
	}
	defer rows.Close()
	events := []projection.Event{}
	for rows.Next() {
		var event projection.Event
		var ledger int64
		var index int64
		var rpcEventID *string
		if err := rows.Scan(&event.Network, &event.TransactionHash, &ledger, &index, &event.Type, &event.Data, &rpcEventID); err != nil {
			return state, fmt.Errorf("scan controller journal: %w", err)
		}
		if rpcEventID == nil || *rpcEventID == "" {
			return state, errors.New("legacy controller journal has no RPC event order; use a fresh bootstrap before resuming")
		}
		if ledger < 0 || ledger > 0xffffffff || index < 0 || index > 0xffffffff {
			return state, errors.New("controller journal contains an out-of-range ledger or event index")
		}
		event.Ledger = uint32(ledger)
		event.Index = uint32(index)
		event.RPCEventID = *rpcEventID
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return state, fmt.Errorf("read controller journal: %w", err)
	}
	var cursor string
	err = s.pool.QueryRow(ctx, `SELECT cursor FROM indexer_checkpoints WHERE controller_id = $1`, controllerID).Scan(&cursor)
	if errors.Is(err, pgx.ErrNoRows) {
		if len(events) != 0 {
			return state, errors.New("controller journal exists without a checkpoint")
		}
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("load indexer checkpoint: %w", err)
	}
	if cursor == "" {
		return state, errors.New("indexer checkpoint has no cursor")
	}
	state, err = projection.ApplyBatch(state, events, cursor)
	if err != nil {
		return projection.NewState(), fmt.Errorf("replay controller journal: %w", err)
	}
	return state, nil
}

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
				(id, controller_id, network_id, ledger_sequence, transaction_hash, event_index, event_type, topics, data, rpc_event_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULLIF($10, ''))
			ON CONFLICT (network_id, transaction_hash, event_index) DO NOTHING
		`, event.Key(), controllerID, event.Network, event.Ledger, event.TransactionHash, event.Index, event.Type, topics, event.Data, event.RPCEventID)
		if err != nil {
			return state, fmt.Errorf("insert controller event: %w", err)
		}
		if err := applyReadModel(ctx, tx, controllerID, event); err != nil {
			return state, fmt.Errorf("project %s: %w", event.Type, err)
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
		    ledger_sequence = GREATEST(indexer_checkpoints.ledger_sequence, EXCLUDED.ledger_sequence),
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
