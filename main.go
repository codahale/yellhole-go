package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"go.uber.org/automaxprocs/maxprocs"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

func main() {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Automatically set GOMAXPROCS.
	undo, err := maxprocs.Set()
	defer undo()
	if err != nil {
		panic(err)
	}
	slog.Info("setting runtime CPU count", "GOMAXPROCS", runtime.GOMAXPROCS(-1))

	// Parse the configuration flags and environment variables.
	config, err := parseConfig()
	if err != nil {
		panic(err)
	}

	// Create a new app.
	app, err := newApp(ctx, config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := app.close(); err != nil {
			slog.Error("error shutting down", "err", err)
		}
	}()

	// Configure an HTTP server with good defaults.
	server := &http.Server{
		Addr:    config.Addr,
		Handler: app,

		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
		IdleTimeout:       120 * time.Second,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	// Listen for connections in a separate goroutine.
	slog.Info("listening for connections", "baseURL", config.BaseURL)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("error listening for requests", "err", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	slog.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish the requests it is
	// currently handling.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("error shutting down", "err", err)
	}

	slog.Info("exiting")
}
