# Development Guide

## Prerequisites

- [mise](https://mise.jdx.dev/) — manages tool versions and tasks
- [Docker](https://docs.docker.com/get-docker/) — runs PostgreSQL (and the dev
  server) in containers

## Getting Started

```sh
mise install       # Install pinned tool versions (Go, sqlc, migrate, air)
mise run dev       # Start everything
```

`mise run dev` runs `docker compose up`, which:

1. Starts PostgreSQL 18 (auto-creates the `camp_scheduler` user + dev/test
   databases on first run via `local-setup.sql`)
2. Builds a dev container with Go, air (hot-reload), and golang-migrate
3. Runs all pending migrations on both dev and test databases
4. Starts the Go server via air — watches for file changes and rebuilds
   automatically

The server is available at `http://localhost:9100`. Press Ctrl+C to stop
everything.

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
| `mise run test:integration` | Run integration tests (requires local database) |
| `mise run sqlc`     | Regenerate Go code from SQL queries                  |
| `mise run migrate`  | Run pending migrations on the dev database           |
| `mise run migrate -- --version N` | Migrate dev database to version N     |
| `mise run migrate:test` | Run pending migrations on the test database      |
| `mise run migrate:all`  | Run migrations on both dev and test databases    |
| `mise run migration <name>` | Create a new migration file                  |
| `mise run db:reset` | Destroy local databases and volumes                  |

### Docker Compose

`docker-compose.yml` defines two services:

- **postgres** — PostgreSQL 18 with a named volume for data persistence. On
  first start, runs `database/local-setup/local-setup.sql` to create the app
  user and databases.
- **server** — Dev container that runs migrations and starts air. Bind-mounts
  the project directory so file changes trigger hot-reload.

### air (hot-reload)

[air](https://github.com/air-verse/air) watches `.go` files and rebuilds/restarts
the server on changes. Configuration lives in `.air.toml`. Air runs inside the
server Docker container when using `mise run dev`, or standalone on the host via
`mise run server`.

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

`DATABASE_URL` and `DATABASE_URL_TEST` are automatically constructed from the
above variables via mise templates.

Note: the Docker Compose server service defines its own environment with
`DATABASE_HOST=postgres` (the Docker network hostname) so the containerized
server connects to the containerized Postgres. The mise env vars with
`DATABASE_HOST=localhost` are used by host-based tools like `mise run migrate`.

## Database

See [database/README.md](../database/README.md) for database-specific details
including migrations and reset instructions.

## Testing

```sh
mise run test              # Unit tests
mise run test:integration  # Integration tests (needs running Postgres with migrations applied)
```

Integration tests connect to a `camp_scheduler_test` database. When using
`mise run dev`, migrations are applied to both databases automatically.

For standalone testing, start Postgres and run migrations first:

```sh
docker compose up -d postgres          # Start just Postgres
mise run migrate:all                   # Apply migrations to both databases
mise run test:integration              # Run integration tests
```

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
