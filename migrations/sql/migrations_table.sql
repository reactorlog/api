create table if not exists migrations (
    id uuid primary key default uuidv7(),
    version integer not null,
    name text not null,
    created_at timestamptz not null default current_timestamp,
    constraint migrations_version_key unique (version)
);
