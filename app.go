package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/db"
	sloghttp "github.com/samber/slog-http"
)

type app struct {
	conn        *sql.DB
	queries     *db.Queries
	purgeTicker *time.Ticker
	http.Handler
}

func newApp(config *config) (*app, error) {
	slog.Default().Info("starting", "dataDir", config.DataDir, "buildTag", buildTag)

	// Connect to the database.
	conn, err := sql.Open("sqlite", filepath.Join(config.DataDir, "yellhole.db"))
	if err != nil {
		return nil, err
	}
	queries := db.New(conn)

	// Run migrations, if any,.
	if err := db.RunMigrations(conn); err != nil {
		return nil, err
	}

	// Set up a purgeTicker to purge old sessions every five minutes.
	purgeTicker := time.NewTicker(5 * time.Minute)
	go purgeOldRows(queries, purgeTicker)

	// Open the data directory as a file system root.
	dataRoot, err := os.OpenRoot(config.DataDir)
	if err != nil {
		return nil, err
	}

	// Load the embedded public assets and create an asset controller.
	assets, err := newAssetController(public, "public")
	if err != nil {
		return nil, err
	}

	// Create a new template set.
	templates, err := newTemplateSet(config, assets)
	if err != nil {
		return nil, err
	}

	// Create the controllers.
	images, err := newImageController(config, dataRoot, queries)
	if err != nil {
		return nil, err
	}

	feeds := newFeedController(config, queries, templates)
	admin := newAdminController(config, queries, templates)
	auth := newAuthController(config, queries, templates)

	// Construct a route map of handlers.
	mux := http.NewServeMux()

	mux.Handle("GET /{$}", http.HandlerFunc(feeds.HomePage))
	mux.Handle("GET /atom.xml", http.HandlerFunc(feeds.AtomFeed))
	mux.Handle("GET /notes/{start}", http.HandlerFunc(feeds.WeekPage))
	mux.Handle("GET /note/{id}", http.HandlerFunc(feeds.NotePage))

	mux.Handle("GET /admin", http.HandlerFunc(admin.AdminPage))
	mux.Handle("POST /admin/new", http.HandlerFunc(admin.NewNote))
	mux.Handle("POST /admin/images/download", http.HandlerFunc(images.DownloadImage))
	mux.Handle("POST /admin/images/upload", http.HandlerFunc(images.UploadImage))

	mux.Handle("GET /register", http.HandlerFunc(auth.RegisterPage))
	mux.Handle("POST /register/start", http.HandlerFunc(auth.RegisterStart))
	mux.Handle("POST /register/finish", http.HandlerFunc(auth.RegisterFinish))
	mux.Handle("GET /login", http.HandlerFunc(auth.LoginPage))
	mux.Handle("POST /login/start", http.HandlerFunc(auth.LoginStart))
	mux.Handle("POST /login/finish", http.HandlerFunc(auth.LoginFinish))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", http.HandlerFunc(images.ServeFeedImage)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", http.HandlerFunc(images.ServeThumbImage)))

	for _, path := range assets.AssetPaths() {
		mux.Handle("GET /"+path, assets)
	}

	// Require authentication for all /admin requests.
	root := auth.requireAuthentication(mux, "/admin")

	// Serve the root from the base URL path.
	if config.BaseURL.Path != "/" {
		nestedPrefix := strings.TrimRight(config.BaseURL.Path, "/")
		root = http.StripPrefix(nestedPrefix, mux)
	}

	// Add logging.
	loggerHandler := slog.DiscardHandler
	if config.requestLog {
		loggerHandler = slog.NewJSONHandler(os.Stdout, nil)
	}
	logger := slog.New(loggerHandler)
	root = sloghttp.Recovery(root)
	root = sloghttp.New(logger)(root)

	return &app{conn, queries, purgeTicker, root}, nil
}

func (a *app) close() error {
	a.purgeTicker.Stop()

	return a.conn.Close()
}
