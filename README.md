# MapCTF

Lean, map-based CTF and cyber-range platform built with a modern React UI and a Go backend.

## Highlights

- React 18 + TypeScript frontend reusing the familiar FBCTF look.
- Go 1.21 API with PostgreSQL and Redis backing stores.
- Docker-first workflow plus opt-in local development for each service.

```text
mapctf/
├── frontend/   React UI
├── backend/    Go API + services
├── deploy/     Docker + scripts
└── static/     Shared assets
```

## Quick start

1. Install Docker, Node.js 18+, and Go 1.21+.
2. Copy env defaults: `cp .env.example .env`.
3. Adjust `.env` as needed (DB creds, ports).
4. Run everything: `docker compose -f docker-compose-dev.yml up` or use `make docker_dev`.
5. Visit [http://localhost:3000](http://localhost:3000) (frontend) or [http://localhost:9001](http://localhost:9001) (API).

### Work on individual services

```bash
# Frontend
cd frontend && npm install && npm run dev    # serves on :3000

# Backend
cd backend && go mod download && make run    # serves on :8080
```

## Feature set

- Players: interactive map, team play, live scoreboard, category filters, progress tracking.
- Admins: browser-based control panel for challenges, teams, stats, and data import/export.

## Tech stack

| Layer    | Tools                                   |
|----------|-----------------------------------------|
| Frontend | React 18, TypeScript, Vite, React Router |
| Backend  | Go 1.21, REST API, PostgreSQL 15, Redis 7 |

## Configuration

Most knobs live in `.env` (versions, DB creds, Redis). Start from `.env.example` and adjust as needed. Backend-specific flags are documented in `backend/README.md`.

## API

Core endpoints cover auth, team/challenge listings, submissions, and admin stats. Full details and request/response schemas live in `backend/README.md`.

## Development helpers

```bash
# Frontend scripts
npm run dev | build | preview | lint

# Backend make targets
make build | run_api | clean
```

## Contributing & license

Fork, branch, commit, push, and open a PR. MapCTF ships under GPL-3.0; see `LICENSE`. Questions or bugs? Open an issue. Design inspiration courtesy of Facebook's archived FBCTF project.
