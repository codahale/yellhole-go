package main

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/db"
	sloghttp "github.com/samber/slog-http"
	_ "modernc.org/libc"
	_ "modernc.org/sqlite"
)

var (
	//go:embed assets
	assetsFS embed.FS
	//go:embed templates
	templatesFS embed.FS
	//go:embed db/migrations/*.sql
	migrationsFS embed.FS
)

type app struct {
	conn        *sql.DB
	queries     *db.Queries
	purgeTicker *time.Ticker
	http.Handler
}

func newApp(ctx context.Context, config *config) (*app, error) {
	slog.Info("starting", "dataDir", config.DataDir, "buildTag", buildTag)

	// Connect to the database.
	conn, err := sql.Open("sqlite", filepath.Join(config.DataDir, "yellhole.db")+"?_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	queries := db.New(conn)

	// Run migrations, if any,.
	if err := db.RunMigrations(conn, migrationsFS, "db/migrations"); err != nil {
		return nil, err
	}

	// Set up a purgeTicker to purge old sessions every five minutes.
	purgeTicker := time.NewTicker(5 * time.Minute)
	go purgeOldRows(ctx, queries, purgeTicker)

	// Open the data directory as a file system root.
	dataRoot, err := os.OpenRoot(config.DataDir)
	if err != nil {
		return nil, err
	}

	// Load the embedded public assets.
	assetsDir, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return nil, err
	}

	assetPaths, assetHashes, assets, err := loadAssets(assetsDir)
	if err != nil {
		return nil, err
	}

	// Load the embedded templates and create a new template set.
	templatesDir, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		return nil, err
	}

	templates, err := newTemplateSet(config, templatesDir, assetHashes)
	if err != nil {
		return nil, err
	}

	// Configure Webauthn.
	webAuthn, err := newWebauthn(config)
	if err != nil {
		return nil, err
	}

	// Create an image store.
	images, err := newImageStore(dataRoot)
	if err != nil {
		return nil, err
	}

	// Construct a route map of handlers.
	mux := http.NewServeMux()
	addRoutes(mux, config, queries, templates, images, webAuthn, assets, assetPaths)

	// Require authentication for all /admin requests.
	handler := requireAuthentication(config, queries, mux, "/admin")

	// Serve the root from the base URL path.
	if config.BaseURL.Path != "/" {
		nestedPrefix := strings.TrimRight(config.BaseURL.Path, "/")
		handler = http.StripPrefix(nestedPrefix, mux)
	}

	// Add logging.
	loggerHandler := slog.DiscardHandler
	if config.requestLog {
		loggerHandler = slog.NewJSONHandler(os.Stdout, nil)
	}
	logger := slog.New(loggerHandler)
	handler = sloghttp.Recovery(handler)
	handler = sloghttp.New(logger)(handler)

	return &app{conn, queries, purgeTicker, handler}, nil
}

func (a *app) close() error {
	a.purgeTicker.Stop()

	return a.conn.Close()
}
