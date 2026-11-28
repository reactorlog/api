update reactors as reactor
set
    capacity_mw = incoming.capacity_mw,
    updated_at = current_timestamp
from sites as site,
    unnest(
        $1::text[],
        $2::text[],
        $3::double precision[]
    ) as incoming(site_name, unit_number, capacity_mw)
where site.name = incoming.site_name
  and reactor.site_id = site.id
  and reactor.unit_number = incoming.unit_number
  and reactor.capacity_mw is distinct from incoming.capacity_mw;
