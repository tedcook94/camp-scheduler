# camp-scheduler Database

## Overview

The local PostgreSQL database runs in a Docker container managed by Docker
Compose. On first start, the container automatically runs `local-setup.sql` to
provision roles, schemas, and both the dev and test databases.

`mise run dev` handles everything automatically: it starts Postgres, runs all
pending migrations, and starts the Go server with hot-reload.

## Schemas and roles

Tables live in two schemas, owned by separate runtime roles:

| Schema | Owner          | Runtime role | Used by                     |
| ------ | -------------- | ------------ | --------------------------- |
| `app`  | `camp_scheduler` | `app_user`  | Go server (sqlc queries)    |
| `auth` | `camp_scheduler` | `auth_user` | BetterAuth auth-server      |

Each runtime role has `search_path` fixed to its schema (plus `public`), so
unqualified identifiers in queries resolve to the right schema without
per-query qualification. The privileged `camp_scheduler` role is reserved for
schema migrations and admin/seed CLI tools.

## First-Time Setup

```sh
mise install       # Install pinned tool versions
mise run dev       # Start everything (Postgres + migrations + server)
```

No additional database setup commands are required.

## Resetting the Database

To destroy all local data and start fresh:

```sh
mise run db:reset  # Stops containers and removes all volumes
mise run dev       # Reinitializes everything from scratch
```

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate).
Migration files live in `database/migrations/`.

The database connection URLs are constructed from the environment variables
defined in `mise.toml`.

```sh
# Run all pending migrations on the dev database
mise run migrate

# Migrate to a specific version
mise run migrate -- --version 3

# Run migrations on both dev and test databases
mise run migrate:all

# Create a new migration
mise run migration add_camper_table
```

Note: the standalone `migrate` tasks assume Postgres is already running. When
using `mise run dev`, migrations are run automatically inside the Docker
container before the server starts.

See [docs/development.md](../docs/development.md) for full dev workflow
documentation.
