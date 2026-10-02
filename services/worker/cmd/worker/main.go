package main

import "log/slog"

func main() {
	slog.Info("worker requires engine and database configuration", "service", "worker")
}
