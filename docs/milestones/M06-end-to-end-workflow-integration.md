# M06 — End-to-End Workflow Integration

## A. Metadata

- **Status:** Planned
- **Dependencies:** M03, M04, M05
- **Result:** A verified GitHub push for a registered application can produce a commit-specific image, start a runtime instance, and reach a documented HTTP endpoint with each outcome visible through the existing API.

## B. Goal

Join the capabilities delivered independently by M03–M05 into one traceable Git-to-HTTP workflow and prove it with a deterministic sample application.

## C. Current problem and context

M02 persists application and deployment state; M03 handles webhooks; M04 builds images; M05 starts and routes containers. Passing unit tests in each component does not prove that the correct commit, image, runtime instance, and route remain associated across the whole lifecycle or that a partial failure is diagnosable.

## D. Scope

**In scope:** component wiring, lifecycle ownership, identifiers across stages, one sample application, success and failure end-to-end tests, operator-facing workflow evidence.

**Out of scope:** new authentication, multi-tenant operation, advanced retries (M07), secret configuration (M08), user-driven rollback (M09), and production release packaging (M17).

## E. Functional requirements

- **FR-01:** A GitHub push accepted by M03 results in one deployment associated with the registered application, push commit SHA, build result, and runtime instance.
- **FR-02:** The worker consumes durable PostgreSQL state; no in-memory-only handoff is required for correctness.
- **FR-03:** Each stage records its own start/end timestamps and a bounded failure reason against the deployment.
- **FR-04:** The runtime is not started unless a successful build artifact is persisted and its immutable local image ID is available.
- **FR-05:** An operator can retrieve the deployment and identify its commit, current stage, image identity, runtime endpoint, and failure reason without querying worker memory.
- **FR-06:** Re-delivery of a webhook does not create an additional workflow, subject to the idempotency guarantees completed in M03.

## F. Architecture and design

The API owns request validation and initial `queued` creation. The build worker owns `queued -> building -> built|failed`. The deployment worker owns `built -> deploying -> healthy|failed`. Each worker persists a transition before performing the corresponding external side effect and records the result afterward. `bootstrap` composes workers and adapters; domain packages must not import each other's persistence adapters.

Use the PostgreSQL deployment row as the workflow's durable aggregate for this single-host design. Add only fields that are required to identify the runtime instance and endpoint; do not mirror image or application data already represented elsewhere. Any event/audit history beyond current state is deferred to M13.

The end-to-end test uses a checked-in minimal fixture or a local test repository pinned to a full commit. It must not depend on public GitHub availability: submit a signed synthetic webhook or create the deployment through the API. Use a real local Docker Engine and Linux host only for the final runtime test.

## G. Data model and state transitions

No separate build-job table is required unless the implementation demonstrates that one deployment can own multiple independently retried build attempts. Persist any M05 runtime instance identifier and endpoint on the deployment or its existing runtime record, with a unique constraint preventing two active runtime records for the same deployment.

The shared lifecycle remains:

| From | To | Owner / trigger |
|---|---|---|
| `queued` | `building`, `cancelled` | Build worker / explicit cancellation |
| `building` | `built`, `failed`, `cancelled` | Build worker |
| `built` | `deploying`, `failed`, `cancelled` | Deployment worker |
| `deploying` | `healthy`, `failed`, `cancelled` | Runtime worker after readiness or failure |
| `healthy`, `failed`, `cancelled` | — | Terminal |

Use compare-and-set updates. Duplicate processing must return or observe the existing outcome; it must not repeat an external side effect blindly. Host restart recovery policy is owned by M07.

## H. API and external interfaces

Keep the existing `/api/v1` application and deployment routes unless the integrated behavior proves a response field is missing. Extend deployment representation only with persisted runtime identity/endpoint and clear readiness status. Preserve existing error envelope and pagination. No live GitHub API calls are allowed in the end-to-end test.

## I. Security requirements

- **SEC-01:** Run only operator-trusted fixture code under the M04/M05 trust boundary.
- **SEC-02:** No Docker socket, database credential, webhook secret, or platform token enters the workload environment.
- **SEC-03:** Bind application listeners as specified in M05; never expose arbitrary container ports directly to the host network.
- **SEC-04:** Do not return internal command output or secret-bearing configuration in ordinary deployment responses.

## J. Failure scenarios

Invalid webhook, duplicate delivery, commit unavailable, build failure, missing image, runtime start failure, readiness timeout, route update failure, database outage before/after an external side effect, and process termination between stages must each leave a persisted diagnosable state. A failed deployment must not silently replace a previously working route. Restart/retry semantics are defined by M07, not improvised here.

## K. Testing strategy

- Unit tests for stage-to-stage identity and allowed transitions.
- HTTP tests for deployment status fields and stable error shapes.
- PostgreSQL integration test for durable state across server/worker restart.
- Deterministic local webhook and duplicate-delivery test, without live GitHub.
- Linux end-to-end test with PostgreSQL and a real Docker Engine; a reverse proxy is required if selected by M05.
- Failure injection at build, runtime, readiness, route update, and persistence boundaries.

## L. Acceptance criteria

- [ ] **AC-01:** A pinned sample commit is returned by HTTP after a signed local webhook and is identical in the deployment and build record.
- [ ] **AC-02:** The image ID recorded by the build is the image used by the runtime instance.
- [ ] **AC-03:** The successful runtime endpoint returns the fixture's expected response.
- [ ] **AC-04:** Injected build, startup, readiness, and route failures produce a failed deployment with the correct stage and non-empty diagnostic.
- [ ] **AC-05:** Restarting the API between stages does not lose persisted identifiers or status.
- [ ] **AC-06:** A duplicate webhook does not create a second deployment.
- [ ] **AC-07:** A failed replacement leaves the prior healthy endpoint reachable.

## M. Documentation and operations

Update the architecture flow, API response documentation, sample application instructions, and a concise end-to-end runbook. Record any newly required schema fields and configuration in their migration/reference docs.

## N. Completion definition

Complete M06 only when all acceptance criteria pass on the documented Linux test host and the persisted records make every successful and failed stage traceable.
