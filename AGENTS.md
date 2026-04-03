# AGENTS.md

## Project

Camp Scheduler -- a Go web application for automating summer camp scheduling
(counselor-to-cabin, camper-to-cabin, counselor-to-activity assignments) using
a constraint satisfaction solver. See `README.md` for the full project plan.

## Tech Stack

- Go 1.24+, stdlib `net/http` router, sqlc, PostgreSQL, golang-migrate

## Architecture

- Layered domain-driven: Handler -> Service -> sqlc queries -> PostgreSQL
- Constraint solver reads state via sqlc, produces ranked solutions with explanations
- Project structure: `server/` (Go app), `database/` (migrations), `sql/` (sqlc queries/schema)

## Code Style

- **Comments:** Minimal but purposeful. Document public API boundaries, complex
  algorithms, and non-obvious "why" decisions. Do not add comments that restate
  what the code does -- code should be self-documenting through naming.
- **Error handling:** Use stdlib error wrapping (`fmt.Errorf` with `%w`). No
  third-party error libraries.
- **Naming:** Follow Go conventions. Exported names get brief doc comments only
  when the name alone isn't sufficient.

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
