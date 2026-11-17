select
    id::text,
    eia_plant_code,
    name,
    timezone,
    latitude,
    longitude,
    address_line_1,
    address_line_2,
    city,
    state,
    location,
    zip,
    phone,
    email,
    created_at,
    updated_at
from sites
where id = $1::uuid;
