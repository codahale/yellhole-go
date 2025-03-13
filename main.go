package main

import (
	"database/sql"
	"embed"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/codahale/yellhole-go/db"
	_ "golang.org/x/image/webp"
	_ "modernc.org/sqlite"
)

//go:generate go tool sqlc generate -f db/sqlc.yaml

//go:embed public
var public embed.FS

func main() {
	// Parse the configuration flags and environment variables.
	config, err := parseConfig()
	if err != nil {
		panic(err)
	}

	// Connect to the database.
	conn, err := sql.Open("sqlite", filepath.Join(config.DataDir, "yellhole.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	queries := db.New(conn)

	// Open the data directory as a file system root.
	log.Printf("storing data in %s", config.DataDir)
	dataRoot, err := os.OpenRoot(config.DataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer dataRoot.Close()

	// Load the embedded public assets and create an asset controller.
	assets := newAssetController(public, "public")

	// Create the controllers.
	images, err := newImageController(dataRoot, queries)
	if err != nil {
		log.Fatal(err)
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
	mux.Handle("GET /login/start", http.HandlerFunc(auth.LoginStart))
	mux.Handle("GET /login/finish", http.HandlerFunc(auth.LoginFinish))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", http.HandlerFunc(images.ServeFeedImage)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", http.HandlerFunc(images.ServeThumbImage)))

	for _, path := range assets.AssetPaths() {
		mux.Handle(fmt.Sprintf("GET /%s", path), assets)
	}

	// Listen for HTTP requests.
	log.Printf("listening on %s", config.Addr)
	if err := http.ListenAndServe(config.Addr, mux); err != nil {
		log.Fatalln("error serving HTTP", err)
	}
}
