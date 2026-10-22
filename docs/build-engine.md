# Build Engine

The Build Engine turns a persisted deployment request into a local container image for the exact source commit recorded on that request. GitHub webhook handling remains short-lived: it validates the event and persists a queued deployment, while a background worker performs checkout and image building.

## Workflow and architecture

1. A GitHub push webhook creates a deployment using the exact commit SHA from the event. A manual API request either supplies an exact SHA or asks the GitHub API for the repository's current default branch and its HEAD commit.
2. The API persists the resolved full commit SHA in the deployment before it is queued. An omitted SHA is never stored as a moving branch reference.
3. The worker atomically claims a queued deployment and changes its status to `building`.
4. It loads the registered application's clone URL and validates the full commit SHA.
5. The local builder creates a unique temporary workspace, clones the repository, fetches the recorded SHA, checks it out detached, and verifies `git rev-parse HEAD`.
6. It validates the root-level `Dockerfile` and repository context, then runs `docker build`.
7. It inspects the resulting local image and records the image reference, Docker image ID, bounded build log, start/finish times, and final status on the deployment.

For an omitted SHA, the resolver obtains `default_branch` from GitHub repository metadata and requests the latest commit on that branch. It does not assume `main` or `master`, and it does not use the application's webhook branch setting. The resolved SHA is immutable for that request even if the branch advances before the worker builds it. If repository metadata, default branch, or branch HEAD cannot be retrieved, request creation fails without queuing a deployment.

Successful builds transition to `built`; failed builds transition to `failed` with an error message. A later runtime stage can consume the `built` deployment and its image identity. There is no runtime deployment or remote registry integration yet.

The application worker coordinates the deployment queue, registered-application lookup, timeout, and result persistence through interfaces. PostgreSQL provides the atomic queue claim and durable result fields. The infrastructure builder owns Git, temporary workspace, Docker CLI, image-reference construction, and output capture. A PostgreSQL advisory lock and active-build check limit the service to one claimed build across server processes; row locking also prevents duplicate claims.

The worker polls immediately at startup and then every two seconds when the queue is empty. An empty queue is represented as an expected result and does not produce an error log. The claim query uses a locked `Find` and row count so GORM does not log its `ErrRecordNotFound` result for this normal case. Genuine claim/database failures are logged with operation context and retried with exponential backoff from two seconds up to 30 seconds. The worker remains active until its context is canceled; shutdown cancels active Git/Docker client commands and waits for the worker to persist the outcome.

Build logs are capped at 64 KiB and stored in `deployments.build_log`. They are not included in the unauthenticated HTTP API response because build output can contain repository-controlled or otherwise sensitive text. Operators with database access can inspect them with:

```sql
SELECT id, status, image_reference, image_digest, error_message, build_log
FROM deployments
WHERE id = 1;
```

## Repository and image requirements

- The registered clone URL must be usable by the server's Git installation.
- To enqueue builds from GitHub push events, register the application's numeric `github_repository_id` through application creation and configure `GITHUB_WEBHOOK_SECRET` as documented for the webhook integration.
- Every queued deployment must contain a full 40- or 64-character Git commit SHA. Manual requests may omit it to resolve the default branch HEAD before queueing; malformed explicit SHA values fail without triggering a fallback.
- The repository must contain a regular file named `Dockerfile` at its root. Nested Dockerfiles are not supported.
- The build context is the checked-out repository root. Multi-stage Dockerfiles are supported by Docker.
- Symlinks and non-regular filesystem entries in the checked-out context are rejected.
- Image tags use `serve-service/app-<application-id>-<sanitized-name>:sha-<full-commit-sha>`. The application ID prevents different registered applications with the same name from colliding.
- The recorded `image_digest` is the local Docker image ID (`sha256:...`). It is not a pushed registry manifest digest. The tag is deterministic but mutable; the image ID is the stronger local content identity.

## Local development and operation

Install Git and the Docker CLI, and make both available on `PATH`. Start a local Docker Engine that the CLI can reach. The implementation does not pin minimum Git or Docker versions; a Docker version supporting the standard `docker build` and `docker image inspect` commands is required.

Apply database migrations and start the service using the existing project targets:

```sh
make migrate-up
make run
```

Create a deployment for the latest commit on the GitHub default branch:

```sh
curl -X POST http://localhost:8000/api/v1/applications/1/deployments \
  -H 'Content-Type: application/json' \
  -d '{}'
```

The `201` response contains the resolved `commit_sha`, which is also persisted before the worker claims the request. To select a specific commit instead, supply its full SHA:

```sh
curl -X POST http://localhost:8000/api/v1/applications/1/deployments \
  -H 'Content-Type: application/json' \
  -d '{"commit_sha":"0123456789abcdef0123456789abcdef01234567"}'
```

