create table note (
    note_id text primary key not null,
    body text not null,
    created_at timestamp not null default current_timestamp
);

create index idx_note_created_at_desc on note (created_at desc);

create table image (
    image_id text primary key not null,
    filename text not null,
    format text not null,
    created_at timestamp not null default current_timestamp
);

create index idx_image_created_at_desc on image (created_at desc);

create table session (
    session_id text primary key not null,
    created_at timestamp not null default current_timestamp
);

create table passkey (
    passkey_id blob primary key not null,
    public_key_spki blob not null,
    created_at timestamp not null default current_timestamp
);

create table challenge (
    challenge_id text primary key not null,
    bytes blob not null,
    created_at timestamp not null default current_timestamp
);