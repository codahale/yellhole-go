package main

import (
	"encoding/json"
	"net/http"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/view"
	"github.com/codahale/yellhole-go/webauthn"
	"github.com/google/uuid"
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
	passkeyIDs, err := ac.queries.PasskeyIDs(r.Context())
	if err != nil {
		panic(err)
	}

	challenge := webauthn.NewRegistrationChallenge(ac.config, uuid.New(), passkeyIDs)
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(&challenge); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	var response webauthn.RegistrationResponse
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
		panic(err)
	}

	passkeyID, publicKeySPKI, err := response.Validate(ac.config)
	if err != nil {
		panic(err)
	}

	if err := ac.queries.CreatePasskey(r.Context(), db.CreatePasskeyParams{
		PasskeyID:     passkeyID,
		PublicKeySPKI: publicKeySPKI,
	}); err != nil {
		panic(err)
	}
	w.WriteHeader(http.StatusOK)
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
