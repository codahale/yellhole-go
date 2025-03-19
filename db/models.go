// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0

package db

type Challenge struct {
	ChallengeID string
	Bytes       []byte
	CreatedAt   int64
}

type Image struct {
	ImageID   string
	Filename  string
	Format    string
	CreatedAt int64
}

type Note struct {
	NoteID    string
	Body      string
	CreatedAt int64
}

type Passkey struct {
	PasskeyID     []byte
	PublicKeySPKI []byte
	CreatedAt     int64
}

type Session struct {
	SessionID string
	CreatedAt int64
}
