# Roadmap

## Project Goal

project-name is a learning-driven Git-to-deployment system. It connects a GitHub repository to an automated workflow that builds an application, runs it on a Linux host, and exposes it over HTTP.

The primary goal is to learn DevOps and system architecture by building and operating a realistic system. This is not initially a commercial product, but its features and engineering practices should be real.

## Guiding Principles

- Start with one Linux host and Docker Compose.
- Deliver a working, testable result in every milestone.
- Document architectural decisions and their trade-offs.
- Treat application source code and build steps as untrusted input.
- Test failure paths as well as successful paths.
- Defer Kubernetes and multi-host orchestration until there is a clear need.

## Milestones

| ID | Milestone | Main outcome | Depends on | Status |
|---|---|---|---|---|
| M00 | [Project Definition](docs/milestones/M00-project-definition.md) | Scope, requirements, initial architecture, and engineering conventions are documented. | — | Planned |
| M01 | [Development Foundation](docs/milestones/M01-development-foundation.md) | A reproducible local development environment and tested project skeleton. | M00 | Planned |
| M02 | [Core API](docs/milestones/M02-core-api.md) | Applications and deployments can be managed through a persistent API. | M01 | Planned |
| M03 | [GitHub Integration](docs/milestones/M03-github-integration.md) | Verified GitHub push events can safely request deployments. | M02 | Planned |
| M04 | [Build Engine](docs/milestones/M04-build-engine.md) | A specific repository commit can be built into a container image. | M03 | Planned |
| M05 | [Deployment Runtime](docs/milestones/M05-deployment-runtime.md) | Built applications can run on the host and be reached over HTTP. | M04 | Planned |

## Milestone Policy

A milestone is complete only when its acceptance criteria pass, relevant tests run successfully, and required documentation is updated. Writing code alone does not complete a milestone.

The roadmap is intentionally limited to the first six milestones. Later work—such as health checks, logs, HTTPS, secrets management, rollback, observability, and recovery—will be planned after the initial end-to-end workflow has been validated.

## Current Focus

Start with M00. Do not treat proposed technologies or architectural choices as final until they have been reviewed and recorded in the relevant documentation or ADR.
