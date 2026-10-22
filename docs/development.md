# Development Guide

## Requirements

- Go version specified in `src/go.mod`
- Git
- PostgreSQL

## Setup

Run Makefile commands from the repository root. Go source and the module are under `src/`.

Copy the example environment file:

```fish
cp src/.env.example src/.env
```

Download dependencies:

```fish
cd src
go mod download
cd ..
```

Create a PostgreSQL role and database for local development (use a local-only password and do not reuse it elsewhere):

```sql
CREATE ROLE serve_service LOGIN PASSWORD 'serve_service_dev';
CREATE DATABASE serve_service OWNER serve_service;
```

The `.env.example` values match this local setup. Copy it to `src/.env` as above and change credentials as appropriate. Set optional `GITHUB_TOKEN` when resolving private repository default branches or to raise GitHub API rate limits; it needs repository metadata and contents read permission. The server and migration command load `.env` only in the development environment; explicitly supplied process environment variables take precedence.

Apply schema migrations from the repository root:

```fish
make migrate-up
```

The server creates one shared GORM database handle for application and deployment requests. Apply migrations before starting it.

## Run the server

```fish
make run
```

The server listens on port 8000 by default.

Override configuration using environment variables:

```fish
env PORT=9000 make run
```

Existing process environment variables take precedence over values
loaded from `.env`.

## Run the frontend

The frontend is a React 19, TypeScript, and Vite application in `web/`. It uses Tailwind CSS v4 and source components from the Space UI registry.

Install frontend dependencies and configure the optional development proxy:

```fish
cd web
npm install
cp .env.example .env
```

Start the Go API in one terminal (`make run` from the repository root). In another terminal, start Vite:

```fish
cd web
npm run dev
```

Open the local Vite URL shown in the terminal (normally `http://localhost:5173`). Vite proxies `/api` and `/health` requests to `BACKEND_URL`, which defaults to `http://localhost:8000`. Set `BACKEND_URL` in `web/.env` if the Go server uses another origin. `VITE_API_BASE_URL` defaults to `/api/v1` and should include the API prefix when overridden.

The development proxy keeps browser requests same-origin; the Go API does not configure CORS. The API has no authentication. Applications can be listed and retrieved by ID; deployments can be created, retrieved, and listed for a known application. There is no global deployment list. Manual and webhook deployment requests are persisted as queued work. A background worker checks out the recorded commit and builds its root-level `Dockerfile` with the local Docker Engine; a successful build is exposed as `built` with an image reference and local image ID. Runtime deployment is not implemented. See [the build engine guide](./build-engine.md) for prerequisites, operations, security assumptions, and troubleshooting.

Space UI components are installed as source with the official registry CLI. After the preset is initialized, components can be added with:

```fish
npx shadcn@latest add @spaceui/primitives-button @spaceui/primitives-card @spaceui/primitives-badge @spaceui/primitives-table @spaceui/primitives-input
```

Run the frontend production build and API-client unit tests:

```fish
npm run build
npm test
```

## Verify the application

Check the health endpoint:

```fish
curl -i http://localhost:8000/health
```

Expected response body:

```json
{"status":"healthy"}
```

## Core API

API routes use the `/api/v1` prefix:

| Method | Path | Success |
|---|---|---|
| `POST` | `/api/v1/applications` | `201 Created` |
| `GET` | `/api/v1/applications?limit=20&offset=0` | `200 OK` |
| `GET` | `/api/v1/applications/{id}` | `200 OK` |
| `POST` | `/api/v1/applications/{id}/deployments` | `201 Created` |
| `GET` | `/api/v1/deployments/{id}` | `200 OK` |
| `GET` | `/api/v1/deployments/{id}/logs` | `200 OK` |
| `GET` | `/api/v1/applications/{id}/deployments?limit=20&offset=0` | `200 OK` |

Create an application:

```fish
curl -i -X POST http://localhost:8000/api/v1/applications \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo","repository_url":"https://github.com/acme/demo.git","branch":"main"}'
```

To receive GitHub push events, include the repository's numeric GitHub ID in the create request and configure `GITHUB_WEBHOOK_SECRET` for the server. The UI never asks for the webhook secret.

Queue the latest commit from the repository's GitHub default branch:

```fish
curl -i -X POST http://localhost:8000/api/v1/applications/1/deployments \
  -H 'Content-Type: application/json' \
  -d '{}'
```

The API resolves the default branch from GitHub repository metadata and persists its current HEAD SHA in the response before queueing. To target a specific full SHA instead:

```fish
curl -i -X POST http://localhost:8000/api/v1/applications/1/deployments \
  -H 'Content-Type: application/json' \
  -d '{"commit_sha":"abcdef0123456789abcdef0123456789abcdef01"}'
```

Deployment requests are persisted with status `queued`. The background worker changes the status to `building`, then `built` or `failed`; the UI polls deployment details while queued or building. History sorts by creation time descending, with ID descending as the tie-breaker. Application and deployment list `limit` defaults to 20 and must be 1–100; `offset` defaults to 0. Logs are available from `/deployments/{id}/logs`, while remaining excluded from the deployment JSON response. The API does not currently authenticate callers, so run it only in a trusted development/operator environment.

Application repository URLs must use HTTPS or SSH and must not embed credentials (except the conventional `git` SSH username). Branch names must be valid Git refs. A supplied commit ID must contain 40 or 64 hexadecimal characters. When omitted, only the GitHub repository's API-reported default branch is used; failure to retrieve repository metadata or its latest commit returns `502` and queues no deployment. An invalid explicit SHA never falls back to the default branch. Webhook requests continue to use their push event SHA.

The deployment transition graph is `queued -> building|cancelled`, `building -> built|deploying|failed|cancelled`, `built -> deploying|failed|cancelled`, and `deploying -> healthy|failed|cancelled`. `healthy`, `failed`, and `cancelled` are terminal. A transition to `failed` requires a diagnostic message. No API endpoint accepts arbitrary status changes.

The frontend repository form and deployment history/details are documented in [`web/README.md`](../web/README.md). The deployment form accepts an optional full SHA; build prerequisites and limitations are in the [Build Engine guide](./build-engine.md).

Errors use `{"error":{"code":"...","message":"..."}}`. Invalid input returns `400`, missing applications/deployments `404`, duplicate names `409`, GitHub default-commit resolution failures `502`, and unexpected failures return a generic `500` without internal database details.

## Quality checks

Run all checks:

```fish
make check
```

Run individual checks:

```fish
make fmt-check
make test
make vet
```

To format Go files:

```fish
make fmt
```

PostgreSQL integration checks are opt-in so ordinary unit tests do not mutate a developer database. They apply pending migrations and create/delete their own records. The queue-claiming integration test uses the real PostgreSQL transaction and locking behavior and requires the database to have no pre-existing `queued` or `building` deployments; it skips when that queue is not idle. With a configured local database, run:

```fish
set -lx RUN_DATABASE_INTEGRATION_TESTS 1
cd src
go test ./internal/platform/database -run '^TestPostgresBuildQueueClaiming$' -count=1 -v
```

Run `go test ./...` from `src` for the full suite, including the existing persistence integration check when enabled.

The integration check applies pending migrations, verifies application/deployment persistence after reopening the database handle, and cleans up its own records.

## Troubleshooting

### Invalid PORT

PORT must be an integer between 1 and 65535.

### Port already in use

Set PORT to another available port.

### Tests fail

Run `go test ./...` from the `src` directory and inspect the
reported package and test name.
