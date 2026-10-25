package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/irvaniamirali/serve-service/internal/domains/applications"
	"github.com/irvaniamirali/serve-service/internal/domains/builds/usecase"
	"github.com/irvaniamirali/serve-service/internal/github"
)

const (
	maxCapturedLogSize = 64 * 1024
)

var credentialURLPattern = regexp.MustCompile(`(?i)(https?|ssh|git)://[^/\s@]+@`)

// Builder clones exact Git revisions and builds images with the local Docker Engine.
type Builder struct{}

// NewBuilder creates the local Git and Docker Engine build adapter.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build checks out and verifies the recorded commit before building from its root Dockerfile.
func (builder *Builder) Build(
	ctx context.Context,
	application applications.Application,
	commitSHA string,
) (result usecase.BuildResult, returnedErr error) {
	if commitSHA != strings.TrimSpace(commitSHA) {
		return result, fmt.Errorf("requested commit is not a full Git SHA")
	}
	repositoryURL := strings.TrimSpace(application.RepositoryURL)
	if !github.IsDeployableCommitSHA(commitSHA) {
		return result, fmt.Errorf("requested commit is not a full Git SHA")
	}
	if application.ID <= 0 {
		return result, fmt.Errorf("application identity is missing")
	}
	if err := validateCloneURL(repositoryURL); err != nil {
		return result, err
	}

	workspace, err := os.MkdirTemp("", "serve-service-build-*")
	if err != nil {
		return result, fmt.Errorf("create temporary build workspace: %w", err)
	}
	if err := os.Chmod(workspace, 0o700); err != nil {
		if cleanupErr := os.RemoveAll(workspace); cleanupErr != nil {
			return result, fmt.Errorf("restrict temporary build workspace: %v; cleanup failed: %w", err, cleanupErr)
		}
		return result, fmt.Errorf("restrict temporary build workspace: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(workspace); err != nil {
			result.Log = appendBuildLog(result.Log, "workspace cleanup failed")
			if returnedErr == nil {
				returnedErr = fmt.Errorf("remove temporary build workspace: %w", err)
			}
		}
	}()

	output := &limitedBuffer{}
	run := func(directory, program, operation string, args ...string) error {
		command := exec.CommandContext(ctx, program, args...)
		command.Dir = directory
		command.Env = commandEnvironment()
		command.Stdout = output
		command.Stderr = output
		if err := command.Run(); err != nil {
			return commandFailure(operation, ctx, err)
		}
		return nil
	}

	if err := run(filepath.Dir(workspace), "git", "clone repository", "clone", "--no-checkout", "--", repositoryURL, workspace); err != nil {
		return resultWithLog(result, output, repositoryURL), err
	}
	if err := run(workspace, "git", "fetch requested commit", "fetch", "--depth", "1", "origin", commitSHA); err != nil {
		return resultWithLog(result, output, repositoryURL), err
	}
	if err := run(workspace, "git", "check out requested commit", "checkout", "--detach", commitSHA); err != nil {
		return resultWithLog(result, output, repositoryURL), err
	}

	var checkedOut bytes.Buffer
	verify := exec.CommandContext(ctx, "git", "-C", workspace, "rev-parse", "HEAD")
	verify.Dir = workspace
	verify.Env = commandEnvironment()
	verify.Stdout = &checkedOut
	verify.Stderr = output
	if err := verify.Run(); err != nil {
		return resultWithLog(result, output, repositoryURL), commandFailure("verify checked-out commit", ctx, err)
	}
	if actual := strings.TrimSpace(checkedOut.String()); !strings.EqualFold(actual, commitSHA) {
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("checked-out commit does not match requested commit")
	}

	dockerfile := filepath.Join(workspace, "Dockerfile")
	info, err := os.Lstat(dockerfile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return resultWithLog(result, output, repositoryURL), fmt.Errorf("repository must contain a root-level Dockerfile")
		}
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("inspect root-level Dockerfile: %w", err)
	}
	if !info.Mode().IsRegular() {
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("root-level Dockerfile must be a regular file, not a symlink or special file")
	}
	if err := validateNoSymlinks(ctx, workspace); err != nil {
		return resultWithLog(result, output, repositoryURL), err
	}
	if file, err := os.Open(dockerfile); err != nil {
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("open root-level Dockerfile: %w", err)
	} else if err := file.Close(); err != nil {
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("close root-level Dockerfile: %w", err)
	}

	imageReference := imageReference(application.ID, application.Name, commitSHA)
	if err := run(workspace, "docker", "build image with local Docker Engine", "build", "--file", dockerfile, "--tag", imageReference, workspace); err != nil {
		return resultWithLog(result, output, repositoryURL), err
	}

	var digest bytes.Buffer
	inspect := exec.CommandContext(
		ctx,
		"docker",
		"image",
		"inspect",
		"--format={{.Id}}",
		imageReference,
	)
	inspect.Dir = workspace
	inspect.Env = commandEnvironment()
	inspect.Stdout = &digest
	inspect.Stderr = output
	if err := inspect.Run(); err != nil {
		return resultWithLog(result, output, repositoryURL), commandFailure("verify built image", ctx, err)
	}
	imageDigest := strings.TrimSpace(digest.String())
	if !strings.HasPrefix(imageDigest, "sha256:") || len(imageDigest) != len("sha256:")+64 {
		return resultWithLog(result, output, repositoryURL), fmt.Errorf("Docker returned an invalid local image ID")
	}

	result.ImageReference = imageReference
	result.ImageDigest = imageDigest
	result.Log = sanitizedOutput(output.String(), repositoryURL)
	return result, nil
}

