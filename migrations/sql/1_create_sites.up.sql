create table sites (
    id uuid primary key default uuidv7(),
    eia_plant_code text,
    name text not null,
    timezone text not null,
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
    constraint sites_name_location unique nulls distinct (name, location) include (id),
    constraint sites_latitude_range check (
        latitude is null or latitude between -90 and 90
    ),
    constraint sites_longitude_range check (
        longitude is null or longitude between -180 and 180
    )
);
