package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/codahale/yellhole-go/build"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/codahale/yellhole-go/db"
	"go.uber.org/automaxprocs/maxprocs"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

func run(args []string, lookupEnv func(string) (string, bool)) error {
	// Create context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Automatically set GOMAXPROCS.
	undo, err := maxprocs.Set()
	defer undo()
	if err != nil {
		return err
	}
	slog.Info("setting runtime CPU count", "GOMAXPROCS", runtime.GOMAXPROCS(-1))

	// Parse the configuration flags and environment variables.
	addr, baseURL, dataDir, author, title, description, lang, err := loadConfig(args, lookupEnv)
	if err != nil {
		return err
	}

	// Connect to the database.
	slog.Info("starting", "dataDir", dataDir, "buildTag", build.Tag)
	queries, err := db.NewWithMigrations(filepath.Join(dataDir, "yellhole.db"))
	if err != nil {
		return err
	}
	defer func() {
		if err := queries.Close(); err != nil {
			slog.Error("error closing database", "err", err)
		}
	}()

	// Create a new app.
	app, err := newApp(ctx, queries, baseURL, dataDir, author, title, description, lang, true)
	if err != nil {
		return err
	}

	// Configure an HTTP server with good defaults.
	server := &http.Server{
		Addr:    addr,
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
	slog.Info("listening for connections", "baseURL", baseURL)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
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

	return nil
}

func main() {
	if err := run(os.Args[1:], os.LookupEnv); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
