# ADR 0001 — Local Docker builds and the initial trust boundary

- **Status:** Accepted for the learning MVP; not suitable for untrusted multi-tenant workloads.
- **Date:** 2026-10-29

## Context

M04 must demonstrate a durable deployment job that checks out a specific Git commit and produces a locally runnable image. The project starts on one Linux host and currently has a local worker that invokes Git and Docker CLI commands through the host's Docker Engine.

Dockerfile instructions are arbitrary code. The worker and control plane share a host and database, while the Docker daemon is a high-privilege system service. A container boundary alone does not provide a defensible isolation guarantee for hostile tenants.

## Decision

For the initial learning workflow, use the host's local Docker Engine and accept builds only from repositories trusted by the operator. Run one active build at a time, use a fresh temporary workspace, pin checkout to the requested full commit SHA, bound build duration and stored output, and document the lack of resource quotas and strong isolation. Do not expose this service as a public build platform.

The local build is an execution model, not a sandbox. Cancellation, cleanup, and a single-worker queue reduce operational risk but do not neutralize malicious build instructions.

## Alternatives considered

- **Per-build isolated virtual machines:** stronger separation, but requires provisioning and lifecycle machinery beyond the initial single-host learning scope.
- **Remote build service:** separates the control plane from execution but introduces an external service, credentials, network failure modes, and a new trust boundary.
- **Kubernetes build pods:** adds cluster operations and multi-host orchestration before the project has a working single-host path.
- **Local Docker daemon with arbitrary public repositories:** rejected because the security boundary and resource controls are inadequate.

## Consequences

- The implementation remains small enough to study Git checkout, durable queueing, Docker CLI behavior, and failure handling.
- Operators must control repository registration and API access.
- No hard CPU, memory, disk, or network quotas are currently enforced.
- A process crash can leave temporary workspaces or daemon-side work; queue staleness is not equivalent to proof of termination.
- Stronger isolation, quotas, authorization, and recovery become prerequisites before accepting untrusted users.

## Revisit when

The project allows untrusted repository owners, exposes registration beyond a trusted operator, supports concurrent independent tenants, or needs production availability guarantees.
