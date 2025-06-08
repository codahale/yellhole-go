package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/codahale/yellhole-go/internal/build"
	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/imgstore"
	"go.uber.org/automaxprocs/maxprocs"
)

//go:generate sqlc generate -f internal/db/sqlc.yaml

func run(args []string, lookupEnv func(string) (string, bool)) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	// Generate the build tag.
	buildTag, err := build.Tag()
	if err != nil {
		return fmt.Errorf("failed to generate build tag: %w", err)
	}

	// Parse the configuration flags and environment variables.
	addr, baseURL, dataDir, author, title, description, lang, err := loadConfig(args, lookupEnv)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create a context that listens for the interrupt signal from the OS.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Automatically set GOMAXPROCS.
	undo, err := maxprocs.Set()
	defer undo()
	if err != nil {
		return fmt.Errorf("failed to set GOMAXPROCS: %w", err)
	}
	logger.Info("setting runtime CPU count", "GOMAXPROCS", runtime.GOMAXPROCS(-1))

	// Connect to the database.
	logger.Info("starting", "dataDir", dataDir, "buildTag", buildTag)
	conn, queries, err := db.NewWithMigrations(logger, filepath.Join(dataDir, "yellhole.db"))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer func() {
		if err := queries.Close(); err != nil {
			slog.Error("error closing queries", "err", err)
		}
		if err := conn.Close(); err != nil {
			slog.Error("error closing database", "err", err)
		}
	}()

	// Create an image store.
	images, err := imgstore.New(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create image store: %w", err)
	}
	defer func() {
		if err := images.Close(); err != nil {
			slog.Error("error closing image store", "err", err)
		}
	}()

	// Create a new app.
	app, err := newApp(ctx, logger, queries, images, baseURL, author, title, description, lang, buildTag, true)
	if err != nil {
		return fmt.Errorf("failed to create application: %w", err)
	}

	// Configure an HTTP server with good defaults.
	server := &http.Server{
		Addr:    addr,
		Handler: app,

		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		IdleTimeout:       120 * time.Second,
		ReadTimeout:       5 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	// Listen for connections in a separate goroutine.
	logger.Info("listening for connections", "baseURL", baseURL)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error listening for requests", "err", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify the user of shutdown.
	stop()
	logger.Info("shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish the requests it is
	// currently handling.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("error shutting down", "err", err)
	}

	logger.Info("exiting")

	return nil
}

func main() {
	if err := run(os.Args[1:], os.LookupEnv); err != nil {
		if !errors.Is(err, flag.ErrHelp) {
			_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(2)
		}
	}
}
