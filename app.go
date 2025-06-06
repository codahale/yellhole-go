package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"filippo.io/csrf"
	"github.com/CAFxX/httpcompression"
	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/imgstore"
	sloghttp "github.com/samber/slog-http"
	"github.com/valyala/bytebufferpool"
)

// newApp constructs an application handler given the various application inputs.
func newApp(ctx context.Context, logger *slog.Logger, queries *db.Queries, images *imgstore.Store, baseURL, author, title, description, lang, buildTag string, requestLog bool) (http.Handler, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL %q: %w", baseURL, err)
	}

	// Ensure the base URL always ends in a slash.
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	// Set up a purgeTicker to purge old sessions every five minutes.
	purgeTicker := time.NewTicker(5 * time.Minute)
	go purgeOldRows(ctx, logger, queries, purgeTicker)

	// Load the embedded public assets.
	assetPaths, assetHashes, assets, err := loadAssets()
	if err != nil {
		return nil, fmt.Errorf("failed to load assets: %w", err)
	}

	// Load the embedded templates.
	templates, err := loadTemplates(author, title, description, lang, buildTag, u, assetHashes)
	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	// Construct a route map of handlers.
	mux := http.NewServeMux()
	addRoutes(mux, author, title, description, u, logger, queries, templates, images, assets, assetPaths)

	// Require authentication for all /admin requests.
	handler := requireAuthentication(queries, mux, u, "/admin")

	// Protect from CSRF attacks.
	handler = csrf.New().Handler(handler)

	// Add compression for responses.
	compress, err := httpcompression.DefaultAdapter(httpcompression.ContentTypes([]string{"text/html", "text/css", "text/javascript"}, false))
	if err != nil {
		return nil, fmt.Errorf("failed to create compression adapter: %w", err)
	}
	handler = compress(handler)

	// Serve the root from the base URL path.
	handler = http.StripPrefix(strings.TrimRight(u.Path, "/"), handler)

	// Recover from panics in handlers.
	handler = sloghttp.Recovery(handler)

	// Add logging.
	loggerHandler := slog.DiscardHandler
	if requestLog {
		loggerHandler = slog.NewTextHandler(os.Stdout, nil)
	}
	return sloghttp.New(slog.New(loggerHandler))(handler), nil
}

type appHandler = func(http.ResponseWriter, *http.Request) error

func handleErrors(handler appHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			slog.ErrorContext(r.Context(), "error handling request", "err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})
}

func htmlResponse(w http.ResponseWriter, t *template.Template, name string, data any) error {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	if err := t.ExecuteTemplate(b, name, data); err != nil {
		return fmt.Errorf("failed to execute template %q: %w", name, err)
	}

	w.Header().Set("Content-Type", "text/html")
	if _, err := w.Write(b.B); err != nil {
		return fmt.Errorf("failed to write HTML response: %w", err)
	}
	return nil
}

func jsonResponse(w http.ResponseWriter, v any) error {
	b := bytebufferpool.Get()
	defer bytebufferpool.Put(b)

	if err := json.NewEncoder(b).Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON response: %w", err)
	}

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(b.B); err != nil {
		return fmt.Errorf("failed to write JSON response: %w", err)
	}
	return nil
}
