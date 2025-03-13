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
	confAddr := ":8080"
	confDataDir := "./data"

	// Connect to the database.
	conn, err := sql.Open("sqlite", filepath.Join(confDataDir, "yellhole.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	queries := db.New(conn)

	// Open the data directory as a file system root.
	log.Printf("storing data in %s", confDataDir)
	dataRoot, err := os.OpenRoot(confDataDir)
	if err != nil {
		log.Fatal(err)
	}
	defer dataRoot.Close()

	// Load the embedded public assets and create an asset controller.
	assets := newAssetController(public, "public")

	// Create a new image controller.
	images, err := newImageController(dataRoot, queries)
	if err != nil {
		log.Fatal(err)
	}
	defer images.Close()

	// Construct a route map of handlers.
	mux := http.NewServeMux()

	// TODO implement feed controller
	// mux.Handle("GET /{$}", http.HandlerFunc(feeds.HomePage))
	// mux.Handle("GET /atom.xml", http.HandlerFunc(feeds.AtomFeed))
	// mux.Handle("GET /notes/{start}", http.HandlerFunc(feeds.WeekPage))
	// mux.Handle("GET /note/{id}", http.HandlerFunc(feeds.NotePage))

	// TODO implement admin controller
	// mux.Handle("GET /admin", http.HandlerFunc(admin.AdminPage))
	// mux.Handle("POST /admin/new", http.HandlerFunc(admin.NewNote))
	// mux.Handle("POST /admin/upload", http.HandlerFunc(admin.UploadImage))
	// mux.Handle("POST /admin/download", http.HandlerFunc(admin.DownloadImage))

	// TODO implement auth controller
	// mux.Handle("GET /register", http.HandlerFunc(auth.RegisterPage))
	// mux.Handle("POST /register/start", http.HandlerFunc(auth.RegisterStart))
	// mux.Handle("POST /register/finish", http.HandlerFunc(auth.RegisterFinish))
	// mux.Handle("GET /login", http.HandlerFunc(auth.LoginPage))
	// mux.Handle("GET /login/start", http.HandlerFunc(auth.LoginStart))
	// mux.Handle("GET /login/finish", http.HandlerFunc(auth.LoginFinish))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", http.HandlerFunc(images.ServeFeedImage)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", http.HandlerFunc(images.ServeThumbImage)))
	mux.Handle("POST /images/download", http.HandlerFunc(images.DownloadImage))
	mux.Handle("POST /images/upload", http.HandlerFunc(images.UploadImage))

	for _, path := range assets.AssetPaths() {
		mux.Handle(fmt.Sprintf("GET /%s", path), assets)
	}

	// Listen for HTTP requests.
	log.Printf("listening on %s", confAddr)
	if err := http.ListenAndServe(confAddr, mux); err != nil {
		log.Fatalln("error serving HTTP", err)
	}
}
