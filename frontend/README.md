# SwasthyaSetu: frontend

Next.js (App Router) built as a **static export** and served by the backend
from the same origin in production. TypeScript, Tailwind CSS v4, shadcn/ui,
and Zustand.

```bash
cp .env.example .env.local   # points the dev server at the API on :8000
npm install
npm run dev                  # http://localhost:3000
npm run lint && npm run typecheck && npm run build   # build writes out/
```

| Path | Contents |
|---|---|
| `app/` | `/` role picker, `/patient/`, `/doctor/` |
| `components/chat/` | Chat panel with the AI assistant |
| `components/call/video-call.tsx` | Call screen: remote and self video, controls |
| `components/patient/`, `components/doctor/` | Each role's flow |
| `lib/use-call.ts` | WebRTC: media, peer connection, signalling, reconnect |
| `lib/api.ts` | Backend client (same origin unless `NEXT_PUBLIC_API_URL` is set) |
| `lib/store/chat.ts` | Chat state (Zustand, kept in sessionStorage) |
| `lib/session.ts` | Active call ticket, so a refresh rejoins the call |

The three emergency levels are theme tokens (`--triage-green`, `-yellow`,
`-red`) in `app/globals.css`. Use them, and the `emergency` button variant,
rather than ad-hoc reds.
