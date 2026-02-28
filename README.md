# MapCTF

<p align="center">
  <img alt="mapctf" src="mapctf.png" width="180" />
  <p align="center">
    Map-based CTF/cyber-range platform.
  </p>
  <p align="center">
    <a href="https://github.com/jmpsec/mapctf/blob/master/LICENSE">
      <img alt="Software License" src="https://img.shields.io/badge/license-MIT-green?style=flat-square&fuckgithubcache=1">
    </a>
    <a href="https://github.com/jmpsec/mapctf">
      <img alt="Build Status" src="https://github.com/jmpsec/mapctf/actions/workflows/build_and_test_main_merge.yml/badge.svg?branch=main&fuckgithubcache=1">
    </a>
    <a href="https://goreportcard.com/report/github.com/jmpsec/mapctf">
      <img alt="Go Report Card" src="https://goreportcard.com/badge/github.com/jmpsec/mapctf?style=flat-square&fuckgithubcache=1">
    </a>
  </p>
</p>

MapCTF is a map-based CTF/cyber-range platform with:

- A Go backend (API + map service)
- A React + TypeScript frontend
- PostgreSQL and Redis for state/session storage

> [!WARNING]
> MapCTF is under active development. Review configuration and deployment settings before running in production.

## Architecture

The development stack is split into four main runtime components:

- `mapctf-api` on `:8081` (REST/API workflows)
- `mapctf-map` on `:8082` (map/gameboard and template/static serving)
- `mapctf-postgres` on `:5432` (persistent relational data)
- `mapctf-redis` on `:6379` (cache and session store)

In this repository, frontend development runs separately with Vite (default `:3000`).

## Project Structure

```text
mapctf/
├── backend/
│   ├── cmd/
│   │   ├── api/            # API service entrypoint
│   │   ├── map/            # Map service entrypoint + handlers/templates
│   │   └── cli/            # CLI utilities
│   ├── pkg/
│   │   ├── backend/        # DB layer
│   │   ├── cache/          # Redis/cache primitives
│   │   ├── challenges/     # Challenge domain logic
│   │   ├── config/         # Config types/flags/validation
│   │   ├── teams/          # Team domain logic
│   │   ├── users/          # User/auth domain logic
│   │   └── version/        # Version metadata
│   └── Makefile            # Backend build/run/test targets
├── frontend/
│   ├── src/
│   │   ├── components/     # Reusable UI components
│   │   ├── pages/          # Route/page components
│   │   ├── services/       # API client calls
│   │   ├── contexts/       # React context state
│   │   ├── styles/         # SCSS/CSS modules
│   │   ├── types/          # TS models/types
│   │   └── utils/          # UI helpers
│   ├── public/             # Public assets
│   └── htmls/              # Legacy/static map HTML assets
├── deploy/
│   ├── config/             # Example configuration files
│   └── docker/
│       ├── conf/
│       └── dockerfiles/    # Dev Dockerfiles for api/map/frontend
├── docker-compose-dev.yml  # Dev services (api, map, postgres, redis)
├── Makefile                # Root convenience targets
└── ARCHITECTURE.md         # Deeper architecture notes
```

## Quick Start (Docker)

1. Copy environment defaults:
```bash
cp .env.example .env
```
2. Start backend services:
```bash
docker compose -f docker-compose-dev.yml up --build
```
3. Start frontend (separate terminal):
```bash
cd frontend
npm install
npm run dev
```
4. Open:
- Frontend: `http://localhost:3000`
- API: `http://localhost:8081`
- Map service: `http://localhost:8082`

## Local Development (No Docker for Go Services)

Start only DB/Redis with Docker:
```bash
make up-backend
```

Run services locally:
```bash
# API
cd backend
make run_api

# Map service
cd backend
make run_map

# Frontend
cd frontend
npm install
npm run dev
```

## Useful Commands

Root `Makefile`:

- `make docker_dev_build` build dev images
- `make docker_dev_up` run compose stack
- `make docker_dev_down` stop compose stack
- `make docker_dev_logs_api` tail API logs
- `make docker_dev_logs_map` tail map logs

Backend `Makefile` (`backend/`):

- `make build` build API + map binaries
- `make run_api` run API with `go run`
- `make run_map` run map service with `go run`
- `make test` run backend package tests

Frontend (`frontend/`):

- `npm run dev` start Vite dev server
- `npm run build` production build
- `npm run lint` lint TS/React code

## Configuration

Primary development variables are in `.env` (see `.env.example`), including:

- Toolchain/image versions (`GOLANG_VERSION`, `NODE_VERSION`)
- Database credentials/ports
- Redis settings
- Bootstrap admin credentials
- Map UUID (`MAP_UUID`)

Backend also supports CLI flags and YAML configuration (example: `deploy/config/mapctf.example.yaml`).

## Additional Docs

- [ARCHITECTURE.md](ARCHITECTURE.md) deeper dives into design decisions, data models, and service interactions.

## License

GPL-3.0. See [LICENSE](LICENSE).
