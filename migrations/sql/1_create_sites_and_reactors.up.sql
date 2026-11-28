create table sites (
    id uuid primary key default uuidv7(),
    eia_plant_code text,
    name text not null,
    timezone text,
    latitude double precision,
    longitude double precision,
    address_line_1 text,
    address_line_2 text,
    city text,
    state text,
    location text,
    zip text,
    phone text,
    email text,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp,
    constraint sites_name_key unique (name) include (id),
    constraint sites_latitude_range check (
        latitude is null or latitude between -90 and 90
    ),
    constraint sites_longitude_range check (
        longitude is null or longitude between -180 and 180
    )
);

create table reactors (
    id uuid primary key default uuidv7(),
    site_id uuid not null,
    name text not null,
    unit_number text not null,
    docket_number text not null,
    reactor_type text not null,
    design text,
    capacity_mw double precision,
    created_at timestamptz not null default current_timestamp,
    updated_at timestamptz not null default current_timestamp,
    constraint reactors_docket_number_key unique (docket_number) include (site_id),
    constraint reactors_site_id_fkey
        foreign key (site_id) references sites (id) on delete restrict
);

create index reactors_site_id_idx on reactors (site_id);
