package main

import (
	"context"
	"database/sql"
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
	sloghttp "github.com/samber/slog-http"
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

	if err := view.Render(w, "auth/register.html", struct{ Config *config.Config }{ac.config}); err != nil {
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
		CreatedAt:     time.Now(),
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

	passkeyIDs, err := ac.queries.PasskeyIDs(r.Context())
	if err != nil {
		panic(err)
	}

	challenge, err := webauthn.NewLoginChallenge(ac.config, passkeyIDs)
	if err != nil {
		panic(err)
	}

	if err := ac.queries.CreateChallenge(r.Context(), db.CreateChallengeParams{
		ChallengeID: challenge.ChallengeID,
		Bytes:       challenge.Challenge,
		CreatedAt:   time.Now(),
	}); err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")
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

	// Decode the body.
	var resp webauthn.LoginResponse
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
		panic(err)
	}

	// Get and remove the challenge value from the database.
	challenge, err := ac.queries.DeleteChallenge(r.Context(), db.DeleteChallengeParams{
		ChallengeID: resp.ChallengeID,
		CreatedAt:   time.Now().Add(-5 * time.Minute),
	})
	if errors.Is(err, sql.ErrNoRows) {
		slog.Error("invalid challenge", "id", sloghttp.GetRequestID(r))
		http.Error(w, "Invalid challenge", http.StatusBadRequest)
		return
	} else if err != nil {
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

	sessionID := uuid.NewString()
	if err := ac.queries.CreateSession(r.Context(), db.CreateSessionParams{
		SessionID: sessionID,
		CreatedAt: time.Now(),
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

	auth, err := queries.SessionExists(r.Context(), db.SessionExistsParams{
		SessionID: cookie.Value,
		CreatedAt: time.Now().AddDate(0, 0, -7),
	})
	if err != nil {
		panic(err)
	}

	return auth, err
}

func purgeOldRows(queries *db.Queries, ticker *time.Ticker) {
	for range ticker.C {
		purgeOldSessions(queries)
		purgeOldChallenges(queries)
	}
}

func purgeOldSessions(queries *db.Queries) {
	res, err := queries.PurgeSessions(context.Background(), time.Now().AddDate(0, 0, -7))
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

func purgeOldChallenges(queries *db.Queries) {
	res, err := queries.PurgeChallenges(context.Background(), time.Now().Add(-5*time.Minute))
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
