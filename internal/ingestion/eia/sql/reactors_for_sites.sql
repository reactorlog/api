select site.name, reactor.unit_number
from reactors as reactor
join sites as site on site.id = reactor.site_id
where site.name = any($1::text[])
order by site.name, reactor.unit_number;
