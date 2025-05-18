package main

import (
	"net/http"

	"github.com/codahale/yellhole-go/db"
	"github.com/go-webauthn/webauthn/webauthn"
)

func addRoutes(mux *http.ServeMux, config *config, queries *db.Queries, templates *templateSet, images *imageStore, webAuthn *webauthn.WebAuthn, assets http.Handler, assetPaths []string) {
	mux.Handle("GET /{$}", handleHomePage(queries, templates))
	mux.Handle("GET /atom.xml", handleAtomFeed(config, queries))
	mux.Handle("GET /notes/{start}", handleWeekPage(queries, templates))
	mux.Handle("GET /note/{id}", handleNotePage(queries, templates))

	mux.Handle("GET /admin", handleAdminPage(queries, templates))
	mux.Handle("POST /admin/new", handleNewNote(config, queries, templates))
	mux.Handle("POST /admin/images/download", handleDownloadImage(config, queries, images))
	mux.Handle("POST /admin/images/upload", handleUploadImage(config, queries, images))

	mux.Handle("GET /register", handleRegisterPage(config, queries, templates))
	mux.Handle("POST /register/start", handleRegisterStart(config, queries, webAuthn))
	mux.Handle("POST /register/finish", handleRegisterFinish(config, queries, webAuthn))
	mux.Handle("GET /login", handleLoginPage(config, queries, templates))
	mux.Handle("POST /login/start", handleLoginStart(config, queries, webAuthn))
	mux.Handle("POST /login/finish", handleLoginFinish(config, queries, webAuthn))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", handleFeedImage(images)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", handleThumbImage(images)))

	for _, path := range assetPaths {
		mux.Handle("GET /"+path, assets)
	}
}
