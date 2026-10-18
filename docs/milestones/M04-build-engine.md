# M04 — Build Engine

## Goal

Build a container image from the exact source commit referenced by a deployment request.

## Initial Build Strategy

Start with repositories that contain a valid Dockerfile. Additional build strategies can be considered after this workflow is reliable.

## Planned Features

- [ ] Fetch the configured repository.
- [ ] Check out the exact commit SHA recorded for the deployment.
- [ ] Validate the build context and Dockerfile location.
- [ ] Build an image using the selected container build technology.
- [ ] Tag images with an immutable commit-based identifier.
- [ ] Capture build output, exit status, and timing.
- [ ] Apply build timeouts.
- [ ] Define resource limits and cleanup behavior.
- [ ] Record the resulting image reference.
- [ ] Handle clone, checkout, and build failures.
- [ ] Add automated tests for build orchestration.
- [ ] Document supported repository layouts and build requirements.

## Security Requirements

Build steps execute project-controlled instructions and must be treated as untrusted.

- Do not expose platform secrets to build processes by default.
- Do not mount the host filesystem or container runtime socket into builds without a documented security review.
- Define how build credentials are supplied, if needed.
- Apply resource and time limits.
- Clean up temporary workspaces and build resources.
- Document the isolation guarantees and their limitations.

## Acceptance Criteria

- A valid supported repository produces a runnable image.
- The image corresponds to the requested commit.
- A failed build produces a failed deployment state and useful diagnostics.
- Build logs can be associated with a deployment.
- Build timeouts and cleanup behavior are tested.
- Temporary source files and resources are handled safely.
- The chosen build isolation model is documented.

## Dependencies

M03.

## Out of Scope

- Supporting every language or buildpack.
- Distributed build workers.
- Advanced build caching.
- Automatic vulnerability policy enforcement.

## Exit Condition

M04 is complete when project can reliably turn a recorded source commit into a traceable container image.
