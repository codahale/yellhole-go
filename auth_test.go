package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/codahale/yellhole-go/internal/db"
	"github.com/descope/virtualwebauthn"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/go-cmp/cmp"
)

func TestRegisterPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/register", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestRegisterPageWithCredential(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	if err := app.queries.CreateWebauthnCredential(t.Context(), &db.JSONCredential{
		Data: &webauthn.Credential{ID: []byte("test-id")},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/register", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := resp.Header.Get("Location"), "http://example.com/login"; got != want {
		t.Errorf("resp.Header.Get(\"location\") = %q, want = %q", got, want)
	}
}

func TestRegistrationFlowWithCredential(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	if err := app.queries.CreateWebauthnCredential(t.Context(), &db.JSONCredential{
		Data: &webauthn.Credential{ID: []byte("test-id")},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("POST", "http://example.com/register/start", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestLoginPage(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)
	if err := app.queries.CreateWebauthnCredential(t.Context(), &db.JSONCredential{
		Data: &webauthn.Credential{ID: []byte("test-id")},
	}, time.Now()); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "http://example.com/login", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}
}

func TestLoginPageWithoutCredential(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	req := httptest.NewRequest("GET", "http://example.com/login", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, req)

	resp := w.Result()

	if got, want := resp.StatusCode, http.StatusSeeOther; got != want {
		t.Errorf("resp.StatusCode = %d, want = %d", got, want)
	}

	if got, want := resp.Header.Get("Location"), "http://example.com/register"; got != want {
		t.Errorf("resp.Header.Get(\"location\") = %q, want = %q", got, want)
	}
}

func TestRegistrationAndLoginFlow(t *testing.T) {
	t.Parallel()

	app := newTestApp(t)

	// Request a registration challenge.
	startReq := httptest.NewRequest("POST", "http://example.com/register/start", nil)
	w := httptest.NewRecorder()
	app.ServeHTTP(w, startReq)
	startResp := w.Result()

	if got, want := startResp.StatusCode, http.StatusOK; got != want {
		t.Errorf("startResp.StatusCode = %d, want = %d", got, want)
	}

	var credCreation protocol.CredentialCreation
	if err := json.NewDecoder(startResp.Body).Decode(&credCreation); err != nil {
		t.Fatal(err)
	}
	_ = startResp.Body.Close()

	attestationOptionsStr, err := json.Marshal(credCreation.Response)
	if err != nil {
		t.Fatal(err)
	}

	// Mimic a browser creating an attestation response.
	rp := virtualwebauthn.RelyingParty{Name: "Test Yell", ID: "example.com", Origin: "http://example.com"}
	authenticator := virtualwebauthn.NewAuthenticator()
	credential := virtualwebauthn.NewCredential(virtualwebauthn.KeyTypeEC2)
	attestationOptions, err := virtualwebauthn.ParseAttestationOptions(string(attestationOptionsStr))
	if err != nil {
		t.Fatal(err)
	}
	attestationResponse := virtualwebauthn.CreateAttestationResponse(rp, authenticator, credential, *attestationOptions)

	// Register the passkey.
	finishReq := httptest.NewRequest("POST", "http://example.com/register/finish", nil)
	finishReq.AddCookie(startResp.Cookies()[0])
	finishReq.Header.Set("Content-Type", "application/json")
	finishReq.Body = io.NopCloser(strings.NewReader(attestationResponse))
	w = httptest.NewRecorder()
	app.ServeHTTP(w, finishReq)
	finishResp := w.Result()

	if got, want := finishResp.StatusCode, http.StatusOK; got != want {
		t.Errorf("finishResp.StatusCode = %d, want = %d", got, want)
	}

	var finishResponse map[string]interface{}
	if err := json.NewDecoder(finishResp.Body).Decode(&finishResponse); err != nil {
		t.Fatal(err)
	}
	_ = finishResp.Body.Close()

	if got, want := finishResponse, map[string]interface{}{"verified": true}; !cmp.Equal(got, want) {
		t.Errorf("finishResponse = %v, want = %v", got, want)
	}

	// Request a login challenge.
	startReq = httptest.NewRequest("POST", "http://example.com/login/start", nil)
	w = httptest.NewRecorder()
	app.ServeHTTP(w, startReq)
	startResp = w.Result()

	var credAssertion protocol.CredentialAssertion
	if err := json.NewDecoder(startResp.Body).Decode(&credAssertion); err != nil {
		t.Fatal(err)
	}
	_ = startResp.Body.Close()

	assertionOptionsStr, err := json.Marshal(credAssertion.Response)
	if err != nil {
		t.Fatal(err)
	}

	// Mimic a browser creating an assertion response.
	assertionOptions, err := virtualwebauthn.ParseAssertionOptions(string(assertionOptionsStr))
	if err != nil {
		t.Fatal(err)
	}
	assertionResponse := virtualwebauthn.CreateAssertionResponse(rp, authenticator, credential, *assertionOptions)

	// Login with the passkey.
	finishReq = httptest.NewRequest("POST", "http://example.com/login/finish", nil)
	finishReq.AddCookie(startResp.Cookies()[0])
	finishReq.Header.Set("Content-Type", "application/json")
	finishReq.Body = io.NopCloser(strings.NewReader(assertionResponse))
	w = httptest.NewRecorder()
	app.ServeHTTP(w, finishReq)
	finishResp = w.Result()

	if got, want := finishResp.StatusCode, http.StatusOK; got != want {
		t.Errorf("finishResp.StatusCode = %d, want = %d", got, want)
	}

	if err := json.NewDecoder(finishResp.Body).Decode(&finishResponse); err != nil {
		t.Fatal(err)
	}
	_ = finishResp.Body.Close()

	// Check session ID.
	sessionID := finishResp.Cookies()[0].Value
	loggedIn, err := app.queries.SessionExists(t.Context(), sessionID, time.Now().Add(-30*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	if !loggedIn {
		t.Errorf("loggedIn = false, want = true")
	}
}
