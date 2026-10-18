# M03 — GitHub Integration

## Goal

Connect GitHub repository events to deployment requests in a verifiable and idempotent way.

## Planned Features

- [ ] Define how a repository is registered with platform.
- [ ] Implement a webhook endpoint.
- [ ] Verify webhook signatures using the configured secret.
- [ ] Parse relevant push events.
- [ ] Filter events by the configured branch.
- [ ] Record the exact commit SHA associated with each request.
- [ ] Handle duplicate delivery IDs idempotently.
- [ ] Record webhook processing outcomes.
- [ ] Ignore unrelated event types safely.
- [ ] Provide a manual deployment trigger through the API.
- [ ] Add tests for valid, invalid, duplicate, and irrelevant events.
- [ ] Document webhook setup and troubleshooting.

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
