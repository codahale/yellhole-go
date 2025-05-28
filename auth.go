package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	sloghttp "github.com/samber/slog-http"
)

func handleRegisterPage(queries *db.Queries, t *template.Template, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Ensure we only register one passkey.
		registered, err := queries.HasWebauthnCredential(r.Context())
		if err != nil {
			return fmt.Errorf("failed to check for existing webauthn credential: %w", err)
		}
		if registered {
			http.Redirect(w, r, baseURL.JoinPath("login").String(), http.StatusSeeOther)
			return nil
		}

		// Ensure the session isn't authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			return fmt.Errorf("failed to check authentication status in register page: %w", err)
		}
		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return nil
		}

		// Respond with the register page.
		return htmlResponse(w, t, "register.gohtml", nil)
	}
}

func handleRegisterStart(queries *db.Queries, author, title string, baseURL *url.URL) appHandler {
	webAuthn := newWebauthn(title, baseURL)

	return func(w http.ResponseWriter, r *http.Request) error {
		// Create a new webauthn attestation challenge.
		creation, session, err := webAuthn.BeginRegistration(
			webauthnUser{author, []*db.JSONCredential{}},
			webauthn.WithCredentialParameters(webauthn.CredentialParametersRecommendedL3()),
		)
		if err != nil {
			return fmt.Errorf("failed to begin webauthn registration: %w", err)
		}

		// Store the webauthn session data in the DB.
		regSessionID := uuid.NewString()
		if err := queries.CreateWebauthnSession(r.Context(), regSessionID, db.JSON(*session), time.Now()); err != nil {
			return fmt.Errorf("failed to create webauthn session: %w", err)
		}

		// Set a registration session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "registrationSessionID", regSessionID, 60))

		// Respond with the attestation challenge.
		return jsonResponse(w, creation)
	}
}

func handleRegisterFinish(queries *db.Queries, author, title string, baseURL *url.URL) appHandler {
	webAuthn := newWebauthn(title, baseURL)

	return func(w http.ResponseWriter, r *http.Request) error {
		// Find the webauthn session ID.
		regSessionID, err := r.Cookie("registrationSessionID")
		if err != nil {
			return fmt.Errorf("failed to get registration session cookie: %w", err)
		}

		// Read, delete, and decode the webauthn session data.
		session, err := queries.DeleteWebauthnSession(r.Context(), regSessionID.Value, time.Now().Add(-1*time.Minute))
		if err != nil {
			return fmt.Errorf("failed to retrieve and delete webauthn session: %w", err)
		}

		// Validate the attestation response.
		cred, err := webAuthn.FinishRegistration(webauthnUser{author, []*db.JSONCredential{}}, session.Data, r)
		if err != nil {
			// If the attestation is invalid, respond with verified=false.
			slog.ErrorContext(r.Context(), "unable to finish passkey registration", "err", err, "id", sloghttp.GetRequestID(r))
			return jsonResponse(w, map[string]bool{"verified": false})
		}

		// Store the new credential in the database.
		if err := queries.CreateWebauthnCredential(r.Context(), db.JSON(cred), time.Now()); err != nil {
			return fmt.Errorf("failed to create webauthn credential: %w", err)
		}

		// Delete the registration session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "registrationSessionID", "", -1))

		// Respond with verified=true.
		return jsonResponse(w, map[string]bool{"verified": true})
	}
}

func handleLoginPage(queries *db.Queries, t *template.Template, baseURL *url.URL) appHandler {
	return func(w http.ResponseWriter, r *http.Request) error {
		// Redirect to registration if no credentials exist.
		registered, err := queries.HasWebauthnCredential(r.Context())
		if err != nil {
			return fmt.Errorf("failed to check for existing webauthn credential in login page: %w", err)
		}
		if !registered {
			http.Redirect(w, r, baseURL.JoinPath("register").String(), http.StatusSeeOther)
			return nil
		}

		// Ensure the session isn't authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			return fmt.Errorf("failed to check authentication status in login page: %w", err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return nil
		}

		// Respond with the login page.
		return htmlResponse(w, t, "login.gohtml", nil)
	}
}

func handleLoginStart(queries *db.Queries, author, title string, baseURL *url.URL) appHandler {
	webAuthn := newWebauthn(title, baseURL)

	return func(w http.ResponseWriter, r *http.Request) error {
		// Ensure the request isn't already authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			return fmt.Errorf("failed to check authentication status in login start: %w", err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return nil
		}

		// Fetch all credentials from the database.
		credentials, err := queries.WebauthnCredentials(r.Context())
		if err != nil {
			return fmt.Errorf("failed to retrieve webauthn credentials: %w", err)
		}

		// Create a webauthn login challenge.
		assertion, session, err := webAuthn.BeginLogin(webauthnUser{author, credentials})
		if err != nil {
			return fmt.Errorf("failed to begin webauthn login: %w", err)
		}

		// Store the challenge in the database.
		loginSessionID := uuid.NewString()
		if err := queries.CreateWebauthnSession(r.Context(), loginSessionID, db.JSON(*session), time.Now()); err != nil {
			return fmt.Errorf("failed to create webauthn login session: %w", err)
		}

		// Assign a login session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "loginSessionID", loginSessionID, 60))

		// Respond with the login challenge.
		return jsonResponse(w, assertion)
	}
}

