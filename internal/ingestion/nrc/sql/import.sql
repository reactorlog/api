-- Stored sites are renamed before new sites are inserted, so list order does not
-- change which site a docket joins. Rows whose stored values already match are
-- left in place.
with incoming as (
    select distinct on (docket_number)
        name,
        unit_number,
        docket_number,
        reactor_type,
        site_name,
        location,
        state,
        ordinality
    from unnest(
        $1::text[],
        $2::text[],
        $3::text[],
        $4::text[],
        $5::text[],
        $6::text[],
        $7::text[]
    ) with ordinality as incoming_row(
        name,
        unit_number,
        docket_number,
        reactor_type,
        site_name,
        location,
        state,
        ordinality
    )
    order by docket_number, ordinality desc
),
site_target as (
    select distinct on (reactor.site_id)
        reactor.site_id,
        incoming.site_name,
        incoming.location,
        incoming.state
    from incoming
    join reactors as reactor
        on reactor.docket_number = incoming.docket_number
    order by reactor.site_id, incoming.ordinality desc
),
renamed as (
    update sites as site
    set
        name = site_target.site_name,
        location = site_target.location,
        state = site_target.state,
        updated_at = current_timestamp
    from site_target
    where site.id = site_target.site_id
      and (site.name, site.location, site.state) is distinct from (
          site_target.site_name,
          site_target.location,
          site_target.state
      )
    returning site.id, site.name, site.location
),
occupied as (
    select renamed.id, renamed.name, renamed.location
    from renamed
    union all
    select site.id, site.name, site.location
    from sites as site
    where not exists (
        select 1
        from renamed
        where renamed.id = site.id
    )
),
created as (
    insert into sites (name, location, state)
    select distinct on (incoming.site_name, incoming.location)
        incoming.site_name,
        incoming.location,
        incoming.state
    from incoming
    where not exists (
        select 1
        from reactors as reactor
        where reactor.docket_number = incoming.docket_number
    )
    and not exists (
        select 1
        from occupied
        where occupied.name = incoming.site_name
          and occupied.location = incoming.location
    )
    order by incoming.site_name, incoming.location, incoming.ordinality
    returning id, name, location
)
insert into reactors (site_id, name, unit_number, docket_number, reactor_type)
select
    coalesce(reactor.site_id, created.id, occupied.id),
    incoming.name,
    incoming.unit_number,
    incoming.docket_number,
    incoming.reactor_type
from incoming
left join reactors as reactor
    on reactor.docket_number = incoming.docket_number
left join created
    on reactor.site_id is null
   and created.name = incoming.site_name
   and created.location = incoming.location
left join occupied
    on reactor.site_id is null
   and created.id is null
   and occupied.name = incoming.site_name
   and occupied.location = incoming.location
on conflict (docket_number) do update
set
    name = excluded.name,
    unit_number = excluded.unit_number,
    reactor_type = excluded.reactor_type,
    updated_at = current_timestamp
where (
    reactors.name,
    reactors.unit_number,
    reactors.reactor_type
) is distinct from (
    excluded.name,
    excluded.unit_number,
    excluded.reactor_type
);
