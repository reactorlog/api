# Reactor capacity_mw

## Goal

Store each reactor’s electrical capacity in megawatts from Form EIA-860 nameplate data, exposed on the read API as `capacity_mw`.

## Decisions

- Field name: `capacity_mw` (not `nameplate_mw`). Source column is EIA “Nameplate Capacity (MW)”.
- Grain: one value per reactor row.
- Source: extend existing `ingest-eia-plants`. No new CLI command.
- Match: after site↔plant match, require a bijection between that site’s reactor `unit_number` values and operable `NUC` Generator IDs at the plant code. Any miss or extra fails the import.
- Schema: add nullable `capacity_mw double precision` on `reactors` by editing migration `1_create_sites_and_reactors` in place. Pre-release: no additive migration. Recreate local DBs.
- NRC import never writes `capacity_mw`. Reimport preserves existing values.
- Unused EIA plants (for example Crane) stay logged and skipped.
- Idempotent EIA updates: write `capacity_mw` only when the stored value differs.

## Data shape

```text
Generator {
  PlantCode   string
  ID          string   // EIA Generator ID
  NameplateMW float64  // EIA Nameplate Capacity (MW)
}

Parse yields []Plant plus generators indexed by plant code.
Plant match is unchanged.
Reactor.CapacityMW *float64  // json capacity_mw
```

## Flow

1. Fetch and parse EIA-860 (plant sheet + operable NUC generators with nameplate).
2. Match every site to a plant. Fail on unmatched sites.
3. For each matched site, load reactors and generators for that plant code.
4. Require unit_number set == Generator ID set. Fail otherwise.
5. In the existing EIA transaction: enrich sites, then update reactor `capacity_mw`.

## Out of scope

- Net summer or winter capacity
- Thermal MWt
- Plant-level summed capacity
- Additive schema migrations
