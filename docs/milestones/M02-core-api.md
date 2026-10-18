# M02 — Core API

## Goal

Implement the persistent backend domain for applications and deployments before connecting external Git events or executing builds.

## Initial Domain Concepts

### Application

Represents a deployable application and its source configuration, including repository URL, configured branch, and application-specific settings.

### Deployment

Represents one attempt to deploy a specific source revision. It should retain a stable identifier, source commit when known, lifecycle status, timestamps, and diagnostic metadata.

## Planned Features

- [ ] Create and retrieve application records.
- [ ] Validate repository URLs and branch configuration.
- [ ] Create deployment requests through the API.
- [ ] Retrieve a deployment and its current status.
- [ ] List deployment history for an application.
- [ ] Persist records in the selected database.
- [ ] Add database migrations.
- [ ] Define allowed deployment state transitions.
- [ ] Use consistent API error responses.
- [ ] Add unit and integration tests.
- [ ] Document API endpoints and request examples.

## Initial Deployment States

The initial lifecycle may include:

- `queued`
- `building`
- `deploying`
- `healthy`
- `failed`
- `cancelled`

Only implement transitions that have defined behavior. Revise the state model as the architecture becomes clearer.

## Acceptance Criteria

- Application and deployment records can be created and retrieved.
- Records persist across backend restarts.
- Invalid requests return predictable errors.
- Illegal state transitions are rejected.
- Concurrent requests do not silently corrupt persistent state.
- Tests cover successful requests and important failure cases.
- API behavior is documented.

## Dependencies

M01.

## Out of Scope

- Actual GitHub webhook processing.
- Container image building.
- Starting application containers.
- A full user authentication and authorization system.

## Exit Condition

M02 is complete when the API can reliably represent applications and deployment attempts, ready for integration with GitHub events.
