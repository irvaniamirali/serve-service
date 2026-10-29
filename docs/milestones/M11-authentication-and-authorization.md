# M11 — Authentication and Authorization

## A. Metadata

- **Status:** Planned
- **Dependencies:** M06
- **Result:** The single-operator control API and frontend require authenticated access, with explicit authorization for configuration, deployment, logs, and administration.

## B. Goal

Protect an initially single-operator system before it exposes secret-management or administrative APIs, without creating an unnecessary multi-tenant identity service.

## C. Current problem and context

The current Fiber API has no authentication; application creation, deployments, and build-log access are unauthenticated. M06 can be tested locally under an operator-only boundary, but M08 secrets and external exposure require authentication and CSRF protection.

## D. Scope

**In scope:** one initial administrator, login/logout, session expiry, password/bootstrap handling, route authorization, frontend session behavior, audit actor identity.

**Out of scope:** public self-registration, teams, organization tenancy, social login, SSO, delegated roles, and identity-provider federation.

## E. Functional requirements

- **FR-01:** The first deployment provisions one administrator through `go run ./cmd/admin bootstrap` from `src/`. The command prompts for username and password without echoing the password, succeeds only when no administrator exists, and creates no public registration route.
- **FR-02:** Store password verifiers using Argon2id with a per-user cryptographically random 16-byte salt, 64 MiB memory, 3 iterations, parallelism 2, and a 32-byte output; encode parameters with the verifier and never store plaintext passwords.
- **FR-03:** Use an opaque random session token in an `HttpOnly`, `SameSite` cookie; use `Secure` in HTTPS deployments and a documented loopback-only development mode.
- **FR-04:** Sessions expire after a configured maximum age and can be explicitly revoked at logout/password change.
- **FR-05:** Mutating browser requests require CSRF protection and same-origin validation; API clients use the same explicit session contract.
- **FR-06:** Application/deployment/config/log/admin operations require an authenticated administrator; public application traffic remains unauthenticated.
- **FR-07:** Unauthorized requests return `401`; authenticated but forbidden requests return `403`; errors reveal no credential or session details.
- **FR-08:** Every security-sensitive mutation records the authenticated actor ID, operation, target, and timestamp without secrets.

## F. Architecture and design

Use an application-owned user/session model and Fiber middleware that populates a typed principal; authorization checks stay in handlers/use cases, not a global service locator. For a single local operator, use one admin role only. Use server-side sessions stored hashed in PostgreSQL so revocation and expiry work across process restart. Do not accept identity from reverse-proxy headers.

Bootstrap runs only once after the operator has configured the database and has host-shell access; it does not accept credentials as command-line arguments or log them. Rate-limit login to five failed attempts per normalized username and source IP per 15-minute window using a bounded in-process limiter; successful login clears that pair's failures. The limiter may reset on process restart and this limitation is documented. No external identity provider is required.

## G. Data model and state transitions

Add admin user and session records with unique normalized username, password hash, created/updated timestamps, session token hash, expiry, revoked timestamp, and indexes for active sessions. Store only a token hash. Session lifecycle is `active -> expired|revoked`; only active and unexpired sessions authenticate. Migration seeds no default password or user.

## H. API and external interfaces

Add `POST /api/v1/auth/login`, `POST /api/v1/auth/logout`, and `GET /api/v1/auth/session`. Login accepts username/password, sets session cookie, and returns a minimal principal; logout revokes server-side session and clears cookie. No password value is returned. Protect all existing admin routes and M08 configuration routes; preserve public M10 application host routing separately.

## I. Security requirements

- **SEC-01:** Session IDs are cryptographically random, stored only as hashes, and compared safely.
- **SEC-02:** Cookies use `HttpOnly`, `SameSite=Lax` or stricter, `Secure` outside loopback development, and bounded lifetime.
- **SEC-03:** CSRF defenses cover all state-changing cookie-authenticated requests.
- **SEC-04:** Passwords, cookies, CSRF tokens, and bootstrap credentials are redacted from logs.
- **SEC-05:** No default credentials, public signup, or secret-bearing error messages.

## J. Failure scenarios

Cover invalid credentials, locked/throttled attempts, database outage, expired/revoked session, cookie theft response through revocation, CSRF mismatch, missing secure transport, bootstrap command rerun, password rotation, and server restart. Authentication fails closed when session storage is unavailable.

## K. Testing strategy

- Unit tests for credential verification, session expiry/revocation, and authorization policy.
- Fiber HTTP tests for 401/403, cookies, CSRF, logout, and existing route protection.
- PostgreSQL integration tests for sessions across server restart and concurrent revocation.
- Frontend tests for login/session expiration and safe error handling.
- Manual Linux verification behind local HTTPS or loopback development settings; no external IdP.

## L. Acceptance criteria

- [ ] **AC-01:** All existing administrative API routes reject anonymous requests.
- [ ] **AC-02:** No user registration or default credential exists.
- [ ] **AC-03:** Session token database value is not the browser token.
- [ ] **AC-04:** Expired and revoked sessions fail; logout revokes the active session.
- [ ] **AC-05:** Cross-site state-changing request without a valid CSRF token is rejected.
- [ ] **AC-06:** Login/session secrets are absent from captured logs and error responses.

## M. Documentation and operations

Document first-admin bootstrap, password/session rotation, cookie/TLS requirements, reverse-proxy trust, recovery after database restore, and frontend login behavior.

## N. Completion definition

Complete M11 when unauthenticated and CSRF tests cover every admin route and the documented single-admin account can be provisioned and recovered without default credentials.
