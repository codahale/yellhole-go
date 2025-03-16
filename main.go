package main

import (
	"database/sql"
	"embed"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	sloghttp "github.com/samber/slog-http"
	_ "golang.org/x/image/webp"
	_ "modernc.org/libc"
	_ "modernc.org/sqlite"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

//go:embed public
var public embed.FS

func main() {
	// Parse the configuration flags and environment variables.
	config, err := config.Parse()
	if err != nil {
		panic(err)
	}

	slog.Default().Info("starting", "dataDir", config.DataDir)

	// Connect to the database.
	conn, err := sql.Open("sqlite", filepath.Join(config.DataDir, "yellhole.db"))
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Run migrations, if any,.
	if err := db.RunMigrations(conn); err != nil {
		panic(err)
	}

	queries := db.New(conn)

	// Set up a ticker to purge old sessions every five minutes.
	go purgeOldSessionsInBackground(queries)

	// Open the data directory as a file system root.
	dataRoot, err := os.OpenRoot(config.DataDir)
	if err != nil {
		panic(err)
	}
	defer dataRoot.Close()

	// Load the embedded public assets and create an asset controller.
	assets := newAssetController(public, "public")

	// Create the controllers.
	images, err := newImageController(config, dataRoot, queries)
	if err != nil {
		panic(err)
	}
	defer images.Close()

	feeds := newFeedController(config, queries)
	admin := newAdminController(config, queries)
	auth := newAuthController(config, queries)

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
		mux.Handle(fmt.Sprintf("GET /%s", path), assets)
	}

	// Require authentication for all /admin requests.
	var root http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, "/admin") {
			auth, err := isAuthenticated(r, queries)
			if err != nil {
				panic(err)
			}

			if !auth {
				slog.Info("unauthenticated request", "uri", r.RequestURI, "id", sloghttp.GetRequestID(r))
				http.Redirect(w, r, config.BaseURL.JoinPath("login").String(), http.StatusSeeOther)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})

	if config.BaseURL.Path != "/" {
		nestedPrefix := strings.TrimRight(config.BaseURL.Path, "/")
		root = http.StripPrefix(nestedPrefix, mux)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	root = sloghttp.Recovery(root)
	root = sloghttp.New(logger)(root)

	// Listen for HTTP requests.
	slog.Info("listening for connections", "baseURL", config.BaseURL)
	if err := http.ListenAndServe(config.Addr, root); err != nil {
		panic(err)
	}
}
