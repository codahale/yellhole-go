package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

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
	registered, err := ac.queries.HasPasskey(r.Context())
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

	if err := view.Render(w, "register.html", struct{ Config *config.Config }{ac.config}); err != nil {
		panic(err)
	}
}

func (ac *authController) RegisterStart(w http.ResponseWriter, r *http.Request) {
	passkeyIDs, err := ac.queries.PasskeyIDs(r.Context())
	if err != nil {
		panic(err)
	}

	challenge, err := webauthn.NewRegistrationChallenge(ac.config, passkeyIDs)
	if err != nil {
		panic(err)
	}

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
	registered, err := ac.queries.HasPasskey(r.Context())
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

	if err := view.Render(w, "login.html", struct{ Config *config.Config }{ac.config}); err != nil {
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

	passkeyIDs, err := ac.queries.PasskeyIDs(r.Context())
	if err != nil {
		panic(err)
	}

	challenge, err := webauthn.NewLoginChallenge(ac.config, passkeyIDs)
	if err != nil {
		panic(err)
	}

	challengeID := uuid.New()
	if err := ac.queries.CreateChallenge(r.Context(), db.CreateChallengeParams{
		ChallengeID: challengeID.String(),
		Bytes:       challenge.Challenge,
	}); err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")
	http.SetCookie(w, &http.Cookie{
		Name:     "challengeID",
		Value:    challengeID.String(),
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   5 * 60, // 5 minutes
	})
	if err := json.NewEncoder(w).Encode(&challenge); err != nil {
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

	cookie, err := r.Cookie("challengeID")
	if err != nil {
		panic(err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "challengeID",
		Value:    "",
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
	})

	// Get and remove the challenge value from the database.
	challenge, err := ac.queries.DeleteChallenge(r.Context(), cookie.Value)
	if err != nil {
		panic(err)
	}

	var resp webauthn.LoginResponse
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		panic(err)
	}

	// Find the passkey by ID.
	passkeySPKI, err := ac.queries.FindPasskey(r.Context(), resp.RawID)
	if err != nil {
		panic(err)
	}

	if err := resp.Validate(ac.config, passkeySPKI, challenge); err != nil {
		panic(err)
	}

	sessionID := uuid.New()
	if err := ac.queries.CreateSession(r.Context(), sessionID.String()); err != nil {
		panic(err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sessionID",
		Value:    sessionID.String(),
		Path:     ac.config.BaseURL.Path,
		HttpOnly: true,
		Secure:   ac.config.BaseURL.Scheme == "https",
		SameSite: http.SameSiteStrictMode,
		MaxAge:   7 * 24 * 60 * 60, // 7 weeks
	})

	w.WriteHeader(http.StatusOK)
}

func isAuthenticated(r *http.Request, queries *db.Queries) (bool, error) {
	cookie, err := r.Cookie("sessionID")
	if err != nil && !errors.Is(err, http.ErrNoCookie) {
		panic(err)
	}

	if cookie == nil {
		return false, nil
	}

	auth, err := queries.SessionExists(r.Context(), cookie.Value)
	if err != nil {
		panic(err)
	}

	return auth, err
}

func purgeOldSessionsInBackground(queries *db.Queries) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		res, err := queries.PurgeSessions(context.Background())
		if err != nil {
			slog.Error("error purging old sessions", "err", err)
			continue
		}

		n, err := res.RowsAffected()
		if err != nil {
			slog.Error("error purging old sessions", "err", err)
			continue
		}
		slog.Info("purged old sessions", "count", n)
	}
}
