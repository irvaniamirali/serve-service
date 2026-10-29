# M08 — Runtime Configuration and Secrets

## A. Metadata

- **Status:** Planned
- **Dependencies:** M05, M11
- **Result:** An authorized operator can manage versioned application runtime configuration without putting secret values in source, images, build arguments, API responses, or ordinary logs.

## B. Goal

Provide a minimal, single-host configuration mechanism for application containers, with explicit secret handling and safe deployment snapshots.

## C. Current problem and context

M05 defines an explicit runtime environment allowlist but no complete storage, update, access, or rotation contract. M04 deliberately does not pass application configuration into builds. Configuration and secret APIs must not be added before M11 protects them.

## D. Scope

**In scope:** non-secret and secret runtime values, versioning, validation, redaction, per-application association, rotation/removal semantics, and restore implications.

**Out of scope:** build-time secrets, arbitrary mounts, secret injection into Docker build, external Vault/KMS integration, and multi-tenant secret delegation.

## E. Functional requirements

- **FR-01:** Configuration is split into non-sensitive values and secrets; key names follow a documented environment-variable grammar and value/total-size limits.
- **FR-02:** Each edit creates a new immutable configuration revision. A deployment references the revision used to start its runtime instance.
- **FR-03:** A changed runtime configuration takes effect only on a new deployment; it never mutates an already running container's environment in place.
- **FR-04:** API reads return non-secret keys and secret-presence metadata only, never secret values.
- **FR-05:** Secret set, replace, and delete operations require an authenticated authorized operator and are audited without recording values.
- **FR-06:** On a single host, encrypt secret values at rest with an authenticated encryption scheme; the key is supplied as a root/service-account-protected file or environment source outside PostgreSQL and is not stored in the database.
- **FR-07:** At container creation, resolve only the referenced configuration revision and pass approved values through the runtime adapter; do not place secrets in image layers, Docker build args, command-line arguments, logs, or diagnostics.
- **FR-08:** Key loss is an explicit unrecoverable condition for encrypted secrets; document restore/key-backup requirements.

## F. Architecture and design

The application configuration use case owns key validation and revision creation. The PostgreSQL adapter stores encrypted secret ciphertext and non-secret values separately. The runtime adapter receives a short-lived in-memory map after authorization and must avoid serializing it into logs. Do not write plaintext temporary files unless the selected Docker API makes that unavoidable; if unavoidable, use mode `0600`, a private directory, bounded lifetime, cleanup, and a threat note.

Use AES-256-GCM with a random 96-bit nonce for each encryption and bind application ID, configuration revision, and key name as associated data. Store the nonce with the ciphertext and reject authentication-tag failures. Supply exactly 32 bytes of key material from a root/service-account-only file; do not auto-generate a key that may be silently lost on restart. Do not add a remote secret manager unless the deployment expands beyond a trusted single host or key custody requirements justify it.

## G. Data model and state transitions

Add an application-owned configuration revision entity, with a monotonic revision number, created timestamp, and non-secret values; secret values are encrypted independently and keyed by application/revision/name. Unique constraints cover `(application_id, revision)` and `(application_id, revision, key)`. A deployment references one revision; the active app pointer identifies the latest desired revision. Retain prior revisions while referenced by an active or retained deployment.

Configuration revision lifecycle: `draft -> current -> superseded`; deletion is allowed only when no active deployment references it, or after a replacement deployment has completed and retention permits deletion. A migration must not introduce plaintext secret columns.

## H. API and external interfaces

Proposed API additions under `/api/v1/applications/{id}/config`:

| Method | Purpose | Response rule |
|---|---|---|
| `GET` | Read current non-secret values and secret key names/presence | Never returns values |
| `PUT` | Create a new config revision from validated values and an expected current revision (`If-Match`) | Returns revision ID, not secret values; stale revision returns `409` |
| `DELETE` | Remove named secret keys in a new revision | Idempotent only for absent keys |

All routes require M11 authorization. A submitted secret is accepted only over TLS at the point of exposure; local HTTP is for loopback development only. Return `400` for invalid names/limits, `404` for missing application, `409` for stale revision, and generic `500` for storage/encryption failures.

## I. Security requirements

- **SEC-01:** Encrypt secret values at rest with authenticated encryption; reject startup if required key material is missing or malformed.
- **SEC-02:** Never expose values through list/get responses, error text, logs, metrics, build output, or deployment metadata.
- **SEC-03:** Prevent plaintext from reaching build subprocesses and images.
- **SEC-04:** Do not use shared host environment configuration as a cross-application secret store.
- **SEC-05:** Audit actor, key name, operation, and revision, never value.
- **SEC-06:** Document that host root and the service account can access runtime secrets; this is not protection against a compromised host.

## J. Failure scenarios

Cover missing/rotated encryption key, authentication-tag failure, database outage, stale revision update, oversized payload, invalid key, secret deletion while referenced, container start failure, and restore without the external key. Fail closed: do not start a container with silently missing or partially decrypted required configuration.

## K. Testing strategy

- Unit tests for key validation, size bounds, encryption/decryption, and redaction.
- HTTP authorization/response tests with secret canaries.
- PostgreSQL integration tests for revisions, uniqueness, ciphertext persistence, and references.
- Runtime adapter test proving secrets are not logged or passed to image build.
- Manual Linux test for key file permissions and restart behavior; no live external secret manager.

## L. Acceptance criteria

- [ ] **AC-01:** A database dump contains no plaintext submitted secret canary.
- [ ] **AC-02:** Application/config responses, errors, and logs contain no secret canary.
- [ ] **AC-03:** A deployment uses exactly the immutable config revision it records.
- [ ] **AC-04:** Rotation affects a newly created deployment, not a running container.
- [ ] **AC-05:** Missing/decryption-invalid key fails closed with a safe diagnostic.
- [ ] **AC-06:** Restore documentation specifies both database and external key recovery.

## M. Documentation and operations

Document configuration API, environment key source/permissions, rotation, deletion, backup/restore, size limits, and host-root residual risk in the development, architecture, and security docs.

## N. Completion definition

Complete M08 after all data, API, log-redaction, authorization, and restart tests pass with no secret values in persistent plaintext or observable responses.
