# Development Guide

## Requirements

- Go version specified in `src/go.mod`
- Git
- PostgreSQL
- golangci-lint v2.1.6 for lint checks

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

The `.env.example` values match this local setup. Copy it to `src/.env` as above and change credentials as appropriate. The server and migration command load `.env` only in the development environment; explicitly supplied process environment variables take precedence.

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

The development proxy keeps browser requests same-origin; the Go API does not configure CORS. The API has no authentication. Applications can be created and retrieved by ID, but there is no application collection endpoint; deployment requests can be created/retrieved and history is available only for a known application ID. There is no global deployment list. The UI makes those limitations explicit and does not display simulated records. A successful manual request means it was queued in the API; builds, webhook processing, and runtime deployment are not implemented.

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
| `GET` | `/api/v1/applications/{id}` | `200 OK` |
| `POST` | `/api/v1/applications/{id}/deployments` | `201 Created` |
| `GET` | `/api/v1/deployments/{id}` | `200 OK` |
| `GET` | `/api/v1/applications/{id}/deployments?limit=20&offset=0` | `200 OK` |

Create an application:

```fish
curl -i -X POST http://localhost:8000/api/v1/applications \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo","repository_url":"https://github.com/acme/demo.git","branch":"main"}'
```

Queue a deployment with a known source commit (the body may be omitted if it is not known):

```fish
curl -i -X POST http://localhost:8000/api/v1/applications/1/deployments \
  -H 'Content-Type: application/json' \
  -d '{"commit_sha":"abcdef0123456789abcdef0123456789abcdef01"}'
```

Deployment requests are persisted with status `queued`; this API does not clone, build, or run the application. Deployment history sorts by creation time descending, with ID descending as the tie-breaker. `limit` defaults to 20 and must be 1–100; `offset` defaults to 0.

Application repository URLs must use HTTPS or SSH and must not embed credentials (except the conventional `git` SSH username). Branch names must be valid Git refs. Commit IDs, when supplied, must contain 40 or 64 hexadecimal characters.

The deployment transition graph is `queued -> building|cancelled`, `building -> deploying|failed|cancelled`, and `deploying -> healthy|failed|cancelled`. `healthy`, `failed`, and `cancelled` are terminal. A transition to `failed` requires a diagnostic message. No API endpoint accepts arbitrary status changes.

Errors use `{"error":{"code":"...","message":"..."}}`. Invalid input returns `400`, missing applications/deployments `404`, duplicate names `409`, and unexpected failures return a generic `500` without internal database details.

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
make lint
```

To format Go files:

```fish
make fmt
```

PostgreSQL integration checks are opt-in so ordinary unit tests do not mutate a developer database. With a configured local database, run:

```fish
set -lx RUN_DATABASE_INTEGRATION_TESTS 1
make test
```

The integration check applies pending migrations, verifies application/deployment persistence after reopening the database handle, and cleans up its own records.

## Troubleshooting

### Invalid PORT

PORT must be an integer between 1 and 65535.

### Port already in use

Set PORT to another available port.

### Missing golangci-lint

Install the version specified in this guide before running `make lint`.

### Tests fail

Run `go test ./...` from the `src` directory and inspect the
reported package and test name.
