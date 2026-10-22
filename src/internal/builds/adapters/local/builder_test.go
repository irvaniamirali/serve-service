package local

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

const fixtureCommitSHA = "0123456789abcdef0123456789abcdef01234567"

func TestBuildChecksOutExactCommitAndCleansWorkspace(t *testing.T) {
	directory := t.TempDir()
	gitArguments := filepath.Join(directory, "git-arguments")
	workspacePath := filepath.Join(directory, "workspace-path")
	dockerArguments := filepath.Join(directory, "docker-arguments")

	writeExecutable(t, filepath.Join(directory, "git"), `#!/bin/sh
printf '%s\n' "$*" >> '`+gitArguments+`'
case "$1" in
  clone)
    printf '%s\n' "$5" > '`+workspacePath+`'
    mkdir -p "$5"
    printf 'FROM scratch\n' > "$5/Dockerfile"
    printf 'cloning %s\n' "$4" >&2
    ;;
  -C)
    printf '`+fixtureCommitSHA+`\n'
    ;;
esac
`)
	writeExecutable(t, filepath.Join(directory, "docker"), `#!/bin/sh
printf '%s\n' "$*" >> '`+dockerArguments+`'
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
  printf 'sha256:`+strings.Repeat("a", 64)+`\n'
else
  printf 'image build completed\n'
fi
`)
	t.Setenv("PATH", directory+":/bin")

	result, err := NewBuilder().Build(
		context.Background(),
		applications.Application{
			ID:            12,
			Name:          "My Test App",
			RepositoryURL: "https://user:fake-token@example.test/org/repo.git",
		},
		fixtureCommitSHA,
	)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if result.ImageReference != "serve-service/app-12-my-test-app:sha-"+fixtureCommitSHA {
		t.Fatalf("image reference = %q", result.ImageReference)
	}
	if result.ImageDigest != "sha256:"+strings.Repeat("a", 64) {
		t.Fatalf("image digest = %q", result.ImageDigest)
	}
	if strings.Contains(result.Log, "fake-token") {
		t.Fatalf("build log leaked clone credentials: %q", result.Log)
	}

	arguments, err := os.ReadFile(gitArguments)
	if err != nil {
		t.Fatalf("read Git arguments: %v", err)
	}
	if !strings.Contains(string(arguments), "fetch --depth 1 origin "+fixtureCommitSHA) ||
		!strings.Contains(string(arguments), "checkout --detach "+fixtureCommitSHA) {
		t.Fatalf("Git did not fetch and check out the requested revision: %s", arguments)
	}
	dockerArgs, err := os.ReadFile(dockerArguments)
	if err != nil {
		t.Fatalf("read Docker arguments: %v", err)
	}
	if !strings.Contains(string(dockerArgs), "build --file ") ||
		!strings.Contains(string(dockerArgs), "serve-service/app-12-my-test-app:sha-"+fixtureCommitSHA) {
		t.Fatalf("Docker build did not receive the root Dockerfile and deterministic tag: %s", dockerArgs)
	}

	workspace, err := os.ReadFile(workspacePath)
	if err != nil {
		t.Fatalf("read workspace path: %v", err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(workspace))); !os.IsNotExist(err) {
		t.Fatalf("temporary workspace still exists or stat failed unexpectedly: %v", err)
	}
}

func TestImageReferenceSanitizesRepositoryName(t *testing.T) {
	value := imageReference(2, " !!!A Name With / Spaces!!! ", fixtureCommitSHA)
	if value != "serve-service/app-2-a-name-with-spaces:sha-"+fixtureCommitSHA {
		t.Fatalf("imageReference() = %q", value)
	}
}

func TestValidateNoSymlinksRejectsRepositorySymlink(t *testing.T) {
	root := t.TempDir()
	if err := os.Symlink("outside", filepath.Join(root, "linked-file")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := validateNoSymlinks(context.Background(), root); err == nil || !strings.Contains(err.Error(), "unsupported symlink") {
		t.Fatalf("validateNoSymlinks() error = %v, want unsupported-symlink error", err)
	}
}

func TestBuildFailureCleansWorkspaceAfterFetchError(t *testing.T) {
	directory := t.TempDir()
	workspacePath := filepath.Join(directory, "workspace-path")
	writeFakeGit(t, directory, workspacePath, fixtureCommitSHA, true, "fetch", false)
	t.Setenv("PATH", directory+":/usr/bin:/bin")

	result, err := NewBuilder().Build(context.Background(), testApplication(), fixtureCommitSHA)
	if err == nil || !strings.Contains(err.Error(), "fetch requested commit") {
		t.Fatalf("Build() error = %v, want fetch failure", err)
	}
	if !strings.Contains(result.Log, "fetch failed") {
		t.Fatalf("build log = %q, want fetch diagnostics", result.Log)
	}
	assertWorkspaceRemoved(t, workspacePath)
}

func TestBuildRejectsCheckoutMismatchAndMissingDockerfile(t *testing.T) {
	tests := []struct {
		name             string
		actualCommit     string
		createDockerfile bool
		wantError        string
	}{
		{
			name:             "commit mismatch",
			actualCommit:     strings.Repeat("f", 40),
			createDockerfile: true,
			wantError:        "does not match requested commit",
		},
		{
			name:             "missing Dockerfile",
			actualCommit:     fixtureCommitSHA,
			createDockerfile: false,
			wantError:        "root-level Dockerfile",
		},
		{
			name:             "checkout failure",
			actualCommit:     fixtureCommitSHA,
			createDockerfile: true,
			wantError:        "check out requested commit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			workspacePath := filepath.Join(directory, "workspace-path")
			failureStep := ""
			if test.name == "checkout failure" {
				failureStep = "checkout"
			}
			writeFakeGit(t, directory, workspacePath, test.actualCommit, test.createDockerfile, failureStep, false)
			t.Setenv("PATH", directory+":/bin")

			_, err := NewBuilder().Build(context.Background(), testApplication(), fixtureCommitSHA)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("Build() error = %v, want %q", err, test.wantError)
			}
			assertWorkspaceRemoved(t, workspacePath)
		})
	}
}

