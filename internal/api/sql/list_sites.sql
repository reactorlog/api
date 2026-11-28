select
    sites.id::text,
    sites.eia_plant_code,
    sites.name,
    sites.timezone,
    sites.latitude,
    sites.longitude,
    sites.address_line_1,
    sites.address_line_2,
    sites.city,
    sites.state,
    sites.location,
    sites.zip,
    sites.phone,
    sites.email,
    sites.created_at,
    sites.updated_at,
    count(reactors.id)::int
from sites
left join reactors on reactors.site_id = sites.id
group by sites.id
order by sites.name, sites.location nulls last, sites.id;
