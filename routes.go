package main

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/codahale/yellhole-go/imgstore"

	"github.com/codahale/yellhole-go/db"
)

func addRoutes(mux *http.ServeMux, author, title, description string, baseURL *url.URL, queries *db.Queries, t *template.Template, images *imgstore.Store, assets http.Handler, assetPaths []string) {
	mux.Handle("GET /{$}", handleHomePage(queries, t))
	mux.Handle("GET /atom.xml", handleAtomFeed(queries, author, title, description, baseURL))
	mux.Handle("GET /notes/{start}", handleWeekPage(queries, t))
	mux.Handle("GET /note/{id}", handleNotePage(queries, t))

	mux.Handle("GET /admin", handleAdminPage(queries, t))
	mux.Handle("POST /admin/new", handleNewNote(queries, t, baseURL))
	mux.Handle("POST /admin/images/download", handleDownloadImage(queries, images, baseURL))
	mux.Handle("POST /admin/images/upload", handleUploadImage(queries, images, baseURL))

	mux.Handle("GET /register", handleRegisterPage(queries, t, baseURL))
	mux.Handle("POST /register/start", handleRegisterStart(queries, author, title, baseURL))
	mux.Handle("POST /register/finish", handleRegisterFinish(queries, author, title, baseURL))
	mux.Handle("GET /login", handleLoginPage(queries, t, baseURL))
	mux.Handle("POST /login/start", handleLoginStart(queries, author, title, baseURL))
	mux.Handle("POST /login/finish", handleLoginFinish(queries, author, title, baseURL))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", handleFeedImage(images)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", handleThumbImage(images)))

	for _, path := range assetPaths {
		mux.Handle("GET /"+path, assets)
	}
}
