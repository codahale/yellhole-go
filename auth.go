package main

import (
	"net/http"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/view"
)

type authController struct {
	config  *config.Config
	queries *db.Queries
}

func newAuthController(config *config.Config, queries *db.Queries) *authController {
	return &authController{config, queries}
}

func (ac *authController) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// TODO check for existing passkey
	// TODO ensure session isn't authenticated

	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "register.html", nil); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterStart(w http.ResponseWriter, r *http.Request) {
	// TODO get passkey IDs
	http.NotFound(w, r)
}

func (ac *authController) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	// TODO insert passkey into DB
	http.NotFound(w, r)
}

func (ac *authController) LoginPage(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	w.Header().Set("content-type", "text/html")
	if err := view.Render(w, "login.html", nil); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginStart(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	// TODO insert challenge into DB
	http.NotFound(w, r)
}

func (ac *authController) LoginFinish(w http.ResponseWriter, r *http.Request) {
	// TODO ensure session isn't authenticated
	// TODO delete challenge from DB
	// TODO insert session into DB
	http.NotFound(w, r)
}
