package main

import (
	"net/http"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
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
	// TODO get current year
	// TODO get recent images
	http.NotFound(w, r)
}

func (ac *adminController) NewNote(w http.ResponseWriter, r *http.Request) {
	// TODO authenticate session
	// TODO insert note into DB
	http.NotFound(w, r)
}
