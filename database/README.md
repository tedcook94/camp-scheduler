# camp-scheduler Database

## Local Setup

To set up a local database, run the setup script from the project root:

```sh
cd database/local-setup && ./local-setup.sh
```

This creates both the `camp_scheduler` development database and the
`camp_scheduler_test` database used by integration tests.

To tear it down:

```sh
cd database/local-setup && ./local-drop.sh
```

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate).
Migration files live in `database/migrations/`.

The database connection URL is built automatically from the `DATABASE_*`
variables in `.env` (host, port, user, password, name, sslmode).

From the project root:

```sh
# Run migrations to a specific version
make migrate v=3

# Run migrations on both dev and test databases
make migrate-all v=3

# Create a new migration
make migration name=add_camper_table
```
