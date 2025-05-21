-- name: CreateNote :exec
insert into note (note_id, body, created_at)
values (:note_id, :body, :created_at);

-- name: NoteByID :one
select note_id,
       body,
       created_at
from note
where note_id = :note_id;

-- name: RecentNotes :many
select note_id,
       body,
       created_at
from note
order by created_at desc
limit :limit;

-- name: WeeksWithNotes :many
select cast(date(datetime(created_at, 'weekday 0', '-7 days')) as text) as start_date,
       cast(date(datetime(created_at, 'weekday 0', '-1 day')) as text)  as end_date
from note
group by 1
order by 1 desc;

-- name: NotesByDate :many
select note_id,
       body,
       created_at
from note
where :start_date <= created_at
  and created_at < :end_date
order by created_at desc;

-- name: RecentImages :many
select *
from image
order by created_at desc
limit :limit;

-- name: CreateImage :exec
insert into image (image_id,
                   filename,
                   original_filename,
                   format,
                   created_at)
values (:image_id, :filename, :original_filename, :format, :created_at);

-- name: CreateSession :exec
insert into session (session_id, created_at)
values (:session_id, :created_at);

-- name: SessionExists :one
select count(1) > 0
from session
where session_id = :session_id
  and created_at > :expiry;

-- name: PurgeSessions :execresult
delete
from session
where created_at < :expiry;

-- name: CreateWebauthnCredential :exec
insert into webauthn_credential (credential_data, created_at)
values (:credential_data, :created_at);

-- name: WebauthnCredentials :many
select credential_data
from webauthn_credential;

-- name: HasWebauthnCredential :one
select count(1) > 0
from webauthn_credential;

-- name: CreateWebauthnSession :exec
insert into webauthn_session (webauthn_session_id, session_data, created_at)
values (:webauthn_session_id, :session_data, :created_at);

-- name: DeleteWebauthnSession :one
delete
from webauthn_session
where webauthn_session_id = :webauthn_session_id
  and created_at > :expiry
returning session_data;

-- name: PurgeWebauthnSessions :execresult
delete
from webauthn_session
where created_at < :expiry;