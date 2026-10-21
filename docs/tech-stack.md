# Technical Stack

## Status

**Status:** Go, Fiber v2, PostgreSQL/pgx, and Goose are implemented for M02. Other proposed integrations remain subject to review.

This document describes the selected stack for implemented M02 behavior and proposed choices for later milestones. Significant new decisions should be reviewed and recorded in Architecture Decision Records (ADRs).

Dependency and tool versions must be pinned in the relevant manifests, lockfiles, and CI configuration.

## Design Priorities

1. Keep the system maintainable for a small engineering team.
2. Provide reproducible development and testing workflows.
3. Separate API handling from deployment execution and infrastructure operations.
4. Make deployment failures observable and diagnosable.
5. Minimize operational dependencies for a single Linux host.
6. Keep the API independent of any particular client interface.
7. Preserve a path toward stronger workload isolation and independent workers.

## Proposed Stack

| Area                        | Proposal                                                      | Rationale                                                                                                            |
| --------------------------- | ------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| Backend language            | Go                                                            | Suitable for infrastructure services, concurrency, and compiled deployment artifacts.                                |
| HTTP framework              | Fiber                                                         | Provides routing, middleware, and an ergonomic API for HTTP services.                                                |
| API format                  | JSON over HTTP                                                | Straightforward integration with CLI tools and future clients.                                                       |
| API specification           | OpenAPI                                                       | Documents request/response contracts and supports contract validation.                                               |
| Database                    | PostgreSQL                                                    | Durable relational storage and transaction support for project and deployment metadata.                              |
| PostgreSQL driver           | `pgx`                                                         | Explicit PostgreSQL support for Go. Compatibility with the selected Fiber and database integration must be verified. |
| Database migrations         | Goose                                                         | Versioned SQL schema changes with a straightforward workflow.                                                        |
| Container build and runtime | Docker Engine                                                 | Supports the initial Dockerfile-based build and single-host runtime model.                                           |
| Source checkout             | Git CLI                                                       | Uses the established Git toolchain for repository operations.                                                        |
| HTTP ingress                | Caddy, pending review                                         | Provides HTTP routing without requiring the ingress proxy to access the Docker socket.                               |
| Configuration               | Environment variables with startup validation                 | Separates runtime configuration from application code.                                                               |
| Logging                     | Go `log/slog`, or a compatible structured logging integration | Structured logs with consistent fields and useful deployment context.                                                |
| Backend tests               | Go `testing` and Fiber-compatible HTTP testing                | Supports unit tests and HTTP-level verification.                                                                     |
| Go formatting               | `gofmt`                                                       | Standardized Go formatting.                                                                                          |
| Go linting                  | `golangci-lint`                                               | Consolidated static analysis and maintainability checks.                                                             |
| Local dependencies          | Docker Compose                                                | Reproducible development services.                                                                                   |
| CI                          | GitHub Actions                                                | Automated checks for proposed changes.                                                                               |

## Backend Architecture

Use a modular Go application rather than microservices.

Keep HTTP handlers, business logic, persistence, deployment orchestration, source checkout, image building, runtime management, and ingress configuration behind explicit package boundaries.

Fiber should be an HTTP delivery layer, not the place where deployment business rules live. Core services should not depend on Fiber-specific request or response types.

The API and deployment worker may initially share a codebase while running in separate process modes. Deployment requests and their states must be persisted rather than held only in memory.

## Fiber Compatibility Requirements

Fiber is based on `fasthttp`, rather than Go's standard `net/http` server implementation. Before adopting integrations, verify their compatibility with the selected Fiber version.

- Use Fiber's supported handler and middleware interfaces.
- Verify the chosen PostgreSQL integration and request-context behavior.
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

- [x] Fiber version and required integrations for the M02 API.
- [x] PostgreSQL driver and migration tool for M02 persistence.
- [ ] Docker Engine build/runtime integration.
- [ ] Ingress proxy and controlled route updates.
- [ ] Worker execution and job recovery strategy.
- [ ] Logging and configuration conventions.
- [ ] GitHub Actions quality-check workflow.
- [ ] Trusted-repository-only security assumption.

Material decisions should be recorded in `docs/adr/`. Unresolved choices must remain explicitly open rather than being decided accidentally during implementation.
