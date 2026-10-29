# M07 — Durable Workers and Recovery

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06
- **Result:** Build and deployment jobs recover predictably after worker or host interruption without claiming exactly-once external execution.

## B. Goal

Make asynchronous work durable, bounded, safe to retry, and observable across worker restarts and competing processes.

## C. Current problem and context

M04's build queue already uses PostgreSQL locking and marks sufficiently old `building` rows failed. M05 introduces external container and routing effects. A database transaction cannot atomically commit with Docker or a proxy, so a crash can leave an external resource whose database state is incomplete. M06 proves the workflow; this milestone defines safe recovery for that workflow.

## D. Scope

**In scope:** explicit ownership/lease policy, attempt accounting, safe retry classes, stale-operation reconciliation, graceful shutdown, competing-worker tests, and deployment-stage recovery.

**Out of scope:** multi-host scheduling, exactly-once guarantees, automatic retry of arbitrary build failures, cross-region recovery, and a distributed message broker.

## E. Functional requirements

- **FR-01:** A worker claims work with a database-atomic operation and records a claim token/lease expiry or an equivalent fenced ownership mechanism.
- **FR-02:** Only the current owner can renew a lease or commit a result; stale owners cannot overwrite newer state.
- **FR-03:** Every attempt has a bounded attempt number and a persisted last error classification.
- **FR-04:** Retry only explicitly transient failures (for example, temporary database/network unavailability); validation, unsupported repository layout, and deterministic build failures are permanent unless an operator creates a new deployment.
- **FR-05:** Retries use capped exponential backoff with jitter and a finite configured attempt ceiling. Exceeding it transitions to `failed`.
- **FR-06:** On startup, a reconciler discovers expired leases and checks external resources by stable platform labels/IDs before deciding whether to resume, finalize, or fail.
- **FR-07:** Shutdown stops new claims, cancels in-flight commands, records a bounded outcome when possible, and exits within a documented grace period.
- **FR-08:** The system documents delivery as at-least-once with idempotent state updates and best-effort external-effect reconciliation, not exactly once.

## F. Architecture and design

Keep PostgreSQL as the queue; do not add Redis or a broker for one Linux host. Use a monotonically increasing lease generation and require every renewal and state commit to compare that generation. Before each external side effect, revalidate ownership; use stable resource IDs/labels and idempotent adapter operations so a side effect racing lease expiry can be reconciled before a successor acts. Docker and the proxy do not enforce database fencing tokens, so document the remaining race rather than claiming external exactly-once behavior.

The worker calls narrow queue, runtime, and build interfaces. It must not hold a database transaction open while running Git, Docker, HTTP probes, or proxy commands. External resources receive stable deployment-scoped labels so recovery can find them. A reconciliation pass is idempotent and can run after every process start.

## G. Data model and state transitions

Add attempt/ownership metadata only where the current deployment schema does not already provide it: attempt count, lease owner/token, lease expiry, and retry-after time. Add indexes for eligible queued work and expired leases. Keep the existing deployment status names; retries within a stage do not introduce new public statuses.

| Condition | Action |
|---|---|
| Eligible queued/ retryable job | Claim and increment attempt atomically |
| Owner lease valid | Only that owner may record result |
| Lease expired, no external effect | Requeue if retryable and attempts remain |
| Lease expired, external resource found | Reconcile resource and state idempotently |
| Permanent failure or attempt ceiling reached | Mark `failed` with safe reason |
| Terminal status | Never reclaim |

Concurrent claims must not return the same valid lease. A crash after Docker creates a resource but before database commit is handled by discovery, not blind duplicate creation.

## H. API and external interfaces

Do not expose claim tokens, worker host details, or raw retry internals through the public deployment API. Add only operator-visible attempt count and last stage/error fields where useful; detailed worker state access is subject to M11 authorization.

## I. Security requirements

- **SEC-01:** Lease tokens are random, stored/compared safely, and never logged or returned.
- **SEC-02:** Resource discovery is limited to resources labeled as owned by Serve Service and the matching deployment ID.
- **SEC-03:** A stale worker cannot overwrite state after lease expiry/reassignment.
- **SEC-04:** Retries cannot amplify resource exhaustion; attempts and backoff are bounded.

## J. Failure scenarios

Cover worker crash before claim commit, after claim, during build, after image creation, during container start, after readiness, during route switch, database loss, lease renewal loss, duplicate reconciler execution, and graceful shutdown timeout. If an external effect cannot be determined safely, mark the job for operator intervention rather than guessing success.

## K. Testing strategy

- Unit tests for retry classification, jitter bounds, lease expiration, and idempotent reconciliation.
- PostgreSQL integration tests with two competing workers and stale ownership.
- Process-level tests that kill/restart a worker around each external side-effect boundary.
- Docker/Linux integration tests for labeled resource discovery and safe cleanup.
- No live GitHub access; webhook test fixtures from M03 remain local.

## L. Acceptance criteria

- [ ] **AC-01:** Two concurrent workers never hold valid ownership of the same job.
- [ ] **AC-02:** A stale worker result is rejected after ownership changes.
- [ ] **AC-03:** Transient errors retry only within the configured maximum and permanent errors do not retry.
- [ ] **AC-04:** Restart reconciliation finds and safely resolves at least one injected post-side-effect/pre-commit crash.
- [ ] **AC-05:** Shutdown stops claims and returns within the documented grace period.
- [ ] **AC-06:** Documentation states the actual at-least-once/external-effect guarantee.

## M. Documentation and operations

Update worker runbooks with lease/attempt configuration, retry classification, restart behavior, stuck-job diagnosis, and safe operator action. Add an ADR only if the lease/fencing choice departs from PostgreSQL row ownership in a material way.

## N. Completion definition

Complete M07 when all retry, concurrency, restart, and stale-owner acceptance tests pass and no documentation claims exactly-once execution.
