# M01 — Development Foundation

## Status

In progress. The Go and frontend foundations, configuration, logging, health endpoint, and Go CI checks exist. A reproducible local service environment and clean-checkout walkthrough have not been verified.

## Goal and scope

Make the repository straightforward to build and test locally, without hiding required services or credentials behind undocumented setup.

## Present implementation

- Go module and server/migration commands under `src/`.
- Fiber HTTP server, environment configuration, structured `slog` logging, and `/health`.
- PostgreSQL through GORM with embedded Goose SQL migrations.
- React 19, TypeScript, and Vite frontend under `frontend/`.
- Make targets for run, migrations, formatting, tests, vet, and aggregate checks.
- GitHub Actions checks Go formatting, tests, and `go vet`.

The repository does not currently contain a Docker Compose file. PostgreSQL must be installed or provisioned separately. CI does not run PostgreSQL integration tests or frontend checks, and no Go linter is configured.

## Implementation stages

### M01.1 — Reproducible backend setup

- **Goal/scope:** A clean checkout can start the API against a documented PostgreSQL instance.
- **Tasks:** Verify required Go version, environment precedence, database role/database creation, migrations, health response, and shutdown. Decide whether local PostgreSQL provisioning remains host-managed or gains a Compose file.
- **Decision/alternative:** Keep the existing host-managed setup until Compose provides a clear learning or reproducibility benefit; do not silently add an orchestration layer.
- **Security/failures:** Use local-only sample credentials; report missing configuration, unavailable PostgreSQL, occupied ports, and migration errors explicitly.
- **Tests:** Configuration unit tests, migration/persistence integration tests, health HTTP test, and a clean-checkout manual run.
- **Acceptance:** Documented commands work from a clean checkout and the API returns its documented health response.
- **Dependencies:** M00.
- **Documentation:** `README.md`, `docs/development.md`, `.env.example`.

### M01.2 — Baseline quality checks

- **Goal/scope:** Local and CI checks run the same available Go quality gates.
- **Tasks:** Keep formatting, `go test ./...`, and `go vet ./...` aligned between Makefile and CI; identify frontend checks and PostgreSQL checks as separate where not yet automated.
- **Decision/alternative:** Do not advertise a linter until one is selected and configured.
- **Security/failures:** No credentials or local environment files in CI; failures must remain visible rather than being skipped by success-shaped fallbacks.
- **Tests:** Run `make check` and compare the CI workflow steps.
- **Acceptance:** All documented local quality targets exist and a CI run executes them.
- **Dependencies:** M01.1.
- **Documentation:** `README.md`, `docs/development.md`, `.github/workflows/ci.yml` if CI scope changes.

### M01.3 — Frontend and service handoff

- **Goal/scope:** Frontend development can reach the local API using the documented proxy.
- **Tasks:** Verify npm install, Vite proxy configuration, frontend tests/build, and the API health check in separate terminals.
- **Decision/alternative:** Keep frontend and Go API as separate development processes; production asset serving is not required for this stage.
- **Security/failures:** The frontend contains no infrastructure credentials; API authentication remains an explicit limitation, so local exposure must be controlled.
- **Tests:** Frontend unit tests and production build plus backend health request.
- **Acceptance:** A clean checkout can run the frontend and API as documented.
- **Dependencies:** M01.1.
- **Documentation:** `frontend/README.md`, `docs/development.md`, `README.md`.

## Exit condition

Complete M01 after a clean-checkout setup, baseline checks, and frontend/API handoff have been run and their environment requirements are accurately documented.
