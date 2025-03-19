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

-- name: WeeksWithNotes :many
select
    cast(date(datetime(created_at, 'localtime'), 'weekday 0', '-7 days') as text) as start_date,
    cast(date(datetime(created_at, 'localtime'), 'weekday 0') as text) as end_date
from note
group by 1 order by 1 desc;

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

-- name: PurgeChallenges :execresult
delete from challenge where created_at < ?;

-- name: HasPasskey :one
select count(passkey_id) > 0 from passkey;

-- name: FindPasskey :one
select public_key_spki from passkey where passkey_id = ?;

-- name: PasskeyIDs :many
select passkey_id from passkey;

-- name: CreatePasskey :exec
insert into passkey (passkey_id, public_key_spki, created_at) values (?, ?, ?);

-- name: CreateChallenge :exec
insert into challenge (challenge_id, bytes, created_at) values (?, ?, ?);

-- name: DeleteChallenge :one
delete from challenge
where challenge_id = ? and created_at > ? 
returning bytes;