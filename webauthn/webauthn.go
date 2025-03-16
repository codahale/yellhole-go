package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/codahale/yellhole-go/config"
)

type RegistrationChallenge struct {
	RpID       string   `json:"rpId"`
	UserID     []byte   `json:"userIdBase64"`
	Username   string   `json:"username"`
	PasskeyIDs [][]byte `json:"passkeyIdsBase64"`
}

func NewRegistrationChallenge(config *config.Config, passkeyIDs [][]byte) (*RegistrationChallenge, error) {
	userID := make([]byte, 16)
	if _, err := rand.Read(userID); err != nil {
		return nil, err
	}

	if passkeyIDs == nil {
		passkeyIDs = make([][]byte, 0)
	}
	return &RegistrationChallenge{
		RpID:       config.BaseURL.Hostname(),
		UserID:     userID,
		Username:   config.Author,
		PasskeyIDs: passkeyIDs,
	}, nil
}

type RegistrationResponse struct {
	ClientDataJSON    []byte `json:"clientDataJSONBase64"`
	AuthenticatorData []byte `json:"authenticatorDataBase64"`
	PublicKey         []byte `json:"publicKeyBase64"`
}

func (r *RegistrationResponse) Validate(config *config.Config) ([]byte, []byte, error) {
	// Decode and validate the public key.
	cert, err := x509.ParsePKIXPublicKey(r.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	publicKey, ok := cert.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return nil, nil, fmt.Errorf("invalid public key type")
	}

	var ccd struct {
		Action      string `json:"type"`
		Origin      string `json:"origin"`
		CrossOrigin bool   `json:"crossOrigin"`
	}
	if err := json.Unmarshal(r.ClientDataJSON, &ccd); err != nil {
		return nil, nil, fmt.Errorf("invalid client data: %w", err)
	}
	if ccd.Action != "webauthn.create" {
		return nil, nil, fmt.Errorf("invalid action type: %q", ccd.Action)
	}
	if ccd.CrossOrigin {
		return nil, nil, fmt.Errorf("cross-origin webauthn attempt")
	}
	origin, err := url.Parse(ccd.Origin)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid origin: %w", err)
	}
	if origin.String() != strings.TrimRight(config.BaseURL.String(), "/") {
		return nil, nil, fmt.Errorf("invalid origin: %q / %q", origin.String(), config.BaseURL.String())
	}

	// Decode and validate the authenticator data.
	h := sha256.New()
	h.Write([]byte(config.BaseURL.Hostname()))
	if subtle.ConstantTimeCompare(h.Sum(nil), r.AuthenticatorData[:h.Size()]) == 0 {
		return nil, nil, fmt.Errorf("invalid RpID")
	}
	if r.AuthenticatorData[32]&1 == 0 {
		return nil, nil, fmt.Errorf("user presence flag not set")
	}
	if len(r.AuthenticatorData) < 55 {
		return nil, nil, fmt.Errorf("no credential ID provided")
	}

	credIDLen := int(binary.BigEndian.Uint16(r.AuthenticatorData[53:55]))
	if len(r.AuthenticatorData) < 55+credIDLen {
		return nil, nil, fmt.Errorf("bad credential ID size")
	}

	return r.AuthenticatorData[55 : 55+credIDLen], r.PublicKey, nil
}

type LoginChallenge struct {
	RpID       string   `json:"rpId"`
	Challenge  []byte   `json:"challengeBase64"`
	PasskeyIDs [][]byte `json:"passkeyIdsBase64"`
}

func NewLoginChallenge(config *config.Config, passkeyIDs [][]byte) (*LoginChallenge, error) {
	challenge := make([]byte, 32)
	if _, err := rand.Read(challenge); err != nil {
		return nil, err
	}
	return &LoginChallenge{
		RpID:       config.BaseURL.Hostname(),
		Challenge:  challenge,
		PasskeyIDs: passkeyIDs,
	}, nil
}

type LoginResponse struct {
	RawID             []byte `json:"rawIdBase64"`
	ClientDataJSON    []byte `json:"clientDataJSONBase64"`
	AuthenticatorData []byte `json:"authenticatorDataBase64"`
	Signature         []byte `json:"signatureBase64"`
}

func (r *LoginResponse) Validate(config *config.Config, passkeySPKI, challenge []byte) error {
	// Validate the collected client data and check the challenge.
	var ccd struct {
		Action      string `json:"type"`
		Origin      string `json:"origin"`
		CrossOrigin bool   `json:"crossOrigin"`
		Challenge   string `json:"challenge"`
	}
	if err := json.Unmarshal(r.ClientDataJSON, &ccd); err != nil {
		return fmt.Errorf("invalid client data: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(ccd.Challenge), []byte(base64.RawURLEncoding.EncodeToString(challenge))) != 1 {
		return fmt.Errorf("invalid challenge")
	}
	if ccd.Action != "webauthn.get" {
		return fmt.Errorf("invalid action type: %q", ccd.Action)
	}
	if ccd.CrossOrigin {
		return fmt.Errorf("cross-origin webauthn attempt")
	}
	origin, err := url.Parse(ccd.Origin)
	if err != nil {
		return fmt.Errorf("invalid origin: %w", err)
	}
	if origin.String() != strings.TrimRight(config.BaseURL.String(), "/") {
		return fmt.Errorf("invalid origin: %q / %q", origin.String(), config.BaseURL.String())
	}

	// Decode and validate the authenticator data.
	// Decode and validate the authenticator data.
	h := sha256.New()
	h.Write([]byte(config.BaseURL.Hostname()))
	if subtle.ConstantTimeCompare(h.Sum(nil), r.AuthenticatorData[:h.Size()]) == 0 {
		return fmt.Errorf("invalid RpID")
	}
	if r.AuthenticatorData[32]&1 == 0 {
		return fmt.Errorf("user presence flag not set")
	}

	// Decode and validate the public key.
	cert, err := x509.ParsePKIXPublicKey(passkeySPKI)
	if err != nil {
		return err
	}

	publicKey, ok := cert.(*ecdsa.PublicKey)
	if !ok || publicKey.Curve != elliptic.P256() {
		return fmt.Errorf("invalid public key type")
	}

	// Re-calculate the signed material.
	h.Reset()
	h.Write(r.ClientDataJSON)
	ccdHash := h.Sum(nil)
	h.Reset()
	h.Write(r.AuthenticatorData)
	h.Write(ccdHash)

	// Verify the signature.
	if !ecdsa.VerifyASN1(publicKey, h.Sum(nil), r.Signature) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
