# M05 — Deployment Runtime

## Status

Planned. The backend records built image references and local image IDs, but it does not start application containers, allocate runtime ports, or configure HTTP routes.

## Goal and scope

Run a successfully built image on one Linux host and make it reachable over HTTP without exposing the Docker API or silently replacing a working deployment with a failed one.

This milestone includes only the readiness check needed to decide whether a new container can receive traffic. Ongoing health monitoring, automated rollback controls, HTTPS automation, advanced logs, multi-host scheduling, and high availability remain future work.

## Recommended initial routing decision

Use an operator-managed reverse proxy on the host and a reserved host-port range bound only to loopback for application containers. Reserve ports transactionally in PostgreSQL to avoid conflicts. This avoids giving the proxy Docker socket access and avoids requiring the first implementation to coordinate a dynamic Docker network for every application. Validate this choice on the target Linux distribution before implementation; alternatives include a shared private Docker network with generated proxy routes, which adds network lifecycle complexity.

## Implementation stages

### M05.1 — Runtime contract and durable resource ownership

- **Goal/scope:** Define the minimum runtime inputs, application port, and mapping between a deployment and its container/host port.
- **Tasks:** Add a schema migration for runtime metadata and unique port reservation; define an explicit allowlist for environment variables; define image-ID use, container labels/names, resource ownership, and lifecycle transitions.
- **Design decision:** Start with loopback-only reserved host ports and an operator-managed reverse proxy. Do not expose arbitrary host ports or add a Docker-socket-aware proxy.
- **Security/failures:** Reject invalid ports/configuration; do not pass control-plane secrets or all host environment variables to workloads; test collisions and abandoned reservations.
- **Tests:** Unit validation and PostgreSQL uniqueness/concurrent reservation integration tests.
- **Acceptance:** Each deployment can be associated with an immutable image identity and at most one reserved loopback port; stale/failed reservations have a defined cleanup path.
- **Dependencies:** M04.4.
- **Documentation:** Runtime contract, schema, host prerequisites, and security notes in this page and `docs/architecture.md`.

### M05.2 — Restricted container start and readiness

- **Goal/scope:** Start one built image with explicit configuration and determine whether it is ready.
- **Tasks:** Create/start the container with a narrow port binding, no privileged mode, no host filesystem mount, no Docker socket, and defined resource/restart policies; poll a bounded HTTP readiness endpoint; persist container ID and state.
- **Design decision:** Use one bounded startup probe for initial acceptance; do not promise continuous health monitoring. Keep container engine calls behind a runtime adapter.
- **Security/failures:** Test unavailable Docker, missing image, invalid application port, startup exit, readiness timeout, cancellation, and container cleanup failure. Never report healthy before the probe succeeds.
- **Tests:** Adapter unit tests with a fake runtime, plus Linux Docker integration test using a trusted fixture image.
- **Acceptance:** A valid fixture responds through its loopback port; failed startup records a failure and leaves unrelated containers untouched.
- **Dependencies:** M05.1.
- **Documentation:** Runtime flags, port behavior, probe timeout, and failure diagnosis.

### M05.3 — Safe routing and replacement

- **Goal/scope:** Route HTTP traffic to the new container only after readiness and preserve the previous working route if replacement fails.
- **Tasks:** Generate and validate proxy configuration; apply/reload it through a controlled host-local mechanism; switch the route only after readiness; retain the old container until the new route succeeds; remove obsolete resources only after confirmation.
- **Design decision:** Use explicit generated configuration rather than Docker labels requiring proxy access to the Docker socket. Keeping the last working container until the route switch succeeds is a correctness prerequisite; a user-driven rollback feature remains future work.
- **Security/failures:** Validate hostnames and generated config; test proxy reload failure, route conflict, old-container stop failure, and interrupted switch. Do not allow arbitrary proxy directives from application input.
- **Tests:** Config generation and route-switch unit tests plus Linux proxy integration tests.
- **Acceptance:** A failed new deployment does not silently remove or reroute the prior healthy deployment; unrelated applications remain reachable.
- **Dependencies:** M05.2.
- **Documentation:** Proxy configuration, operator DNS setup, and replacement semantics.

### M05.4 — End-to-end Linux acceptance

- **Goal/scope:** Demonstrate the complete trusted Git-to-HTTP path on one host.
- **Tasks:** Register a fixture repository, submit a verified push, build the exact commit, start the image, verify the assigned HTTP route, inspect state, and exercise a failed replacement.
- **Design decision:** Keep the acceptance topology single-host and operator-managed; defer Kubernetes, remote workers, automatic TLS, and high availability.
- **Security/failures:** Run only trusted fixture code; verify API and Docker daemon exposure, workload environment allowlist, loopback bindings, cleanup, and failure retention.
- **Tests:** Full Linux end-to-end test and all backend/frontend checks.
- **Acceptance:** A documented clean setup returns the fixture response over HTTP and a failed subsequent deployment leaves the working version reachable.
- **Dependencies:** M05.3.
- **Documentation:** README status, operating procedure, architecture flow, and troubleshooting.

## Exit condition

M05 is complete only after the Linux end-to-end scenario succeeds and failure replacement preserves the previous working route. This completes the initial implementation phase. M06 proves the integrated workflow, and M07–M18 specify the subsequent recovery, operator controls, hardening, release, and readiness work.
