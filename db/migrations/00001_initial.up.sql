create table
    note (
        note_id text primary key not null,
        body text not null,
        created_at datetime not null
    );

create index idx_note_created_at_desc on note (created_at desc);

create table
    image (
        image_id text primary key not null,
        filename text not null,
        original_filename text not null,
        format text not null,
        created_at datetime not null
    );

create index idx_image_created_at_desc on image (created_at desc);

create table
    session (
        session_id text primary key not null,
        created_at datetime not null
    );

create table
    webauthn_credential (
        credential_data blob not null,
        created_at datetime not null
    );

create table
    webauthn_session (
        webauthn_session_id text primary key not null,
        session_data blob not null,
        created_at datetime not null
    );