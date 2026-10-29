# M15 — Operational Reliability and Backup/Restore

## A. Metadata

- **Status:** Planned
- **Dependencies:** M08, M09, M13, M14
- **Result:** An operator can back up and restore platform metadata and required configuration on the same or replacement single host, with a demonstrated restore test.

## B. Goal

Define a recoverable operating procedure for database, encryption key, application metadata, deployment history, and local build/runtime artifacts.

## C. Current problem and context

PostgreSQL is the source of truth for application/deployment records, but database restore alone cannot recreate local Docker images or secret decryption keys. A backup command without a successful restore test does not provide a recovery capability.

## D. Scope

**In scope:** backup consistency, protected storage, database/key/config backup, image/artifact relationship, restore on clean host, migration upgrade order, startup ordering, recovery objectives.

**Out of scope:** high availability, point-in-time recovery service, automated cross-region replication, and commercial disaster recovery guarantees.

## E. Functional requirements

- **FR-01:** Define a scheduled PostgreSQL logical backup procedure using available PostgreSQL tooling and a consistent snapshot.
- **FR-02:** Protect backups with host permissions and encryption at rest; do not write credentials in command history or logs.
- **FR-03:** Back up external encryption key material from M08 through a separate protected procedure; database backups do not contain the key.
- **FR-04:** Explicitly define which Docker images are rebuildable from pinned commit versus retained locally; document that database restore alone may leave a deployment without its image.
- **FR-05:** Restore to a disposable database and clean Linux host using a documented sequence; run schema migrations only according to a compatible-version policy.
- **FR-06:** Verify restored application records, deployment state, current route pointers, secret decryption, image availability/rebuild path, and HTTP response.
- **FR-07:** Set learning-project recovery objectives (target RPO/RTO) as measurable values and record observed restore time in each restore test.
- **FR-08:** Startup ordering waits for PostgreSQL migrations/availability; API/worker readiness fails explicitly when required dependencies are unavailable.

## F. Architecture and design

Use `pg_dump --format=custom` and `pg_restore` rather than adding a backup service. Create an encrypted backup every 24 hours at 02:00 UTC, retain the latest seven daily and four weekly backups, and encrypt each archive with `age` to an operator-managed recipient key; store archives outside the database host's ephemeral container filesystem. Keep both the age identity and M08 encryption key separate from database backups; losing the M08 key means stored secrets cannot be decrypted. Treat local image archives as optional; rebuild from the exact immutable commit under the trusted-source policy.

Set the initial objective to an RPO of at most 24 hours and an RTO of at most 4 hours for a single-host learning deployment. These are targets measured by the restore exercise, not availability guarantees; revise them only with recorded test evidence and an operational reason.

## G. Data model and state transitions

No application schema change is required unless M13/M14 add retention metadata. Backup manifests should identify timestamp, database version, schema migration version, key identifier (never key), and included artifact references. Restore state is operational, not a deployment status. Never mark a restored deployment healthy until runtime readiness is reverified.

## H. API and external interfaces

No public backup API. Use documented operator commands/system service configuration. Health/readiness endpoints follow M13. Restore operations require admin/host access and must not be exposed through the application API.

## I. Security requirements

- **SEC-01:** Backup artifacts and key copies are access-restricted and encrypted at rest.
- **SEC-02:** Backup/restore command output never includes credentials or decrypted secrets.
- **SEC-03:** Restore target is explicitly verified before destructive restore; production/local source databases are not overwritten accidentally.
- **SEC-04:** Keep encryption key backup separate from database data and document custody risk.

## J. Failure scenarios

Cover partial backup, corrupted archive, wrong database target, incompatible schema, missing encryption key, missing image, unavailable repository/commit, disk full, host loss, migration failure, and route/config restore mismatch. A failed restore is reported and never presented as successful.

## K. Testing strategy

- Script/unit tests for manifest parsing, validation, and target safeguards.
- PostgreSQL restore test into a disposable database on every planned release or scheduled cadence.
- Linux clean-host manual/automated test restores configuration/key and serves the fixture endpoint.
- Verify encrypted backup can be read only with intended credentials; no production external account required.

## L. Acceptance criteria

- [ ] **AC-01:** A clean disposable PostgreSQL instance restores from the documented backup and has expected migrations/data.
- [ ] **AC-02:** Restored secret ciphertext decrypts only when the separately recovered key is supplied.
- [ ] **AC-03:** The fixture app is reachable after image restore or documented exact-commit rebuild.
- [ ] **AC-04:** Restore test records observed RPO/RTO against declared targets.
- [ ] **AC-05:** Wrong-target and corrupted-backup tests fail safely without altering source data.

## M. Documentation and operations

Add backup schedule, protected storage, restore checklist, migration compatibility, encryption-key custody, image rebuild, and host replacement procedures. Record restore evidence and measured recovery times.

## N. Completion definition

Complete M15 only after a repeatable restore exercise passes on a clean environment and proves secrets, state, and an application endpoint are recoverable as documented.
