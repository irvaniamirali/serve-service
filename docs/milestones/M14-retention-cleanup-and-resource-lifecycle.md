# M14 — Retention, Cleanup, and Resource Lifecycle

## A. Metadata

- **Status:** Planned
- **Dependencies:** M07, M09, M12, M13
- **Result:** A scheduled, auditable cleanup process bounds database/log/image/container growth without deleting active or recovery-required resources.

## B. Goal

Make resource cleanup predictable and safe as deployments accumulate, including after interrupted worker operations.

## C. Current problem and context

The current system stores deployment history and bounded build logs and creates local images/workspaces. M05 adds containers and routes. Without explicit retention, image layers, logs, stopped resources, and database history can grow without bound; naive pruning can destroy the current or rollback version.

## D. Scope

**In scope:** retention defaults, cleanup eligibility, dry-run preview, idempotent cleanup, orphan detection, audit of destructive actions, bounded resource use.

**Out of scope:** user-configurable arbitrary retention schedules, object storage, legal/compliance retention, and deletion of audit-relevant records without explicit policy.

## E. Functional requirements

- **FR-01:** Define operator-configurable age/count retention for deployment records, build logs, images, stopped runtime instances, and temporary workspaces, with conservative defaults and hard bounds.
- **FR-02:** Cleanup excludes current/desired deployments, active jobs/leases, configured rollback candidates, and resources referenced by a route.
- **FR-03:** Cleanup runs in bounded batches and is safe to repeat after process interruption.
- **FR-04:** Provide a dry-run report listing resource type, count, and total estimated size before deletion.
- **FR-05:** Destructive cleanup requires an operator action or a documented scheduled policy and is recorded with actor/time/counts; never log secret data.
- **FR-06:** Delete database rows only after owned external resources are removed or explicitly marked orphaned for retry.
- **FR-07:** Build/deployment history required for diagnosis is retained according to M13/M15 policy; removal of a deployment record must not break foreign-key integrity or hide active state.
- **FR-08:** Temporary workspace cleanup removes only paths created under the configured Serve Service root and never follows symlinks outside it.

## F. Architecture and design

Implement a cleanup use case and adapters for PostgreSQL and Docker; do not put pruning logic in HTTP handlers or worker shutdown. Run it daily at 02:00 UTC using a single-host scheduler and a PostgreSQL advisory lock to prevent overlap. Default retention is 90 days or 1,000 terminal deployments per application (whichever limit is reached first), retaining the current deployment and the three newest healthy rollback candidates; remove stopped unreferenced containers after 24 hours, images only when no retained/current deployment references them and they have been unused for 30 days, and abandoned workspaces after 24 hours. Keep retention limits operator-configurable only within documented hard bounds. Docker commands must use Serve Service ownership labels and never target global unused images.

## G. Data model and state transitions

Use M07 labels/ownership IDs and M09 current/rollback references. Persist a cleanup-run record containing run ID, effective OS user, start/end time, dry-run flag, per-resource counts, and outcome; retain it for one year and purge it under a separate bound. Resource lifecycle is `eligible -> deletion_requested -> deleted` or `cleanup_failed`; active resources are never eligible. Avoid a new deployment status for cleanup.

## H. API and external interfaces

Provide the CLI command `go run ./cmd/maintenance cleanup` from `src/`, with explicit `--dry-run` and `--execute` modes; do not expose destructive cleanup through the application API. Restrict execution to the host operator/service account, record the effective OS user and timestamp for each run, and make the result a summary rather than arbitrary host paths or secret values. Re-evaluate eligibility immediately before deletion so a resource that became active after planning is skipped.

## I. Security requirements

- **SEC-01:** All deletions are scoped to platform-owned resource labels and validated workspace roots.
- **SEC-02:** Recheck active/current/rollback references within the deletion transaction or fencing boundary.
- **SEC-03:** Do not execute shell strings assembled from database values.
- **SEC-04:** Destructive operations are authenticated, authorized, bounded, and audited.

## J. Failure scenarios

Cover disk full, Docker unavailable, database outage, partial external deletion, stale dry-run plan, concurrent deployment, symlink/path traversal, corrupted image metadata, scheduler overlap, and cleanup crash. Failed cleanup remains retryable and does not mark a resource deleted prematurely.

## K. Testing strategy

- Unit tests for retention eligibility and dry-run planning.
- PostgreSQL tests for active reference exclusion and concurrent state change.
- Fake Docker tests prove commands are label-scoped and bounded.
- Linux integration test creates owned and unrelated containers/images and verifies only eligible owned resources are removed.
- Filesystem tests for symlink and workspace boundary behavior.

## L. Acceptance criteria

- [ ] **AC-01:** Dry run reports eligible resources and performs no deletions.
- [ ] **AC-02:** Current, active, routed, and rollback-retained images/containers are never selected.
- [ ] **AC-03:** Cleanup deletes only platform-owned resources in the fixture test.
- [ ] **AC-04:** Interrupted cleanup can resume without corrupting current deployment state.
- [ ] **AC-05:** Batch and retention bounds prevent one run from monopolizing the host.

## M. Documentation and operations

Document defaults, dry-run/execute commands, resource ownership labels, recovery after partial cleanup, history retention, and disk-pressure response. Update the backup/restore and rollback runbooks.

## N. Completion definition

Complete M14 after dry-run, deletion-scope, interruption, active-resource protection, and Linux cleanup tests pass.
