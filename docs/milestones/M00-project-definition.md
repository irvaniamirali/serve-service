# M00 — Project Definition

## Status

Partially documented. The learning goal and initial scope are clear; the working threat model and a full set of verified acceptance scenarios are still being consolidated.

## Goal

Define a deliberately small but realistic Git-to-deployment project that can be built, operated, and understood incrementally.

## Project purpose

Serve Service is a learning project for understanding Go services, Linux, networking, databases, Git, Docker, build systems, security boundaries, and operational reliability. It is not initially a commercial product.

## First user journey

1. An operator registers an application, a repository, and its deployment branch.
2. A verified GitHub push or manual API request creates a durable deployment for an immutable commit.
3. A worker checks out and builds that exact commit.
4. A later runtime stage starts the image and routes HTTP traffic to it.
5. The operator can inspect lifecycle status and useful failure details.

Steps 1–3 are represented in current code. Container runtime and HTTP routing are planned, not implemented.

## Scope and constraints

- One Linux host and PostgreSQL for the first end-to-end system.
- GitHub is the initial source provider.
- Dockerfile-based builds use the host's local Docker Engine.
- The initial trust model is operator-trusted repositories only.
- API and worker may share the Go codebase; durable state belongs in PostgreSQL.
- Each milestone delivers a testable increment and covers failures as well as success.
- Kubernetes, multi-host scheduling, commercial account features, and broad buildpack support are out of scope.

The local Docker build is not a security boundary for hostile code. See [ADR 0001](../adr/0001-single-host-build-trust-boundary.md).

## Architectural principles

- HTTP handlers translate transport requests; use cases own application rules.
- Domain packages declare persistence contracts; infrastructure implements PostgreSQL adapters.
- `bootstrap` composes concrete dependencies.
- Deployment records, not an in-memory queue, are the durable work source.
- A deployment identifies the exact commit, not a branch that can move during execution.
- Do not add a package, interface, or dependency without a concrete responsibility.

## Acceptance criteria

- Implemented, planned, and future behavior is distinguished in user-facing docs.
- Every milestone has dependency-aware, testable exit criteria.
- Security assumptions and accepted trade-offs are recorded.
- A clean end-to-end Linux acceptance scenario can be run from documented steps.

## Dependencies and follow-up

No prerequisite. M01 depends on M00. Complete the threat-model and acceptance-scenario review before declaring this milestone complete.

## Documentation

See the [architecture](../architecture.md), [roadmap](../roadmap.md), and [technology stack](../tech-stack.md). The roadmap links every milestone from [M01](./M01-development-foundation.md) through [M18](./M18-final-end-to-end-validation-and-operational-readiness.md).
