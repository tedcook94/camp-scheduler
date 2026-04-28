# Development Guide

## Prerequisites

- [mise](https://mise.jdx.dev/) — manages tool versions and tasks
- [Docker](https://docs.docker.com/get-docker/) — runs PostgreSQL in a container

## Getting Started

```sh
mise install        # Install pinned tool versions (Go, sqlc, migrate, air, node, pnpm)
mise run auth:install  # Install auth-server dependencies (first time only)
mise run dev        # Start Postgres + auth-server + Go server + Vite frontend with hot-reload
```

`mise run dev` starts the full local development environment:

1. Starts PostgreSQL 18 in Docker (auto-creates the `camp_scheduler` user +
   dev/test databases on first run via `local-setup.sql`)
2. Runs all pending Go-side migrations on both dev and test databases
3. Runs BetterAuth schema migrations against the dev database
4. Starts the BetterAuth auth-server on port 9101 (TypeScript service in `auth/`)
5. Starts the Go server via air on port 9100 — watches for file changes and rebuilds
6. Starts the Vite dev server for the frontend on port 5173 — proxies `/api/auth/*`
   to the auth-server and `/api/*` to the Go server

The frontend is accessible at `http://localhost:5173` during development
and at `http://localhost:9100` in production builds.

To run a single component:

```sh
mise run server    # Go server only (assumes Postgres is running)
mise run auth:dev  # Auth-server only (assumes Postgres is running)
mise run web       # Vite frontend only
```

All environment variables are provided by mise (defined in `mise.toml`), so the
server runs directly on the host. Press Ctrl+C to stop everything — Postgres is
automatically shut down via `docker compose down`.

## Tooling

### mise

[mise](https://mise.jdx.dev/) pins tool versions and provides task commands.
Configuration lives in `mise.toml`.

Activate mise in your shell to get the pinned tool versions on your PATH:

```sh
mise activate zsh  # or bash/fish — add to your shell rc file
```

Override any value locally with `mise.local.toml` (gitignored).

### Available Tasks

| Task                | Description                                          |
| ------------------- | ---------------------------------------------------- |
| `mise run dev`      | Start Postgres, auth-server, Go server, and Vite frontend |
| `mise run server`   | Run Go server standalone (assumes Postgres is running) |
| `mise run auth:install` | Install auth-server dependencies                |
| `mise run auth:dev`     | Run auth-server with hot-reload (port 9101)     |
| `mise run auth:migration <name>` | Generate a new auth-schema migration from the live BetterAuth config |
| `mise run build`    | Build frontend + server binary to `bin/server`       |
| `mise run test`     | Run unit tests                                       |
| `mise run test:integration` | Run integration tests (auto-starts Postgres) |
| `mise run sqlc`     | Regenerate Go code from SQL queries                  |
| `mise run migrate`  | Run pending app-schema migrations on the dev database |
| `mise run migrate -- --version N` | Migrate dev app schema to version N    |
| `mise run migrate:test` | Run pending app-schema migrations on the test database |
| `mise run migrate:auth` | Run pending auth-schema migrations on the dev database |
| `mise run migrate:auth:test` | Run pending auth-schema migrations on the test database |
| `mise run migrate:all`  | Run all migrations (app + auth) on both dev and test |
| `mise run migration <name>` | Create a new app-schema migration file       |
| `mise run db:reset` | Destroy local databases and volumes                  |
| `mise run seed:super-admin` | Create a super-admin user (interactive)        |
| `mise run seed:demo` | Seed demo camp data (idempotent: deletes and recreates) |
| `mise run web`      | Start Vite dev server with HMR (port 5173)           |
| `mise run web:build`| Build frontend static files                          |
| `mise run web:check`| Run svelte-check type checking                       |

### Docker Compose

`docker-compose.yml` defines the Postgres service:

- **postgres** — PostgreSQL 18 (built from `database/postgres/Dockerfile`).
  Uses a named volume for data persistence. On first start, runs
  `database/local-setup/local-setup.sql` to create the app user and databases.

### Auth server

The BetterAuth auth-server (`auth/`) is a TypeScript service that:

- Mints short-lived (15m) Ed25519-signed JWTs the Go server validates via JWKS
- Owns the BetterAuth-managed tables (`user`, `session`, `account`,
  `verification`, `organization`, `member`, `invitation`, `jwks`) in the same
  Postgres database used by the Go server
- Exposes BetterAuth's standard `/api/auth/*` endpoints (sign-in, sessions,
  admin user CRUD, organization member management, JWT issuance via the `jwt`
  plugin)
- Exposes a small `/internal/*` API (protected by `AUTH_SHARED_SECRET`) that the
  Go server calls to keep the `organization` table in sync with `camps`

See `auth/README.md` for endpoint details.

### air (hot-reload)

[air](https://github.com/air-verse/air) watches `.go` files and rebuilds/restarts
the server on changes. Configuration lives in `.air.toml`. Air runs on the host
when using `mise run dev` or standalone via `mise run server`.

## Environment Variables

All environment variables are defined in `mise.toml` under `[env]`. Key
variables:

| Variable              | Default           | Description                |
| --------------------- | ----------------- | -------------------------- |
| `PORT`                | `9100`            | Server listen port         |
| `MODE`                | `local`           | Runtime mode               |
| `LOG_LEVEL`           | `info`            | Log level (debug/info/warn/error) |
| `DATABASE_HOST`       | `localhost`        | Postgres host              |
| `DATABASE_PORT`       | `5432`            | Postgres port              |
| `DATABASE_USER`       | `camp_scheduler`  | Privileged Postgres role (migrations, admin/seed CLIs) |
| `DATABASE_PASSWORD`   | `p@ss123`         | Password for `camp_scheduler` |
| `DATABASE_NAME`       | `camp_scheduler`  | Database name              |
| `DATABASE_SSL_MODE`   | `disable`         | Postgres SSL mode          |
| `DATABASE_TIMEOUT`    | `10s`             | Connection timeout         |
| `APP_DATABASE_URL`    | (constructed)     | Runtime URL for the Go server (uses `app_user` role, search_path = app, public) |
| `AUTH_DATABASE_URL`   | (constructed)     | Runtime URL for the BetterAuth auth-server (uses `auth_user` role, search_path = auth, public) |
| `APP_MIGRATE_URL`     | (constructed)     | golang-migrate URL for the `app` schema migrations (privileged + `search_path=app`) |
| `AUTH_MIGRATE_URL`    | (constructed)     | golang-migrate URL for the `auth` schema migrations (privileged + `search_path=auth`) |
| `AUTH_PORT`           | `9101`            | Auth-server listen port    |
| `AUTH_BASE_URL`       | `http://localhost:9101` | Public URL the auth-server is reachable at |
| `AUTH_SERVER_URL`     | `http://localhost:9101` | URL the Go server uses to reach the auth-server (JWKS + `/internal/*`) |
| `AUTH_SHARED_SECRET`  | (dev value)       | Shared bearer token for Go ↔ auth-server `/internal/*` calls |
| `BETTER_AUTH_SECRET`  | (dev value)       | BetterAuth signing secret for cookies/CSRF (min 32 chars) |

`DATABASE_URL`, `DATABASE_URL_TEST`, and the `APP_*` / `AUTH_*` URL variants
are automatically constructed from the discrete `DATABASE_*` variables via
mise templates. See `mise.toml` for the exact composition.

## Database

See [database/README.md](../database/README.md) for database-specific details
including migrations and reset instructions.

## Testing

```sh
mise run test              # Unit tests
mise run test:integration  # Integration tests
```

Integration tests connect to a `camp_scheduler_test` database.
`mise run test:integration` automatically starts Postgres (if not already
running), applies migrations, runs the tests, and stops Postgres afterwards if
it was started by the task. If Postgres was already running (e.g., from
`mise run dev`), it is left as-is.

## Demo Data

```sh
mise run seed:demo
```

Seeds a fully configured "Demo Camp" with realistic data for testing and
demonstration. The tool is idempotent — running it again deletes all existing
demo data and recreates it from scratch.

What it creates:

- 1 camp ("Demo Camp") with 1 season and 2 linked sessions
- 3 age groups, 6 cabins, 6 activities, 4 time slots, 3 certifications
- Full session configuration (age groups, cabins, time slots, activities per
  session)
- 12 counselors with certifications and preferences (age group, co-counselor,
  activity) varying between sessions
- 4 counselor session history entries (from Session 1)
- 36 campers with friend preferences, enrolled in both sessions

> **Note:** When `AUTH_SERVER_URL` and `AUTH_SHARED_SECRET` are set
> (which they are in the default `mise.toml`), `seed:demo` also provisions
> the demo camp's organization on the auth-server and creates a demo admin
> user assigned to it. The demo admin signs in with username `demo` and
> password `demo1234`. Re-running `seed:demo` deletes and recreates that
> user. If the auth-server env vars are not set, the auth-side
> provisioning is skipped (the camp data still seeds).

**Intentional constraint failure:** Session 2's activity schedule is deliberately
unsolvable. Its Morning 2 time slot has both Swimming and Canoeing, which each
require Lifeguard certification (4 Lifeguard slots total), but only 3 counselors
hold that certification. This demonstrates what happens when hard constraints
cannot be satisfied. Session 1's activity schedule is solvable.

## API Testing

API definitions are maintained in `yaak/` as a Yaak workspace sync directory.
See [yaak/README.md](../yaak/README.md) for usage instructions. Keep these
definitions in sync when adding or changing endpoints.

Yaak auto-saves every UI change to the sync directory, including request body
edits made during testing. Review staged `yaak/` files before committing to
verify the changes are intentional and not leftover test data.

## Code Generation

After modifying SQL queries in `database/queries/`:

```sh
mise run sqlc
```

This regenerates the Go code in `internal/db/`.
