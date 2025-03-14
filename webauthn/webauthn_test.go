package webauthn

import (
	"bytes"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/codahale/yellhole-go/config"
)

func TestRegistrationResponseValidate(t *testing.T) {
	response := RegistrationResponse{
		ClientDataJSON: []byte{
			123, 34, 99, 104, 97, 108, 108, 101, 110, 103, 101, 34, 58, 110, 117, 108, 108, 44, 34,
			111, 114, 105, 103, 105, 110, 34, 58, 34, 104, 116, 116, 112, 58, 47, 47, 101, 120, 97,
			109, 112, 108, 101, 46, 99, 111, 109, 47, 34, 44, 34, 116, 121, 112, 101, 34, 58, 34,
			119, 101, 98, 97, 117, 116, 104, 110, 46, 99, 114, 101, 97, 116, 101, 34, 44, 34, 99,
			114, 111, 115, 115, 79, 114, 105, 103, 105, 110, 34, 58, 102, 97, 108, 115, 101, 125,
		},
		AuthenticatorData: []byte{
			163, 121, 166, 246, 238, 175, 185, 165, 94, 55, 140, 17, 128, 52, 226, 117, 30, 104,
			47, 171, 159, 45, 48, 171, 19, 210, 18, 85, 134, 206, 25, 71, 1, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 32, 87, 207, 13, 187, 205, 19, 155, 141, 218,
			199, 42, 32, 144, 44, 221, 198, 179, 28, 225, 162, 251, 38, 54, 219, 199, 97, 206, 248,
			32, 193, 235, 79,
		}, PublicKey: []byte{
			48, 89, 48, 19, 6, 7, 42, 134, 72, 206, 61, 2, 1, 6, 8, 42, 134, 72, 206, 61, 3, 1, 7,
			3, 66, 0, 4, 116, 167, 148, 124, 165, 138, 250, 212, 60, 247, 239, 66, 215, 50, 255,
			2, 176, 1, 161, 237, 71, 151, 51, 47, 123, 62, 148, 226, 227, 107, 14, 58, 26, 172,
			188, 226, 202, 52, 150, 247, 91, 187, 170, 172, 125, 143, 18, 36, 227, 3, 218, 158, 179,
			167, 151, 204, 37, 227, 149, 15, 13, 223, 107, 127,
		},
	}
	baseURL, err := url.Parse("http://example.com/")
	if err != nil {
		t.Fatal(err)
	}
	keyID, publicKey, err := response.Validate(&config.Config{
		BaseURL: baseURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	expectedKeyID := []byte{
		0x57, 0xcf, 0xd, 0xbb, 0xcd, 0x13, 0x9b, 0x8d, 0xda, 0xc7, 0x2a, 0x20, 0x90, 0x2c, 0xdd,
		0xc6, 0xb3, 0x1c, 0xe1, 0xa2, 0xfb, 0x26, 0x36, 0xdb, 0xc7, 0x61, 0xce, 0xf8, 0x20, 0xc1,
		0xeb, 0x4f,
	}
	if !bytes.Equal(expectedKeyID, keyID) {
		t.Errorf("expected %#v but was %#v", expectedKeyID, keyID)
	}

	expectedPublicKey := []byte{
		0x30, 0x59, 0x30, 0x13, 0x6, 0x7, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x2, 0x1, 0x6, 0x8, 0x2a,
		0x86, 0x48, 0xce, 0x3d, 0x3, 0x1, 0x7, 0x3, 0x42, 0x0, 0x4, 0x74, 0xa7, 0x94, 0x7c, 0xa5,
		0x8a, 0xfa, 0xd4, 0x3c, 0xf7, 0xef, 0x42, 0xd7, 0x32, 0xff, 0x2, 0xb0, 0x1, 0xa1, 0xed,
		0x47, 0x97, 0x33, 0x2f, 0x7b, 0x3e, 0x94, 0xe2, 0xe3, 0x6b, 0xe, 0x3a, 0x1a, 0xac, 0xbc,
		0xe2, 0xca, 0x34, 0x96, 0xf7, 0x5b, 0xbb, 0xaa, 0xac, 0x7d, 0x8f, 0x12, 0x24, 0xe3, 0x3,
		0xda, 0x9e, 0xb3, 0xa7, 0x97, 0xcc, 0x25, 0xe3, 0x95, 0xf, 0xd, 0xdf, 0x6b, 0x7f,
	}
	if !bytes.Equal(expectedPublicKey, publicKey) {
		t.Errorf("expected %#v but was %#v", expectedPublicKey, publicKey)
	}
}

func TestFuckMe(t *testing.T) {
	s := `{"clientDataJSONBase64":"eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoiQUEiLCJvcmlnaW4iOiJodHRwOi8vbG9jYWxob3N0OjMwMDAiLCJjcm9zc09yaWdpbiI6ZmFsc2V9","authenticatorDataBase64":"SZYN5YgOjGh0NBcPZHZgW4/krrmihjLHmVzzuoMdl2NdAAAAAPv8MAcVTk7MjAtuAgVX170AFB3WJiTPSNVFj/E0ATQz6D/d8IE5pQECAyYgASFYIG1xiQ820++WsxWSIKBalAzSorYooNDClonyckl2+sgbIlgge5hMP90Xrb7QpkDd/6DLnvixFlrVf3QSWO15AzlhJjU=","publicKeyBase64":"MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEbXGJDzbT75azFZIgoFqUDNKitiig0MKWifJySXb6yBt7mEw/3RetvtCmQN3/oMue+LEWWtV/dBJY7XkDOWEmNQ=="}`
	var resp struct {
		ClientDataJSON    []byte `json:"clientDataJSONBase64"`
		AuthenticatorData []byte `json:"authenticatorDataBase64"`
		PublicKey         []byte `json:"publicKeyBase64"`
	}
	if err := json.Unmarshal([]byte(s), &resp); err != nil {
		t.Fatal(err)
	}

	var clientData struct {
		Action      string `json:"type"`
		Challenge   string `json:"challenge"`
		Origin      string `json:"origin"`
		CrossOrigin bool   `json:"crossOrigin"`
	}
	t.Log(string(resp.ClientDataJSON))
	if err := json.Unmarshal(resp.ClientDataJSON, &clientData); err != nil {
		t.Fatal(err)
	}
	t.Log(clientData)
}
