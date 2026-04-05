# AGENTS.md

## Project

Camp Scheduler -- a Go web application for automating summer camp scheduling
(counselor-to-cabin, camper-to-cabin, counselor-to-activity assignments) using
a constraint satisfaction solver. See `README.md` for the full project plan.

## Tech Stack

- Go 1.24+, Gin, sqlc, PostgreSQL, golang-migrate

## Architecture

- Layered domain-driven: Controller -> Service -> sqlc queries -> PostgreSQL
- Constraint solver reads state via sqlc, produces ranked solutions with explanations
- Project structure: `cmd/server/` (entrypoint), `internal/` (app code), `database/` (migrations, sqlc queries)

## Code Style

- **Comments:** Minimal but purposeful. Document public API boundaries, complex
  algorithms, and non-obvious "why" decisions. Do not add comments that restate
  what the code does -- code should be self-documenting through naming.
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

  // multiple fields -- context first, error last, one per line
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
- Integration tests that exercise real flows (API -> DB -> solver -> results) are
  preferred over isolated unit tests with heavy mocking.

## API Testing

- API definitions live in `insomnia/` (Insomnia workspace export)
- Keep the Insomnia definitions up to date as endpoints are added or changed

## Things to Avoid

- Excessive boilerplate comments (e.g., `// Create creates a new camp`)
- Unit tests on every function just for coverage numbers
- Heavy mocking -- prefer integration tests with a real test database
- Over-abstraction -- keep things simple until complexity is proven necessary

## Documentation

- Keep `README.md`, `database/README.md`, and other docs up to date when making
  changes that affect project structure, setup steps, or developer workflows.
- Keep `insomnia/` API definitions in sync with endpoint changes.
