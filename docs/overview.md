# Project Overview

Serve Service is a learning-driven project for building and operating a Git-to-deployment workflow on one Linux host. It is intended to teach practical Go, databases, Linux, networking, Git, Docker, build systems, security boundaries, and operational reliability.

## Current implementation

The Go API persists application registrations and deployment requests in PostgreSQL. GitHub webhook verification and a background worker that builds the exact recorded commit using the local Docker Engine are present. The frontend is a separate React/TypeScript/Vite development application. Application containers and HTTP routing are not implemented.

## Intended workflow

Register a repository and branch → verify a GitHub push → create an idempotent deployment request → check out the exact commit → build an image → start it on the host → route HTTP traffic.

Each arrow after image build is planned work. Current integration status and dependencies are in the [roadmap](roadmap.md).

## Constraints

- Start with one Linux host; do not introduce Kubernetes or a multi-host scheduler without a demonstrated need.
- Treat repository content and Dockerfile instructions as untrusted.
- The current host Docker Engine build is acceptable only for operator-trusted repositories; it is not strong isolation.
- Test adverse cases and preserve explicit failure state.
- Keep each milestone independently verifiable and document its design trade-offs.

See the [architecture](architecture.md), [M00 project definition](milestones/M00-project-definition.md), and [M04 security boundary](milestones/M04-build-engine.md).
