# M03 — GitHub Integration

## Goal

Connect GitHub repository events to deployment requests in a verifiable and idempotent way.

## Planned Features

- [x] Define how a repository is registered with the platform.
- [x] Implement a webhook endpoint.
- [x] Verify webhook signatures using the configured secret.
- [x] Parse relevant push events.
- [x] Filter events by the configured branch.
- [x] Record the exact commit SHA associated with each request.
- [x] Handle duplicate delivery IDs idempotently.
- [x] Record webhook processing outcomes.
- [x] Ignore unrelated event types safely.
- [x] Provide a manual deployment trigger through the API.
- [x] Add tests for valid, invalid, duplicate, and irrelevant events.
- [x] Document webhook setup and troubleshooting.

## Repository registration

Applications can include a canonical GitHub repository identity by setting `github_repository_id` when creating the record. The repository URL and deployment branch remain the primary registration data. The webhook secret is configured in the environment as `GITHUB_WEBHOOK_SECRET` and is never serialized in API responses.

## Webhook configuration

Configure GitHub to send `push` events to one of the platform endpoints:

- `POST /api/v1/webhooks/github`
- `POST /api/v1/github/webhooks`

Use the payload format `application/json` and a secret matching `GITHUB_WEBHOOK_SECRET`. The server computes the SHA256 HMAC over the original raw request body and compares it to the `X-Hub-Signature-256` header using constant-time comparison.

## Branch and deployment validation

The webhook handler accepts pushes only when the event `ref` matches the configured deployment branch for the application, such as `refs/heads/main` matching an application branch of `main`. Branch deletion events and non-branch refs are ignored. The actual deployment request stores the immutable `after` SHA from the push payload instead of trusting the branch name alone.

## Delivery idempotency

Every webhook delivery is recorded with a unique `X-GitHub-Delivery` value in the database. If the same delivery is retried, the server returns a safe duplicate response instead of creating a second deployment. Processed, ignored, and failed outcomes are persisted in `github_webhook_deliveries` for investigation and safe retry decisions.

## Manual deployment trigger

The existing deployment endpoint remains the manual trigger path and accepts `commit_sha` for an explicit immutable revision. The same validation rules apply as for webhook-triggered deployments. Authentication remains outside the scope of this milestone unless the surrounding deployment environment provides it.

## Reliability Requirements

- Invalid signatures must be rejected.
- Duplicate deliveries must not create duplicate deployment requests.
- The system must not trust a branch name alone as an immutable source revision.
- Webhook handling must not perform long-running image builds synchronously.
- Failures must leave enough diagnostic information for investigation.

## Acceptance Criteria

- A valid push event for the configured branch creates a deployment request.
- Events for other branches do not trigger deployments.
- Invalid signatures are rejected.
- Duplicate webhook deliveries are handled safely.
- The requested commit SHA is persisted.
- Webhook processing failures are observable and documented.
- Automated tests cover expected and adverse cases.

## Dependencies

M02.

## Out of Scope

- Building container images.
- Running application containers.
- Supporting Git providers other than GitHub.

## Exit Condition

M03 is complete when GitHub changes reliably create traceable deployment requests without duplicate processing.
