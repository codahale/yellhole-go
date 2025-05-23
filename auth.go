package main

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/codahale/yellhole-go/db"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	sloghttp "github.com/samber/slog-http"
)

func newWebauthn(title string, baseURL *url.URL) (*webauthn.WebAuthn, error) {
	return webauthn.New(&webauthn.Config{
		RPID:          baseURL.Hostname(),
		RPDisplayName: title,
		RPOrigins:     []string{strings.TrimRight(baseURL.String(), "/")},
	})
}

func handleRegisterPage(queries *db.Queries, t *template.Template, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure we only register one passkey.
		registered, err := queries.HasWebauthnCredential(r.Context())
		if err != nil {
			panic(err)
		}
		if registered {
			http.Redirect(w, r, baseURL.JoinPath("login").String(), http.StatusSeeOther)
			return
		}

		// Ensure session isn't authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			panic(err)
		}
		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return
		}

		// Respond with the register page.
		htmlResponse(w, t, "register.gohtml", nil)
	})
}

func handleRegisterStart(queries *db.Queries, author, title string, baseURL *url.URL) http.Handler {
	webAuthn, err := newWebauthn(title, baseURL)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new webauthn attestation challenge.
		creation, session, err := webAuthn.BeginRegistration(
			webauthnUser{author, []*db.JSONCredential{}},
			webauthn.WithCredentialParameters(webauthn.CredentialParametersRecommendedL3()),
		)
		if err != nil {
			panic(err)
		}

		// Store the webauthn session data in the DB.
		regSessionID := uuid.NewString()
		if err := queries.CreateWebauthnSession(r.Context(), regSessionID, db.JSON(*session), time.Now()); err != nil {
			panic(err)
		}

		// Set a registration session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "registrationSessionID", regSessionID, 60))

		// Respond with the attestation challenge.
		jsonResponse(w, creation)
	})
}

func handleRegisterFinish(queries *db.Queries, author, title string, baseURL *url.URL) http.Handler {
	webAuthn, err := newWebauthn(title, baseURL)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Find the webauthn session ID.
		regSessionID, err := r.Cookie("registrationSessionID")
		if err != nil {
			panic(err)
		}

		// Read, delete, and decode the webauthn session data.
		session, err := queries.DeleteWebauthnSession(r.Context(), regSessionID.Value, time.Now().Add(-1*time.Minute))
		if err != nil {
			panic(err)
		}

		// Validate the attestation response.
		cred, err := webAuthn.FinishRegistration(webauthnUser{author, []*db.JSONCredential{}}, session.Data, r)
		if err != nil {
			// If the attestation is invalid, respond with verified=false.
			slog.Error("unable to finish passkey registration", "err", err, "id", sloghttp.GetRequestID(r))
			jsonResponse(w, map[string]bool{"verified": false})
			return
		}

		// Store the new credential in the database.
		if err := queries.CreateWebauthnCredential(r.Context(), db.JSON(cred), time.Now()); err != nil {
			panic(err)
		}

		// Delete the registration session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "registrationSessionID", "", -1))

		// Respond with verified=true.
		jsonResponse(w, map[string]bool{"verified": true})
	})
}

func handleLoginPage(queries *db.Queries, t *template.Template, baseURL *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Redirect to registration if no credentials exist.
		registered, err := queries.HasWebauthnCredential(r.Context())
		if err != nil {
			panic(err)
		}
		if !registered {
			http.Redirect(w, r, baseURL.JoinPath("register").String(), http.StatusSeeOther)
			return
		}

		// Ensure session isn't authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			panic(err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return
		}

		// Respond with the login page.
		htmlResponse(w, t, "login.gohtml", nil)
	})
}

func handleLoginStart(queries *db.Queries, author, title string, baseURL *url.URL) http.Handler {
	webAuthn, err := newWebauthn(title, baseURL)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure request isn't already authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			panic(err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return
		}

		// Fetch all credentials from the database.
		credentials, err := queries.WebauthnCredentials(r.Context())
		if err != nil {
			panic(err)
		}

		// Create a webauthn login challenge.
		assertion, session, err := webAuthn.BeginLogin(webauthnUser{author, credentials})
		if err != nil {
			panic(err)
		}

		// Store the challenge in the database.
		loginSessionID := uuid.NewString()
		if err := queries.CreateWebauthnSession(r.Context(), loginSessionID, db.JSON(*session), time.Now()); err != nil {
			panic(err)
		}

		// Assign a login session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "loginSessionID", loginSessionID, 60))

		// Respond with the login challenge.
		jsonResponse(w, assertion)
	})
}

func handleLoginFinish(queries *db.Queries, author, title string, baseURL *url.URL) http.Handler {
	webAuthn, err := newWebauthn(title, baseURL)
	if err != nil {
		panic(err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure request isn't already authenticated.
		auth, err := isAuthenticated(r, queries)
		if err != nil {
			panic(err)
		}

		if auth {
			http.Redirect(w, r, baseURL.JoinPath("admin").String(), http.StatusSeeOther)
			return
		}

		// Fetch all credentials from the database.
		credentials, err := queries.WebauthnCredentials(r.Context())
		if err != nil {
			panic(err)
		}

		// Get the ID of the login session.
		loginSessionID, err := r.Cookie("loginSessionID")
		if err != nil {
			panic(err)
		}

		// Find and delete it from the database.
		session, err := queries.DeleteWebauthnSession(r.Context(), loginSessionID.Value, time.Now().Add(-1*time.Minute))
		if err != nil {
			panic(err)
		}

		// Validate the webauthn challenge.
		_, err = webAuthn.FinishLogin(webauthnUser{author, credentials}, session.Data, r)
		if err != nil {
			// Respond with verified=false if the challenge response was invalid.
			slog.Error("unable to finish passkey login", "err", err, "id", sloghttp.GetRequestID(r))
			jsonResponse(w, map[string]bool{"verified": false})
			return
		}

		// Create a new web session and assign a session cookie.
		sessionID := uuid.NewString()
		if err := queries.CreateSession(r.Context(), sessionID, time.Now()); err != nil {
			panic(err)
		}
		http.SetCookie(w, secureCookie(baseURL, "sessionID", sessionID, 60*60*24*7))

		// Delete the login session ID cookie.
		http.SetCookie(w, secureCookie(baseURL, "loginSessionID", "", -1))

		// Respond with verified=true.
		jsonResponse(w, map[string]bool{"verified": true})
	})
}

func requireAuthentication(queries *db.Queries, h http.Handler, baseURL *url.URL, prefix string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, prefix) {
			auth, err := isAuthenticated(r, queries)
			if err != nil {
				panic(err)
			}

			if !auth {
				slog.Info("unauthenticated request", "uri", r.RequestURI, "id", sloghttp.GetRequestID(r))
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
		panic(err)
	}

	if cookie == nil {
		return false, nil
	}

	auth, err := queries.SessionExists(r.Context(), cookie.Value, time.Now().AddDate(0, 0, -7))
	if err != nil {
		panic(err)
	}

	return auth, err
}

func purgeOldRows(ctx context.Context, queries *db.Queries, ticker *time.Ticker) {
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			purgeOldSessions(ctx, queries)
			purgeOldWebauthnSessions(ctx, queries)
		}
	}
}

func purgeOldSessions(ctx context.Context, queries *db.Queries) {
	res, err := queries.PurgeSessions(ctx, time.Now().AddDate(0, 0, -7))
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

func purgeOldWebauthnSessions(ctx context.Context, queries *db.Queries) {
	res, err := queries.PurgeWebauthnSessions(ctx, time.Now().Add(-5*time.Minute))
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
