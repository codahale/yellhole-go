package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/db"
	sloghttp "github.com/samber/slog-http"
	_ "modernc.org/libc"
	_ "modernc.org/sqlite"
)

func newApp(ctx context.Context, config *config, queries *db.Queries) (http.Handler, error) {
	// Set up a purgeTicker to purge old sessions every five minutes.
	purgeTicker := time.NewTicker(5 * time.Minute)
	go purgeOldRows(ctx, queries, purgeTicker)

	// Open the data directory as a file system root.
	dataRoot, err := os.OpenRoot(config.DataDir)
	if err != nil {
		return nil, err
	}

	// Load the embedded public assets.
	assetPaths, assetHashes, assets, err := loadAssets()
	if err != nil {
		return nil, err
	}

	// Load the embedded templates and create a new template set.
	templates, err := newTemplateSet(config, assetHashes)
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
	return handler, nil
}
