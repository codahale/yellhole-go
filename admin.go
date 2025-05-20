package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

func handleAdminPage(queries *db.Queries, templates *templateSet) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		images, err := queries.RecentImages(r.Context(), 10)
		if err != nil {
			panic(err)
		}

		templates.render(w, "new.html", images)
	})
}

func handleNewNote(queries *db.Queries, templates *templateSet, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("preview") == fmt.Sprint(true) {
			templates.render(w, "preview.html", r.FormValue("body"))
			return
		}

		if v := r.Header.Get("sec-fetch-site"); v != "same-origin" && v != "none" {
			slog.Error("invalid sec-fetch-site value", "sec-fetch-site", v)
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		id := uuid.New().String()
		if err := queries.CreateNote(r.Context(), id, r.FormValue("body"), time.Now()); err != nil {
			panic(err)
		}

		http.Redirect(w, r, baseURL.JoinPath("note", id).String(), http.StatusSeeOther)
	})
}
