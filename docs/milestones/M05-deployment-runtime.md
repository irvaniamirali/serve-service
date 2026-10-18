# M05 — Deployment Runtime

## Goal

Run a successfully built application container and make it reachable through an HTTP endpoint.

## Initial Runtime Model

The first implementation targets a single Linux host. Docker Compose may be used to operate Platform's own services, while deployed application containers are managed through a clearly defined runtime component. The exact boundary must be documented before implementation.

## Planned Features

- [ ] Define the application port configuration contract.
- [ ] Create application containers from built images.
- [ ] Assign stable application and deployment identifiers.
- [ ] Configure environment variables for the runtime.
- [ ] Apply container restart and resource policies.
- [ ] Configure reverse-proxy routing.
- [ ] Route HTTP requests to the intended application.
- [ ] Track container creation, startup, exit, and removal.
- [ ] Detect startup failures and record diagnostics.
- [ ] Prevent hostname and routing conflicts.
- [ ] Stop obsolete deployment containers safely.
- [ ] Add integration tests for runtime operations.
- [ ] Document host setup, networking, and troubleshooting.

## Acceptance Criteria

- A successfully built image can be started as an application container.
- The application is reachable through its assigned HTTP hostname or documented local test URL.
- Multiple applications can be routed independently.
- Container startup failure is reflected in deployment status.
- Stopping one application does not interrupt unrelated applications.
- Runtime resources can be inspected and cleaned up.
- Runtime operations do not block API request handling indefinitely.
- The runtime security model and host requirements are documented.

## Dependencies

M04.

## Out of Scope

- Kubernetes.
- Multi-host scheduling.
- Automatic horizontal scaling.
- Fully automated HTTPS provisioning, unless it is selected as a small, well-defined extension after the HTTP workflow works.

## End-to-End Acceptance Scenario

1. Register a supported GitHub repository.
2. Push a change to the configured branch.
3. Platform receives and verifies the webhook.
4. Platform records a deployment for the exact commit.
5. The build engine creates an image.
6. The runtime starts the application.
7. A request to the assigned HTTP endpoint returns the expected application response.
8. Build and deployment outcomes can be inspected.

## Exit Condition

M05 is complete when the initial Git-to-HTTP workflow works end to end and can be reproduced using documented steps.
