# Auth Server

TypeScript service powering authentication and organization management for
Camp Scheduler. Built on [BetterAuth](https://better-auth.com) with the
`organization`, `admin`, `jwt`, and `username` plugins.

## Architecture

- **Camps as organizations.** Every camp is a BetterAuth organization with
  `organization.id == camps.id`. The Go server owns the `camps` table; this
  service owns the `organization`/`member`/`invitation` tables. The Go camp
  CRUD handlers call this service over `/internal/organizations` to keep the
  two in sync (shared-secret auth via `AUTH_SHARED_SECRET`).
- **JWT bridge to Go.** The frontend obtains a short-lived (15 min) JWT via
  `/api/auth/token` and sends it to the Go API as `Authorization: Bearer ...`.
  Go validates against this server's JWKS endpoint
  (`/api/auth/jwks`). Sessions are long-lived (30 days, rolling).
- **Roles.** `super_admin` is a global role on the user (admin plugin).
  Within an organization, members have a per-org role (`owner`/`admin`/
  `member`); the active org's role is exposed as the `org_role` JWT claim.

## Endpoints

- `/api/auth/*` — BetterAuth (sign-in, sign-up, session, organization, admin,
  jwt). See [BetterAuth docs](https://better-auth.com/docs).
- `/internal/organizations` — service-to-service org create/delete (Go only).
- `/internal/members` — service-to-service member add/remove (Go only).
- `/internal/users` — service-to-service user create/delete (Go only).
- `/internal/users/:id/revoked-after` — returns the user's revocation
  timestamp (Go only).
- `/health`, `/ready` — liveness / readiness.

## Token revocation

The user table has a `revokedAfter` timestamp column (defaults to row
creation). The Go server caches this per user (~30s TTL) and rejects any JWT
whose `iat` is at or before the value. Bumping the column to `now()`
invalidates all outstanding JWTs for that user within one cache TTL.

The `/internal/members DELETE` handler bumps `revokedAfter` whenever a user
is removed from a camp so the user can't keep acting on the removed camp via
a JWT minted before the removal. Future "kick session" / "log out
everywhere" features can use the same hook.

The Go server fails open on transient unavailability of the revocation
endpoint: a stale cache entry continues to be honored until the auth-server
becomes reachable again. This trades a small additional revocation window
for Go-side availability during auth-server restarts.

## Local development

```sh
cd auth
pnpm install
pnpm dev        # http://localhost:9101
```

Schema changes are managed by the unified golang-migrate stream
(`mise run migrate:auth`). When you change `auth.ts` plugins or fields,
regenerate the migration:

```sh
mise run auth:migration <name>   # writes database/migrations/auth/NNNNNN_<name>.up.sql
mise run migrate:auth            # apply
```

The generator (`scripts/generate-migration.ts`) connects to a throwaway
Postgres database with the privileged role, asks BetterAuth's Kysely adapter
to compile the full schema, and writes the resulting SQL wrapped in a
transaction with `SET LOCAL search_path = auth, public;` so it always lands
in the `auth` schema.

Environment variables (provided by `mise.toml` at the repo root):

| Var | Purpose |
| --- | --- |
| `AUTH_PORT` | listen port (default 9101) |
| `AUTH_BASE_URL` | external URL of this service |
| `AUTH_TRUSTED_ORIGINS` | comma-separated CORS allowlist |
| `BETTER_AUTH_SECRET` | BetterAuth signing/encryption secret (32+ chars) |
| `AUTH_SHARED_SECRET` | bearer secret accepted on `/internal/*` |
| `AUTH_DATABASE_URL` | runtime Postgres URL (uses `auth_user` role, search_path = auth, public) |
| `DATABASE_URL` | privileged Postgres URL (used by the migration generator only) |

## JWT claim shape

```jsonc
{
  "sub": "<user uuid>",
  "username": "demo",
  "email": "demo@example.com",
  "role": "user",                  // global role: "user" | "super_admin"
  "camp_id": "<organization uuid>", // active organization, "" for super_admin without active org
  "org_role": "admin",             // role in the active organization
  "impersonated_by": "<admin user id>", // present when impersonating
  "first_name": "Demo",
  "last_name": "User",
  "iat": ..., "exp": ..., "iss": "...", "aud": "..."
}
```
