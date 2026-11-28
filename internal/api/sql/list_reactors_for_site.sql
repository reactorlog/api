select
    id::text,
    site_id::text,
    name,
    unit_number,
    docket_number,
    reactor_type,
    design,
    capacity_mw,
    created_at,
    updated_at
from reactors
where site_id = $1::uuid
order by name, unit_number, id;
