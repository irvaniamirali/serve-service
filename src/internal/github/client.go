package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

const (
	defaultAPIURL      = "https://api.github.com"
	githubRequestLimit = 1 << 20
)

// Client resolves trusted repository metadata and commit information from GitHub.
type Client struct {
	apiURL     string
	token      string
	httpClient *http.Client
}

// NewClient creates a GitHub API client. Token is optional for public repositories;
// private repositories require a token with repository contents read permission.
func NewClient(token string) *Client {
	return &Client{
		apiURL: strings.TrimRight(defaultAPIURL, "/"),
		token:  strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ResolveDefaultBranchHead returns the full commit SHA at the repository's GitHub
// default branch when the API is queried. Callers persist this SHA before queuing work.
func (client *Client) ResolveDefaultBranchHead(
	ctx context.Context,
	application applications.Application,
) (string, error) {
	owner, repository, err := parseGitHubRepository(application.RepositoryURL)
	if err != nil {
		return "", err
	}

	repositoryPath := "/repos/" + url.PathEscape(owner) + "/" + url.PathEscape(repository)
	if application.GitHubRepositoryID != nil {
		repositoryPath = "/repositories/" + strconv.FormatInt(*application.GitHubRepositoryID, 10)
	}

	var metadata struct {
		ID            int64  `json:"id"`
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := client.getJSON(ctx, repositoryPath, nil, &metadata); err != nil {
		return "", fmt.Errorf("retrieve GitHub repository metadata: %w", err)
	}
	if application.GitHubRepositoryID != nil && metadata.ID != *application.GitHubRepositoryID {
		return "", fmt.Errorf("GitHub repository metadata ID does not match the registered repository ID")
	}
	if metadata.FullName == "" || metadata.DefaultBranch == "" {
		return "", fmt.Errorf("GitHub repository metadata has no full name or default branch")
	}
	if !strings.EqualFold(metadata.FullName, owner+"/"+repository) {
		return "", fmt.Errorf("GitHub repository metadata does not match the configured repository URL")
	}

	parts := strings.Split(metadata.FullName, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("GitHub returned an invalid repository full name")
	}
	query := url.Values{
		"per_page": {"1"},
		"sha":      {metadata.DefaultBranch},
	}
	var commits []struct {
		SHA string `json:"sha"`
	}
	if err := client.getJSON(
		ctx,
		"/repos/"+url.PathEscape(parts[0])+"/"+url.PathEscape(parts[1])+"/commits",
		query,
		&commits,
	); err != nil {
		return "", fmt.Errorf("retrieve HEAD of GitHub default branch %q: %w", metadata.DefaultBranch, err)
	}
	if len(commits) == 0 || !IsDeployableCommitSHA(commits[0].SHA) {
		return "", fmt.Errorf("GitHub returned no valid commit for default branch %q", metadata.DefaultBranch)
	}
	return strings.ToLower(commits[0].SHA), nil
}

func (client *Client) getJSON(
	ctx context.Context,
	path string,
	query url.Values,
	destination any,
) error {
	requestURL := client.apiURL + path
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return fmt.Errorf("create GitHub API request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("Cache-Control", "no-cache")
	if client.token != "" {
		request.Header.Set("Authorization", "Bearer "+client.token)
	}

	response, err := client.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send GitHub API request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, githubRequestLimit+1))
	if err != nil {
		return fmt.Errorf("read GitHub API response: %w", err)
	}
	if len(body) > githubRequestLimit {
		return fmt.Errorf("GitHub API response exceeded the size limit")
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiError struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &apiError); err != nil {
			apiError.Message = ""
		}
		if response.StatusCode == http.StatusForbidden && response.Header.Get("X-RateLimit-Remaining") == "0" {
			apiError.Message = "GitHub API rate limit exceeded"
			if reset := response.Header.Get("X-RateLimit-Reset"); reset != "" {
				apiError.Message += " (resets at Unix time " + reset + ")"
			}
		} else if response.StatusCode == http.StatusUnauthorized {
			apiError.Message = "GitHub API authentication failed"
		} else if apiError.Message == "" {
			apiError.Message = http.StatusText(response.StatusCode)
		}
		return fmt.Errorf("GitHub API returned HTTP %d: %s", response.StatusCode, apiError.Message)
	}
	if err := json.Unmarshal(body, destination); err != nil {
		return fmt.Errorf("decode GitHub API response: %w", err)
	}
	return nil
}

func parseGitHubRepository(repositoryURL string) (string, string, error) {
	value := strings.TrimSpace(repositoryURL)
	if strings.HasPrefix(value, "git@github.com:") {
		value = strings.TrimPrefix(value, "git@github.com:")
		return splitRepositoryPath(value)
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Hostname() != "github.com" ||
		(parsed.Scheme != "https" && parsed.Scheme != "ssh") ||
		parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", "", fmt.Errorf("default-branch resolution requires a GitHub.com HTTPS or SSH repository URL")
	}
	if parsed.User != nil {
		if parsed.Scheme != "ssh" || parsed.User.Username() != "git" {
			return "", "", fmt.Errorf("GitHub repository URL must not contain credentials")
		}
		if _, hasPassword := parsed.User.Password(); hasPassword {
			return "", "", fmt.Errorf("GitHub repository URL must not contain credentials")
		}
	}
	return splitRepositoryPath(parsed.Path)
}

func splitRepositoryPath(value string) (string, string, error) {
	value = strings.Trim(strings.TrimSpace(value), "/")
	value = strings.TrimSuffix(value, ".git")
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("default-branch resolution requires a GitHub.com owner/repository URL")
	}
	return parts[0], parts[1], nil
}
