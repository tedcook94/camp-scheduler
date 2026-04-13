# Development Guide

## Prerequisites

- [mise](https://mise.jdx.dev/) — manages tool versions and tasks
- [Docker](https://docs.docker.com/get-docker/) — runs PostgreSQL in a container

## Getting Started

```sh
mise install       # Install pinned tool versions (Go, sqlc, migrate, air)
mise run dev       # Start everything
```

`mise run dev` starts the full local development environment:

1. Starts PostgreSQL 18 with pg_cron in Docker (auto-creates the
   `camp_scheduler` user + dev/test databases on first run via
   `local-setup.sql`)
2. Runs all pending migrations on both dev and test databases
3. Starts the Go server via air — watches for file changes and rebuilds
   automatically

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
| `mise run dev`      | Start Postgres + server with hot-reload              |
| `mise run server`   | Run server standalone (assumes Postgres is running)  |
| `mise run build`    | Build the server binary to `bin/server`              |
| `mise run test`     | Run unit tests                                       |
| `mise run test:integration` | Run integration tests (auto-starts Postgres) |
| `mise run sqlc`     | Regenerate Go code from SQL queries                  |
| `mise run migrate`  | Run pending migrations on the dev database           |
| `mise run migrate -- --version N` | Migrate dev database to version N     |
| `mise run migrate:test` | Run pending migrations on the test database      |
| `mise run migrate:all`  | Run migrations on both dev and test databases    |
| `mise run migration <name>` | Create a new migration file                  |
| `mise run db:reset` | Destroy local databases and volumes                  |

### Docker Compose

`docker-compose.yml` defines the Postgres service:

- **postgres** — PostgreSQL 18 with pg_cron (built from
  `database/postgres/Dockerfile`). Uses a named volume for data persistence. On
  first start, runs `database/local-setup/local-setup.sql` to create the app
  user and databases.

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
| `DATABASE_USER`       | `camp_scheduler`  | Postgres user              |
| `DATABASE_PASSWORD`   | `p@ss123`         | Postgres password          |
| `DATABASE_NAME`       | `camp_scheduler`  | Database name              |
| `DATABASE_SSL_MODE`   | `disable`         | Postgres SSL mode          |
| `DATABASE_TIMEOUT`    | `10s`             | Connection timeout         |
| `JWT_SECRET`          | (dev value)       | Signing key for JWT tokens |

`DATABASE_URL` and `DATABASE_URL_TEST` are automatically constructed from the
above variables via mise templates.

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

## API Testing

API definitions are maintained in `insomnia/` as an Insomnia workspace export.
See [insomnia/README.md](../insomnia/README.md) for usage instructions. Keep
these definitions in sync when adding or changing endpoints.

## Code Generation

After modifying SQL queries in `database/queries/`:

```sh
mise run sqlc
```

This regenerates the Go code in `internal/db/`.
