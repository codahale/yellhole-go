package main

import (
	"net/http"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/view"
)

type adminController struct {
	config  *config.Config
	queries *db.Queries
}

func newAdminController(config *config.Config, queries *db.Queries) *adminController {
	return &adminController{config, queries}
}

func (ac *adminController) AdminPage(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session

	images, err := ac.queries.RecentImages(r.Context(), 10)
	if err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "new.html", struct {
		Config *config.Config
		Images []db.Image
	}{
		ac.config,
		images,
	}); err != nil {
		panic(err)
	}
}

func (ac *adminController) NewNote(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO insert note into DB
	http.NotFound(w, r)
}
