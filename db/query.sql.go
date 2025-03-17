// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.28.0
// source: query.sql

package db

import (
	"context"
	"database/sql"
	"time"
)

const createChallenge = `-- name: CreateChallenge :exec
insert into challenge (challenge_id, bytes, created_at) values (?, ?, ?)
`

type CreateChallengeParams struct {
	ChallengeID string
	Bytes       []byte
	CreatedAt   time.Time
}

func (q *Queries) CreateChallenge(ctx context.Context, arg CreateChallengeParams) error {
	_, err := q.db.ExecContext(ctx, createChallenge, arg.ChallengeID, arg.Bytes, arg.CreatedAt)
	return err
}

const createImage = `-- name: CreateImage :exec
insert into image (image_id, filename, format, created_at)
values (?, ?, ?, ?)
`

type CreateImageParams struct {
	ImageID   string
	Filename  string
	Format    string
	CreatedAt time.Time
}

func (q *Queries) CreateImage(ctx context.Context, arg CreateImageParams) error {
	_, err := q.db.ExecContext(ctx, createImage,
		arg.ImageID,
		arg.Filename,
		arg.Format,
		arg.CreatedAt,
	)
	return err
}

const createNote = `-- name: CreateNote :exec
insert into note (note_id, body, created_at) values (?, ?, ?)
`

type CreateNoteParams struct {
	NoteID    string
	Body      string
	CreatedAt time.Time
}

func (q *Queries) CreateNote(ctx context.Context, arg CreateNoteParams) error {
	_, err := q.db.ExecContext(ctx, createNote, arg.NoteID, arg.Body, arg.CreatedAt)
	return err
}

const createPasskey = `-- name: CreatePasskey :exec
insert into passkey (passkey_id, public_key_spki, created_at) values (?, ?, ?)
`

type CreatePasskeyParams struct {
	PasskeyID     []byte
	PublicKeySPKI []byte
	CreatedAt     time.Time
}

func (q *Queries) CreatePasskey(ctx context.Context, arg CreatePasskeyParams) error {
	_, err := q.db.ExecContext(ctx, createPasskey, arg.PasskeyID, arg.PublicKeySPKI, arg.CreatedAt)
	return err
}

const createSession = `-- name: CreateSession :exec
insert into session (session_id, created_at)
values (?, ?)
`

type CreateSessionParams struct {
	SessionID string
	CreatedAt time.Time
}

func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) error {
	_, err := q.db.ExecContext(ctx, createSession, arg.SessionID, arg.CreatedAt)
	return err
}

const deleteChallenge = `-- name: DeleteChallenge :one
delete from challenge
where challenge_id = ? and created_at > datetime('now', '-5 minutes')
returning bytes
`

func (q *Queries) DeleteChallenge(ctx context.Context, challengeID string) ([]byte, error) {
	row := q.db.QueryRowContext(ctx, deleteChallenge, challengeID)
	var bytes []byte
	err := row.Scan(&bytes)
	return bytes, err
}

const findPasskey = `-- name: FindPasskey :one
select public_key_spki from passkey where passkey_id = ?
`

func (q *Queries) FindPasskey(ctx context.Context, passkeyID []byte) ([]byte, error) {
	row := q.db.QueryRowContext(ctx, findPasskey, passkeyID)
	var public_key_spki []byte
	err := row.Scan(&public_key_spki)
	return public_key_spki, err
}

const hasPasskey = `-- name: HasPasskey :one
select count(passkey_id) > 0 from passkey
`

func (q *Queries) HasPasskey(ctx context.Context) (bool, error) {
	row := q.db.QueryRowContext(ctx, hasPasskey)
	var column_1 bool
	err := row.Scan(&column_1)
	return column_1, err
}

const noteByID = `-- name: NoteByID :one
select note_id, body, created_at
from note
where note_id = ?
`

func (q *Queries) NoteByID(ctx context.Context, noteID string) (Note, error) {
	row := q.db.QueryRowContext(ctx, noteByID, noteID)
	var i Note
	err := row.Scan(&i.NoteID, &i.Body, &i.CreatedAt)
	return i, err
}

const notesByDate = `-- name: NotesByDate :many
select note_id, body, created_at
from note
where created_at >= ?1 and created_at < ?2 
order by created_at desc
`

type NotesByDateParams struct {
	Start time.Time
	End   time.Time
}

func (q *Queries) NotesByDate(ctx context.Context, arg NotesByDateParams) ([]Note, error) {
	rows, err := q.db.QueryContext(ctx, notesByDate, arg.Start, arg.End)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Note
	for rows.Next() {
		var i Note
		if err := rows.Scan(&i.NoteID, &i.Body, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const passkeyIDs = `-- name: PasskeyIDs :many
select passkey_id from passkey
`

func (q *Queries) PasskeyIDs(ctx context.Context) ([][]byte, error) {
	rows, err := q.db.QueryContext(ctx, passkeyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items [][]byte
	for rows.Next() {
		var passkey_id []byte
		if err := rows.Scan(&passkey_id); err != nil {
			return nil, err
		}
		items = append(items, passkey_id)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const purgeSessions = `-- name: PurgeSessions :execresult
delete from session where created_at < datetime('now', '-7 days')
`

func (q *Queries) PurgeSessions(ctx context.Context) (sql.Result, error) {
	return q.db.ExecContext(ctx, purgeSessions)
}

const recentImages = `-- name: RecentImages :many
select image_id, filename, format, created_at
from image
order by created_at desc
limit ?
`

func (q *Queries) RecentImages(ctx context.Context, limit int64) ([]Image, error) {
	rows, err := q.db.QueryContext(ctx, recentImages, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Image
	for rows.Next() {
		var i Image
		if err := rows.Scan(
			&i.ImageID,
			&i.Filename,
			&i.Format,
			&i.CreatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const recentNotes = `-- name: RecentNotes :many
select note_id, body, created_at
from note
order by created_at desc
limit ?
`

func (q *Queries) RecentNotes(ctx context.Context, limit int64) ([]Note, error) {
	rows, err := q.db.QueryContext(ctx, recentNotes, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Note
	for rows.Next() {
		var i Note
		if err := rows.Scan(&i.NoteID, &i.Body, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const sessionExists = `-- name: SessionExists :one
select count(1) > 0
from session
where session_id = ? and created_at > datetime('now', '-7 days')
`

func (q *Queries) SessionExists(ctx context.Context, sessionID string) (bool, error) {
	row := q.db.QueryRowContext(ctx, sessionExists, sessionID)
	var column_1 bool
	err := row.Scan(&column_1)
	return column_1, err
}

const weeksWithNotes = `-- name: WeeksWithNotes :many
select
    cast(date(datetime(created_at, 'localtime'), 'weekday 0', '-7 days') as text) as start_date,
    cast(date(datetime(created_at, 'localtime'), 'weekday 0') as text) as end_date
from note
group by 1 order by 1 desc
`

type WeeksWithNotesRow struct {
	StartDate string
	EndDate   string
}

func (q *Queries) WeeksWithNotes(ctx context.Context) ([]WeeksWithNotesRow, error) {
	rows, err := q.db.QueryContext(ctx, weeksWithNotes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []WeeksWithNotesRow
	for rows.Next() {
		var i WeeksWithNotesRow
		if err := rows.Scan(&i.StartDate, &i.EndDate); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