func handleLoginFinish(queries *db.Queries, author, title string, baseURL *url.URL) appHandler {
	webAuthn := newWebauthn(title, baseURL)

	return func(w http.ResponseWriter, r *http.Request) error {
		// Ensure the request isn't already authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			return fmt.Errorf("failed to check authentication status in login finish: %w", err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return nil
		}

		// Fetch all credentials from the database.
		credentials, err := queries.WebauthnCredentials(r.Context())
		if err != nil {
			return fmt.Errorf("failed to retrieve webauthn credentials in login finish: %w", err)
		}

		// Get the ID of the login session.
		loginSessionID, err := r.Cookie("loginSessionID")
		if err != nil {
			return fmt.Errorf("failed to get login session cookie: %w", err)
		}

		// Find and delete it from the database.
		session, err := queries.DeleteWebauthnSession(r.Context(), loginSessionID.Value, time.Now().Add(-1*time.Minute))
		if err != nil {
			return fmt.Errorf("failed to retrieve and delete webauthn login session: %w", err)
		}

		// Validate the webauthn challenge.
		_, err = webAuthn.FinishLogin(webauthnUser{author, credentials}, session.Data, r)
		if err != nil {
			// Respond with verified=false if the challenge response was invalid.
			slog.ErrorContext(r.Context(), "unable to finish passkey login", "err", err, "id", sloghttp.GetRequestID(r))
			return jsonResponse(w, map[string]bool{"verified": false})
		}

		// Create a new web session and assign a session cookie.
		sessionID := uuid.NewString()
		if err := queries.CreateSession(r.Context(), sessionID, time.Now()); err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
		http.SetCookie(w, secureCookie(baseURL, "sessionID", sessionID, 60*60*24*7))

		// Delete the login session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "loginSessionID", "", -1))

		// Respond with verified=true.
		return jsonResponse(w, map[string]bool{"verified": true})
	}
}

func requireAuthentication(queries *db.Queries, h http.Handler, baseURL *url.URL, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, prefix) {
			auth, err := isAuthenticated(r, queries)
			if err != nil {
				slog.ErrorContext(r.Context(), "error handling request", "err", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			if !auth {
				slog.InfoContext(r.Context(), "unauthenticated request", "uri", r.RequestURI, "id", sloghttp.GetRequestID(r))
				http.Redirect(w, r, baseURL.JoinPath("login").String(), http.StatusSeeOther)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func secureCookie(baseURL *url.URL, name, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     baseURL.Path,
		HttpOnly: true,
		Secure:   baseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   maxAge,
	}
}

func isAuthenticated(r *http.Request, queries *db.Queries) (bool, error) {
	cookie, err := r.Cookie("sessionID")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		return false, fmt.Errorf("failed to get session cookie: %w", err)
	}

	if cookie == nil {
		return false, nil
	}

	return queries.SessionExists(r.Context(), cookie.Value, time.Now().AddDate(0, 0, -7))
}

func purgeOldRows(ctx context.Context, queries *db.Queries, ticker *time.Ticker) {
	purge := func(ctx context.Context, name string, expiry time.Time, f func(context.Context, time.Time) (sql.Result, error)) {
		res, err := f(ctx, expiry)
		if err != nil {
			slog.ErrorContext(ctx, "error purging old "+name, "err", err)
			return
		}

		n, err := res.RowsAffected()
		if err != nil {
			slog.ErrorContext(ctx, "error purging old "+name, "err", err)
			return
		}
		slog.InfoContext(ctx, "purged old "+name, "count", n)
	}

	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			purge(ctx, "sessions", time.Now().AddDate(0, 0, -7), queries.PurgeSessions)
			purge(ctx, "challenges", time.Now().Add(-5*time.Minute), queries.PurgeWebauthnSessions)
		}
	}
}

func newWebauthn(title string, baseURL *url.URL) *webauthn.WebAuthn {
	webAuthn, err := webauthn.New(&webauthn.Config{
		RPID:          baseURL.Hostname(),
		RPDisplayName: title,
		RPOrigins:     []string{strings.TrimRight(baseURL.String(), "/")},
	})
	if err != nil {
		// If the configuration is invalid, panic. This will always be a programming error, not a runtime error.
		panic(err)
	}
	return webAuthn
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
