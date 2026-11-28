# Site reactor_count on list cards

## Goal

Expose how many reactors belong to each site on `GET /sites`, and show that count on the web sites list cards. Clients must not re-derive the count by downloading `/reactors`.

## Decisions

- Add required JSON field `reactor_count` (non-negative int) on `Site`.
- `GET /sites` computes it in SQL with one `LEFT JOIN reactors` + `GROUP BY` site (not a correlated per-row subquery). Sites with no reactors return `0`. List scanning reads `reactor_count` from that result.
- `GET /sites/{id}` returns the same field on the embedded `Site`. Keep `get_site.sql` without a count expression: load the site row and its reactors as today, then set `reactor_count = len(reactors)` in `SiteByID` before responding. Invariant: `reactor_count == len(reactors)`.
- No migration and no stored column — count is derived from `reactors`.
- Web mirrors the field at the HTTP parse boundary and shows it on list cards only. Site detail keeps using `reactors[]` and does not add separate count chrome.
- Card copy: place line stays `city · state`; add a reactor line such as `1 reactor` / `N reactors` (English pluralization). Omit invented labels when count is shown as a number alone.

## Out of scope

- Map links, fleet map, timezone fill, address on cards.
- Changing `/reactors` or nesting `reactors[]` on the list response.
- Caching or materialized counts.

## API shape

`Site` gains:

```text
reactor_count int  // json: reactor_count, always present, >= 0
```

`SiteDetail` embeds `Site` unchanged aside from that field; still includes `reactors`.

## Web

- Extend `Site` in `src/sites/types.ts` and require `reactor_count` in `parseSite`.
- `SitesListPage` renders the count on each card beside existing place text.
- Update colocated Vitest parse (and screen/UI tests if they assert card model fields).

## Tests

- API: list SQL/store returns correct counts including zero; detail count matches `len(reactors)`; handler JSON includes `reactor_count`.
- Web: parser rejects missing/non-integer `reactor_count`; accepts `0`.

## Performance

At current fleet size (~55 sites / ~95 reactors) both correlated subquery and join-aggregate complete in well under 1 ms locally. Prefer join-aggregate for a single pass as the fleet grows. Detail derives count in memory from the reactors already loaded.
