# reactorlog web MVP design

Explanation of the first frontend for reactorlog. Implementation lives in a new sibling repo, not in this API tree.

## Goal

Ship a TypeScript SPA that lets someone browse sites and open a site to see its reactors. Layout and brand leave room for charts later via owlplot. Smallest change that meets that goal.

## Product scope

In scope:

- Sites list at `/`
- Site detail at `/sites/:id` with that site's reactors
- Dark-first UI
- Empty chart slot on site detail for a future series

Out of scope:

- Auth
- Write APIs
- Real time-series charts until the API exposes them
- TanStack Query / Router / Start
- Bun as package manager or runtime
- SSR

## Stack

- Vite
- Svelte
- TypeScript
- Tailwind CSS
- Svelte client routing (two routes)
- Svelte writable / readable / derived stores for server-backed UI state
- owlplot (SVG renderer) behind a `base` chart slot when series data exists

Repo path: `Github/reactorlog/web`, GitHub `reactorlog/web` when published. Local API stays `HTTP_ADDR` default `:8080`. Vite proxies `/api` to that origin so the browser talks same-origin during development.

## Brand

| Role | Value |
| --- | --- |
| Brand / Cherenkov | `#00A8E8` |
| Accent / Amber | `#E6A817` |
| Page | Tailwind `slate-950` (`#020617`) |
| Panel | Tailwind `slate-900` (`#0f172a`) |
| Border | Tailwind `slate-800` (`#1e293b`) |

Dark mode first. Chrome stays near-black slate. Color in the UI comes from Cherenkov and Amber, not from a navy page wash.

## Architecture

Bounded contexts are top-level folders. No folder named `domain` or `domains`.

```
src/
  app/       shell, router, theme tokens (CSS variables + Tailwind theme)
  base/      shared functional UI contexts compose (PageShell, list chrome, ChartSlot)
  sites/     types, HTTP client, store, SitesListPage, SiteDetailPage
  reactors/  types and list presentation used on site detail
```

### Data shapes

Mirror the API JSON. Parse at the HTTP boundary. Components trust the parsed types.

```ts
type Site = {
  id: string
  eia_plant_code: string | null
  name: string
  timezone: string | null
  latitude: number | null
  longitude: number | null
  address_line_1: string | null
  address_line_2: string | null
  city: string | null
  state: string | null
  location: string | null
  zip: string | null
  phone: string | null
  email: string | null
  created_at: string
  updated_at: string
}

type Reactor = {
  id: string
  site_id: string
  name: string
  unit_number: string
  docket_number: string
  reactor_type: string
  design: string | null
  created_at: string
  updated_at: string
}

type SiteDetail = Site & { reactors: Reactor[] }

type LoadState<T> =
  | { status: 'idle' }
  | { status: 'loading' }
  | { status: 'ready'; data: T }
  | { status: 'error'; message: string }
```

### Stores

One store module per context that owns server-backed page state. `sites/store` holds `LoadState<Site[]>` for the list and `LoadState<SiteDetail>` for the selected site. Actions: `loadSites()`, `loadSite(id)`. No third-party state library.

Owlplot does not live in the store. The store holds domain data. `ChartSlot` receives series props when the API has them. MVP passes an empty series and renders a labeled empty state.

### Data flow

1. Page mounts and calls a store action.
2. Store sets `loading`. Client fetches through the Vite `/api` proxy.
3. Client parses JSON into the types above or fails with a message.
4. Store sets `ready` or `error`.
5. Pages subscribe. Site detail renders `reactors` from `SiteDetail`. Chart slot stays empty until series exist.

### Errors

- Network or 5xx: `error` status and a short message plus retry.
- 404 on site detail: `error` with not-found copy and a link to `/`.
- Optional `/health` indicator in the shell. It does not block navigation.

### API coupling

MVP reads:

- `GET /health`
- `GET /sites`
- `GET /sites/{id}`
- `GET /reactors` only if a page needs the flat list. Site detail uses reactors embedded in `SiteDetail`.

If the browser cannot reach the API without a proxy in some environments, fix CORS in the API repo in a follow-up. Dev path assumes the Vite proxy.

## Testing

- Vitest: JSON parse helpers and store transitions with a fake `fetch`.
- Manual smoke: list → detail against a local API.
- Browser E2E is not required to scaffold the repo.

## Non-goals revisited

Do not add Recharts. Do not add TanStack. Do not add Bun. Do not name packages or folders after DDD jargon. Encode the model in types, stores, and folder boundaries instead.
