# Camp Scheduler

A Go web application for automating summer camp scheduling: assigning counselors
to cabins, campers to cabins, and counselors to activities. The system uses a
constraint satisfaction solver to produce ranked, fully-explainable assignment
solutions that respect hard constraints and optimize soft preferences.

## Tech Stack

| Layer       | Technology                               |
| ----------- | ---------------------------------------- |
| Language    | Go 1.26                                  |
| HTTP        | Gin                                      |
| Database    | PostgreSQL 18                            |
| Query Layer | sqlc (type-safe SQL code generation)     |
| Migrations  | golang-migrate                           |
| Auth        | BetterAuth (TypeScript service, Hono)    |
| Frontend    | SvelteKit, Svelte 5, Tailwind CSS v4     |
| Solver      | Custom Go constraint satisfaction engine |
| Dev Tooling | mise, air (hot-reload), Docker Compose   |

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│   Web UI    │───▶│  Go REST API │────▶│ PostgreSQL  │
│ (SvelteKit) │     │  (Gin :9100) │     │             │
└──────┬──────┘     └──────┬───────┘     └──────┬──────┘
       │                   │                    ▲
       │            JWT validation              │
       │            (JWKS lookup)               │
       │                   │                    │
       ▼                   ▼                    │
┌──────────────────────────────┐                │
│  BetterAuth Auth Server      │────────────────┘
│  (TypeScript / Hono :9101)   │ shares Postgres (own tables)
│  • sign-in, sessions         │
│  • JWT minting (Ed25519)     │
│  • organizations = camps     │
│  • admin user CRUD           │
└──────────────────────────────┘
```

**Request flow:** Browser ↔ BetterAuth (cookie session + short-lived JWT) →
Go API validates JWT via JWKS → Service → sqlc queries → PostgreSQL.

**Solver flow:** Reads a snapshot of current state via sqlc, runs constraint
satisfaction search, produces ranked solutions with per-assignment explanations,
and stores the selected solution back.

## Domain Model

- **Camp** — top-level org; all entities are scoped to a camp
- **Age Group** (village) — grouping of campers by age range
- **Cabin** — physical cabin, belongs to an age group
- **Season / Session** — time hierarchy (e.g., "Summer 2026" → week-long sessions)
- **Counselor** — staff member (may be junior); has preferences and certifications
- **Camper** — individual camper with age, gender, and friend preferences
- **Activity** — e.g., "Archery", with capacity and certification requirements
- **Time Slot** — scheduling block within a session for activities
- **Assignment Results** — stored solver output with scores and explanations

## Solver

The solver is a constraint satisfaction + optimization engine that handles two
assignment types:

1. **Cabin Assignment** — assign counselors and campers to cabins simultaneously,
   respecting cabin capacity (counselors + campers), seniority, gender, and age
   group constraints while optimizing for counselor and camper preferences
2. **Activity Scheduling** — assign counselors to activities in time slots
   respecting certifications and avoiding conflicts

Each solver run produces top-N ranked solutions with full score breakdowns and
per-assignment explanations, allowing camp directors to compare viable
configurations and understand trade-offs.

## Project Structure

```
cmd/server/          → Application entrypoint
internal/            → All application code
  server/            → HTTP server setup and routing
  config/            → Configuration (env-based via envconfig)
  auth/              → JWT/JWKS validation + middleware
  solver/            → Constraint satisfaction solver (3 solver types)
  db/                → sqlc-generated database code
  api/               → Shared API helpers
  assignment/        → Assignment run orchestration
  <domain>/          → Domain packages (camp, cabin, counselor, camper, activity, timeslot, sessionconfig, etc.)
auth/                → BetterAuth auth-server (TypeScript / Hono)
  src/               → BetterAuth config, JWT plugin, internal admin API
  scripts/           → Migrate + seed-super-admin scripts
web/                 → Frontend (SvelteKit SPA)
  src/lib/api/       → API client layer (typed fetch wrapper)
  src/lib/components/→ UI components (shadcn-svelte)
  src/routes/        → SvelteKit routes (login, admin, app dashboard + domain CRUD)
database/
  migrations/        → SQL migration files (golang-migrate)
  queries/           → SQL query files (sqlc)
  local-setup/       → Local Postgres initialization scripts
yaak/                → API workspace sync directory (Yaak)
docs/                → Development and roadmap documentation
```

## Documentation

| Document                              | Contents                             |
| ------------------------------------- | ------------------------------------ |
| [docs/development.md](docs/development.md) | Local setup, dev workflow, tooling  |
| [docs/roadmap.md](docs/roadmap.md)    | Product roadmap and development phases |
| [database/README.md](database/README.md) | Database setup, migrations, reset  |
| [yaak/README.md](yaak/README.md)         | API testing with Yaak              |
| [AGENTS.md](AGENTS.md)               | Coding conventions for AI agents     |

## Quick Start

```sh
mise install       # Install pinned tool versions
mise run dev       # Start Postgres + Go server with hot-reload
```

See [docs/development.md](docs/development.md) for full setup instructions.
