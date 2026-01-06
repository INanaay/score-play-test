-- UUID extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE tags (
                      id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                      name VARCHAR(50) NOT NULL,
                      created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX tags_name_uk ON tags (name);


create table file_metadata (
                               id uuid primary key default gen_random_uuid(),
                               filename varchar(255) not null,
                               mime_type varchar(127) not null,
                               file_type varchar(50) not null,
                               size_bytes bigint not null check (size_bytes >= 0),
                               storage_key varchar(512) not null unique,
                               checksum varchar(128), -- allow different encodings or algos
                               status varchar(30) not null check (status in ('uploading','completed','failed')),
                               created_at timestamptz not null default now(),
                               updated_at timestamptz not null default now(),
                               deleted_at timestamptz
);
CREATE INDEX file_metadata_status_updated_at_idx
    ON file_metadata (status, updated_at)
    WHERE deleted_at IS NULL;


create table upload_session (
                                 id uuid primary key default gen_random_uuid(),
                                 file_id uuid not null references file_metadata(id) on delete cascade,
                                 provider_upload_id text not null,
                                 part_size int not null,
                                 expires_at timestamptz not null,
                                 status text not null check (status in ('open','completed','aborted')),
                                 created_at timestamptz not null default now(),
                                 updated_at timestamptz not null default now()
);

create index idx_upload_session_file on upload_session(file_id);
create index idx_upload_session_expiry on upload_session(status, expires_at);
create unique index uniq_open_session on upload_session(file_id) where status = 'open';


CREATE TABLE file_metadata_tags (
                           file_id UUID NOT NULL REFERENCES file_metadata(id) ON DELETE CASCADE,
                           tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
                           PRIMARY KEY (file_id, tag_id)
);

CREATE INDEX filetags_tag_file_idx ON file_metadata_tags (tag_id, file_id);
CREATE INDEX filetags_file_idx ON file_metadata_tags (file_id);


CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
RETURN NEW;
END;
$$ language 'plpgsql';


CREATE TRIGGER update_file_metadata_updated_at BEFORE UPDATE ON upload_session
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_upload_session_updated_at BEFORE UPDATE ON file_metadata
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();