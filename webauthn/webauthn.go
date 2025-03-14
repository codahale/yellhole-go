package webauthn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/codahale/yellhole-go/config"
	"github.com/google/uuid"
)

type RegistrationChallenge struct {
	RpID       string   `json:"rpId"`
	UserID     []byte   `json:"userIdBase64"`
	Username   string   `json:"username"`
	PasskeyIDs [][]byte `json:"passkeyIdsBase64"`
}

func NewRegistrationChallenge(config *config.Config, userID uuid.UUID, passkeyIDs [][]byte) RegistrationChallenge {
	if passkeyIDs == nil {
		passkeyIDs = make([][]byte, 0)
	}
	rc := RegistrationChallenge{
		RpID:       config.BaseURL.Hostname(),
		UserID:     userID[:],
		Username:   config.Author,
		PasskeyIDs: passkeyIDs,
	}
	return rc
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

	// TODO Decode and validate the client data.
	var ccd collectedClientData
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
	if origin.String() != config.BaseURL.String() {
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

type collectedClientData struct {
	Action      string `json:"type"`
	Challenge   string `json:"challenge"`
	Origin      string `json:"origin"`
	CrossOrigin bool   `json:"crossOrigin"`
}
