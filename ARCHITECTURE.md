# MapCTF Architecture

## Overview

MapCTF is split into two backend services:

- `mapctf-api` for JSON API workflows.
- `mapctf-map` for the browser-facing map experience, session-backed gameplay pages, JSON feeds, and admin operations.

The current backend design is UUID-scoped. A single map service instance is configured for one active game UUID, and nearly every browser route and persistence lookup is filtered through that UUID.

This document focuses on the backend runtime, with emphasis on the current `mapctf-map` service. Frontend implementation details are intentionally omitted.

## Runtime Topology

```text
┌─────────────────────┐
│   Browser Client    │
│ login/gameboard/    │
│ admin pages         │
└──────────┬──────────┘
           │ HTTP + session cookie
           ▼
┌──────────────────────────────┐
│        mapctf-map            │
│           (:8082)            │
│                              │
│ - HTML template rendering    │
│ - static asset delivery      │
│ - session auth via SCS       │
│ - JSON feeds for game UI     │
│ - admin import/export flows  │
└──────────┬───────────┬────────┘
           │           │
           │           └──────────────────────┐
           │                                  │
           ▼                                  ▼
┌─────────────────────┐              ┌─────────────────────┐
│     PostgreSQL      │              │        Redis        │
│ domain + settings + │              │ session store via   │
│ logs + map metadata │              │ goredisstore/SCS    │
└─────────────────────┘              └─────────────────────┘
```

## Repository Areas

```text
mapctf/
├── backend/
│   ├── cmd/
│   │   ├── api/            # JSON API service
│   │   ├── cli/            # CLI utilities
│   │   └── map/            # Map service entrypoint, handlers, templates
│   └── pkg/
│       ├── backend/        # DB lifecycle and retry logic
│       ├── cache/          # Redis lifecycle and retry logic
│       ├── challenges/     # Challenges and categories
│       ├── config/         # Flags, YAML config, validation
│       ├── countries/      # SVG/map country metadata and assignment state
│       ├── logs/           # Activity, announcement, scoreboard, hint, failure, registration logs
│       ├── settings/       # UUID-scoped runtime settings and audit logs
│       ├── teams/          # Teams, memberships, scores, logos
│       ├── users/          # Users, auth, sessions metadata
│       └── version/        # Build metadata
├── deploy/
├── docker-compose-dev.yml
└── tools/
```

## Map Service Startup

`backend/cmd/map/main.go` wires the service in this order:

1. Load config and ensure a UUID exists.
2. Connect to PostgreSQL with retry support.
3. Connect to Redis with retry support.
4. Initialize managers:
   - `teams`
   - `users`
   - `challenges`
   - `settings`
   - `logs`
   - `countries`
5. Seed mutable runtime data:
   - team logos from `Map.LogosFile`
   - countries from `Map.CountriesFile`
   - default settings via `SettingsManager.Initialization()`
6. Create an SCS session manager backed by Redis.
7. Build a Chi router and register public, authenticated, JSON, and admin routes.

The map service also optionally enables request-level HTTP debug logging through a rotating `lumberjack` logger when `DebugHTTP.Enabled` is configured.

## Map Service Responsibilities

### Public/browser routes

The map service renders the user-facing HTML templates under `backend/cmd/map/templates`:

- landing page: `GET /{uuid}/`
- login page: `GET /{uuid}/login`
- registration page: `GET /{uuid}/registration`
- countdown page: `GET /{uuid}/countdown`
- rules page: `GET /{uuid}/rules`
- form posts:
  - `POST /{uuid}/login`
  - `POST /{uuid}/registration`
  - `POST /{uuid}/logout`

Service-level utility routes are also served directly:

- `GET /health`
- `GET /error`
- `GET /forbidden`
- `GET /favicon.ico`
- `GET /static/*`

### Authenticated gameplay routes

After session authentication, the service exposes:

- `GET /{uuid}/gameboard`
- JSON feeds under `/{uuid}/json/*`:
  - `GET /activity`
  - `GET /announcements`
  - `GET /challenges`
  - `GET /teams`

These JSON endpoints are used by the gameboard/admin templates to refresh dynamic state without moving that logic into the API service.

### Admin control plane

The map service now owns a substantial admin surface under `/{uuid}/admin`, gated by both session auth and admin role checks.

Main admin pages:

- `GET /admin/`
- `GET /admin/settings`
- `GET /admin/controls`
- `GET /admin/challenges`
- `GET /admin/users`
- `GET /admin/teams`
- `GET /admin/activity`
- `GET /admin/announcements`
- `GET /admin/countries`

Admin write paths include:

- settings update, export, import, and reset-to-defaults
- full game export/import
- challenge CRUD, category CRUD, bulk enable/disable/delete, import/export
- user CRUD, bulk enable/disable/delete, import/export
- team CRUD, team visibility toggles, team bulk delete, import/export
- team logo CRUD, bulk enable/disable/delete, custom SVG uploads
- country active-state updates and challenge assignment views

