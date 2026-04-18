# AGENTS.md

## Project

Camp Scheduler — a Go web application for automating summer camp scheduling
(counselor-to-cabin, camper-to-cabin, counselor-to-activity assignments) using
a constraint satisfaction solver. See `README.md` for a high-level overview and
`docs/roadmap.md` for current status and planned work.

## Tech Stack

- Go 1.26, Gin, sqlc, PostgreSQL 18, golang-migrate
- Local dev: mise (tooling + tasks), air (hot-reload), Docker Compose

## Architecture

- Layered domain-driven: Controller → Service → sqlc queries → PostgreSQL
- Constraint solver reads state via sqlc, produces ranked solutions with explanations
- Project structure: `cmd/server/` (entrypoint), `internal/` (app code), `database/` (migrations, sqlc queries)

## Code Style

- **Comments:** Minimal but purposeful. Document public API boundaries, complex
  algorithms, and non-obvious "why" decisions. Do not add comments that restate
  what the code does — code should be self-documenting through naming.
- **Error handling:** Use stdlib error wrapping (`fmt.Errorf` with `%w`). No
  third-party error libraries. Error format strings should start with "error"
  (e.g., `fmt.Errorf("error listing camps: %w", err)`), matching the logging
  convention.
- **Naming:** Follow Go conventions. Exported names get brief doc comments only
  when the name alone isn't sufficient.
- **Logging:** Use `log/slog` with the `slog.With()` chaining style for
  attaching context fields to log calls. Prefer this over inline variadic args
  for readability. When chaining multiple `With()` calls, place each on its own
  line to minimize diffs. Place `"error"` as the last `With()` in the chain.
  Error-level log messages should start with "error" (e.g.,
  `"error loading config"`).
  ```go
  // single field
  slog.With("error", err).Error("error loading config")

  // multiple fields — context first, error last, one per line
  slog.
      With("id", id).
      With("error", err).
      Error("error getting camp")
  ```

## Testing

- Tests should provide value greater than the cost of maintaining them.
- Do NOT generate unit tests for trivial CRUD operations, simple getters/setters,
  or thin wrapper functions.
- DO write tests for: solver logic, complex constraint evaluation, non-obvious
  business rules, edge cases that have caused or could cause bugs.
- Integration tests that exercise real flows (API → DB → solver → results) are
  preferred over isolated unit tests with heavy mocking.

## Git Workflow

- New features and refactors go on topic branches off `main`. Do not commit
  directly to `main`.
- Branch names: lowercase kebab-case describing the change
  (e.g., `jwt-camp-context`, `add-solver-weights`).
- Commit messages: lowercase imperative, concise, focused on *what changed* at a
  high level. No function/type/file names — describe the intent, not the
  mechanics. Abbreviations like `jwt`, `id`, `db` stay lowercase.
  ```
  # good
  add helper to extract camp id from jwt claims
  derive camp id from jwt claims in resource routes
  update integration tests for jwt-based camp context

  # bad
  Add GetCampID() helper to middleware.go
  Update agegroup/controller.go, cabin/controller.go, ...
  ```
- One logical change per commit. Group related file edits into a single commit;
  split unrelated changes into separate commits.

## Dev Workflow

- Tool versions are pinned in `mise.toml`. Run `mise install` to set up.
- `mise run dev` starts everything (Postgres + migrations + server with hot-reload).
- See `docs/development.md` for full setup and all available tasks.

## API Testing

- API definitions live in `yaak/` (Yaak workspace sync directory)
- Keep the Yaak definitions up to date as endpoints are added or changed

## Things to Avoid

- Excessive boilerplate comments (e.g., `// Create creates a new camp`)
- Unit tests on every function just for coverage numbers
- Heavy mocking — prefer integration tests with a real test database
- Over-abstraction — keep things simple until complexity is proven necessary

## Documentation

- Keep `README.md`, `docs/`, `database/README.md`, and other docs up to date
  when making changes that affect project structure, setup steps, or developer
  workflows.
- Keep `yaak/` API definitions in sync with endpoint changes.
- Keep `docs/roadmap.md` up to date when completing or adding development phases.
- When making changes, review all project documentation for necessary updates as
  part of each logical change — don't defer doc updates to the end.
