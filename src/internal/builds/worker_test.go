package builds

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/irvaniamirali/serve-service/internal/applications"
	"github.com/irvaniamirali/serve-service/internal/deployments"
)

const testCommitSHA = "0123456789abcdef0123456789abcdef01234567"

func TestRunBuildPersistsImageIdentityForRequestedCommit(t *testing.T) {
	queue := &buildQueueFake{}
	apps := &applicationRepositoryFake{application: applications.Application{
		ID:            17,
		Name:          "Example App",
		RepositoryURL: "https://github.com/example/app.git",
	}}
	imageBuilder := &builderFake{result: BuildResult{
		ImageReference: "serve-service/app-17-example-app:sha-" + testCommitSHA,
		ImageDigest:    "sha256:" + strings.Repeat("a", 64),
		Log:            "build completed",
	}}
	worker := newTestWorker(queue, apps, imageBuilder)

	err := worker.runBuild(context.Background(), deployments.Deployment{
		ID:            9,
		ApplicationID: 17,
		Status:        deployments.StatusBuilding,
		CommitSHA:     stringPointer(testCommitSHA),
	})
	if err != nil {
		t.Fatalf("runBuild() error = %v", err)
	}
	if imageBuilder.commitSHA != testCommitSHA {
		t.Fatalf("builder commit SHA = %q, want %q", imageBuilder.commitSHA, testCommitSHA)
	}
	if queue.recordedStatus != deployments.StatusBuilt {
		t.Fatalf("recorded status = %q, want %q", queue.recordedStatus, deployments.StatusBuilt)
	}
	if queue.imageReference == nil || *queue.imageReference != imageBuilder.result.ImageReference {
		t.Fatalf("recorded image reference = %v, want %q", queue.imageReference, imageBuilder.result.ImageReference)
	}
	if queue.imageDigest == nil || *queue.imageDigest != imageBuilder.result.ImageDigest {
		t.Fatalf("recorded image digest = %v, want %q", queue.imageDigest, imageBuilder.result.ImageDigest)
	}
	if queue.buildLog != "build completed" || queue.failureMessage != nil {
		t.Fatalf("recorded log/error = %q/%v", queue.buildLog, queue.failureMessage)
	}
}

func TestRunBuildRejectsMissingCommitAndPersistsFailure(t *testing.T) {
	queue := &buildQueueFake{}
	apps := &applicationRepositoryFake{}
	imageBuilder := &builderFake{}
	worker := newTestWorker(queue, apps, imageBuilder)

	if err := worker.runBuild(context.Background(), deployments.Deployment{ID: 9, Status: deployments.StatusBuilding}); err != nil {
		t.Fatalf("runBuild() error = %v", err)
	}
	if imageBuilder.calls != 0 {
		t.Fatalf("builder calls = %d, want 0", imageBuilder.calls)
	}
	if queue.recordedStatus != deployments.StatusFailed || queue.failureMessage == nil {
		t.Fatalf("missing SHA was not persisted as a failed build: status=%q error=%v", queue.recordedStatus, queue.failureMessage)
	}
	if !strings.Contains(*queue.failureMessage, "full commit SHA") {
		t.Fatalf("failure message = %q, want missing-SHA context", *queue.failureMessage)
	}
}

func TestRunBuildPersistsBuilderFailureAndDiagnostics(t *testing.T) {
	queue := &buildQueueFake{}
	apps := &applicationRepositoryFake{application: applications.Application{
		ID:            17,
		RepositoryURL: "https://github.com/example/app.git",
	}}
	imageBuilder := &builderFake{
		result: BuildResult{Log: "Docker Engine is unavailable"},
		err:    errors.New("docker build image failed"),
	}
	worker := newTestWorker(queue, apps, imageBuilder)

	if err := worker.runBuild(context.Background(), deployments.Deployment{
		ID:            9,
		ApplicationID: 17,
		Status:        deployments.StatusBuilding,
		CommitSHA:     stringPointer(testCommitSHA),
	}); err != nil {
		t.Fatalf("runBuild() error = %v", err)
	}
	if queue.recordedStatus != deployments.StatusFailed {
		t.Fatalf("recorded status = %q, want %q", queue.recordedStatus, deployments.StatusFailed)
	}
	if queue.failureMessage == nil || !strings.Contains(*queue.failureMessage, "Docker image build or inspection failed") {
		t.Fatalf("recorded failure = %v", queue.failureMessage)
	}
	if !strings.Contains(queue.buildLog, "Docker Engine is unavailable") ||
		!strings.Contains(queue.buildLog, "docker build image failed") {
		t.Fatalf("recorded log = %q", queue.buildLog)
	}
	if queue.imageReference != nil || queue.imageDigest != nil {
		t.Fatalf("failed build recorded image identity: %v / %v", queue.imageReference, queue.imageDigest)
	}
}

