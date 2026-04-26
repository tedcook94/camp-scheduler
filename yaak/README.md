# camp-scheduler Yaak

API request definitions for the camp-scheduler project, maintained via
[Yaak](https://yaak.app) workspace data sync.

## Setup

1. Open Yaak and go to **Settings > General > Data Sync Directory**.
2. Point the sync directory to this `yaak/` folder.
3. The workspace and all requests will load automatically.

## Environments

The **Base Environment** (committed to git) contains:

- `base_url` — the Go API base URL (default: `http://localhost:9100/api/v1`)
- `auth_base_url` — the BetterAuth auth-server URL (default: `http://localhost:9101`)
- `access-token` — auto-populated via response chaining from the **Fetch JWT** request
- 21 test ID variables for referencing created resources in requests

To override any variable for your local setup, create a private (non-sharable)
sub-environment in Yaak. Private environments are not synced to disk and won't
appear in git.

## Authentication

The Go API requires Bearer JWTs minted by the BetterAuth auth-server (port 9101).
The committed `auth/` folder has two requests that handle the flow:

1. **Sign in** — POSTs to the auth-server's username sign-in endpoint. Edit the
   body to your username/password (or override via a private sub-environment).
   The auth-server sets a session cookie scoped to `localhost:9101`, which Yaak
   persists in its per-workspace cookie jar.

   Important: sign in directly against the auth-server (`auth_base_url`), not
   through the Vite proxy (`localhost:5173`). A cookie set via the proxy is
   scoped to that origin, which Yaak can't reach when calling the auth-server
   on `localhost:9101`.

2. **Fetch JWT** — GETs `/api/auth/token`. Yaak attaches the cookie from step 1
   automatically and the response body is `{"token": "..."}`. The
   `access-token` environment variable chains off this response, so subsequent
   API calls automatically use the latest JWT.

JWTs are short-lived (15m) — re-run **Fetch JWT** when calls start returning
401. Re-run **Sign in** if the session itself has expired (default 30 days).

## Keeping Definitions in Sync

Changes made in the Yaak UI are automatically written to this directory. Commit
the updated files as part of any PR that adds or changes API endpoints.