The admin handlers support both browser form workflows and JSON/XHR-style responses depending on request headers.

## Routing and Middleware

The `mapctf-map` router uses Chi with:

- request IDs
- real IP extraction
- request logging
- panic recovery
- 30-second request timeout
- `sessionManager.LoadAndSave` for SCS session persistence

Route protection is layered:

- public routes are available under `/{uuid}` without auth
- `RequireAuth` redirects unauthenticated users to `/{uuid}/login`
- `RequireAdmin` blocks non-admin users from `/admin` routes

Every template and mutation handler also validates that the URL UUID matches the configured service UUID before continuing.

## Authentication and Session Model

The map service does not use JWTs for browser flows. It uses SCS server-side sessions stored in Redis.

Session behavior from `backend/cmd/map/main.go`:

- cookie name: `session_id`
- `HttpOnly`: enabled
- `Secure`: enabled
- path: `/`
- persistent cookie: enabled
- lifetime: 24 hours
- idle timeout: 30 minutes

On login:

- credentials are checked through `pkg/users`
- the SCS token is renewed
- username and admin status are stored in the session
- user session metadata is updated with IP address and user agent

On logout:

- the SCS session is destroyed
- the client is redirected back to `/{uuid}/login`

## Settings Subsystem

The map service now depends heavily on `pkg/settings`, which auto-migrates:

- `platform_settings`
- `setting_logs`

`SettingsManager.Initialization()` creates missing default settings for the active UUID. Current setting groups include:

- booleans:
  - `login_enabled`
  - `login_strong_passwords`
  - `registration_enabled`
  - `registration_names`
  - `registration_emails`
  - `scoring_enabled`
  - `game_paused`
  - `game_started`
- strings:
  - `custom_org`
  - `custom_logo`
  - `language`
  - `registration_token`
- dates:
  - `game_start_time`
  - `game_end_time`
- integers:
  - `registration_type`
  - `leaderboard_limit`

These settings directly control browser and admin behavior such as:

- whether login is open to non-admin users
- whether registration is open, token-gated, or requires extra fields
- event timing and paused/started state
- scoring visibility and leaderboard size
- custom branding strings/assets

All settings mutations are audited through `setting_logs`.

## Countries and Map Metadata

`pkg/countries` is a newer part of the runtime model and auto-migrates `map_countries`.

Countries are seeded from a JSON file referenced by `Map.CountriesFile`. Each record stores:

- country name and code
- active/inactive state
- whether it is assigned to a challenge
- the assigned challenge ID
- SVG/vector metadata used to render the map:
  - land path/class/style
  - marker path/class/style/transform

The admin countries page combines this country data with challenge lookups so operators can understand which challenges are attached to which map regions.

## Logs and Activity Data

`pkg/logs` now provides the data backing both JSON feeds and admin reporting. It auto-migrates:

- `activity_logs`
- `announcements`
- `scoreboard_logs`
- `hints_logs`
- `failures_logs`
- `registration_logs`

The map service currently surfaces:

- activity streams through `/json/activity` and the admin activity page
- announcements through `/json/announcements` and the admin announcements page

This keeps event-style operational data close to the browser-facing service rather than routing it through the API service.

## Teams, Users, and Challenges in the Map Service

The map service composes several domain managers rather than reimplementing domain logic:

- `pkg/teams`
  - teams
  - memberships
  - scores
  - team logos
  - initial logo seeding
- `pkg/users`
  - credential validation
  - user registration
  - session metadata updates
- `pkg/challenges`
  - challenge CRUD
  - category CRUD
  - active challenge retrieval for the gameboard

Admin import/export handlers in `backend/cmd/map/handlers/admin.go` orchestrate these managers to move settings, users, teams, logos, categories, and challenges as versioned JSON payloads.

## Persistence Model

The map-side backend currently persists at least the following table families through GORM auto-migrations in the manager packages:

- users:
  - `platform_users`
- teams:
  - `platform_teams`
  - `team_memberships`
  - `team_scores`
  - `team_logos`
- challenges:
  - `challenges`
  - `categories`
- map metadata:
  - `map_countries`
- settings/audit:
  - `platform_settings`
  - `setting_logs`
- event logs:
  - `activity_logs`
  - `announcements`
  - `scoreboard_logs`
  - `hints_logs`
  - `failures_logs`
  - `registration_logs`

UUID scoping is enforced across the core game data so multiple game instances can coexist in the same database without route or query collisions.

## Deployment Notes

For local development, `docker-compose-dev.yml` runs:

- `mapctf-api`
- `mapctf-map`
- PostgreSQL
- Redis

The map service depends on both PostgreSQL and Redis being reachable during startup because it initializes domain managers before serving traffic.

## Known Constraints

- The map service is configured for one active UUID at a time, even though the database schema itself is UUID-scoped.
- Some legacy naming and older documentation may still refer to earlier entity or frontend-driven flows.
- `FaviconHandler` serves `/static/img/favicon.png`, so static asset pathing needs to remain aligned with deployment layout.