func TestRunBuildPropagatesCancellationAndPersistsFailure(t *testing.T) {
	queue := &buildQueueFake{}
	apps := &applicationRepositoryFake{application: applications.Application{
		ID:            17,
		RepositoryURL: "https://github.com/example/app.git",
	}}
	imageBuilder := &builderFake{err: context.Canceled}
	worker := newTestWorker(queue, apps, imageBuilder)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := worker.runBuild(ctx, deployments.Deployment{
		ID:            9,
		ApplicationID: 17,
		Status:        deployments.StatusBuilding,
		CommitSHA:     stringPointer(testCommitSHA),
	}); err != nil {
		t.Fatalf("runBuild() error = %v", err)
	}
	if !imageBuilder.sawCanceledContext {
		t.Fatal("builder did not receive the canceled context")
	}
	if queue.recordedStatus != deployments.StatusFailed {
		t.Fatalf("recorded status = %q, want %q", queue.recordedStatus, deployments.StatusFailed)
	}
}

func TestRunBuildReturnsPersistenceFailure(t *testing.T) {
	queue := &buildQueueFake{recordError: errors.New("database unavailable")}
	apps := &applicationRepositoryFake{application: applications.Application{
		ID:            17,
		RepositoryURL: "https://github.com/example/app.git",
	}}
	imageBuilder := &builderFake{result: BuildResult{ImageReference: "image:tag", ImageDigest: "sha256:digest"}}
	worker := newTestWorker(queue, apps, imageBuilder)

	err := worker.runBuild(context.Background(), deployments.Deployment{
		ID:            9,
		ApplicationID: 17,
		Status:        deployments.StatusBuilding,
		CommitSHA:     stringPointer(testCommitSHA),
	})
	if !errors.Is(err, queue.recordError) {
		t.Fatalf("runBuild() error = %v, want persistence error", err)
	}
}

func TestRunBuildRejectsInvalidDeploymentState(t *testing.T) {
	queue := &buildQueueFake{}
	apps := &applicationRepositoryFake{}
	imageBuilder := &builderFake{}
	worker := newTestWorker(queue, apps, imageBuilder)

	err := worker.runBuild(context.Background(), deployments.Deployment{
		ID:     9,
		Status: deployments.StatusBuilt,
	})
	if err == nil || !strings.Contains(err.Error(), "not in building state") {
		t.Fatalf("runBuild() error = %v, want invalid-state error", err)
	}
	if imageBuilder.calls != 0 || queue.recordedStatus != "" {
		t.Fatal("invalid deployment state was processed or persisted")
	}
}

func TestRunContinuesAfterEmptyQueueAndProcessesClaimedDeployment(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := &buildQueueFake{
		claimResults: []claimResult{
			{err: deployments.ErrNoQueuedBuild},
			{deployment: deployments.Deployment{
				ID:            9,
				ApplicationID: 17,
				Status:        deployments.StatusBuilding,
				CommitSHA:     stringPointer(testCommitSHA),
			}},
		},
	}
	queue.afterRecord = cancel
	apps := &applicationRepositoryFake{application: applications.Application{
		ID:            17,
		RepositoryURL: "https://github.com/example/app.git",
	}}
	imageBuilder := &builderFake{result: BuildResult{
		ImageReference: "example/app:" + testCommitSHA,
		ImageDigest:    "sha256:" + strings.Repeat("a", 64),
	}}
	var logs bytes.Buffer
	worker := newTestWorker(queue, apps, imageBuilder)
	worker.pollInterval = time.Millisecond
	worker.logger = slog.New(slog.NewTextHandler(&logs, nil))

	if err := worker.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if queue.claimCalls != 2 {
		t.Fatalf("claim calls = %d, want empty poll followed by claimed deployment", queue.claimCalls)
	}
	if imageBuilder.calls != 1 || queue.recordedStatus != deployments.StatusBuilt {
		t.Fatalf("worker did not complete the claimed build: calls=%d status=%q", imageBuilder.calls, queue.recordedStatus)
	}
	if strings.Contains(logs.String(), "level=ERROR") {
		t.Fatalf("empty queue produced an error log: %s", logs.String())
	}
}

func TestRunCancellationStopsEmptyQueuePollingPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	firstClaim := make(chan struct{})
	queue := &buildQueueFake{
		onClaim: func(call int) {
			if call == 1 {
				close(firstClaim)
			}
		},
	}
	worker := newTestWorker(queue, &applicationRepositoryFake{}, &builderFake{})
	worker.pollInterval = time.Second
	done := make(chan error, 1)
	go func() {
		done <- worker.Run(ctx)
	}()

	select {
	case <-firstClaim:
	case <-time.After(time.Second):
		t.Fatal("worker did not perform its initial queue poll")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("worker did not stop promptly after cancellation")
	}
}

