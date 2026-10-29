# M12 — Resource Limits and Runtime Isolation

## A. Metadata

- **Status:** Planned
- **Dependencies:** M04, M05, M07
- **Result:** Runtime instances and builds have explicit, tested resource ceilings and restrictive container defaults suitable for controlled single-host use.

## B. Goal

Bound the host impact and privileges of workloads while distinguishing runtime container restrictions from build-time execution risks.

## C. Current problem and context

The current build worker has a timeout and bounded logs but no CPU, memory, disk, or network quotas. M05 proposes restricted runtime containers, but resource defaults and enforcement are not yet specified. Docker containers reduce accidental exposure but do not make hostile Dockerfiles safe.

## D. Scope

**In scope:** configurable hard limits, safe defaults, runtime capabilities/filesystem/network controls, build concurrency and resource policy, orphan identification, Linux verification.

**Out of scope:** claiming a container is a VM, untrusted public build service, cgroups management outside supported Docker controls, multi-host isolation, and arbitrary user-selected limits.

## E. Functional requirements

- **FR-01:** Define operator-configurable maximum CPU, memory, PID count, open files where supported, writable layer size where supported, build duration, and concurrent jobs.
- **FR-02:** Enforce safe defaults and hard upper bounds; application input cannot raise them.
- **FR-03:** Runtime containers run non-root where the image supports it, with no privileged mode, no host network, no host filesystem mount by default, all unnecessary capabilities dropped, and a read-only root filesystem where compatible.
- **FR-04:** Any writable paths are explicit, bounded, and deployment-owned; no Docker socket is mounted.
- **FR-05:** Build-time and runtime settings are separate. Build execution remains trusted-repository-only unless a stronger isolation mechanism is reviewed and tested.
- **FR-06:** Resource exhaustion results in a bounded failure reason and cleanup attempt without terminating unrelated platform services.
- **FR-07:** On startup, identify only labeled Serve Service containers and report/clean orphaned resources according to M07/M14 policy.

## F. Architecture and design

Implement resource settings in Docker/runtime adapters and central validated configuration. Use Docker Engine features available on the supported Linux version; startup must verify required capabilities and fail clearly rather than silently omit a configured limit. Select one supported Linux distribution/version for acceptance and record kernel/cgroup prerequisites.

Use these initial runtime defaults unless the supported host cannot enforce them: 1 CPU, 512 MiB memory, 256 PIDs, 64 open files per process where supported, and one active build with the existing 20-minute timeout. Keep the writable-layer ceiling operator-configurable and enable it only on a storage driver that proves enforcement; never claim it when the daemon ignores the option. The operator may lower limits but cannot configure an unlimited value through application input.

For builds, enforce concurrency and timeout immediately. Select a Docker build configuration that demonstrably limits build CPU/memory on the supported host; if the host Docker Engine cannot enforce those limits, do not claim build isolation or accept untrusted repositories. A missing resource control is a documented acceptance blocker for broadening beyond trusted builds. Do not mount the Docker socket into build containers or application containers.

## G. Data model and state transitions

Persist the effective policy version/values on a deployment only when needed to reproduce or explain runtime behavior. Do not duplicate every daemon-reported statistic as authoritative state. Runtime lifecycle remains `deploying -> healthy|failed`; resource termination is a runtime failure reason, not a new deployment status. Migrations should add only policy reference/version metadata if required.

## H. API and external interfaces

Resource policy is operator configuration, not public per-application input. Expose sanitized effective limits in authorized operator diagnostics only. Reject unsupported policy at startup with a clear configuration error; do not accept caller-selected host capabilities, mounts, network modes, or daemon options.

## I. Security requirements

- **SEC-01:** Container creation explicitly sets privilege, capability, user, mount, network, and resource controls.
- **SEC-02:** No arbitrary Docker flags or host paths are accepted from application configuration.
- **SEC-03:** Build and runtime resource budgets are independent and both documented.
- **SEC-04:** Only labeled platform resources are enumerated or removed.
- **SEC-05:** State residual risks for kernel/daemon vulnerabilities and host-level denial of service.

## J. Failure scenarios

Test unsupported cgroups, daemon ignoring/denying a limit, memory/CPU/PID exhaustion, full disk, runaway build, unavailable Docker, cleanup failure, orphaned containers, incompatible non-root image, and host restart. The service must not claim enforcement when daemon inspection does not confirm the requested settings.

## K. Testing strategy

- Unit tests for policy bounds and runtime request construction.
- Fake Docker CLI/API tests for explicit settings and unsafe-option rejection.
- Linux Docker integration test inspects actual container configuration and induces bounded resource failures.
- Build test verifies concurrency limit and timeout without executing untrusted source.
- PostgreSQL test confirms effective policy reference if persisted.

## L. Acceptance criteria

- [ ] **AC-01:** Runtime inspection confirms configured memory, CPU, PID, user, privilege, and mount policy.
- [ ] **AC-02:** Caller input cannot enable privileged mode, host networking, arbitrary bind mounts, or socket access.
- [ ] **AC-03:** Unsupported host enforcement fails startup or operation explicitly.
- [ ] **AC-04:** Resource exhaustion fails only the affected deployment and produces a bounded diagnostic.
- [ ] **AC-05:** Docs distinguish build risk from runtime isolation and retain trusted-source restriction.

## M. Documentation and operations

Document supported Linux/Docker/cgroup prerequisites, policy defaults and limits, resource monitoring, incompatible images, and residual host risks.

## N. Completion definition

Complete M12 when the supported Linux host demonstrably enforces the declared runtime limits and automated tests prevent unsafe overrides.
