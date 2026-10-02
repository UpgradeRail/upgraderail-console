package main

import "log/slog"

func main() {
	slog.Info("indexer requires controller and database configuration", "service", "indexer")
}
