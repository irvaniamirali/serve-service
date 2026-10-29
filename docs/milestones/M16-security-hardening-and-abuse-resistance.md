# M16 — Security Hardening and Abuse Resistance

## A. Metadata

- **Status:** Planned
- **Dependencies:** M03, M08, M10, M11, M12, M13, M14
- **Result:** A prioritized threat model and regression suite cover the complete single-host platform, with residual risks explicitly accepted by the operator.

## B. Goal

Consolidate and independently verify security controls across the API, webhook, workers, build/runtime, database, proxy, and host before calling the system responsibly operable.

## C. Current problem and context

Earlier milestones implement individual protections: webhook signatures, auth, encrypted runtime secrets, container restrictions, redaction, and cleanup. Cross-component composition can still introduce replay, privilege, leakage, injection, and denial-of-service paths.

## D. Scope

**In scope:** maintained threat model, security regression checklist, security tests, ingress/request limits, dependency/image update policy, and review of residual risks.

**Out of scope:** certification, penetration testing as a service, production multi-tenant guarantee, cluster hardening, and claiming the local Docker build is safe for hostile tenants.

## E. Functional requirements

- **FR-01:** Threat model identifies assets, trust boundaries, attacker capabilities, abuse cases, mitigations, owner milestone, and residual risk for API, webhook, database, workers, build, runtime, proxy, host, and secrets.
- **FR-02:** Verify webhook HMAC over raw bytes, delivery idempotency, repository/ref identity, request-body limit, and rate control against replay and abuse.
- **FR-03:** Verify all admin APIs and sensitive logs/config require M11 authorization; public application traffic cannot cross into admin routes.
- **FR-04:** Verify repository URL/ref handling, subprocess argument safety, filesystem containment, output bounds, and no secret forwarding.
- **FR-05:** Verify container privilege/resource restrictions and Docker socket exposure from M12.
- **FR-06:** Add bounded request/time/resource policies where absent and document trusted-proxy assumptions.
- **FR-07:** Establish dependency update and vulnerability triage policy using existing Go/npm manifests and CI; do not auto-upgrade unreviewed runtime dependencies in production.
- **FR-08:** Keep an explicit residual-risk list and operator sign-off; do not remove operator-trusted-repository restriction unless a separate isolation review supports it.

## F. Architecture and design

Maintain a compact prioritized threat model under `docs/security/` and a testable checklist in the security review procedure. Controls are implemented by their owning earlier milestones; M16 closes cross-boundary gaps and proves them, rather than duplicating their feature work. Prioritize direct host compromise, secret disclosure, unauthorized deployment, event forgery/replay, and resource exhaustion.

Set request limits/rate limits in the existing Fiber server and host proxy. Use in-process rate controls only for single-host endpoints; do not add a distributed limiter. Require external TLS termination or loopback access for credential-bearing requests and state the trusted proxy header rules.

## G. Data model and state transitions

No new lifecycle status is introduced. Add schema only when a specific finding requires a missing integrity or audit constraint. Security-relevant changes use versioned migrations and verify existing rows before tightening constraints.

## H. API and external interfaces

No broad new public API. Define request limits, login limits, webhook response behavior, and protected diagnostics in existing endpoint docs. Rate limiting returns a stable `429` without disclosing implementation details; webhook retry guidance must distinguish retryable server failures from rejected payloads.

## I. Security requirements

- **SEC-01:** Maintain the threat model and link each mitigation to a regression test or operator control.
- **SEC-02:** Ensure HMAC/replay/idempotency and authentication/authorization fail closed on storage errors.
- **SEC-03:** Run static analysis, dependency vulnerability checks, and secret scanning in CI without requiring production credentials.
- **SEC-04:** Verify build and runtime process environment allowlists and container restrictions.
- **SEC-05:** Document that host root, Docker daemon compromise, kernel vulnerabilities, and denial of service remain residual risks.

## J. Failure scenarios

Exercise invalid/replayed webhooks, oversized requests, brute-force login, expired sessions, database outages during auth/idempotency, malicious repository URL/ref/path, secret canary output, Docker daemon failure, resource exhaustion, proxy misconfiguration, stale cleanup, and dependency alert response. Each must fail closed or have a bounded, documented operator recovery.

## K. Testing strategy

- Unit and HTTP security tests for validation, auth, rate limits, HMAC, headers, and redaction.
- PostgreSQL tests for replay, duplicate delivery, session revocation, and constraints.
- Linux/Docker inspection for privileges, resource limits, mounts, and port binding.
- CI static/vulnerability/secret checks that use deterministic local inputs or public advisory databases without repository secrets.
- Manual threat-model walkthrough before acceptance; no external penetration vendor required.

## L. Acceptance criteria

- [ ] **AC-01:** Every high-priority threat has a tested mitigation or explicitly accepted residual risk and owner.
- [ ] **AC-02:** Security regression tests cover API auth, webhook replay, secret leakage, unsafe runtime options, and bounded requests.
- [ ] **AC-03:** CI reports static-analysis and dependency/secret-check results without secrets.
- [ ] **AC-04:** No known critical/high applicable vulnerability is left without documented disposition.
- [ ] **AC-05:** The service still explicitly restricts builds to trusted repositories.

## M. Documentation and operations

Create/update `docs/security/threat-model.md`, security checklist, vulnerability response instructions, trusted proxy/TLS setup, and residual-risk sign-off. Link each control to its owning milestone rather than duplicating the milestone specification.

## N. Completion definition

Complete M16 after threat-model review, regression checks, vulnerability disposition, and operator-facing residual risk documentation are all recorded.
