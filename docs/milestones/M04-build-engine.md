# M04 — Build Engine

## Status

In progress. A PostgreSQL queue worker and local Git/Docker builder exist. Unit tests use fake executables, but the PostgreSQL locking test and a real Linux/Docker build acceptance run were not performed during this work. CPU, memory, disk, and network quotas are not enforced.

## Goal and scope

Build a Docker image from the exact immutable commit referenced by a queued deployment, persist its result, and clearly limit the trust model to operator-controlled repositories.

The current implementation uses one active build at a time across processes sharing the database, a 20-minute worker timeout, a mode-0700 temporary workspace, a root-level Dockerfile, and bounded build logs. It uses the host Docker Engine and is not an isolation boundary.

## Implementation stages

### M04.1 — Durable queue claim and worker lifecycle

- **Goal/scope:** Claim one queued build safely and record lifecycle state in PostgreSQL.
- **Tasks:** Verify atomic claiming, single-host concurrency policy, startup/shutdown cancellation, retry/backoff for claim failures, stale-build handling, and result persistence.
- **Design decision:** Keep one active build on the initial single host; do not add a multi-host scheduler or parallel build pool. The current advisory lock plus row lock is intentionally stronger than needed for a single process but coordinates multiple service processes using the same database.
- **Security/failures:** Test database outage, duplicate claim, worker cancellation, stale `building` row, persistence failure, and a process interruption. Do not automatically retry ambiguous builds.
- **Tests:** Unit worker tests and opt-in PostgreSQL queue/locking/result integration tests.
- **Acceptance:** Each queued deployment is claimed once, transitions to a recorded result, and interrupted work is diagnosable without being silently marked successful.
- **Dependencies:** M03 and M02.2.
- **Documentation:** Queue policy, recovery limits, and database setup in `docs/build-engine.md`.

### M04.2 — Exact checkout and bounded workspace

- **Goal/scope:** Provide a clean source tree at the requested commit without shell interpolation.
- **Tasks:** Validate repository URL and full SHA; clone/fetch/check out in a fresh temporary directory; verify `HEAD`; require a regular root `Dockerfile`; reject unsupported symlinks/special files; sanitize credentials from diagnostics; clean workspaces on all exits.
- **Design decision:** Use argument-based Git subprocess calls and the installed Git CLI. Alternatives such as an in-process Git library add dependency and behavior surface without removing the need to execute Dockerfiles.
- **Security/failures:** Repository content is hostile input; exercise malformed URLs, absent/inaccessible commits, command cancellation, symlink entries, cleanup failure, and output containing credentials.
- **Tests:** Unit tests with fake Git commands for exact SHA, mismatch, missing Dockerfile, cancellation, output bounds, and cleanup; no live remote repository required.
- **Acceptance:** The builder never builds a different commit and never uses a shared writable workspace.
- **Dependencies:** M04.1.
- **Documentation:** Supported repository layout and checkout guarantees in `docs/build-engine.md`.

### M04.3 — Docker build and result recording

- **Goal/scope:** Turn the checked-out context into a locally identifiable image.
- **Tasks:** Run `docker build` with a deterministic commit tag; inspect the resulting image ID; persist image reference, image ID, timing, bounded logs, and sanitized failure reason.
- **Design decision:** The recorded ID is the local Docker image ID, not a pushed registry digest. Defer a registry until the single-host flow works.
- **Security/failures:** Do not forward platform tokens, application secrets, arbitrary host environment, or user-supplied build arguments. Test missing Docker, build failure, timeout, malformed image ID, excessive output, and result database failure.
- **Tests:** Fake Docker executable tests plus PostgreSQL result persistence. Ordinary tests must not require privileged Docker execution.
- **Acceptance:** A build is marked `built` only after a valid local image identity is inspected and the result is persisted; failures are marked failed with bounded diagnostics.
- **Dependencies:** M04.2.
- **Documentation:** Build inputs, environment allowlist, timeout, logs, image identity, and failure behavior in `docs/build-engine.md`.

### M04.4 — Linux operational acceptance

- **Goal/scope:** Verify a trusted sample repository on the target Linux host.
- **Tasks:** Run migrations, start the worker, build a pinned test commit, inspect the image and deployment row, stop/restart the service, and verify cleanup and documented recovery.
- **Design decision:** Accept a deliberately narrow Dockerfile-only workflow. Do not claim general buildpack support or hard isolation.
- **Security/failures:** Use a disposable, operator-controlled repository and host. Confirm Docker socket access is restricted to the service account; record that cancellation and daemon-side work are best effort.
- **Tests:** End-to-end Linux build with a real Docker Engine, alongside all unit and PostgreSQL integration tests.
- **Acceptance:** The exact sample commit produces a runnable local image and all failure/cleanup behavior is documented. Public/untrusted repositories remain unsupported.
- **Dependencies:** M04.3.
- **Documentation:** Linux prerequisites and measured limitations in the README and build guide.

## Security boundary

Dockerfile instructions execute arbitrary commands through the host Docker Engine. Containers do not guarantee isolation from the daemon, kernel, host resource exhaustion, or implementation vulnerabilities. The current worker imposes a timeout and log cap, but no CPU, memory, disk, or network quotas. Restrict it to trusted repositories and operators. Do not expose it as a public multi-tenant build service.

## Exit condition

M04 is complete only after M04.1–M04.4 pass and the limitation above is accepted. Runtime and HTTP routing are M05.