func commandFailure(operation string, ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("%s timed out: %w", operation, context.DeadlineExceeded)
	}
	if errors.Is(ctx.Err(), context.Canceled) {
		return fmt.Errorf("%s canceled: %w", operation, context.Canceled)
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}

func validateCloneURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" || strings.ContainsAny(value, "\r\n\x00") {
		return fmt.Errorf("repository clone URL is missing or invalid")
	}
	if strings.Contains(value, "://") {
		parsed, err := url.Parse(value)
		if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "ssh" && parsed.Scheme != "git") || parsed.Host == "" {
			return fmt.Errorf("repository clone URL must use HTTPS, SSH, or Git protocol")
		}
		return nil
	}
	if strings.HasPrefix(value, "git@") && strings.Contains(value, ":") {
		return nil
	}

	return fmt.Errorf("repository clone URL must use HTTPS, SSH, or Git protocol")
}

func validateNoSymlinks(ctx context.Context, root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if contextErr := ctx.Err(); contextErr != nil {
			return contextErr
		}
		if err != nil {
			return fmt.Errorf("inspect repository build context: %w", err)
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("repository build context contains unsupported symlink %q", filepath.Base(path))
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("repository build context contains unsupported file %q", filepath.Base(path))
		}
		return nil
	})
}

func imageReference(applicationID int64, name, commitSHA string) string {
	var sanitized strings.Builder
	lastDash := false
	for _, character := range strings.ToLower(name) {
		valid := character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
		if valid {
			sanitized.WriteRune(character)
			lastDash = false
		} else if sanitized.Len() > 0 && !lastDash {
			sanitized.WriteByte('-')
			lastDash = true
		}
	}
	name = strings.Trim(sanitized.String(), "-")
	if name == "" {
		name = "application"
	}
	if len(name) > 100 {
		name = name[:100]
	}

	return fmt.Sprintf("serve-service/app-%d-%s:sha-%s", applicationID, name, strings.ToLower(commitSHA))
}

func commandEnvironment() []string {
	// Git and Docker need selected local auth/configuration, but build arguments and host
	// environment variables are never forwarded into Dockerfile instructions.
	environment := make([]string, 0, 8)
	for _, key := range []string{
		"PATH",
		"HOME",
		"SSH_AUTH_SOCK",
		"DOCKER_CONFIG",
		"DOCKER_HOST",
		"DOCKER_TLS_VERIFY",
		"DOCKER_CERT_PATH",
	} {
		if value, exists := os.LookupEnv(key); exists && value != "" {
			environment = append(environment, key+"="+value)
		}
	}
	environment = append(environment, "GIT_TERMINAL_PROMPT=0")
	return environment
}

func resultWithLog(result usecase.BuildResult, output *limitedBuffer, repositoryURL string) usecase.BuildResult {
	result.Log = sanitizedOutput(output.String(), repositoryURL)
	return result
}

func sanitizedOutput(output, repositoryURL string) string {
	if repositoryURL != "" {
		if parsed, err := url.Parse(repositoryURL); err == nil && parsed.User != nil {
			parsed.User = url.User("[REDACTED]")
			output = strings.ReplaceAll(output, repositoryURL, parsed.String())
		}
		output = strings.ReplaceAll(output, repositoryURL, "[repository URL]")
	}
	return credentialURLPattern.ReplaceAllString(output, "$1://[REDACTED]@")
}

func appendBuildLog(current, line string) string {
	if current == "" {
		return line
	}
	return current + "\n" + line
}

type limitedBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func (buffer *limitedBuffer) Write(value []byte) (int, error) {
	if len(value) >= maxCapturedLogSize {
		buffer.buffer.Reset()
		_, _ = buffer.buffer.Write(value[len(value)-maxCapturedLogSize:])
		buffer.truncated = true
		return len(value), nil
	}
	if buffer.buffer.Len()+len(value) > maxCapturedLogSize {
		overflow := buffer.buffer.Len() + len(value) - maxCapturedLogSize
		remaining := append([]byte(nil), buffer.buffer.Bytes()[overflow:]...)
		buffer.buffer.Reset()
		_, _ = buffer.buffer.Write(remaining)
		buffer.truncated = true
	}
	_, _ = buffer.buffer.Write(value)
	return len(value), nil
}

func (buffer *limitedBuffer) String() string {
	output := strings.ToValidUTF8(buffer.buffer.String(), "\uFFFD")
	if buffer.truncated {
		const marker = "[earlier build output truncated]\n"
		if len(output)+len(marker) > maxCapturedLogSize {
			output = output[len(output)-(maxCapturedLogSize-len(marker)):]
		}
		return marker + output
	}
	return output
}