The GitHub REST API must be able to read repository metadata and commits. Public repositories work without credentials subject to GitHub's unauthenticated rate limit. For private repositories or higher rate limits, set optional `GITHUB_TOKEN` to a least-privilege token with repository metadata and contents read access. Authentication failures, rate limits, missing repositories, absent default branches, and unavailable branch heads fail the API request with `502 commit_resolution_failed`; no deployment is queued.

In the web UI, register or select an application, leave the commit field empty to deploy the default branch's latest commit or enter a full SHA for a specific revision, then open deployment details to inspect the resolved SHA, status, image identity, timing, failure details, and build output. The page polls the deployment and log endpoints every 4 seconds while it is queued or building.

The API also supports:

- `GET /api/v1/applications?limit=20&offset=0` for registered repositories.
- `GET /api/v1/deployments/<deployment-id>/logs` for bounded persisted build output.
- `GET /api/v1/deployments/<deployment-id>` for status, image reference, image ID, and timestamps.

The default worker timeout is 20 minutes. The service allows one active build globally. Docker build CPU and memory limits are not configured by this implementation.

Run the Go tests from `src`:

```sh
go test ./...
```

The local builder tests use controlled fake Git and Docker executables; the default test suite does not require a running Docker daemon. PostgreSQL queue-claim integration coverage is opt-in and exercises the actual locking, empty-queue, claim, and result-persistence behavior. Set `RUN_DATABASE_INTEGRATION_TESTS=1` and run `go test ./internal/platform/database -run '^TestPostgresBuildQueueClaiming$' -count=1 -v` from `src`; the configured test database must have no other queued or building deployments. It skips if the queue is not idle.

## Credentials and security limitations

Builds are intended only for repositories trusted by the operator. Dockerfile instructions execute arbitrary commands. Building with the host Docker Engine is not a strong isolation boundary: a malicious build can consume local resources and may exploit the daemon, kernel, or Docker build implementation. Do not offer this worker to untrusted or multi-tenant users.

The builder does not add platform tokens, application secrets, host environment variables, or build arguments to `docker build`. The Git and Docker CLI subprocesses receive a small environment allowlist needed for local tool and credential configuration; Docker build instructions do not inherit those subprocess environment variables. Git access to private repositories depends on credentials already configured for the service account (for example, its Git credential helper or SSH agent); `GITHUB_TOKEN` is used only for GitHub API resolution and is not forwarded to Git or Docker. Private base-image access depends on the local Docker CLI's existing configuration. Credentials embedded in clone URLs are not included in build output or worker logs.

Each build uses a new mode-0700 directory under the operating system's temporary directory. Cleanup is attempted on success, failure, timeout, and cancellation; a cleanup error is appended to diagnostics and causes an otherwise successful build to be recorded as failed. A process crash can leave a temporary directory behind. The queue marks a deployment still in `building` after 30 minutes as failed; it is not automatically retried. Operators should remove stale `serve-service-build-*` directories and create a new deployment to retry.

The 20-minute context timeout is propagated to Git and Docker CLI subprocesses, so the client commands are canceled when the deadline expires. Cancellation of a Docker client request is best effort and does not establish a hard resource boundary for work already accepted by the daemon. No CPU, memory, disk, or network resource limits are enforced. The database lock coordinates processes using the same database, but does not constrain work that may continue inside Docker after a client cancellation.

Build output is repository-controlled data. It is bounded before persistence and is not part of the deployment JSON response, but the logs endpoint returns it to callers. The API has no authentication, so only run it in a trusted operator environment and treat build output as potentially sensitive. Image tags can be reassigned; use the recorded Docker image ID when identifying the exact local image.

Before accepting builds from untrusted users, move builds to a hardened isolation boundary with explicit CPU, memory, disk, network, and concurrency quotas; authenticate and authorize API access; review credential separation and retention; and provide a recovery strategy for interrupted builds.

## Troubleshooting

- **Docker Engine unavailable:** run `docker info` as the same account that runs the service. Confirm the configured local Docker context is reachable.
- **Repository access denied:** verify the service account's Git credential helper or SSH agent. Do not place long-lived credentials in a repository URL.
- **Requested commit unavailable:** confirm the commit belongs to the configured repository and is fetchable by the service account. Builds do not fall back to a branch.
- **Dockerfile missing or unsupported layout:** place a regular `Dockerfile` at the repository root; nested paths and symlinks in the build context are rejected.
- **Build timed out:** inspect the persisted bounded log and simplify or optimize the Dockerfile. The timeout is fixed at 20 minutes; resource controls are not enforced.
- **Insufficient local resources:** inspect Docker Engine disk, memory, and CPU availability. This worker does not impose per-build resource limits.
- **Image inspection or persistence failure:** verify Docker can inspect the local tag and that the database is reachable and migrated. A build is not reported as successful until its result is persisted.
