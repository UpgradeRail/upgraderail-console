package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/projection"
	indexerrpc "github.com/UpgradeRail/upgraderail-console/services/indexer/internal/rpc"
	"github.com/UpgradeRail/upgraderail-console/services/indexer/internal/store"
)

type config struct {
	databaseURL, endpoint, network, passphrase, controller string
	startLedger                                            uint32
	pollInterval                                           time.Duration
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		slog.Error("indexer stopped", "error", redact(err.Error()))
		os.Exit(1)
	}
}

func readConfig() (config, error) {
	c := config{
		databaseURL:  os.Getenv("DATABASE_URL"),
		endpoint:     os.Getenv("STELLAR_RPC_URL"),
		network:      os.Getenv("STELLAR_NETWORK"),
		passphrase:   os.Getenv("STELLAR_NETWORK_PASSPHRASE"),
		controller:   os.Getenv("UPGRADERAIL_CONTROLLER_ID"),
		pollInterval: 5 * time.Second,
	}
	for name, value := range map[string]string{
		"DATABASE_URL":               c.databaseURL,
		"STELLAR_RPC_URL":            c.endpoint,
		"STELLAR_NETWORK":            c.network,
		"STELLAR_NETWORK_PASSPHRASE": c.passphrase,
		"UPGRADERAIL_CONTROLLER_ID":  c.controller,
		"UPGRADERAIL_START_LEDGER":   os.Getenv("UPGRADERAIL_START_LEDGER"),
	} {
		if value == "" {
			return c, fmt.Errorf("%s is required", name)
		}
	}
	endpoint, err := url.Parse(c.endpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "https" && !(endpoint.Scheme == "http" && (endpoint.Hostname() == "127.0.0.1" || endpoint.Hostname() == "localhost"))) {
		return c, errors.New("STELLAR_RPC_URL must be HTTPS or local HTTP")
	}
	start, err := strconv.ParseUint(os.Getenv("UPGRADERAIL_START_LEDGER"), 10, 32)
	if err != nil || start == 0 {
		return c, errors.New("UPGRADERAIL_START_LEDGER must be a positive ledger number")
	}
	c.startLedger = uint32(start)
	if text := os.Getenv("INDEXER_POLL_INTERVAL"); text != "" {
		interval, err := time.ParseDuration(text)
		if err != nil || interval < time.Second {
			return c, errors.New("INDEXER_POLL_INTERVAL must be at least one second")
		}
		c.pollInterval = interval
	}
	return c, nil
}

func run(ctx context.Context) error {
	c, err := readConfig()
	if err != nil {
		return err
	}
	db, err := store.Open(ctx, c.databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	controllerID, err := db.ControllerID(ctx, c.network, c.controller)
	if err != nil {
		return err
	}
	client := indexerrpc.Client{Endpoint: c.endpoint, Network: c.network}
	if err := client.CheckNetwork(ctx, c.passphrase); err != nil {
		return err
	}
	state, err := db.LoadControllerState(ctx, controllerID)
	if err != nil {
		return err
	}
	slog.Info("indexer ready", "service", "indexer", "network", c.network, "controller", c.controller, "resuming", state.Cursor != "")
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		state, err = indexOnce(ctx, client, db, c, controllerID, state)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		timer := time.NewTimer(c.pollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}

func indexOnce(ctx context.Context, client indexerrpc.Client, db *store.Store, c config, controllerID string, state projection.State) (projection.State, error) {
	var batch indexerrpc.Batch
	var err error
	if state.Cursor == "" {
		batch, err = client.FetchControllerEvents(ctx, c.controller, c.startLedger, 200)
	} else {
		batch, err = client.FetchControllerEventsAfter(ctx, c.controller, state.Cursor, 200)
	}
	if err != nil {
		return state, err
	}
	if batch.Cursor == state.Cursor && len(batch.Events) == 0 {
		return state, nil
	}
	next, err := db.ApplyControllerBatch(ctx, state, controllerID, batch.Events, batch.Cursor)
	if err != nil {
		return state, err
	}
	slog.Info("indexer batch committed", "service", "indexer", "events", len(batch.Events), "controller", c.controller)
	return next, nil
}

func redact(message string) string {
	for _, name := range []string{"DATABASE_URL", "STELLAR_RPC_URL", "SESSION_SECRET"} {
		if secret := os.Getenv(name); secret != "" {
			message = strings.ReplaceAll(message, secret, "[redacted]")
		}
	}
	return message
}
