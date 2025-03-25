package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/config"
	"github.com/codahale/yellhole-go/db"
	"github.com/codahale/yellhole-go/view"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	sloghttp "github.com/samber/slog-http"
)

type authController struct {
	config    *config.Config
	queries   *db.Queries
	webauthn  *webauthn.WebAuthn
	templates *view.TemplateSet
}

func newAuthController(config *config.Config, queries *db.Queries, templates *view.TemplateSet) *authController {
	webauthn, err := webauthn.New(&webauthn.Config{
		RPID:          config.BaseURL.Hostname(),
		RPDisplayName: config.Title,
		RPOrigins:     []string{strings.TrimRight(config.BaseURL.String(), "/")},
	})
	if err != nil {
		panic(err)
	}
	return &authController{config, queries, webauthn, templates}
}

func (ac *authController) RegisterPage(w http.ResponseWriter, r *http.Request) {
	// Ensure we only register one passkey.
	registered, err := ac.queries.HasWebauthnCredential(r.Context())
	if err != nil {
		panic(err)
	}
	if registered {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("login").String(), http.StatusSeeOther)
		return
	}

	// Ensure session isn't authenticated.
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}
	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Respond with the login page.
	if err := ac.templates.Render(w, "auth/register.html", struct{ Config *config.Config }{ac.config}); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterStart(w http.ResponseWriter, r *http.Request) {
	// Create a new webauthn attestation challenge.
	creation, session, err := ac.webauthn.BeginRegistration(
		webauthnUser{
			name:        ac.config.Author,
			credentials: []*db.JSONCredential{},
		},
		webauthn.WithCredentialParameters(webauthn.CredentialParametersRecommendedL3()),
	)
	if err != nil {
		panic(err)
	}

	// Store the webauthn session data in the DB.
	registrationSessionID := uuid.NewString()
	if err := ac.queries.CreateWebauthnSession(r.Context(), db.CreateWebauthnSessionParams{
		WebauthnSessionID: registrationSessionID,
		SessionData:       &db.JSONSessionData{Data: *session},
		CreatedAt:         time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	// Set a registration session ID cookie.
	http.SetCookie(w, ac.secureCookie("registrationSessionID", registrationSessionID, 60))

	// Respond with the attestation challenge.
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(creation); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterFinish(w http.ResponseWriter, r *http.Request) {
	// Find the webauthn session ID.
	registrationSessionID, err := r.Cookie("registrationSessionID")
	if err != nil {
		panic(err)
	}

	// Read, delete, and decode the webauthn session data.
	session, err := ac.queries.DeleteWebauthnSession(r.Context(), db.DeleteWebauthnSessionParams{
		WebauthnSessionID: registrationSessionID.Value,
		CreatedAt:         time.Now().Add(-1 * time.Minute).Unix(),
	})
	if err != nil {
		panic(err)
	}

	// Validate the attestation response.
	cred, err := ac.webauthn.FinishRegistration(
		webauthnUser{
			name:        ac.config.Author,
			credentials: []*db.JSONCredential{},
		},
		session.Data,
		r)
	if err != nil {
		// If the attestation is invalid, respond with verified=false.
		slog.Error("unable to finish passkey registration", "err", err, "id", sloghttp.GetRequestID(r))
		w.Header().Set("content-type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]bool{"verified": false}); err != nil {
			panic(err)
		}
		return
	}

	// Store the new credential in the database.
	if err := ac.queries.CreateWebauthnCredential(r.Context(), db.CreateWebauthnCredentialParams{
		CredentialData: &db.JSONCredential{Data: cred},
		CreatedAt:      time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	// Delete the registration session ID cookie.
	http.SetCookie(w, ac.secureCookie("registrationSessionID", "", -1))

	// Respond with verified=true.
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"verified": true}); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginPage(w http.ResponseWriter, r *http.Request) {
	// Redirect to registration if no credentials exist.
	registered, err := ac.queries.HasWebauthnCredential(r.Context())
	if err != nil {
		panic(err)
	}
	if !registered {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("register").String(), http.StatusSeeOther)
		return
	}

	// Ensure session isn't authenticated.
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Respond with the login page.
	if err := ac.templates.Render(w, "auth/login.html", struct{ Config *config.Config }{ac.config}); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginStart(w http.ResponseWriter, r *http.Request) {
	// Ensure request isn't already authenticated.
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Fetch all credentials from the database.
	credentials, err := ac.queries.WebauthnCredentials(r.Context())
	if err != nil {
		panic(err)
	}

	// Create a webauthn login challenge.
	assertion, session, err := ac.webauthn.BeginLogin(webauthnUser{
		name:        ac.config.Author,
		credentials: credentials,
	})
	if err != nil {
		panic(err)
	}

	// Store the challenge in the database.
	loginSessionID := uuid.NewString()
	if err := ac.queries.CreateWebauthnSession(r.Context(), db.CreateWebauthnSessionParams{
		WebauthnSessionID: loginSessionID,
		SessionData:       &db.JSONSessionData{Data: *session},
		CreatedAt:         time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	// Assign a login session ID cookie.
	http.SetCookie(w, ac.secureCookie("loginSessionID", loginSessionID, 60))

	// Respond with the login challenge.
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(assertion); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginFinish(w http.ResponseWriter, r *http.Request) {
	// Ensure request isn't already authenticated.
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Fetch all credentials from the database.
	credentials, err := ac.queries.WebauthnCredentials(r.Context())
	if err != nil {
		panic(err)
	}

	// Get the ID of the login session.
	loginSessionID, err := r.Cookie("loginSessionID")
	if err != nil {
		panic(err)
	}

	// Find and delete it from the database.
	session, err := ac.queries.DeleteWebauthnSession(r.Context(), db.DeleteWebauthnSessionParams{
		WebauthnSessionID: loginSessionID.Value,
		CreatedAt:         time.Now().Add(-1 * time.Minute).Unix(),
	})
	if err != nil {
		panic(err)
	}

	// Validate the webauthn challenge.
	_, err = ac.webauthn.FinishLogin(
		webauthnUser{
			name:        ac.config.Author,
			credentials: credentials,
		},
		session.Data,
		r,
	)
	if err != nil {
		// Respond with verified=false if the challenge response was invalid.
		slog.Error("unable to finish passkey login", "err", err, "id", sloghttp.GetRequestID(r))
		w.Header().Set("content-type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]bool{"verified": false}); err != nil {
			panic(err)
		}
		return
	}

	// Create a new web session and assign a session cookie.
	sessionID := uuid.NewString()
	if err := ac.queries.CreateSession(r.Context(), db.CreateSessionParams{
		SessionID: sessionID,
		CreatedAt: time.Now().Unix(),
	}); err != nil {
		panic(err)
	}
	http.SetCookie(w, ac.secureCookie("sessionID", sessionID, 60*60*24*7))

	// Delete the login session ID cookie.
	http.SetCookie(w, ac.secureCookie("loginSessionID", "", -1))

	// Respond with verified=true.
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"verified": true}); err != nil {
		panic(err)
	}
}

func (ac *authController) secureCookie(name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

func isAuthenticated(r *http.Request, queries *db.Queries) (bool, error) {
	cookie, err := r.Cookie("sessionID")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		panic(err)
	}

	if cookie == nil {
		return false, nil
	}

	auth, err := queries.SessionExists(r.Context(), db.SessionExistsParams{
		SessionID: cookie.Value,
		CreatedAt: time.Now().AddDate(0, 0, -7).Unix(),
	})
	if err != nil {
		panic(err)
	}

	return auth, err
}

func purgeOldRows(queries *db.Queries, ticker *time.Ticker) {
	for range ticker.C {
		purgeOldSessions(queries)
		purgeOldWebauthnSessions(queries)
	}
}

func purgeOldSessions(queries *db.Queries) {
	res, err := queries.PurgeSessions(context.Background(), time.Now().AddDate(0, 0, -7).Unix())
	if err != nil {
		slog.Error("error purging old sessions", "err", err)
		return
	}

	n, err := res.RowsAffected()
	if err != nil {
		slog.Error("error purging old sessions", "err", err)
		return
	}
	slog.Info("purged old sessions", "count", n)
}

func purgeOldWebauthnSessions(queries *db.Queries) {
	res, err := queries.PurgeWebauthnSessions(context.Background(), time.Now().Add(-5*time.Minute).Unix())
	if err != nil {
		slog.Error("error purging old challenge", "err", err)
		return
	}

	n, err := res.RowsAffected()
	if err != nil {
		slog.Error("error purging old challenge", "err", err)
		return
	}
	slog.Info("purged old challenges", "count", n)
}

type webauthnUser struct {
	name        string
	credentials []*db.JSONCredential
}

func (w webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, len(w.credentials))
	for i := range w.credentials {
		creds[i] = *w.credentials[i].Data
	}
	return creds
}

func (w webauthnUser) WebAuthnDisplayName() string {
	return w.name
}

func (w webauthnUser) WebAuthnID() []byte {
	return make([]byte, 16)
}

func (w webauthnUser) WebAuthnName() string {
	return w.name
}

var _ webauthn.User = webauthnUser{}
