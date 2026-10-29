# Serve Service

Serve Service is a learning-driven Git-to-deployment system. The project explores how repository events, durable jobs, builds, containers, and HTTP routing fit together on a single Linux host. The aim is to learn practical Go, DevOps, networking, build systems, security boundaries, and operational reliability—not to launch a commercial platform.

## Current state

The Go API persists applications and deployment requests. GitHub webhook handling and a local Docker build worker are present, but their acceptance work is not complete. There is no runtime manager or public application routing yet.

| Milestone | Current status |
|---|---|
| M00 — Project Definition | Scope documented; decision and threat-model details are being completed. |
| M01 — Development Foundation | Go and frontend foundations exist; a reproducible local service environment is still open. |
| M02 — Core API | API and unit/HTTP tests are present; PostgreSQL integration verification is environment-dependent. |
| M03 — GitHub Integration | Webhook and GitHub API code exist; delivery processing still needs reliability and integration verification. |
| M04 — Build Engine | Single-host Git/Docker worker exists; Linux operation and security acceptance remain open. |
| M05 — Deployment Runtime | Planned; application containers are not started or routed by Serve Service. |

See the [roadmap](docs/roadmap.md) for the complete M00–M18 sequence, dependencies, and status. Milestone documents distinguish verified behavior from plans.

## Architecture and flow

The current path is: register an application and deployment branch → accept a signed GitHub push event or a manual deployment request → persist a deployment with a commit SHA → claim it from PostgreSQL → check out that exact revision and build its root `Dockerfile` with the local Docker Engine. A successful build is recorded as `built`. Starting the resulting image and exposing it over HTTP belong to M05 and are not implemented.

The planned full lifecycle adds runtime start/readiness and routing, durable recovery, versioned configuration and rollback, authentication, resource controls, diagnostics, retention/cleanup, backup/restore, security verification, and release procedures. M06–M18 specify this work; those capabilities are not yet implemented.

The backend is a Go modular monolith using Fiber, PostgreSQL/GORM, and Goose migrations. GitHub API access uses Go's HTTP client. The frontend is React 19, TypeScript, and Vite. See the [architecture overview](docs/architecture.md), [development guide](docs/development.md), and [build engine guide](docs/build-engine.md).

## Development

Prerequisites: Go 1.26.5 (see `src/go.mod`), PostgreSQL, Git, and Node.js/npm for the frontend. A reachable Docker Engine is additionally required to run the build worker; ordinary Go unit tests use controlled fake Git/Docker commands.

From the repository root, using Fish:

```fish
cp src/.env.example src/.env
make migrate-up
make run
```

The example database credentials are for local development only. Provision the PostgreSQL role/database described in the [development guide](docs/development.md) before applying migrations. The API listens on port 8000 by default.

Backend checks:

```fish
make check
cd src
go test ./...
```

PostgreSQL integration tests are opt-in and write test records to the configured database. See the [development guide](docs/development.md) before enabling them. Frontend checks run from `frontend/` with `npm test` and `npm run build`. CI currently runs Go formatting, tests, and `go vet`; it does not run PostgreSQL integration tests or frontend checks.

## Security and limitations

The API has no authentication and must only be exposed in a trusted operator environment. Builds execute repository-controlled Dockerfile instructions through the host Docker Engine. Docker is not a strong isolation boundary, no build CPU/memory/disk limits are enforced, and only operator-trusted repositories are appropriate. The current worker records images but does not deploy them. These are deliberate constraints of the learning MVP, not production security claims.

## Documentation

- [Architecture](docs/architecture.md)
- [Development setup and API](docs/development.md)
- [Build behavior and security limitations](docs/build-engine.md)
- [Technology choices](docs/tech-stack.md)
- [Roadmap](docs/roadmap.md)
- [M00 — Project Definition](docs/milestones/M00-project-definition.md)
- [M01 — Development Foundation](docs/milestones/M01-development-foundation.md)
- [M02 — Core API](docs/milestones/M02-core-api.md)
- [M03 — GitHub Integration](docs/milestones/M03-github-integration.md)
- [M04 — Build Engine](docs/milestones/M04-build-engine.md)
- [M05 — Deployment Runtime](docs/milestones/M05-deployment-runtime.md)
- [M06 — End-to-End Workflow Integration](docs/milestones/M06-end-to-end-workflow-integration.md)
- [M07 — Durable Workers and Recovery](docs/milestones/M07-durable-workers-and-recovery.md)
- [M08 — Runtime Configuration and Secrets](docs/milestones/M08-runtime-configuration-and-secrets.md)
- [M09 — Deployment Safety and Version Management](docs/milestones/M09-deployment-safety-and-version-management.md)
- [M10 — Application Routing and Readiness](docs/milestones/M10-application-routing-and-readiness.md)
- [M11 — Authentication and Authorization](docs/milestones/M11-authentication-and-authorization.md)
- [M12 — Resource Limits and Runtime Isolation](docs/milestones/M12-resource-limits-and-runtime-isolation.md)
- [M13 — Observability and Diagnostics](docs/milestones/M13-observability-and-diagnostics.md)
- [M14 — Retention, Cleanup, and Resource Lifecycle](docs/milestones/M14-retention-cleanup-and-resource-lifecycle.md)
- [M15 — Operational Reliability and Backup/Restore](docs/milestones/M15-operational-reliability-and-backup-restore.md)
- [M16 — Security Hardening and Abuse Resistance](docs/milestones/M16-security-hardening-and-abuse-resistance.md)
- [M17 — CI/CD and Release Engineering](docs/milestones/M17-ci-cd-and-release-engineering.md)
- [M18 — Final End-to-End Validation and Operational Readiness](docs/milestones/M18-final-end-to-end-validation-and-operational-readiness.md)
- [ADR 0001 — Local build trust boundary](docs/adr/0001-single-host-build-trust-boundary.md)
