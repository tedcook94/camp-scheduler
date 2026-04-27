# Product Roadmap

## Current Status

Phases 0–3 are complete. The core platform supports counselor-to-cabin,
camper-to-cabin, and activity scheduling solvers with full constraint
satisfaction, scoring, and explanation generation. Phase 4 (polish and
integration) is in progress.

## Domain Model

### Entities

- **Camp** — top-level org; all entities scoped to a camp
- **Age Group** (village) — grouping of campers by age range
- **Cabin** — physical cabin, belongs to an age group
- **Season** — e.g., "Summer 2026"
- **Session** — a time period within a season (linked list via `previous_session`)
- **Counselor** — staff member, may be junior
- **Camper** — individual camper with age, gender, preferences
- **Activity** — e.g., "Archery", "Swimming", with capacity and certification requirements
- **Time Slot** — scheduling block within a session for activities
- **Certification** — what a counselor is qualified to teach
- **Preferences** — counselor age group/co-counselor/activity preferences; camper friend requests
- **Assignment Results** — stored solver output with scores and explanations

### Schema

Tables are organized by domain. See `database/migrations/` for the complete
schema (37 migrations). Key groups:

- **Core:** camps, age_groups, cabins, seasons, sessions, counselors
- **Preferences:** counselor_age_group_preferences, counselor_cocounselor_preferences, counselor_activity_preferences, camper_friend_preferences
- **History:** counselor_session_history
- **Campers:** campers, camper_session_enrollments
- **Activities:** activities, time_slots, counselor_certifications, session_time_slots, session_activities
- **Session config:** session_age_groups, session_cabins, session_counselors (per-session counselor roster)
- **Assignments:** assignment_runs, assignment_run_selected_solutions, counselor_cabin_solutions/assignments/explanations, camper_cabin_solutions/assignments/explanations, activity_solutions/assignments/explanations

## Constraints

### Hard Constraints (must be satisfied)

- Cabin capacity cannot be exceeded (counselors and campers)
- A cabin must have at least one senior (non-junior) counselor assigned
- Campers can only be assigned to cabins within their enrolled age group
- Counselors must have required certification for an activity
- A counselor cannot be double-booked in the same time slot
- Activity capacity cannot be exceeded

### Soft Constraints (optimized, weighted)

- Returning counselor prefers same village (weight: high)
- Returning counselor prefers same cabin (weight: medium)
- Counselor co-counselor preference (weight: medium)
- Counselor village/age-group preference (weight: medium)
- Prefer multiple senior counselors per cabin over one senior with many juniors (weight: low)
- Camper friend requests — be in same cabin (weight: high)
- Counselor activity preferences (weight: high)

## Solver Design

The solver is a constraint satisfaction + optimization engine:

1. **Input:** Snapshot of all entities, preferences, and constraints for a given session
2. **Search:** Backtracking search with heuristic ordering (most-constrained-first)
3. **Scoring:** Each candidate solution scored by weighted sum of satisfied soft constraints
4. **Output:** Top N solutions ranked by score, each with:
   - Full assignment map
   - Total score + breakdown by constraint category
   - Per-assignment explanation (e.g., "Counselor X assigned to Cabin Y because: returning to same village (+10), co-counselor preference satisfied (+5)")
   - List of unsatisfied soft constraints with reasons

The system presents multiple ranked alternatives so the director can compare
viable configurations and understand trade-offs.

## Development Phases

### Phase 0: Foundation Refresh ✓

- Upgrade to Go 1.24+
- Replace Bun ORM with sqlc
- Switch HTTP framework to Gin
- Replace `pkg/errors` with stdlib errors (Go 1.13+ wrapping)
- Replace logrus with `log/slog`
- Update project structure (`cmd/server/`, `internal/`, `database/queries/`)
- Re-implement Camp CRUD with new stack
- Set up basic test infrastructure

### Phase 1: Counselor-to-Cabin Solver ✓

