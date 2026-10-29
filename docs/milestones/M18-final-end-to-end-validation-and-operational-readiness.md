# M18 — Final End-to-End Validation and Operational Readiness

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06, M07, M08, M09, M10, M11, M12, M13, M14, M15, M16, M17
- **Result:** A clean Linux installation completes a documented trusted-repository Git-to-HTTP deployment and demonstrates failure handling, access control, cleanup, and restore.

## B. Goal

Validate the project as a coherent, usable single-host platform rather than adding another feature set.

## C. Current problem and context

Passing milestone-specific tests does not prove that a new operator can install, configure, secure, operate, diagnose, and recover the combined system. M18 verifies one complete lifecycle and records remaining limitations without claiming production-grade multi-tenant isolation or high availability.

## D. Scope

**In scope:** clean-host setup, pinned demo repository, end-to-end workflow, important failure injection, worker restart recovery, auth/secrets, backup/restore, cleanup, final acceptance matrix, operator handoff.

**Out of scope:** new product features, arbitrary public tenants, high availability, multi-host orchestration, automated TLS unless already accepted, and unresolved security risk remediation by documentation alone.

## E. Functional requirements

- **FR-01:** A clean supported Linux host can be provisioned from the published prerequisites without undocumented repository-local state.
- **FR-02:** The operator can register a pinned demo application, configure webhook delivery, and trigger a verified commit-specific deployment.
- **FR-03:** The deployed demo responds at the documented endpoint and deployment metadata identifies commit, image, config revision, runtime instance, and current status.
- **FR-04:** Injected invalid signature, duplicate delivery, build failure, startup/readiness failure, route failure, database interruption, and worker restart produce documented outcomes.
- **FR-05:** Failed replacement preserves the previous working endpoint; explicit rollback works for a retained eligible deployment.
- **FR-06:** Anonymous callers cannot access admin operations, logs, or configuration; secret canaries do not appear in outputs.
- **FR-07:** Resource limits, cleanup, backup, restore, and key recovery are demonstrated on disposable data.
- **FR-08:** Publish a goal-to-evidence acceptance matrix and list residual risks that prevent a production-grade claim.

## F. Architecture and design

Use the system components and state model delivered by M06–M17. The demonstration repository must be small, pinned to a full commit, contain no credentials, and use only documented runtime configuration. Tests that require GitHub use a configured test repository/webhook; deterministic CI tests continue to use local payloads and fakes.

No architecture change is made in this milestone unless a blocking inconsistency is discovered. Any such change is filed as an explicit bug fix with a regression test and corresponding documentation update.

## G. Data model and state transitions

No new schema is expected. Verify migrations from an empty database and upgrade from the prior supported release. Confirm deployment states, worker attempts, config revisions, current deployment pointers, route state, cleanup ownership, and backup manifest all agree after success, failure, restart, and restore.

## H. API and external interfaces

Exercise documented application, deployment, logs/config, authentication, rollback, readiness, and application endpoint interfaces. Record methods, routes, authentication requirement, status codes, and expected payload in the acceptance matrix. Do not introduce a new API solely for the test.

## I. Security requirements

- **SEC-01:** Perform only with a disposable host and trusted demonstration repository.
- **SEC-02:** Verify host Docker API exposure, container bindings/privileges, TLS or loopback admin access, and firewall assumptions.
- **SEC-03:** Verify secrets are excluded from Git, image layers, build arguments, logs, API responses, and backup manifests.
- **SEC-04:** Confirm the trusted-repository restriction and residual single-host risks are visible to operators.

## J. Failure scenarios

Run failure injection at webhook validation, duplicate handling, Git checkout/build, persistence, runtime start, readiness, proxy reload, worker crash, host restart, cleanup, and restore. For each, record expected state, observed state, operator action, and whether a previous healthy route remains available.

## K. Testing strategy

- Full Go unit/HTTP/vet/lint and frontend test/build suites.
- PostgreSQL integration/migration suite on disposable instances.
- Real Linux + Docker Engine + configured proxy end-to-end test.
- GitHub delivery test against an operator-controlled test repository; ordinary CI must not need GitHub credentials.
- Backup/restore test on a separate disposable database/host environment.
- Manual acceptance review using the published matrix and runbooks.

## L. Acceptance criteria

- [ ] **AC-01:** Clean-host install and setup succeed from documented instructions.
- [ ] **AC-02:** A verified push for the pinned commit builds and serves the expected HTTP response.
- [ ] **AC-03:** Duplicate delivery creates no second deployment; failures at every listed stage are diagnosable.
- [ ] **AC-04:** Failed replacement preserves the prior healthy endpoint and rollback restores a retained version.
- [ ] **AC-05:** Anonymous access and secret-canary checks pass.
- [ ] **AC-06:** Worker/host restart recovers or safely marks every in-progress operation according to M07.
- [ ] **AC-07:** Cleanup preserves current and rollback-required resources.
- [ ] **AC-08:** A separate restore exercise meets the documented learning-project recovery objectives.
- [ ] **AC-09:** Acceptance matrix links each project goal to a test or recorded manual evidence.
- [ ] **AC-10:** README lists known limitations and does not claim production-grade multi-tenant security or high availability.

## M. Documentation and operations

Publish a clean-install guide, demo procedure, failure/recovery runbook, final acceptance matrix, supported host/software versions, and known-limitations list. Update README and roadmap statuses only after evidence is recorded.

## N. Completion definition

Complete M18 only when every acceptance item has recorded evidence, the full workflow is repeatable on a clean supported Linux host, and remaining risks are explicit and accepted for controlled experimentation.
