create table reactors (
    id uuid primary key default uuidv7(),
    site_id uuid not null,
    name text not null,
    unit_number text not null,
    docket_number text not null,
    reactor_type text not null,
    design text not null,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp,
    constraint reactors_docket_number_key unique (docket_number) include (site_id),
    constraint reactors_site_id_fkey
        foreign key (site_id) references sites (id) on delete restrict
);

create index reactors_site_id_idx on reactors (site_id);
