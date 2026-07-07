# Gigpurse

A gig marketplace connecting clients with musicians. This repo is a monorepo with two
independent projects:

```
Gigpurse/
├── backend/    # Go REST API (Clean Architecture, MongoDB, JWT auth)
└── frontend/   # Next.js web app
```

The API is the source of truth for behavior — see [`backend/docs/API.md`](backend/docs/API.md)
for the full endpoint reference.

## Running locally

**Backend** (needs a local or remote MongoDB instance):

```bash
cd backend
export MONGODB_URI="mongodb://localhost:27017"
export JWT_SECRET="dev-secret-change-me"
go run ./cmd/gigpurse
```

The API listens on `:8080` by default (override with `PORT`).

**Frontend** (needs the backend running first):

```bash
cd frontend
export GIGPURSE_API_URL="http://localhost:8080"
npm install
npm run dev
```

The app listens on `:3000` by default.

## Backend

Built with **Clean Architecture** principles:

```
backend/
├── cmd/
│   ├── gigpurse/         # Application entry point and dependency injection
│   └── simulator/        # Standalone simulation client
├── internal/
│   ├── domain/           # Core enterprise business rules (Entities & Interfaces)
│   ├── usecase/          # Application business rules (orchestrates domain entities)
│   ├── repository/       # Data storage implementations (mongodb, in-memory)
│   └── delivery/         # Transport layer (HTTP handlers, routes)
├── docs/                 # API.md + Postman collection
├── go.mod
└── render.yaml           # Render.com deploy config
```

1. **Domain** (`internal/domain`): Core business objects and contract interfaces,
   independent of external libraries, databases, and frameworks.
2. **Usecase** (`internal/usecase`): Application-specific business logic orchestrating
   domain entities.
3. **Repository** (`internal/repository`): Data access implementations — MongoDB for
   everything except the prototype wallet, which is in-memory.
4. **Delivery** (`internal/delivery`): HTTP handlers and routes using the standard
   library, with a unified JSON response envelope.

Run tests: `cd backend && go test ./...`

## Frontend

A Next.js (App Router) app that talks to the Go API through a small BFF layer: Route
Handlers under `frontend/app/api/` exchange credentials for a JWT server-side and store
it in an httpOnly cookie, then proxy authenticated requests to the backend. See
`frontend/README.md` (once scaffolded) for details.

## Deploying

The backend deploys to Render via `backend/render.yaml`. Frontend deployment
configuration is not yet set up.
