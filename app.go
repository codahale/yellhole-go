package main

import (
	"context"
	"filippo.io/csrf"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/db"
	sloghttp "github.com/samber/slog-http"
	_ "modernc.org/libc"
	_ "modernc.org/sqlite"
)

func newApp(ctx context.Context, queries *db.Queries, dataDir, author, title, description, lang string, baseURL *url.URL, requestLog bool) (http.Handler, error) {
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
	templates, err := loadTemplates(author, title, description, lang, baseURL, assetHashes)
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
	addRoutes(mux, author, title, description, baseURL, queries, templates, images, assets, assetPaths)

	// Require authentication for all /admin requests.
	handler := requireAuthentication(queries, mux, baseURL, "/admin")

	// Protect from CSRF attacks.
	handler = csrf.New().Handler(handler)

	// Serve the root from the base URL path.
	if baseURL.Path != "/" {
		nestedPrefix := strings.TrimRight(baseURL.Path, "/")
		handler = http.StripPrefix(nestedPrefix, handler)
	}

	// Add logging.
	loggerHandler := slog.DiscardHandler
	if requestLog {
		loggerHandler = slog.NewJSONHandler(os.Stdout, nil)
	}
	logger := slog.New(loggerHandler)
	handler = sloghttp.Recovery(handler)
	handler = sloghttp.New(logger)(handler)
	return handler, nil
}
