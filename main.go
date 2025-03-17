package main

import (
	"embed"
	_ "image/gif"
	_ "image/jpeg"
	"log/slog"
	"net/http"

	"github.com/codahale/yellhole-go/config"
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

	app, err := newApp(config)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := app.close(); err != nil {
			slog.Error("error shutting down", "err", err)
		}
	}()

	// Listen for HTTP requests.
	slog.Info("listening for connections", "baseURL", config.BaseURL)
	if err := http.ListenAndServe(config.Addr, app); err != nil {
		panic(err)
	}
}
