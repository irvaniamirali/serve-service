# M01 — Development Foundation

## Goal

Create a reproducible development environment and a maintainable project skeleton with basic quality checks.

## Scope

- Establish the repository structure.
- Set up the backend application skeletons.
- Configure local dependencies and services.
- Establish configuration, logging, health checks, and automated quality checks.

## Planned Tasks

- [ ] Create the initial repository layout.
- [ ] Add a Go backend skeleton.
- [ ] Add a React and TypeScript frontend skeleton.
- [ ] Add Docker Compose for required local dependencies.
- [ ] Configure PostgreSQL and a database migration approach, if confirmed by M00.
- [ ] Implement configuration loading and validation.
- [ ] Add structured application logging.
- [ ] Add liveness and readiness endpoints.
- [ ] Add formatting, linting, and automated test commands.
- [ ] Add CI checks for proposed changes.
- [ ] Document setup, common commands, and troubleshooting.

## Deliverables

- A working local development environment.
- A documented repository layout.
- Basic backend and frontend entry points.
- Repeatable quality checks.
- A CI workflow.
- Updated development documentation.

## Acceptance Criteria

- A new contributor can follow the setup guide from a clean checkout.
- Required local services start using documented commands.
- Configuration errors are reported clearly.
- Health endpoints return the documented status.
- Automated checks run locally and in CI.
- No credentials, local environment files, or generated build artifacts are committed.

## Dependencies

M00.

## Exit Condition

M01 is complete when the project can be developed and tested reproducibly, and the skeleton is ready for core domain functionality.
