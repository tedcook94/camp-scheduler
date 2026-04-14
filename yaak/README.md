# camp-scheduler Yaak

API request definitions for the camp-scheduler project, maintained via
[Yaak](https://yaak.app) workspace data sync.

## Setup

1. Open Yaak and go to **Settings > General > Data Sync Directory**.
2. Point the sync directory to this `yaak/` folder.
3. The workspace and all requests will load automatically.

## Environments

The **Base Environment** (committed to git) contains:

- `base_url` — the API base URL (default: `http://10.0.4.14:9100/api/v1`)
- `access-token` — auto-populated via request chaining from the Login response
- 18 test ID variables for referencing created resources in requests

To override `base_url` or any other variable for your local setup, create a
private (non-sharable) sub-environment in Yaak. Private environments are not
synced to disk and won't appear in git.

## Authentication

Bearer token auth is configured at the workspace level. All requests inherit it
automatically. The `access-token` variable uses Yaak's request chaining to pull
`$.access_token` from the Login request's most recent response.

To authenticate:

1. Send the **Login** request (in the `auth` folder).
2. All subsequent requests automatically use the returned token.

## Keeping Definitions in Sync

Changes made in the Yaak UI are automatically written to this directory. Commit
the updated files as part of any PR that adds or changes API endpoints.
