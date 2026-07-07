# GigPurse Frontend

A Next.js (App Router) app for the GigPurse musician marketplace. Talks to the
Go backend in `../backend` through a small BFF layer.

## Getting started

```bash
cp .env.example .env.local   # point GIGPURSE_API_URL at your running backend
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). The Go backend must be
running (see `../backend/README.md` or the root README) — the app has no
mock data layer.

## How auth works here

The Go backend issues bearer JWTs. This app never stores the raw token in
client-accessible storage:

- `app/api/auth/*` — Route Handlers that call the Go `/auth/*` endpoints.
  `login` extracts the JWT and sets it as an httpOnly cookie (`gigpurse_token`).
- `app/api/proxy/[...path]` — a generic authenticated proxy. Client code calls
  `/api/proxy/<backend-path>`; the handler reads the cookie server-side and
  forwards it as `Authorization: Bearer <token>` to the Go backend.
- `proxy.js` (Next.js 16's renamed `middleware.js`) — redirects to `/login`
  for protected routes when there's no valid session cookie.
- The one exception is the chat WebSocket, which connects directly to the Go
  backend from the browser. `app/api/auth/ws-token` hands the client the raw
  JWT only for that purpose.

See `lib/api.js` for the client-side fetch wrapper used by every page.

## Project structure

- `app/` — routes (App Router), including the `api/` BFF layer above.
- `components/` — `ui/` is shadcn/ui primitives; the rest are feature
  components grouped by domain (booking, chat, jobs, talent, wallet, ...).
- `lib/` — `api.js` (client fetch wrapper), `backend.js`/`session.js`
  (server-only helpers for the BFF routes), `ws.js` (chat socket helper).
- `hooks/` — `use-current-user.js` wraps `GET /users/profile` via React Query.

## Known gaps (by design)

Wallet/escrow/milestones/transaction-ledger UI is present but intentionally
disabled ("Coming Soon") — the Go backend only has a prototype balance-only
wallet today. See the root README and `../backend/docs/API.md` for details.
