package main

import (
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
)

var (
	//go:embed public
	publicDir embed.FS
	//go:embed templates
	templatesDir embed.FS
	//go:embed db/migrations/*.sql
	migrationsDir embed.FS
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
	conn, err := sql.Open("sqlite", filepath.Join(config.DataDir, "yellhole.db")+"?_time_format=sqlite")
	if err != nil {
		return nil, err
	}
	queries := db.New(conn)

	// Run migrations, if any,.
	if err := db.RunMigrations(conn, migrationsDir, "db/migrations"); err != nil {
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
	assetsDir, err := fs.Sub(publicDir, "public")
	if err != nil {
		return nil, err
	}

	assets, err := newAssetController(assetsDir)
	if err != nil {
		return nil, err
	}

	// Create a new template set.
	templates, err := newTemplateSet(config, templatesDir, assets)
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

	mux.Handle("GET /{$}", http.HandlerFunc(feeds.homePage))
	mux.Handle("GET /atom.xml", http.HandlerFunc(feeds.atomFeed))
	mux.Handle("GET /notes/{start}", http.HandlerFunc(feeds.weekPage))
	mux.Handle("GET /note/{id}", http.HandlerFunc(feeds.notePage))

	mux.Handle("GET /admin", http.HandlerFunc(admin.adminPage))
	mux.Handle("POST /admin/new", http.HandlerFunc(admin.newNote))
	mux.Handle("POST /admin/images/download", http.HandlerFunc(images.downloadImage))
	mux.Handle("POST /admin/images/upload", http.HandlerFunc(images.uploadImage))

	mux.Handle("GET /register", http.HandlerFunc(auth.registerPage))
	mux.Handle("POST /register/start", http.HandlerFunc(auth.registerStart))
	mux.Handle("POST /register/finish", http.HandlerFunc(auth.registerFinish))
	mux.Handle("GET /login", http.HandlerFunc(auth.loginPage))
	mux.Handle("POST /login/start", http.HandlerFunc(auth.loginStart))
	mux.Handle("POST /login/finish", http.HandlerFunc(auth.loginFinish))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", http.HandlerFunc(images.feedImage)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", http.HandlerFunc(images.thumbImage)))

	for _, path := range assets.assetPaths() {
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