func TestBuildTimeoutCancelsGitAndCleansWorkspace(t *testing.T) {
	directory := t.TempDir()
	workspacePath := filepath.Join(directory, "workspace-path")
	writeFakeGit(t, directory, workspacePath, fixtureCommitSHA, true, "", true)
	t.Setenv("PATH", directory+":/bin")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := NewBuilder().Build(ctx, testApplication(), fixtureCommitSHA)
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("Build() error = %v, want timeout", err)
	}
	if strings.Contains(result.Log, "fake-token") {
		t.Fatalf("build log leaked credentials: %q", result.Log)
	}
	assertWorkspaceRemoved(t, workspacePath)
}

func TestBuildReportsDockerUnavailable(t *testing.T) {
	directory := t.TempDir()
	workspacePath := filepath.Join(directory, "workspace-path")
	writeFakeGit(t, directory, workspacePath, fixtureCommitSHA, true, "", false)
	t.Setenv("PATH", directory+":/bin")

	_, err := NewBuilder().Build(context.Background(), testApplication(), fixtureCommitSHA)
	if err == nil || !strings.Contains(err.Error(), "docker") {
		t.Fatalf("Build() error = %v, want Docker-unavailable error", err)
	}
	assertWorkspaceRemoved(t, workspacePath)
}

func TestBuildReportsImageInspectionFailure(t *testing.T) {
	directory := t.TempDir()
	workspacePath := filepath.Join(directory, "workspace-path")
	writeFakeGit(t, directory, workspacePath, fixtureCommitSHA, true, "", false)
	writeExecutable(t, filepath.Join(directory, "docker"), `#!/bin/sh
if [ "$1" = "image" ] && [ "$2" = "inspect" ]; then
  printf 'image inspect failed\n' >&2
  exit 1
fi
printf 'image build completed\n'
`)
	t.Setenv("PATH", directory+":/bin")

	result, err := NewBuilder().Build(context.Background(), testApplication(), fixtureCommitSHA)
	if err == nil || !strings.Contains(err.Error(), "verify built image") {
		t.Fatalf("Build() error = %v, want image-inspection failure", err)
	}
	if !strings.Contains(result.Log, "image inspect failed") {
		t.Fatalf("build log = %q, want inspection diagnostics", result.Log)
	}
	assertWorkspaceRemoved(t, workspacePath)
}

func testApplication() applications.Application {
	return applications.Application{
		ID:            12,
		Name:          "Test App",
		RepositoryURL: "https://github.com/example/app.git",
	}
}

func writeFakeGit(
	t *testing.T,
	directory string,
	workspacePath string,
	actualCommit string,
	createDockerfile bool,
	failureStep string,
	slowClone bool,
) {
	t.Helper()
	dockerfileCreation := ""
	if createDockerfile {
		dockerfileCreation = `printf 'FROM scratch\n' > "$5/Dockerfile"`
	}
	failure := ""
	if failureStep != "" {
		failure = `if [ "$1" = "` + failureStep + `" ]; then
	  printf '` + failureStep + ` failed\n' >&2
	  exit 1
	fi`
	}
	slow := ""
	if slowClone {
		slow = "while :; do :; done"
	}
	script := `#!/bin/sh
	case "$1" in
	  clone)
	    printf '%s\n' "$5" > '` + workspacePath + `'
	    mkdir -p "$5"
	    ` + dockerfileCreation + `
	    ` + slow + `
	    ;;
	  -C)
	    printf '` + actualCommit + `\n'
	    ;;
	esac
	` + failure + `
	`
	writeExecutable(t, filepath.Join(directory, "git"), script)
}

func assertWorkspaceRemoved(t *testing.T, workspacePathFile string) {
	t.Helper()
	workspace, err := os.ReadFile(workspacePathFile)
	if err != nil {
		t.Fatalf("read workspace path: %v", err)
	}
	if _, err := os.Stat(strings.TrimSpace(string(workspace))); !os.IsNotExist(err) {
		t.Fatalf("temporary workspace still exists or stat failed unexpectedly: %v", err)
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o700); err != nil {
		t.Fatalf("write executable fixture: %v", err)
	}
}
