package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/google/uuid"
)

type adminController struct {
	config    *Config
	queries   *db.Queries
	templates *templateSet
}

func newAdminController(config *Config, queries *db.Queries, templates *templateSet) *adminController {
	return &adminController{config, queries, templates}
}

func (ac *adminController) adminPage(w http.ResponseWriter, r *http.Request) {
	images, err := ac.queries.RecentImages(r.Context(), 10)
	if err != nil {
		panic(err)
	}

	ac.templates.render(w, "new.html", images)
}

func (ac *adminController) newNote(w http.ResponseWriter, r *http.Request) {
	if r.FormValue("preview") == fmt.Sprint(true) {
		ac.templates.render(w, "preview.html", r.FormValue("body"))
		return
	}

	if v := r.Header.Get("sec-fetch-site"); v != "same-origin" && v != "none" {
		slog.Error("invalid sec-fetch-site value", "sec-fetch-site", v)
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := uuid.New().String()
	if err := ac.queries.CreateNote(r.Context(), id, r.FormValue("body"), time.Now()); err != nil {
		panic(err)
	}

	http.Redirect(w, r, ac.config.BaseURL.JoinPath("note", id).String(), http.StatusSeeOther)
}
