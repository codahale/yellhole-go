package main

import (
	"net/http"

	"github.com/codahale/yellhole-go/db"
)

type authController struct {
	config  *config
	queries *db.Queries
}

func newAuthController(config *config, queries *db.Queries) *authController {
	return &authController{config, queries}
}

func (ac *authController) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// TODO get current year
	// TODO check for existing passkey
	// TODO ensure session isn't authenticated
	http.NotFound(w, r)
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
	// TODO get current year
	http.NotFound(w, r)
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
