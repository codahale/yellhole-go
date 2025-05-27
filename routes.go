package main

import (
	"github.com/codahale/yellhole-go/internal/db"
	"github.com/codahale/yellhole-go/internal/imgstore"
	"html/template"
	"net/http"
	"net/url"
)

func addRoutes(mux *http.ServeMux, author, title, description string, baseURL *url.URL, queries *db.Queries, t *template.Template, images *imgstore.Store, assets http.Handler, assetPaths []string) {
	mux.Handle("GET /{$}", handleErrors(handleHomePage(queries, t)))
	mux.Handle("GET /notes/{start}", handleErrors(handleWeekPage(queries, t)))
	mux.Handle("GET /note/{id}", handleErrors(handleNotePage(queries, t)))
	mux.Handle("GET /atom.xml", handleErrors(handleAtomFeed(queries, author, title, description, baseURL)))

	mux.Handle("GET /admin", handleErrors(handleAdminPage(queries, t)))
	mux.Handle("POST /admin/new", handleErrors(handleNewNote(queries, t, baseURL)))
	mux.Handle("POST /admin/images/download", handleErrors(handleDownloadImage(queries, images, baseURL)))
	mux.Handle("POST /admin/images/upload", handleErrors(handleUploadImage(queries, images, baseURL)))

	mux.Handle("GET /register", handleErrors(handleRegisterPage(queries, t, baseURL)))
	mux.Handle("POST /register/start", handleErrors(handleRegisterStart(queries, author, title, baseURL)))
	mux.Handle("POST /register/finish", handleErrors(handleRegisterFinish(queries, author, title, baseURL)))
	mux.Handle("GET /login", handleErrors(handleLoginPage(queries, t, baseURL)))
	mux.Handle("POST /login/start", handleErrors(handleLoginStart(queries, author, title, baseURL)))
	mux.Handle("POST /login/finish", handleErrors(handleLoginFinish(queries, author, title, baseURL)))

	mux.Handle("GET /images/feed/", http.StripPrefix("/images/feed/", handleFeedImage(images)))
	mux.Handle("GET /images/thumb/", http.StripPrefix("/images/thumb/", handleThumbImage(images)))

	for _, path := range assetPaths {
		mux.Handle("GET /"+path, assets)
	}
}
