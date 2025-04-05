package main

import (
	"log/slog"
	"net/http"

	"github.com/ory/graceful"
	_ "modernc.org/libc"
	_ "modernc.org/sqlite"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

func main() {
	// Parse the configuration flags and environment variables.
	config, err := parseConfig()
	if err != nil {
		panic(err)
	}

	app, err := newApp(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := app.close(); err != nil {
			slog.Error("error shutting down", "err", err)
		}
	}()

	// Listen for HTTP requests.
	slog.Info("listening for connections", "baseURL", config.BaseURL)
	server := graceful.WithDefaults(&http.Server{
		Addr:    config.Addr,
		Handler: app,
	})
	if err := graceful.Graceful(server.ListenAndServe, server.Shutdown); err != nil {
		panic(err)
	}
}
