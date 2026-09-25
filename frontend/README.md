# SwasthyaSetu — Frontend

Next.js (App Router) configured as a Progressive Web App. TypeScript, Tailwind CSS v4,
shadcn/ui, and Zustand.

See the [root README](../README.md) to run the whole stack with Docker Compose, or
[docs/architecture.md](../docs/architecture.md) for the system design.

## Development

```bash
npm install
npm run dev        # http://localhost:3000
```

Copy `.env.example` to `.env.local` and point `NEXT_PUBLIC_API_URL` /
`NEXT_PUBLIC_WS_URL` at your backend.

```bash
npm run lint       # eslint
npm run typecheck  # tsc --noEmit
npm run build      # production build (standalone output for Docker)
```

## Layout

| Path | Contents |
|---|---|
| `app/` | App Router pages and layouts |
| `components/ui/` | shadcn/ui components (`npx shadcn@latest add <name>` to add more) |
| `lib/utils.ts` | `cn()` — class merging |
| `lib/store/` | Zustand stores |
| `public/sw.js` | Service worker — offline caching |
| `public/manifest.webmanifest` | PWA manifest |

## Styling

Tailwind v4 is configured in CSS, not JavaScript — the theme lives in
[`app/globals.css`](app/globals.css) and there is no `tailwind.config.js`.

Alongside the usual shadcn/ui tokens, the theme defines the three emergency levels from
[architecture §4](../docs/architecture.md#4-emergency-detection-layer) as
`--triage-green`, `--triage-yellow`, and `--triage-red`. Use those tokens (and the
`emergency` button variant) rather than picking a red — a Red-level triage result must
never be rendered in a shade that reads as merely a warning.

## PWA and offline

The service worker is registered in production only, from
[`components/service-worker-registrar.tsx`](components/service-worker-registrar.tsx). It
caches the app shell cache-first, first-aid articles stale-while-revalidate, and
everything else network-first — doctor availability and triage results must never be
served stale.

To test it, run a production build (`npm run build && npm start`), then use DevTools →
Application → Service Workers, or throttle to Offline. `CACHE_VERSION` in
[`public/sw.js`](public/sw.js) must be bumped whenever the worker or the shell changes.