- CRUD for all core entities (age groups, cabins, seasons, sessions, counselors)
- Delete behavior for entities with FK dependencies
- Camp-scoped access (camp_id on Get/Update/Delete queries)
- Counselor preference tables + CRUD
- Counselor session history tracking
- Constraint solver engine (hard validation, soft scoring, backtracking, ranking)
- Explanation/audit trail generation
- SessionSnapshot loading from database
- Assignment persistence + run management
- API endpoints for solver (trigger, results, select/apply)
- Integration tests with realistic camp data

### Phase 2: Camper-to-Cabin Assignment ✓

- Camper table + CRUD
- Camper preferences (friend requests)
- Camper assignment solver
- Cabin capacity constraints with campers
- Friend request soft constraints

### Phase 3: Activity Scheduling ✓

- Activity, time slot, certification tables + CRUD
- Counselor certifications
- Counselor-to-activity-to-timeslot solver
- Time conflict detection
- Activity capacity constraints

### Phase 4: Polish & Integration (in progress)

- [x] Reward repeated preferences that were previously unmet
- [x] Docker Compose for local dev
- [x] JWT authentication
  - [x] Remove camp_id from REST endpoint paths; derive camp context from JWT
- [x] Migrate to Yaak for API testing
- [x] Super-admin management
  - [x] Camp management (create, update, delete camps)
  - [x] User management (create, update, delete users)
  - [x] Frontend panel for managing camps/users
  - [x] Ability to revoke a user's tokens/log out all devices
    - [x] Revoke refresh tokens on password reset
    - [x] Revoke access tokens on password reset (token version)
  - [x] Impersonation (ability to log in as any user)
- [ ] Web frontend
  - [x] Foundation: base path migration (`/admin` → `/`), route groups (`(admin)`, `(user)`), SPA handler update
  - [x] Login: unified login with role-based routing (super_admin → /admin, all other roles → /app)
  - [x] Dashboard: landing page with navigation cards to each section
  - [x] Camp settings: view and edit camp details
  - [x] Certifications: list, create, edit, delete
  - [x] Age groups: list, create, edit, delete
  - [x] Cabins: list, create, edit, delete (with age group assignment)
  - [x] Seasons & sessions: CRUD for seasons; manage sessions within a season
  - [x] Counselors: list, create, edit, delete; manage preferences (age group, co-counselor, activity); certifications; session history
  - [x] Campers: list, create, edit, delete; friend preferences; session enrollments
  - [x] Activities & time slots: CRUD for both; certification requirements; assign time slots and activities to sessions; copy activities between time slots; drag-and-drop reorder
  - [x] Session configuration: assign age groups, cabins, activities, and time slots to a session
  - [x] Copy a session's structural configuration into a new session
  - [x] Assignment runs: trigger solver, view run history, compare ranked solutions, select a solution
- [x] Demo camp seeding tool
- [x] Add dedicated camper enrollments endpoint to eliminate N+1 loading on camper detail page
- [x] Filter counselor preference options to session-configured age groups and activities
- [x] Filter camper preference options to session-enrolled campers
- [x] Enforce gender for cabins
- [x] Assign counselors and campers to cabin at once (respect max cabin size of campers + counselors)
- [x] Assign all counselors for cabins/activities
- [x] Reports
- [x] First and last name for campers and counselors
- [x] "Dirty" flag for when changes have been made since assignment run
- [x] Archive (soft delete) objects
- [x] Overrides (locking assignments before solver runs)
- [x] Migrate auth to BetterAuth (camps as orgs)
- [ ] Data import (CSV/spreadsheet)
- [ ] Rebrand to Leiri
- [ ] Reimplement solver in Prolog?
- [ ] Reimplement frontend in HTMX?
- [ ] Auditing
- [ ] Email sending (verification, password resets, notifications)
- [ ] Camp admin user management (assign account to counselors)
- [ ] Enable MFA
- [ ] OAuth logins
- [ ] External system integration (Campminder, etc.)
- [ ] LLM conversational layer (MCP) for "why was X assigned to Y?" queries
