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
	config   *config.Config
	queries  *db.Queries
	webauthn *webauthn.WebAuthn
}

func newAuthController(config *config.Config, queries *db.Queries) *authController {
	return &authController{config, queries, &webauthn.WebAuthn{
		Config: &webauthn.Config{
			RPID:          config.BaseURL.Hostname(),
			RPDisplayName: config.Title,
			RPOrigins:     []string{strings.TrimRight(config.BaseURL.String(), "/")},
		},
	}}
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

	// Render the page.
	if err := view.Render(w, "auth/register.html", struct{ Config *config.Config }{ac.config}); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterStart(w http.ResponseWriter, r *http.Request) {
	// Create a new webauthn attestation challenge.
	creation, session, err := ac.webauthn.BeginRegistration(webauthnUser{
		name:        ac.config.Author,
		credentials: []webauthn.Credential{},
	})
	if err != nil {
		panic(err)
	}

	// Store the webauthn session data in the DB.
	registrationSessionID := uuid.NewString()
	registrationSessionJSON, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}
	if err := ac.queries.CreateWebauthnSession(r.Context(), db.CreateWebauthnSessionParams{
		WebauthnSessionID: registrationSessionID,
		SessionData:       registrationSessionJSON,
		CreatedAt:         time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	// Pass the webauthn session ID to the browser via a cookie.
	http.SetCookie(w, &http.Cookie{
		Name:     "registrationSessionID",
		Value:    registrationSessionID,
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60,
	})

	// Render the attestation challenge as a JSON object.
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

	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(session), &sessionData); err != nil {
		panic(err)
	}

	// Validate the attestation response.
	cred, err := ac.webauthn.FinishRegistration(
		webauthnUser{
			name:        ac.config.Author,
			credentials: []webauthn.Credential{},
		},
		sessionData,
		r)
	if err != nil {
		slog.Error("unable to finish passkey registration", "err", err, "id", sloghttp.GetRequestID(r))
		w.Header().Set("content-type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]bool{"verified": false}); err != nil {
			panic(err)
		}
		return
	}

	credJSON, err := json.Marshal(cred)
	if err != nil {
		panic(err)
	}

	if err := ac.queries.CreateWebauthnCredential(r.Context(), db.CreateWebauthnCredentialParams{
		CredentialData: credJSON,
		CreatedAt:      time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"verified": true}); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginPage(w http.ResponseWriter, r *http.Request) {
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

	if err := view.Render(w, "auth/login.html", struct{ Config *config.Config }{ac.config}); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginStart(w http.ResponseWriter, r *http.Request) {
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Fetch all credentials from the database.
	credBytes, err := ac.queries.WebauthnCredentials(r.Context())
	if err != nil {
		panic(err)
	}

	// Decode them from JSON.
	credentials := make([]webauthn.Credential, len(credBytes))
	for i := range credBytes {
		if err := json.Unmarshal(credBytes[i], &credentials[i]); err != nil {
			panic(err)
		}
	}

	// Create a webauthn login challenge.
	assertion, session, err := ac.webauthn.BeginLogin(webauthnUser{
		name:        ac.config.Author,
		credentials: credentials,
	})
	if err != nil {
		panic(err)
	}

	sessionID := uuid.NewString()
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		panic(err)
	}
	if err := ac.queries.CreateWebauthnSession(r.Context(), db.CreateWebauthnSessionParams{
		WebauthnSessionID: sessionID,
		SessionData:       sessionJSON,
		CreatedAt:         time.Now().Unix(),
	}); err != nil {
		panic(err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "loginSessionID",
		Value:    sessionID,
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   60,
	})

	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(assertion); err != nil {
		panic(err)
	}
}

func (ac *authController) LoginFinish(w http.ResponseWriter, r *http.Request) {
	auth, err := isAuthenticated(r, ac.queries)
	if err != nil {
		panic(err)
	}

	if auth {
		http.Redirect(w, r, ac.config.BaseURL.JoinPath("admin").String(), http.StatusSeeOther)
		return
	}

	// Fetch all credentials from the database.
	credBytes, err := ac.queries.WebauthnCredentials(r.Context())
	if err != nil {
		panic(err)
	}

	// Decode them from JSON.
	credentials := make([]webauthn.Credential, len(credBytes))
	for i := range credBytes {
		if err := json.Unmarshal(credBytes[i], &credentials[i]); err != nil {
			panic(err)
		}
	}

	// Find, delete, and decode the webauthn session data.
	webauthnSessionID, err := r.Cookie("loginSessionID")
	if err != nil {
		panic(err)
	}

	session, err := ac.queries.DeleteWebauthnSession(r.Context(), db.DeleteWebauthnSessionParams{
		WebauthnSessionID: webauthnSessionID.Value,
		CreatedAt:         time.Now().Add(-1 * time.Minute).Unix(),
	})
	if err != nil {
		panic(err)
	}

	var sessionData webauthn.SessionData
	if err := json.Unmarshal([]byte(session), &sessionData); err != nil {
		panic(err)
	}

	// Validate the webauthn challenge.
	_, err = ac.webauthn.FinishLogin(
		webauthnUser{
			name:        ac.config.Author,
			credentials: credentials,
		},
		sessionData,
		r,
	)
	if err != nil {
		// Return an error object if the login attempt failed.
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
	http.SetCookie(w, &http.Cookie{
		Name:     "sessionID",
		Value:    sessionID,
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 weeks
	})

	// Return a success object.
	w.Header().Set("content-type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]bool{"verified": true}); err != nil {
		panic(err)
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
	credentials []webauthn.Credential
}

func (w webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return w.credentials
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