func TestRunBacksOffAndLogsClaimFailures(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var logs bytes.Buffer
	const expectedFailure = "database connection lost"
	queue := &buildQueueFake{
		claimResults: []claimResult{
			{err: errors.New(expectedFailure)},
			{err: errors.New(expectedFailure)},
			{err: errors.New(expectedFailure)},
		},
		onClaim: func(call int) {
			if call == 3 {
				cancel()
			}
		},
	}
	worker := newTestWorker(queue, &applicationRepositoryFake{}, &builderFake{})
	worker.pollInterval = 10 * time.Millisecond
	worker.maxRetryWait = 20 * time.Millisecond
	worker.logger = slog.New(slog.NewTextHandler(&logs, nil))

	started := time.Now()
	if err := worker.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed < 20*time.Millisecond {
		t.Fatalf("claim retries completed in %v, want exponential backoff", elapsed)
	}
	if queue.claimCalls != 3 {
		t.Fatalf("claim calls = %d, want 3 failed attempts", queue.claimCalls)
	}
	if got := logs.String(); !strings.Contains(got, "claim queued deployment build") ||
		!strings.Contains(got, expectedFailure) {
		t.Fatalf("claim failure was not logged with context: %s", got)
	}
}

func newTestWorker(queue *buildQueueFake, apps *applicationRepositoryFake, imageBuilder *builderFake) *Worker {
	return NewWorker(
		queue,
		apps,
		imageBuilder,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
}

type buildQueueFake struct {
	recordedStatus string
	imageReference *string
	imageDigest    *string
	buildLog       string
	failureMessage *string
	recordError    error
	claimResults   []claimResult
	claimCalls     int
	onClaim        func(int)
	afterRecord    func()
}

type claimResult struct {
	deployment deployments.Deployment
	err        error
}

func (queue *buildQueueFake) ClaimNextQueued(context.Context) (deployments.Deployment, error) {
	queue.claimCalls++
	if queue.onClaim != nil {
		queue.onClaim(queue.claimCalls)
	}
	if len(queue.claimResults) > 0 {
		result := queue.claimResults[0]
		queue.claimResults = queue.claimResults[1:]
		return result.deployment, result.err
	}
	return deployments.Deployment{}, deployments.ErrNoQueuedBuild
}

func (queue *buildQueueFake) RecordBuildResult(
	_ context.Context,
	_ int64,
	succeeded bool,
	imageReference *string,
	imageDigest *string,
	buildLog string,
	failureMessage *string,
) (deployments.Deployment, error) {
	if queue.recordError != nil {
		return deployments.Deployment{}, queue.recordError
	}
	if succeeded {
		queue.recordedStatus = deployments.StatusBuilt
	} else {
		queue.recordedStatus = deployments.StatusFailed
	}
	queue.imageReference = imageReference
	queue.imageDigest = imageDigest
	queue.buildLog = buildLog
	queue.failureMessage = failureMessage
	if queue.afterRecord != nil {
		queue.afterRecord()
	}
	return deployments.Deployment{Status: queue.recordedStatus}, nil
}

type applicationRepositoryFake struct {
	application applications.Application
	err         error
}

func (repository *applicationRepositoryFake) CreateApplication(
	context.Context,
	applications.Application,
) (applications.Application, error) {
	return applications.Application{}, errors.New("not used")
}

func (repository *applicationRepositoryFake) GetApplicationByID(context.Context, int64) (applications.Application, error) {
	return repository.application, repository.err
}

func (repository *applicationRepositoryFake) GetApplicationByGitHubRepositoryID(
	context.Context,
	int64,
) (applications.Application, error) {
	return applications.Application{}, errors.New("not used")
}

func (repository *applicationRepositoryFake) ListApplications(
	context.Context,
	int,
	int,
) ([]applications.Application, error) {
	return nil, errors.New("not used")
}

type builderFake struct {
	result             BuildResult
	err                error
	commitSHA          string
	calls              int
	sawCanceledContext bool
}

func (builder *builderFake) Build(ctx context.Context, _ applications.Application, commitSHA string) (BuildResult, error) {
	builder.calls++
	builder.commitSHA = commitSHA
	select {
	case <-ctx.Done():
		builder.sawCanceledContext = true
	default:
		builder.sawCanceledContext = ctx.Err() != nil
	}
	return builder.result, builder.err
}

func stringPointer(value string) *string {
	return &value
}
