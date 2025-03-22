-- name: CreateNote :exec
insert into note (note_id, body, created_at) values (?, ?, ?);

-- name: NoteByID :one
select note_id, body, created_at
from note
where note_id = ?;

-- name: RecentNotes :many
select note_id, body, created_at
from note
order by created_at desc
limit ?;

-- name: AllNoteTimestamps :many
select created_at
from note;

-- name: NotesByDate :many
select note_id, body, created_at
from note
where created_at >= sqlc.arg(start) and created_at < sqlc.arg(end) 
order by created_at desc;

-- name: RecentImages :many
select *
from image
order by created_at desc
limit ?;

-- name: CreateImage :exec
insert into image (image_id, filename, format, created_at)
values (?, ?, ?, ?);

-- name: CreateSession :exec
insert into session (session_id, created_at)
values (?, ?);

-- name: SessionExists :one
select count(1) > 0
from session
where session_id = ? and created_at > ?;

-- name: PurgeSessions :execresult
delete from session where created_at < ?;

-- name: CreateWebauthnCredential :exec
insert into webauthn_credential (credential_data, created_at) values (?, ?);

-- name: WebauthnCredentials :many
select credential_data from webauthn_credential;

-- name: HasWebauthnCredential :one
select count(1) > 0 from webauthn_credential;

-- name: CreateWebauthnSession :exec
insert into webauthn_session (webauthn_session_id, session_data, created_at) values (?, ?, ?);

-- name: DeleteWebauthnSession :one
delete from webauthn_session 
where webauthn_session_id = ? and created_at > ? 
returning session_data;

-- name: PurgeWebauthnSessions :execresult
delete from webauthn_session where created_at < ?;