# camp-scheduler

A web application for automating summer camp scheduling: assigning counselors to
cabins, campers to cabins, and counselors to activities. The system uses a
constraint satisfaction solver to produce ranked, fully-explainable assignment
solutions that respect hard constraints and optimize soft preferences.

## Tech Stack

| Layer      | Technology                                  |
| ---------- | ------------------------------------------- |
| Language   | Go 1.24+                                    |
| HTTP       | Gin                                         |
| Database   | PostgreSQL                                  |
| Query Layer| sqlc (type-safe SQL code generation)        |
| Migrations | golang-migrate                              |
| Solver     | Custom Go constraint satisfaction engine    |
| Future     | LLM conversational layer (MCP) for Q&A over assignments |

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Web UI    │────>│  REST API   │────>│ PostgreSQL  │
│  (future)   │     │  (Go/Gin)   │     │             │
└─────────────┘     └──────┬──────┘     └──────┬──────┘
                           │                    │
                    ┌──────▼──────┐      ┌──────▼──────┐
                    │   Solver    │<─────│    sqlc     │
                    │   (CSP/Go)  │      │  (queries)  │
                    └──────┬──────┘      └─────────────┘
                           │
                    ┌──────▼──────┐
                    │  Solution   │
                    │  Ranker +   │
                    │  Explainer  │
                    └─────────────┘
```

**Request flow:** API controller -> Service -> sqlc queries -> PostgreSQL

**Solver flow:** Solver reads a snapshot of current state via sqlc, runs
constraint satisfaction, produces ranked solutions with explanations, stores
the selected solution back.

## Domain Model

### Core Entities (existing)

- **Camp** -- top-level org; all entities scoped to a camp
- **Age Group** (aka "village") -- grouping of campers by age range
- **Cabin** -- physical cabin, belongs to an age group
- **Season** -- e.g., "Summer 2026"
- **Session** -- a time period within a season (linked list via `previous_session`)
- **Counselor** -- staff member, may be junior

### New Entities (to add)

- **Camper** -- individual camper with age, gender, preferences
- **Activity** -- e.g., "Archery", "Swimming", with capacity and certification requirements
- **Time Slot** -- scheduling block within a session for activities
- **Counselor Preferences** -- per-session ranked age group preferences, co-counselor preferences
- **Camper Preferences** -- friend requests, activity preferences
- **Certification** -- what a counselor is qualified to teach
- **Assignment Results** -- stored solver output with scores and explanations

### Schema Changes (Phase 1)

New tables:

- `counselor_age_group_preferences` -- per-session ranked age group preferences for a counselor
- `counselor_cocounselor_preferences` -- per-session ranked co-counselor preferences (directional)
- `counselor_session_history` -- tracks which cabin/age group a counselor was in previously
- `counselor_cabin_assignments` -- solver output: counselor -> cabin for a session
- `assignment_runs` -- metadata about each solver run (timestamp, score, status)
- `assignment_explanations` -- per-assignment reasoning trail

### Schema Changes (Phase 2+)

- `campers` -- individual camper records (name, age, gender, etc.)
- `camper_preferences` -- friend requests, cabin requests
- `camper_cabin_assignments` -- solver output: camper -> cabin
- `activities` -- activity definitions with capacity, certifications required
- `time_slots` -- scheduling blocks within sessions
- `counselor_certifications` -- join table: counselor has certification
- `activity_schedule` -- solver output: activity + time slot + counselor(s)

## Constraints

### Hard Constraints (must be satisfied)

- Cabin capacity cannot be exceeded
- Junior counselors cannot be the sole counselor in a cabin
- Counselors must have required certification for an activity (Phase 2+)
- A counselor cannot be double-booked in the same time slot (Phase 2+)
- Activity capacity cannot be exceeded (Phase 2+)

### Soft Constraints (optimized, weighted)

- Returning counselor prefers same village (weight: high)
- Returning counselor prefers same cabin (weight: medium)
- Counselor co-counselor preference (weight: medium)
- Counselor village/age-group preference (weight: medium)
- Camper friend requests -- be in same cabin (Phase 2+, weight: high)
- Camper activity preferences (Phase 2+, weight: medium)

## Solver Design

The solver is a constraint satisfaction + optimization engine:

1. **Input:** Snapshot of all entities, preferences, and constraints for a given session
2. **Search:** Backtracking search with heuristic ordering (most-constrained-first)
3. **Scoring:** Each candidate solution scored by weighted sum of satisfied soft constraints
4. **Output:** Top N solutions ranked by score, each with:
   - Full assignment map (counselor -> cabin)
   - Total score + breakdown by constraint category
   - Per-assignment explanation (e.g., "Counselor X assigned to Cabin Y because:
     returning to same village (+10), co-counselor preference satisfied (+5)")
   - List of unsatisfied soft constraints with reasons (if any)

The system always presents multiple ranked alternatives so the director can
compare viable configurations. This is especially valuable when several
solutions satisfy all hard constraints and score equally well on soft
constraints -- the director can see what trade-offs exist between options.

## Development Phases

### Phase 0: Foundation Refresh

- [x] Upgrade to Go 1.24+
- [x] Replace Bun ORM with sqlc
- [x] Switch HTTP framework to Gin
- [x] Replace `pkg/errors` with stdlib errors (Go 1.13+ wrapping)
- [x] Replace logrus with `log/slog`
- [x] Update project structure (`cmd/server/`, `internal/`, `database/queries/`)
- [x] Re-implement Camp CRUD with new stack
- [x] Set up basic test infrastructure

### Phase 1: Counselor-to-Cabin Solver (MVP)

- [x] Implement CRUD for all existing entities (age groups, cabins, seasons, sessions, counselors)
- [x] Determine delete behavior for entities with FK dependencies (block, cascade, reassign, etc.)
- [x] Add camp_id scoping to Get/Update/Delete queries for camp-owned entities
  - Ensures entities can only be accessed within their owning camp
  - Will align with JWT-based camp scoping when auth is implemented
- [x] Add counselor preference tables + CRUD
- [x] Add counselor session history tracking
- [ ] Implement the constraint solver engine
  - [ ] Hard constraint validation
  - [ ] Soft constraint scoring with configurable weights
  - [ ] Backtracking search with heuristics
  - [ ] Solution ranking
- [ ] Implement explanation/audit trail generation
- [ ] Add assignment run management (trigger, view results, select solution)
- [ ] API endpoints for solver: trigger run, get results, select/apply solution
- [ ] Integration tests with realistic camp data

### Phase 2: Camper-to-Cabin Assignment

- [ ] Add camper table + CRUD
- [ ] Add camper preferences (friend requests, etc.)
- [ ] Extend solver to handle camper assignments
- [ ] Additional hard constraints (cabin capacity with campers)
- [ ] Additional soft constraints (friend requests)

### Phase 3: Activity Scheduling

- [ ] Add activity, time slot, certification tables + CRUD
- [ ] Add counselor certifications
- [ ] Extend solver for counselor-to-activity-to-timeslot assignments
- [ ] Time conflict detection
- [ ] Activity capacity constraints

### Phase 4: Polish & Integration

- [ ] JWT authentication
  - [ ] Remove camp_id from REST endpoint paths; derive camp context from JWT
- [ ] Web frontend
- [ ] Data import (CSV/spreadsheet)
- [ ] External system integration (Campminder, etc.)
- [ ] LLM conversational layer (MCP) for "why was X assigned to Y?" queries

## API Testing

API definitions are maintained in the `insomnia/` directory as an Insomnia
workspace export. These definitions should be kept up to date as endpoints are
added or changed.
