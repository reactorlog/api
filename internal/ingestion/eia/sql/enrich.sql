-- Enrich existing sites from matched EIA plants. Site identity is name.
-- Rows whose enrichment already matches are left untouched.
with incoming as (
    select *
    from unnest(
        $1::text[],
        $2::text[],
        $3::text[],
        $4::text[],
        $5::text[],
        $6::text[],
        $7::double precision[],
        $8::double precision[]
    ) as incoming_row(
        site_name,
        eia_plant_code,
        address_line_1,
        city,
        state,
        zip,
        latitude,
        longitude
    )
)
update sites as site
set
    eia_plant_code = incoming.eia_plant_code,
    address_line_1 = incoming.address_line_1,
    city = incoming.city,
    state = incoming.state,
    zip = incoming.zip,
    latitude = incoming.latitude,
    longitude = incoming.longitude,
    updated_at = current_timestamp
from incoming
where site.name = incoming.site_name
  and (
      site.eia_plant_code,
      site.address_line_1,
      site.city,
      site.state,
      site.zip,
      site.latitude,
      site.longitude
  ) is distinct from (
      incoming.eia_plant_code,
      incoming.address_line_1,
      incoming.city,
      incoming.state,
      incoming.zip,
      incoming.latitude,
      incoming.longitude
  );
