# M02 — Core API

## Status

Implemented. 

## Goal

Implement the persistent backend domain for applications and deployments before connecting external Git events or executing builds.

## Initial Domain Concepts

### Application

Represents a deployable application and its source configuration, including repository URL, configured branch, and application-specific settings.

### Deployment

Represents one attempt to deploy a specific source revision. It should retain a stable identifier, source commit when known, lifecycle status, timestamps, and diagnostic metadata.

## Planned Features

- [x] Create and retrieve application records.
- [x] Validate HTTPS/SSH repository URLs and Git branch configuration.
- [x] Create deployment requests through the API without executing them.
- [x] Retrieve a deployment and its current status.
- [x] List deployment history for an application with deterministic pagination.
- [x] Persist records in PostgreSQL.
- [x] Add a forward-only schema migration for the configured branch, commit SHA validation, and history index.
- [x] Define allowed deployment state transitions and atomically persist them.
- [x] Use consistent, non-sensitive API error responses.
- [x] Add focused unit, HTTP, and opt-in PostgreSQL integration tests.
- [x] Document API endpoints, request examples, and state transitions.

## Initial Deployment States

The lifecycle contains `queued`, `building`, `deploying`, `healthy`, `failed`, and `cancelled`.

Allowed transitions:

```text
queued    -> building | cancelled
building  -> deploying | failed | cancelled
deploying -> healthy | failed | cancelled
```

`healthy`, `failed`, and `cancelled` are terminal. Failure transitions require a concise diagnostic message. The API creates deployments in `queued`; it does not expose a status-update endpoint. Internal worker code can use the deployment use case's validated, compare-and-set transition operation.

## API

All API paths are prefixed with `/api/v1`.

| Method | Path | Behavior |
|---|---|---|
| `POST` | `/applications` | Create an application; returns `201`. |
| `GET` | `/applications/{id}` | Retrieve an application; returns `200`. |
| `POST` | `/applications/{id}/deployments` | Queue a deployment request; returns `201`. The JSON body is optional. |
| `GET` | `/deployments/{id}` | Retrieve deployment status and metadata; returns `200`. |
| `GET` | `/applications/{id}/deployments?limit=20&offset=0` | List deployment history, newest first; returns `200`. |

Application creation body:

```json
{
  "name": "demo",
  "repository_url": "https://github.com/acme/demo.git",
  "branch": "main"
}
```

Deployment creation can include a commit SHA when known:

```json
{
  "commit_sha": "abcdef0123456789abcdef0123456789abcdef01"
}
```

Repository URLs must be valid HTTPS or SSH URLs without embedded credentials (SSH may use the `git` username); branches follow Git ref-name restrictions. Commit object IDs may be 40 or 64 hexadecimal characters.

Errors have one stable shape:

```json
{
  "error": {
    "code": "application_not_found",
    "message": "Application not found"
  }
}
```

Invalid bodies/fields return `400`, missing records return `404`, duplicate application names return `409`, and unexpected errors return a generic `500` without database details. History `limit` is 1–100 (default 20), and `offset` is non-negative (default 0).

## Acceptance Criteria

- Application and deployment records can be created and retrieved.
- Records persist across backend restarts.
- Deployment history is queryable in deterministic order.
- Invalid requests return predictable errors.
- Illegal state transitions are rejected.
- Concurrent requests do not silently corrupt persistent state.
- Tests cover successful requests and important failure cases.
- API behavior is documented.

The PostgreSQL integration check is opt-in: set `RUN_DATABASE_INTEGRATION_TESTS=1` and configure the database environment variables before running `make test`. It applies pending Goose migrations, writes test records, reopens the database handle to verify persistence, and removes its records.

## Dependencies

M01.

## Out of Scope

- Actual GitHub webhook processing.
- Container image building.
- Starting application containers.
- A full user authentication and authorization system.

## Exit Condition

M02 is complete when the API can reliably represent applications and deployment attempts, ready for integration with GitHub events.
