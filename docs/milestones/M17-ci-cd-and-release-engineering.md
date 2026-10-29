# M17 — CI/CD and Release Engineering

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06, M07, M13, M15, M16
- **Result:** Changes to Serve Service are checked consistently and produce versioned, migration-aware release artifacts with a documented single-host upgrade and rollback procedure.

## B. Goal

Make the platform itself reproducible to build, test, package, upgrade, and recover without relying on production secrets or live GitHub/Docker services in ordinary CI.

## C. Current problem and context

The existing GitHub Actions workflow runs Go formatting, tests, and `go vet`. It does not run frontend tests/build, PostgreSQL integration tests, security scans, linter, release artifact builds, schema compatibility checks, or an upgrade procedure.

## D. Scope

**In scope:** CI quality gates, PostgreSQL service-backed tests, frontend build/tests, selected lint/security checks, versioning, reproducible artifact metadata, migration compatibility, release and rollback checklist.

**Out of scope:** automatic production deployment, Kubernetes releases, signing infrastructure that requires a third-party service, and broad fleet management.

## E. Functional requirements

- **FR-01:** CI runs Go formatting, `go test ./...`, `go vet ./...`, frontend tests/build, and PostgreSQL integration tests against an ephemeral service where deterministic.
- **FR-02:** CI runs a selected Go linter and dependency/secret checks with pinned action/tool versions; findings have documented severity handling.
- **FR-03:** CI does not require production credentials, live GitHub API calls, or an available Docker daemon for its default verification path.
- **FR-04:** Release version derives from an immutable tag/commit and is reported by the binary/API.
- **FR-05:** Release artifacts include checksums, source revision, Go/frontend dependency lockfiles, migration bundle, and build metadata.
- **FR-06:** Database changes follow expand/contract compatibility where a release can overlap old/new binaries; unsafe migration ordering is blocked by review/checks.
- **FR-07:** Operator upgrade procedure backs up first, applies migrations in documented order, restarts services, verifies readiness, and provides a tested rollback/recovery path.
- **FR-08:** Dependency updates are reviewed and tested; generated dependency changes are not silently applied during release.

## F. Architecture and design

Retain GitHub Actions and current Makefile targets. Add a PostgreSQL service container for integration tests, with a clean database per run. Use pinned `golangci-lint` for Go linting and pinned `govulncheck` for Go vulnerability analysis; use the existing npm lockfile for frontend checks and dependency audit. Pin tool/action versions and review upgrades as source changes. Build Go binaries with embedded migration files as the current module requires. Publish artifacts to the repository's existing release mechanism only after protected tag creation; do not introduce a registry for application workload images.

Database rollback is not assumed to be safe merely because Goose has down migrations. Prefer forward-compatible expand/contract migrations; platform binary rollback must be tested against the post-migration schema.

## G. Data model and state transitions

No product schema is added solely for CI. Migration checks must detect duplicate versions, missing down/up documentation, and incompatible destructive changes. Release state is represented by immutable version/tag and artifact checksums, not a mutable `latest` tag alone.

## H. API and external interfaces

Expose a non-sensitive version/build-info response or equivalent CLI output containing version, commit, and build timestamp. Do not include secrets, host details, or full environment in build info. Release artifacts are consumed by the documented single-host installer.

## I. Security requirements

- **SEC-01:** CI permissions are least privilege; pull-request workflows do not receive release credentials.
- **SEC-02:** Pin third-party actions by immutable version/commit according to repository policy.
- **SEC-03:** Scan for accidental secrets without printing discovered secret values.
- **SEC-04:** Release artifacts are tied to source revision and checksummed; signing is added only if key custody can be documented.
- **SEC-05:** Release/upgrade instructions keep database backups and M08 key material protected.

## J. Failure scenarios

Cover failed integration DB startup, migration incompatibility, linter/test failure, missing frontend lockfile, release tag mismatch, artifact checksum mismatch, upgrade interruption, migration failure, and binary rollback against a newer schema. CI must fail visibly rather than skip required checks.

## K. Testing strategy

- CI workflow validation and local Make target tests.
- Unit/frontend tests, Go vet/linter, PostgreSQL migration/integration tests.
- Build artifact reproducibility comparison where practical (document any nondeterministic metadata).
- Fresh install and old-version-to-new-version migration tests.
- Upgrade/rollback rehearsal on disposable Linux/PostgreSQL environment; no production credentials.

## L. Acceptance criteria

- [ ] **AC-01:** Pull request CI runs the documented Go, frontend, PostgreSQL, and security quality gates without secrets.
- [ ] **AC-02:** A tagged release produces a versioned artifact and checksum tied to the tag's source commit.
- [ ] **AC-03:** Fresh install and supported upgrade tests pass on clean fixtures.
- [ ] **AC-04:** Binary rollback after migration is either demonstrated safe or explicitly disallowed with a tested forward-recovery procedure.
- [ ] **AC-05:** Release checklist includes backup, migration, readiness verification, and failure recovery.

## M. Documentation and operations

Update CI, versioning, migration, artifact, dependency-update, upgrade/rollback, and release checklist docs. Keep current CI claims in the README accurate.

## N. Completion definition

Complete M17 after a clean release candidate passes all gates and a disposable Linux host is upgraded and recovered using only the documented procedure.
