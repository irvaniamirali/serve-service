# M02 — Core API

## Status

Implementation is present. Go unit and HTTP tests pass. PostgreSQL integration tests are opt-in and were not enabled during the current work, so database acceptance remains environment-dependent.

## Goal

Persist application registrations and deployment lifecycle state before relying on external events or executing builds.

## Implemented behavior

The API is under `/api/v1`:

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/applications` | Create an application. |
| `GET` | `/applications` | List applications with limit/offset pagination. |
| `GET` | `/applications/{id}` | Retrieve an application. |
| `POST` | `/applications/{id}/deployments` | Queue a deployment for an explicit SHA or resolve a repository default-branch SHA. |
| `GET` | `/deployments/{id}` | Retrieve deployment status and metadata. |
| `GET` | `/deployments/{id}/logs` | Retrieve bounded build output. |
| `GET` | `/applications/{id}/deployments` | List application deployment history. |

Applications store a repository URL, configured branch, and optional GitHub repository ID. Deployments store a commit SHA, status, failure details, and—when a worker has built it—image identity and bounded build output.

Duplicate application names and duplicate registered GitHub repository IDs return `409` with error code `application_conflict`.

The lifecycle states are `queued`, `building`, `built`, `deploying`, `healthy`, `failed`, and `cancelled`. The HTTP API creates queued work but does not permit arbitrary status changes. Allowed transitions are defined by the deployment model and enforced by use cases and compare-and-set persistence operations.

The API has no authentication. The build-log endpoint is therefore suitable only for a trusted operator environment.

## Architecture and test layout

- `internal/applications` and `internal/deployments` keep related domain models, validation, use cases, HTTP handlers, and contracts together.
- `internal/infrastructure/database` owns GORM records, PostgreSQL queries, and persistence adapters.
- `internal/bootstrap` wires concrete adapters into handlers and workers.
- Use-case unit tests are split by operation (`create_application_test.go`, `get_application_by_id_test.go`, `list_applications_test.go`; the equivalent deployment files are similarly separated).
- HTTP tests stay with handlers/bootstrap; PostgreSQL persistence and queue tests stay in `database_integration_test.go`.

This keeps the domain packages independent of GORM and PostgreSQL while avoiding a directory per architectural layer.

## Remaining verification stages

### M02.1 — Unit and HTTP behavior

- **Goal/scope:** Verify API contracts, validation, state rules, and error mapping.
- **Tasks:** Keep boundary request/response names explicit; cover invalid input, not-found, conflict, dependency failure, pagination, and state transitions.
- **Decision/alternative:** Keep transport-specific structures in the current flat domain package rather than introducing a controller package solely for naming.
- **Security/failures:** Avoid exposing database diagnostics or build logs in general deployment JSON; reject invalid IDs before repository operations.
- **Tests:** Focused use-case tests and Fiber HTTP tests.
- **Acceptance:** Invalid inputs do not reach persistence; expected errors map to stable HTTP responses.
- **Dependencies:** M01.
- **Documentation:** This page and `docs/development.md`.

### M02.2 — PostgreSQL behavior

- **Goal/scope:** Validate migrations, durable persistence, ordering, and concurrency against PostgreSQL.
- **Tasks:** Run the opt-in integration suite on an isolated database; verify migration state, persisted application/deployment data, duplicate constraints, concurrent transitions, queue claiming, and failure propagation.
- **Decision/alternative:** Keep integration tests opt-in because they apply migrations and write records; do not substitute mocks for PostgreSQL locking behavior.
- **Security/failures:** Use a disposable test database and verify cleanup. The queue claim test requires no pre-existing queued/building deployments.
- **Tests:** `RUN_DATABASE_INTEGRATION_TESTS=1 go test ./internal/infrastructure/database -count=1`.
- **Acceptance:** The complete integration suite passes against the supported PostgreSQL version and leaves no test records.
- **Dependencies:** M01 and M02.1.
- **Documentation:** `docs/development.md` and this page.

## Dependencies

M01.

## Out of scope

Public authentication/authorization, building or running workloads as part of an HTTP request, and multi-host scheduling.

## Exit condition

Declare M02 complete only after M02.1 and M02.2 pass in the target environment and the API's trust limitation remains clearly documented.
