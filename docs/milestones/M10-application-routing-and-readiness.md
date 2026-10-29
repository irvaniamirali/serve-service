# M10 — Application Routing and Readiness Hardening

## A. Metadata

- **Status:** Planned
- **Dependencies:** M05, M06, M07, M09
- **Result:** Each application has a stable, documented HTTP endpoint whose route reflects the confirmed runtime state and fails safely when that runtime is unavailable.

## B. Goal

Harden M05's initial loopback-port and operator-managed proxy strategy for stable application addressing, ongoing unavailability, and controlled route lifecycle.

## C. Current problem and context

M05 owns the first route and startup probe; this milestone does not create a second routing implementation. The initial local endpoint may rely on a port, and basic startup readiness does not define the behavior of stopped or later-unhealthy containers, endpoint stability, or stale routes.

## D. Scope

**In scope:** stable single-host HTTP endpoint mapping, route ownership/reconciliation, readiness versus process-running semantics, unavailable application behavior, endpoint conflict and stale route handling.

**Out of scope:** HTTPS automation, public DNS automation, multiple ingress strategies, global load balancing, canary traffic, and high availability.

## E. Functional requirements

- **FR-01:** Each application has one stable endpoint mapping to its confirmed current deployment; proposed public hostname format is `<validated-app-slug>.<operator-configured-domain>`.
- **FR-02:** Slugs are 1–63 ASCII characters, lowercase letters, digits, or hyphens; the first and last characters must be a letter or digit. Normalize ASCII case before uniqueness checks and reject conflicts without changing existing routing.
- **FR-03:** Routing points only to a loopback-bound host port allocated under M05 and a current deployment confirmed by M09.
- **FR-04:** Route configuration is generated from persisted platform records, validated before activation, and never accepts arbitrary proxy directives.
- **FR-05:** A periodic or startup reconciliation compares desired route records with actual proxy configuration and repairs only platform-owned entries.
- **FR-06:** A stopped or unhealthy runtime is not reported as application-ready. The endpoint returns the proxy's documented unavailable response without routing to another application's port.
- **FR-07:** Route deletion or application removal follows an explicit operator action and never deletes unrelated proxy configuration.
- **FR-08:** The system reports startup readiness separately from ongoing runtime health; liveness does not imply readiness.

## F. Architecture and design

Continue the M05 host-managed reverse proxy design. The proxy has no Docker socket and receives only validated loopback upstreams. The API/runtime manager owns endpoint metadata; a narrow proxy adapter renders and atomically validates/applies generated config. The selected proxy must support config validation before reload; retain the last valid configuration if validation or reload fails.

Expose one operator-configured domain and HTTP only. Operator-managed DNS and host firewall remain prerequisites. If the M05 implementation instead uses an alternative, document the migration to this single strategy before implementation; do not support both.

## G. Data model and state transitions

Reuse application slug/endpoint and runtime port fields introduced by M05 where possible. Add a unique normalized endpoint key and ownership marker only if the schema lacks them. Endpoint lifecycle is `pending -> active -> unavailable -> active|removed`; changes are driven by verified route activation, runtime status, and explicit removal. Reconciliation is idempotent and does not change deployment health status by itself.

## H. API and external interfaces

Expose the assigned endpoint through application/deployment read responses; do not add a user-controlled proxy-config API. Application create/update validation rejects invalid/duplicate slugs. Public requests route through the configured proxy; admin API paths remain protected by M11. Return documented `404` for unknown host and `503` for a known but unavailable application.

## I. Security requirements

- **SEC-01:** Bind all application host ports to loopback; verify this by inspecting actual Docker port bindings.
- **SEC-02:** Validate hostname labels and prevent CRLF/config injection.
- **SEC-03:** Proxy access to the Docker socket is prohibited.
- **SEC-04:** Route reconciliation modifies only generated, platform-owned configuration.
- **SEC-05:** Do not expose host management or application private ports through wildcard interfaces.

## J. Failure scenarios

Cover duplicate hostname, invalid slug/domain, proxy validation/reload failure, stale config, API/proxy restart, missing upstream, stopped container, host-port conflict, route deletion interruption, and unknown host. A failed config update preserves the last known-good config and deployment route.

## K. Testing strategy

- Unit tests for slug, endpoint, and config generation validation.
- Proxy adapter tests with a local fake executable/config validator.
- Integration test for config apply/reload and rollback to last valid config.
- Linux networking tests proving loopback-only binding and unavailable responses.
- No live public DNS or certificate authority required.

## L. Acceptance criteria

- [ ] **AC-01:** Two applications cannot claim the same endpoint.
- [ ] **AC-02:** A valid endpoint routes to only its current deployment.
- [ ] **AC-03:** Stopped/unhealthy upstream returns documented `503`; unknown host returns documented `404`.
- [ ] **AC-04:** Invalid generated configuration is rejected and the prior valid config remains active.
- [ ] **AC-05:** Restart reconciliation restores only owned platform routes.
- [ ] **AC-06:** Linux test confirms application port binds to loopback, not a wildcard host address.

## M. Documentation and operations

Update routing diagram, domain/DNS and firewall setup, proxy config ownership, readiness meanings, route failure troubleshooting, and cleanup procedures.

## N. Completion definition

Complete M10 when endpoint conflicts, unavailable runtimes, proxy failures, and restart reconciliation meet the acceptance tests on Linux.
