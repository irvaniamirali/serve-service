# M09 — Deployment Safety and Version Management

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06, M07, M08
- **Result:** Operators can identify the current and desired application version, preserve a working version during replacement, and explicitly roll back to a retained healthy deployment.

## B. Goal

Turn successful deployments into safely versioned runtime changes with deliberate rollback and retention-aware cleanup.

## C. Current problem and context

M05 keeps an old runtime available until a replacement is ready, and M06 proves the flow. That protection does not define a durable current-version pointer, user-requested rollback, concurrent deployment policy, or how image/config versions remain available after cleanup.

## D. Scope

**In scope:** immutable image identity, desired/current deployment pointers, serialized application deployment, explicit rollback, retained known-good runtime, artifact ownership.

**Out of scope:** automatic rollback based on ongoing health monitoring, multi-region promotion, traffic canaries, and artifact registry replication.

## E. Functional requirements

- **FR-01:** A deployment's build artifact is identified by immutable local image ID, not tag alone.
- **FR-02:** Each application has at most one deployment operation changing its active route at a time; if one is active, reject a new non-duplicate request with `409` and create no deployment record.
- **FR-03:** The current deployment pointer changes only after the new runtime passes M05 readiness and routing activation is confirmed.
- **FR-04:** The previous healthy deployment and image remain available until replacement is confirmed and the configured retention policy permits removal.
- **FR-05:** An authorized operator can request rollback to a retained healthy deployment by deployment ID; rollback creates a new deployment/audit operation referencing the existing image and configuration revision rather than mutating history.
- **FR-06:** Rollback uses the same readiness and route-switch checks as a normal deployment.
- **FR-07:** Failed replacement or failed rollback leaves the last confirmed healthy route active.
- **FR-08:** Cleanup cannot delete an image or runtime instance referenced by current/desired pointers, an active operation, or a retained rollback candidate.

## F. Architecture and design

Model a deployment as an immutable attempt and runtime instance as a deployment-owned external resource. Application state contains desired and current deployment references; the latter changes only after route activation. An explicit rollback creates a new operation with a `rollback_of_deployment_id` reference; it does not rewrite the old deployment status.

Serialize route-changing operations per application using a database lock/version check. Keep old runtime until route activation commits. A crash at route switch is reconciled by M07 and must not infer the active version from a mutable tag.

## G. Data model and state transitions

Add desired/current deployment references to application/runtime ownership only if equivalent M05 records do not already exist. Add nullable rollback source reference and an application-scoped operation lock/version. Add foreign keys, no self-referential rollback, and indexes for active deployment lookup.

| Operation | State / pointer behavior |
|---|---|
| Normal request | `queued -> building -> built -> deploying -> healthy`; desired set when accepted |
| Successful route activation | Current pointer atomically advances to new healthy deployment |
| Failed replacement | New deployment `failed`; current pointer unchanged |
| Rollback request | New deployment references retained healthy target; runs through `deploying` and readiness |
| Successful rollback | Current pointer advances to rollback operation; prior deployment records remain immutable |
| Concurrent request | Reject with `409`; create no second route-changing deployment |

## H. API and external interfaces

Add `POST /api/v1/applications/{id}/deployments/{deploymentId}/rollback` (authorized by M11). It returns `202 Accepted` with a new queued deployment ID and rollback source. Reject non-healthy, non-retained, or cross-application targets with `409` or `404` without disclosing unrelated records.

## I. Security requirements

- **SEC-01:** Only the runtime adapter can activate routes or stop platform-owned resources.
- **SEC-02:** Rollback authorization is application-scoped; requesters cannot select another application's image.
- **SEC-03:** Never use mutable image tags as sole rollback identity.
- **SEC-04:** User-provided identifiers cannot become Docker names or host paths without validation/controlled mapping.

## J. Failure scenarios

Cover concurrent deploy/rollback, missing image, removed image, readiness failure, route activation failure, worker restart before/after pointer update, database failure at pointer change, old container stop failure, and attempted rollback to a failed/foreign deployment. Preserve the prior route and report orphan cleanup separately.

## K. Testing strategy

- Unit tests for eligibility, same-application ownership, and transition policy.
- PostgreSQL tests for serialization, pointer atomicity, and referential integrity.
- Runtime integration tests for image reuse and failed activation preserving current.
- Linux test that deliberately fails a replacement and verifies the old HTTP response remains.
- No live GitHub required.

## L. Acceptance criteria

- [ ] **AC-01:** Current pointer advances only after successful readiness and route activation.
- [ ] **AC-02:** Failed replacement leaves current pointer and endpoint unchanged.
- [ ] **AC-03:** A rollback produces a new deployment record and retains immutable history.
- [ ] **AC-04:** Concurrent operations follow the documented serialized policy.
- [ ] **AC-05:** Cleanup tests prove current, active, and retained rollback images are not eligible.
- [ ] **AC-06:** Cross-application rollback attempts are denied.

## M. Documentation and operations

Document current/desired semantics, rollback endpoint, eligibility, image retention relation, concurrent request policy, and recovery steps in API, architecture, and operator guides.

## N. Completion definition

Complete M09 when normal deployment, failed replacement, and explicit rollback pass database and Linux runtime tests and retained image identity is unambiguous.
