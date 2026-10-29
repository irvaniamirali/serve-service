# Technical Stack

## Status

The Go API, Fiber v2, PostgreSQL through GORM/pgx, Goose migrations, GitHub REST client, and local Git/Docker build worker are present. The worker and webhook milestones are still in progress. No container runtime manager, ingress proxy, Docker Compose file, or OpenAPI contract is implemented.

Dependency and tool versions must be pinned in the relevant manifests, lockfiles, and CI configuration.

## Design Priorities

1. Keep the system maintainable for a small engineering team.
2. Provide reproducible development and testing workflows.
3. Separate API handling from deployment execution and infrastructure operations.
4. Make deployment failures observable and diagnosable.
5. Minimize operational dependencies for a single Linux host.
6. Keep the API independent of any particular client interface.
7. Preserve a path toward stronger workload isolation and independent workers.

## Stack inventory

| Area | Current implementation | Planned or open |
|---|---|---|
| Backend | Go 1.26.5, Fiber v2, JSON over HTTP | OpenAPI contract is not selected. |
| Persistence | PostgreSQL, GORM, pgx driver, Goose SQL migrations | Local PostgreSQL provisioning is not containerized. |
| GitHub | Go `net/http` client, raw-body HMAC-SHA256 verification | Reliable transactional webhook idempotency remains M03 work. |
| Build | Git CLI and host Docker Engine; root Dockerfile only | Runtime management and ingress are not implemented. |
| Frontend | React 19, TypeScript, Vite, Tailwind CSS v4 | Production asset delivery is open. |
| Configuration/logging | Environment variables, startup validation, `log/slog` | More complete secret handling remains future work. |
| Tests/quality | Go `testing`, Fiber HTTP tests, `gofmt`, `go vet`, GitHub Actions | PostgreSQL tests are opt-in; frontend checks and a Go linter are not in CI. |
| M05 ingress | None in code | The M05 plan recommends a host-managed proxy and loopback-only reserved ports; validate before implementation. |

## Backend Architecture

Use a modular Go application rather than microservices.

Keep HTTP handlers, business logic, persistence, deployment orchestration, source checkout, image building, runtime management, and ingress configuration behind explicit package boundaries. Current PostgreSQL adapters are in `internal/infrastructure/database`; application and deployment contracts remain in their flat domain packages.

Fiber should be an HTTP delivery layer, not the place where deployment business rules live. Core services should not depend on Fiber-specific request or response types.

The API and deployment worker may initially share a codebase while running in separate process modes. Deployment requests and their states must be persisted rather than held only in memory.

## Fiber Compatibility Requirements

Fiber is based on `fasthttp`, rather than Go's standard `net/http` server implementation. Before adopting integrations, verify their compatibility with the selected Fiber version.

- Use Fiber's supported handler and middleware interfaces.
- Verify request-context behavior for the selected Fiber and GORM integration.
- Use Fiber-compatible HTTP testing utilities for handler tests.
- Check OpenAPI generation or validation tooling before committing to a particular integration.
- Avoid unnecessary adapters between `fasthttp` and `net/http`.
- Do not assume that every standard-library HTTP middleware or handler can be used directly with Fiber.

## Dependency Policy

- Prefer the standard library for concerns that do not require a framework or external package.
- Add dependencies when they reduce meaningful complexity or risk.
- Review maintenance, licensing, compatibility, and security history.
- Pin versions in manifests and CI where practical.
- Commit applicable dependency lockfiles.
- Never commit credentials, local environment files, generated binaries, or build artifacts.

Access to the Docker Engine API is highly privileged. A read-only socket mount does not make that API a safe security boundary.

## Security Constraint

Building and running arbitrary repository code on the same host as the control plane creates a significant security risk. Docker containers are not an absolute security boundary against malicious workloads.

The initial release must be restricted to repositories trusted by the operator. Unrestricted public self-service deployments remain out of scope until stronger build isolation and workload security controls have been designed and tested.

## Decisions Still to Confirm

- [x] Fiber version and PostgreSQL/GORM/Goose integration.
- [x] Local Docker Engine for the trusted-repository build MVP; see [ADR 0001](adr/0001-single-host-build-trust-boundary.md).
- [x] Durable PostgreSQL queue with one active local build at a time.
- [ ] Manual deployment branch policy versus webhook-configured branch policy.
- [ ] M05 runtime port reservation and host-managed ingress details.
- [ ] Reproducible local PostgreSQL setup.
- [ ] Authentication and authorization before any non-local deployment.

Material decisions belong in `docs/adr/`. Unresolved choices must remain explicitly open rather than being decided accidentally during implementation.
