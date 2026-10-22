package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/irvaniamirali/serve-service/internal/applications"
)

func TestResolveDefaultBranchHeadUsesGitHubMetadataAndNonMainDefaultBranch(t *testing.T) {
	const wantSHA = "abcdef0123456789abcdef0123456789abcdef01"
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests = append(requests, request.Method+" "+request.URL.RequestURI())
		switch request.URL.Path {
		case "/repos/acme/service":
			_, _ = fmt.Fprint(writer, `{"id":42,"full_name":"acme/service","default_branch":"release/next"}`)
		case "/repos/acme/service/commits":
			if request.URL.Query().Get("sha") != "release/next" || request.URL.Query().Get("per_page") != "1" {
				t.Errorf("commit query = %v, want default branch and one result", request.URL.Query())
			}
			_, _ = fmt.Fprintf(writer, `[{"sha":%q}]`, wantSHA)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := testAPIClient(server.URL)
	sha, err := client.ResolveDefaultBranchHead(context.Background(), applications.Application{
		RepositoryURL: "https://github.com/acme/service.git",
		Branch:        "configured-not-default",
	})
	if err != nil {
		t.Fatalf("ResolveDefaultBranchHead() error = %v", err)
	}
	if sha != wantSHA {
		t.Fatalf("resolved SHA = %q, want %q", sha, wantSHA)
	}
	if len(requests) != 2 || requests[0] != "GET /repos/acme/service" ||
		!strings.HasPrefix(requests[1], "GET /repos/acme/service/commits?") {
		t.Fatalf("GitHub API requests = %v", requests)
	}
}

func TestResolveDefaultBranchHeadUsesRepositoryIDAndAuthentication(t *testing.T) {
	const wantSHA = "0123456789abcdef0123456789abcdef01234567"
	repositoryID := int64(42)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Authorization = %q, want bearer token", request.Header.Get("Authorization"))
		}
		switch request.URL.Path {
		case "/repositories/42":
			_, _ = fmt.Fprint(writer, `{"id":42,"full_name":"acme/service","default_branch":"stable"}`)
		case "/repos/acme/service/commits":
			if request.URL.Query().Get("sha") != "stable" {
				t.Errorf("commit branch = %q, want stable", request.URL.Query().Get("sha"))
			}
			_, _ = fmt.Fprintf(writer, `[{"sha":%q}]`, wantSHA)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	client := testAPIClient(server.URL)
	client.token = "test-token"
	sha, err := client.ResolveDefaultBranchHead(context.Background(), applications.Application{
		RepositoryURL:      "https://github.com/acme/service.git",
		GitHubRepositoryID: &repositoryID,
	})
	if err != nil {
		t.Fatalf("ResolveDefaultBranchHead() error = %v", err)
	}
	if sha != wantSHA {
		t.Fatalf("resolved SHA = %q, want %q", sha, wantSHA)
	}
}

func TestResolveDefaultBranchHeadRejectsMissingMetadataAndCommit(t *testing.T) {
	tests := []struct {
		name         string
		metadataJSON string
		commitJSON   string
		wantError    string
	}{
		{
			name:         "missing default branch",
			metadataJSON: `{"id":42,"full_name":"acme/service"}`,
			commitJSON:   `[{"sha":"abcdef0123456789abcdef0123456789abcdef01"}]`,
			wantError:    "no full name or default branch",
		},
		{
			name:         "empty commit list",
			metadataJSON: `{"id":42,"full_name":"acme/service","default_branch":"stable"}`,
			commitJSON:   `[]`,
			wantError:    `no valid commit for default branch "stable"`,
		},
		{
			name:         "invalid commit",
			metadataJSON: `{"id":42,"full_name":"acme/service","default_branch":"stable"}`,
			commitJSON:   `[{"sha":"not-a-commit"}]`,
			wantError:    `no valid commit for default branch "stable"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/repos/acme/service" {
					_, _ = fmt.Fprint(writer, test.metadataJSON)
					return
				}
				_, _ = fmt.Fprint(writer, test.commitJSON)
			}))
			defer server.Close()

			_, err := testAPIClient(server.URL).ResolveDefaultBranchHead(
				context.Background(),
				applications.Application{RepositoryURL: "https://github.com/acme/service"},
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("ResolveDefaultBranchHead() error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestResolveDefaultBranchHeadSurfacesGitHubAPIAndRepositoryErrors(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		headers    map[string]string
		message    string
		wantError  string
		repository string
	}{
		{
			name:       "repository not found",
			status:     http.StatusNotFound,
			message:    `{"message":"Not Found"}`,
			wantError:  "HTTP 404: Not Found",
			repository: "https://github.com/acme/missing",
		},
		{
			name:       "rate limited",
			status:     http.StatusForbidden,
			headers:    map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "123"},
			message:    `{"message":"API rate limit exceeded"}`,
			wantError:  "rate limit exceeded (resets at Unix time 123)",
			repository: "https://github.com/acme/service",
		},
		{
			name:       "unsupported repository",
			repository: "https://example.com/acme/service.git",
			wantError:  "requires a GitHub.com HTTPS or SSH repository URL",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				for key, value := range test.headers {
					writer.Header().Set(key, value)
				}
				writer.WriteHeader(test.status)
				_, _ = fmt.Fprint(writer, test.message)
			}))
			defer server.Close()

			_, err := testAPIClient(server.URL).ResolveDefaultBranchHead(
				context.Background(),
				applications.Application{RepositoryURL: test.repository},
			)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("ResolveDefaultBranchHead() error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestResolveDefaultBranchHeadRejectsRepositoryIDMismatch(t *testing.T) {
	repositoryID := int64(42)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(writer, `{"id":43,"full_name":"acme/service","default_branch":"main"}`)
	}))
	defer server.Close()

	_, err := testAPIClient(server.URL).ResolveDefaultBranchHead(
		context.Background(),
		applications.Application{
			RepositoryURL:      "https://github.com/acme/service",
			GitHubRepositoryID: &repositoryID,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("ResolveDefaultBranchHead() error = %v, want repository ID mismatch", err)
	}
}

func TestResolveDefaultBranchHeadRejectsRepositoryURLMismatch(t *testing.T) {
	repositoryID := int64(42)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(writer, `{"id":42,"full_name":"acme/other","default_branch":"main"}`)
	}))
	defer server.Close()

	_, err := testAPIClient(server.URL).ResolveDefaultBranchHead(
		context.Background(),
		applications.Application{
			RepositoryURL:      "https://github.com/acme/service",
			GitHubRepositoryID: &repositoryID,
		},
	)
	if err == nil || !strings.Contains(err.Error(), "does not match the configured repository URL") {
		t.Fatalf("ResolveDefaultBranchHead() error = %v, want repository URL mismatch", err)
	}
}

func testAPIClient(serverURL string) *Client {
	client := NewClient("")
	client.apiURL = serverURL
	client.httpClient = http.DefaultClient
	return client
}
