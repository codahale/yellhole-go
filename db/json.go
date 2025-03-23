package db

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/go-webauthn/webauthn/webauthn"
)

type JSONColumn[T any] struct {
	Data T
}

func (j *JSONColumn[T]) Value() (driver.Value, error) {
	return json.Marshal(&j.Data)
}

func (j *JSONColumn[T]) Scan(src any) error {
	b, ok := src.([]byte)
	if !ok {
		return fmt.Errorf("unknown input type: %#v", src)
	}
	return json.Unmarshal(b, &j.Data)
}

type JSONCredential = JSONColumn[*webauthn.Credential]
type JSONSessionData = JSONColumn[webauthn.SessionData]

var (
	_ sql.Scanner   = &JSONColumn[string]{Data: ""}
	_ driver.Valuer = &JSONColumn[string]{Data: ""}
)
