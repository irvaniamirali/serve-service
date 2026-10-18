# M00 — Project Definition

## Goal

Define the initial problem, scope, requirements, architecture boundaries, and engineering workflow before implementation begins.

## Problem Statement

Deploying an application from a Git repository often requires developers to configure builds, runtime environments, networking, and repeatable deployments themselves. Platform aims to automate this workflow for a deliberately limited initial environment.

## Initial User Journey

1. An operator registers a GitHub repository and a deployment branch.
2. A push to the configured branch triggers a deployment request.
3. Platform checks out the exact commit and builds a container image.
4. Platform starts the application from that image.
5. The application becomes reachable through an HTTP endpoint.
6. The operator can inspect the deployment status and diagnose failures.

This is the intended workflow, not a claim that these capabilities already exist.

## Initial Scope

- One Linux host.
- Docker-based application builds and runtime.
- GitHub as the initial Git provider.
- Dockerfile-based builds.
- HTTP routing to deployed applications.
- Persistent application and deployment metadata.
- A documented, testable deployment workflow.

## Out of Scope

- Kubernetes and multi-host scheduling.
- Multi-region or high-availability operation.
- Billing and commercial account management.
- Supporting every language or buildpack.
- Unrestricted execution of untrusted workloads.
- Advanced autoscaling.

## Deliverables

- [ ] `docs/overview.md`
- [ ] `docs/requirements.md`
- [ ] `docs/tech-stack.md`
- [ ] `docs/architecture.md`
- [ ] Initial threat model and security assumptions
- [ ] Documentation and coding conventions
- [ ] Initial architecture decisions recorded as ADRs

## Acceptance Criteria

- The problem and initial user journey are clearly documented.
- Functional and non-functional requirements are distinguishable.
- Each proposed component has a defined responsibility.
- Major technology choices have a rationale and alternatives.
- Security boundaries and known limitations are documented.
- The first end-to-end acceptance scenario is agreed upon.

## Exit Condition

M00 is complete when the project scope and initial technical direction are documented well enough to begin M01 without relying on undocumented assumptions.

## Notes

Technology choices remain proposals until recorded in `docs/technical-stack.md` and, where significant, in an ADR.
