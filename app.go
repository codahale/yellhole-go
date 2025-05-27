package main

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"filippo.io/csrf"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/imgstore"
	sloghttp "github.com/samber/slog-http"
)

// newApp constructs an application handler given the various application inputs.
func newApp(ctx context.Context, queries *db.Queries, baseURL, dataDir, author, title, description, lang string, requestLog bool) (http.Handler, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	// Ensure baseURL always ends in a slash.
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	// Set up a purgeTicker to purge old sessions every five minutes.
	purgeTicker := time.NewTicker(5 * time.Minute)
	go purgeOldRows(ctx, queries, purgeTicker)

	// Open the data directory as a file system root.
	dataRoot, err := os.OpenRoot(dataDir)
	if err != nil {
		return nil, err
	}

	// Load the embedded public assets.
	assetPaths, assetHashes, assets, err := loadAssets()
	if err != nil {
		return nil, err
	}

	// Load the embedded templates.
	templates, err := loadTemplates(author, title, description, lang, u, assetHashes)
	if err != nil {
		return nil, err
	}

	// Create an image store.
	images, err := imgstore.New(dataRoot)
	if err != nil {
		return nil, err
	}

	// Construct a route map of handlers.
	mux := http.NewServeMux()
	addRoutes(mux, author, title, description, u, queries, templates, images, assets, assetPaths)

	// Require authentication for all /admin requests.
	handler := requireAuthentication(queries, mux, u, "/admin")

	// Protect from CSRF attacks.
	handler = csrf.New().Handler(handler)

	// Serve the root from the base URL path.
	handler = http.StripPrefix(strings.TrimRight(u.Path, "/"), handler)

	// Recover from panics in handlers.
	handler = sloghttp.Recovery(handler)

	// Add logging.
	loggerHandler := slog.DiscardHandler
	if requestLog {
		loggerHandler = slog.NewJSONHandler(os.Stdout, nil)
	}
	return sloghttp.New(slog.New(loggerHandler))(handler), nil
}
