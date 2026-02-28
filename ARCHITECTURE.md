# MapCTF Architecture

## Overview

MapCTF is a map-based CTF platform with two backend services:
- `mapctf-api` (JSON API for frontend/admin workflows)
- `mapctf-map` (server-rendered map/gameboard pages + static assets)

Data is persisted in PostgreSQL and session/cache state is stored in Redis.

## Key Design Points

- UUID-scoped game instances (not entity-based tenancy)
- JWT authentication for API endpoints
- Redis-backed sessions (SCS) for map web flows
- React + TypeScript frontend for API-driven UI
- Docker-first local development

## Runtime Architecture

```text
┌─────────────────┐      HTTP       ┌────────────────┐
│  React Frontend │◄───────────────►│   mapctf-api   │
│   (Vite :3000)  │                 │     (:8081)    │
└─────────────────┘                 └───────┬────────┘
                                            │
┌─────────────────┐      HTTP               │
│ Browser (Map UI)│◄────────────────────────┘
│     /{uuid}     │                         ┌─────────────────┐
└────────┬────────┘                         │   PostgreSQL    │
         │                                  │   persistent    │
         └──────────────► mapctf-map (:8082)│     data        │
                              │             └─────────────────┘
                              │
                              ▼
                        ┌──────────────┐
                        │    Redis     │
                        │ cache/session│
                        └──────────────┘
```

## Repository Structure

```text
mapctf/
├── backend/
│   ├── cmd/
│   │   ├── api/            # API service entrypoint and handlers
│   │   ├── map/            # Map service entrypoint and handlers/templates
│   │   └── cli/            # CLI utilities
│   ├── pkg/
│   │   ├── backend/        # DB connection and setup
│   │   ├── cache/          # Redis manager
│   │   ├── challenges/     # Challenge + category domain logic
│   │   ├── config/         # Flags, YAML config, validation
│   │   ├── teams/          # Team domain logic
│   │   ├── users/          # User/auth domain logic
│   │   └── version/        # Build/version metadata
│   └── Makefile
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── contexts/
│   │   ├── pages/
│   │   ├── services/
│   │   ├── styles/
│   │   ├── types/
│   │   └── utils/
│   ├── public/
│   └── htmls/
├── deploy/
│   ├── config/
│   └── docker/
├── docker-compose-dev.yml
└── tools/
```

## Backend Services

### `mapctf-api`

Responsibilities:
- Login/logout and JWT token issuance
- Team/challenge read endpoints for gameboard
- Admin endpoints for team/challenge management

Route shape:
- Base prefix: `/api/v1/{uuid}`
- Public:
  - `GET /api/v1/{uuid}/checks-no-auth`
  - `POST /api/v1/{uuid}/auth/login`
  - `GET /api/v1/{uuid}/auth/logout`
- JWT-protected:
  - `GET /api/v1/{uuid}/checks-auth`
  - `GET /api/v1/{uuid}/teams`
  - `GET /api/v1/{uuid}/challenges`
  - `GET /api/v1/{uuid}/admin/teams`
  - `POST /api/v1/{uuid}/admin/teams`
  - `GET /api/v1/{uuid}/admin/challenges`
  - `POST /api/v1/{uuid}/admin/challenges`

### `mapctf-map`

Responsibilities:
- Server-rendered pages (`login`, `registration`, `countdown`, `gameboard`)
- Static/template asset serving
- Cookie-based session flow for browser users

Route shape:
- Service-level:
  - `GET /health`
  - `GET /favicon.ico`
  - `GET /static/*`
- UUID-scoped pages:
  - `GET /{uuid}/`
  - `GET /{uuid}/login`
  - `POST /{uuid}/login`
  - `GET /{uuid}/registration`
  - `POST /{uuid}/registration`
  - `GET /{uuid}/countdown`
  - `POST /{uuid}/logout`
  - `GET /{uuid}/gameboard` (authenticated)

## Domain Packages

- `pkg/users`: user CRUD, password hashing (bcrypt), JWT create/verify, UUID-scoped lookup
- `pkg/teams`: team CRUD, membership and score models, UUID-scoped query paths
- `pkg/challenges`: challenge/category CRUD, UUID-scoped query paths
- `pkg/backend`: DB lifecycle and retries
- `pkg/cache`: Redis lifecycle and retries

## Authentication Model

### API (`mapctf-api`)

- JWT-based authentication.
- Client sends `Authorization: Bearer <token>`.
- Middleware validates token and injects user claims into request context.

### Map (`mapctf-map`)

- Session-based authentication via `github.com/alexedwards/scs/v2`.
- Session data stored server-side in Redis (`goredisstore`).
- Browser receives a secure HTTP-only session cookie.

## Data Model

Primary tables are created through GORM automigrations in manager packages:
- `platform_users`
- `platform_teams`
- `team_memberships`
- `team_scores`
- `team_logos`
- `challenges`
- `categories`

Instance isolation is implemented with a `uuid` field in core records and query filters.

## UUID Scope (Replaces Entity Model)

MapCTF now scopes data and routes by UUID:
- URL paths use `{uuid}` as instance selector.
- Manager queries filter by `uuid`.
- Same username/team names can exist across different UUID instances.

Legacy naming (`entID`) still appears in some frontend types/variables, but active backend route and data scoping is UUID-based.

## Frontend Architecture

Frontend is a React + TypeScript SPA (`frontend/src`) with:
- `pages/` for route screens
- `components/` for reusable UI blocks
- `services/` for API calls
- `contexts/` for app state
- `utils/api.ts` for request wrapper + auth header injection

`VITE_API_URL` controls API target (defaults to `http://localhost:8081`).

## Deployment Model

`docker-compose-dev.yml` defines:
- `mapctf-api`
- `mapctf-map`
- `mapctf-postgres`
- `mapctf-redis`

Frontend dev server is typically run separately with `npm run dev`.

## Security Notes

- Passwords are hashed with bcrypt.
- JWT secret/expiration are configurable via backend config.
- Map sessions use secure cookie flags (`HttpOnly`, `Secure`) and Redis-backed server-side storage.
- Protected routes enforce authentication middleware per service model.

## Testing and Tooling

- Backend unit tests are in package-level `*_test.go` files.
- `backend/Makefile` provides `build`, `run_api`, `run_map`, `test`.
- Root `Makefile` provides Docker-oriented workflows.
- `tools/api-tester.py` can be used for API-level checks.

## Current Gaps / Follow-ups

- Some comments/docs and frontend symbols still use old `entID` terminology.
- Frontend API path construction should stay aligned with UUID-scoped backend routes.
- Swagger comments in handlers may need cleanup where they reference outdated route forms.
