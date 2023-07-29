# camp-scheduler Database

## Development

- First, run `make deps` to install golang-migrate.
- Create a new migration with `make migration name=migration_name`.
- Run a migration with `make migrate`. Ensure your `.env` file is configured with the proper URL and target version.

## Local Setup

To set up a local database, `cd` into the `local-setup` directory and run the `local-setup.sh` script. Then you can `cd` back to this directory and follow the migration steps above to get the latest schema.

You can also remove the database and roles with `local-drop.sh`.
