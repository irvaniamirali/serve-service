# M03 — GitHub Integration

## Status

In progress. The repository has a GitHub API client, HMAC-SHA256 verification, push parsing, branch filtering, a webhook delivery table, and deployment creation. Remaining gaps include repository-name verification, atomic delivery/deployment persistence, recovery of deliveries left in `processing`, and PostgreSQL-backed duplicate-delivery tests. Do not treat the current webhook code as fully reliable yet.

## Goal and scope

Turn authenticated GitHub push deliveries into exactly traceable deployment requests without building synchronously or depending on live GitHub during ordinary tests.

The existing webhook endpoints are `/api/v1/webhooks/github` and `/api/v1/github/webhooks`. `GITHUB_WEBHOOK_SECRET` is a server-wide secret; the per-application `webhook_secret` column is not currently used. Manual deployment without a SHA resolves GitHub's repository default branch, while webhook deployments use the application's configured branch. Decide whether that difference is intentional before completion; the recommended alternative is to make the configured branch the single deployment target.

## Implementation stages

### M03.1 — Repository identity and registration

- **Goal/scope:** Bind a webhook to exactly one registered GitHub repository and branch.
- **Tasks:** Decide when `github_repository_id` is required; compare payload repository ID and canonical `full_name` to registration; reject invalid/non-positive IDs; document global-secret configuration and remove or use the currently unused per-application secret field in a schema-safe way.
- **Design decision:** Numeric GitHub repository ID is the stable lookup key; canonical owner/name is an additional mismatch check. Alternatives are URL-only matching (rename-sensitive) or storing GitHub API metadata (more schema and sync work).
- **Security/failures:** A valid signature does not make repository fields trustworthy; reject identity mismatches and never select an application from a branch name alone. Do not store or return secrets unnecessarily.
- **Tests:** Repository ID match/mismatch, owner/name mismatch, unknown repository, missing registration ID, duplicate ID conflict, and no-live-GitHub handler tests.
- **Acceptance:** Only an event matching the registered repository can reach branch filtering or deployment creation.
- **Dependencies:** M02.
- **Documentation:** `docs/development.md`, `docs/architecture.md`, API examples, and migration notes if schema changes.

### M03.2 — Webhook contract and branch/SHA validation

- **Goal/scope:** Accept only well-formed, signed push events for the configured deployment branch.
- **Tasks:** Preserve raw-body HMAC verification; validate event and delivery headers, JSON shape, repository identity, `refs/heads/...`, deletion markers, and full commit SHA; define precise 2xx/4xx/5xx responses; resolve the manual-versus-webhook branch policy.
- **Design decision:** Use the push payload's immutable `after` SHA for webhook deployments. Do not fetch a mutable branch tip during processing. Keep webhook processing short; building stays in the worker.
- **Security/failures:** Reject invalid signatures before writing a delivery record; bound request size; do not log secrets or raw untrusted payloads. Treat malformed payloads and ignored events distinctly.
- **Tests:** Unit tests for signature, payload, SHA, and branch cases; HTTP tests for valid push, wrong branch, deletion, irrelevant event, malformed JSON, missing headers, invalid signature, and error responses.
- **Acceptance:** Valid signed pushes create queued deployments only for the selected branch and exact event SHA; other events create none.
- **Dependencies:** M03.1.
- **Documentation:** Webhook setup/troubleshooting in `docs/development.md`; behavior in this milestone and `docs/architecture.md`.

### M03.3 — Atomic idempotency and recovery

- **Goal/scope:** Make GitHub retries safe and observable across process/database failures.
- **Tasks:** Make delivery registration, deployment insertion, and terminal delivery outcome one transaction; on unique delivery conflict, return the persisted outcome/deployment; define how any committed `processing` record is recovered; propagate persistence errors instead of returning success-shaped responses.
- **Design decision:** Prefer a single PostgreSQL transaction for database-only processing; do not hold it open across GitHub network calls. If lease-based processing is needed, add an explicit lease and retry rule rather than treating every `processing` row as a completed duplicate.
- **Security/failures:** Handle concurrent duplicate requests, transaction rollback, database outage, crash between steps, and delivery-ID collision safely. Keep bounded, non-secret failure reasons.
- **Tests:** PostgreSQL integration tests for concurrent duplicate delivery, retry after rollback, completed duplicate response, ignored event persistence, and worker/database failure; handler tests remain independent of live GitHub.
- **Acceptance:** One delivery creates at most one deployment; retries return a result that accurately reflects the committed processing outcome; no silent stuck delivery remains.
- **Dependencies:** M03.2 and M02 PostgreSQL integration setup.
- **Documentation:** Transaction/idempotency behavior, operational retry guidance, and updated API responses.

## Exit condition

M03 is complete when each stage passes, webhook outcome persistence is reliable under retries and process failure, and manual/webhook branch semantics are explicit. This milestone does not build images.
