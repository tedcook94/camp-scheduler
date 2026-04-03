# camp-scheduler Database

## Local Setup

To set up a local database, run the setup script from the `local-setup`
directory:

```sh
cd database/local-setup && ./local-setup.sh
```

To tear it down:

```sh
cd database/local-setup && ./local-drop.sh
```

## Migrations

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate).
Migration files live in `database/migrations/`.

From the project root (requires `DATABASE_URL` and `VERSION` env vars):

```sh
# Run migrations to a specific version
make migrate

# Create a new migration
make migration name=add_camper_table
```

The `DATABASE_URL` should be a Postgres connection string, e.g.:

```
postgres://camp_scheduler:p@ss123@localhost:5432/camp_scheduler?sslmode=disable
```
