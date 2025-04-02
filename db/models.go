// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

import (
	"time"
)

type Image struct {
	ImageID   string
	Filename  string
	Format    string
	CreatedAt time.Time
}

type Note struct {
	NoteID    string
	Body      string
	CreatedAt time.Time
}

type Session struct {
	SessionID string
	CreatedAt time.Time
}

type WebauthnCredential struct {
	CredentialData *JSONCredential
	CreatedAt      time.Time
}

type WebauthnSession struct {
	WebauthnSessionID string
	SessionData       *JSONSessionData
	CreatedAt         time.Time
}
