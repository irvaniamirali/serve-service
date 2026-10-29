# M13 — Observability and Diagnostics

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06, M07, M09, M10, M11
- **Result:** An operator can correlate requests, worker attempts, build output, runtime state, and health signals without exposing secrets or deploying an unnecessary observability stack.

## B. Goal

Make normal operation and failure diagnosis possible using the application's existing structured logging and persisted deployment data.

## C. Current problem and context

Current code uses `log/slog` and bounded build logs, but correlation conventions, retention/access policy, runtime diagnostics, queue health, and dependency readiness are incomplete. A process-level `/health` endpoint alone does not reveal whether the database or worker is usable.

## D. Scope

**In scope:** correlation fields, worker/runtime logs, readiness and liveness semantics, bounded diagnostic access, useful internal metrics, secret redaction, operator troubleshooting.

**Out of scope:** a large hosted telemetry stack, distributed tracing vendor, public dashboards, indefinite log storage, and application-level log aggregation.

## E. Functional requirements

- **FR-01:** Every request and worker attempt carries a generated correlation ID; logs include request ID, application ID, deployment ID, stage, attempt, and operation where applicable.
- **FR-02:** Correlation IDs from clients are length/character validated; invalid values are replaced, not trusted as log structure.
- **FR-03:** Liveness reports only process liveness. Readiness reports required local dependencies and worker startup state separately, with bounded timeouts.
- **FR-04:** Logs use stable field names and levels; errors include wrapped operation context but not credentials or raw secrets.
- **FR-05:** Build logs remain bounded, access-controlled by M11, and tied to a deployment/attempt. Runtime logs are bounded and access-controlled if exposed.
- **FR-06:** Provide minimal metrics for queue depth/age, active/stale jobs, stage durations/failures, DB pool health, and runtime count; labels must not include unbounded user input.
- **FR-07:** Metrics are exposed only on an operator-protected/local endpoint and do not contain secret values, payloads, or arbitrary repository names.
- **FR-08:** Define log retention and cleanup ownership in M14.

## F. Architecture and design

Use `slog` consistently and inject a logger at composition boundaries. Expose metrics in Prometheus text format using the maintained `prometheus/client_golang` library; do not create a custom exposition format. Require M11 authentication for metrics and administrative diagnostics. Do not build a dashboard until metrics have been exercised with real failures.

Use bounded in-process health probes and database `Ping`/pool stats; never perform a long external call on every readiness request. Detailed logs remain in their current durable store only within the configured cap; large log streaming/storage is future work.

## G. Data model and state transitions

Do not add a generic event table merely for logging. Persist stage timestamps, attempt identifiers, and bounded failure information already needed by M06/M07. If an append-only audit/attempt table is needed for multiple retries, define its ownership and retention; do not duplicate deployment's current status.

## H. API and external interfaces

Keep `/health` as liveness or document its existing use; add a separate readiness route and protected metrics endpoint. Admin diagnostics require M11. Error response envelope remains stable. Logs endpoint must enforce output bounds and no-store cache policy.

## I. Security requirements

- **SEC-01:** Redact known secret/config keys and credential-bearing repository URLs before logging.
- **SEC-02:** Never log webhook signature, session cookie, password, encryption key, secret value, or full environment.
- **SEC-03:** Bound log fields, metric cardinality, and health check duration.
- **SEC-04:** Protect diagnostic and metrics endpoints; public app routes cannot reach admin diagnostics.

## J. Failure scenarios

Cover database unavailable, worker stopped, queue stuck, log output over limit, malformed correlation header, redaction bypass attempt, metrics endpoint error, and readiness timeout. Liveness remains responsive while readiness accurately fails for required dependencies.

## K. Testing strategy

- Unit tests for correlation validation, field consistency, log bounds, and redaction canaries.
- HTTP tests for liveness/readiness/metrics auth and timeout.
- Worker tests assert identifiers survive context propagation.
- PostgreSQL tests for queue age/active count queries.
- Manual Linux failure injection and inspection; no hosted telemetry account.

## L. Acceptance criteria

- [ ] **AC-01:** Logs from one request/build/runtime share a traceable request/deployment identifier.
- [ ] **AC-02:** Secret canaries do not appear in logs, metrics, errors, or build diagnostics.
- [ ] **AC-03:** Readiness fails on required dependency failure while liveness remains available.
- [ ] **AC-04:** Queue age/worker failure are diagnosable without SQL access.
- [ ] **AC-05:** Metrics cardinality and response size are bounded.

## M. Documentation and operations

Add a logging/metrics field reference, endpoint access rules, health semantics, common failure queries, and links to the M07/M14 runbooks.

## N. Completion definition

Complete M13 when the listed failure scenarios can be diagnosed from protected application diagnostics and redaction tests pass.
