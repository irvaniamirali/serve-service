package applications

import "strconv"

// CreateApplicationRequest is the JSON payload for creating an application.
type CreateApplicationRequest struct {
	Name               string `json:"name"`
	RepositoryURL      string `json:"repository_url"`
	Branch             string `json:"branch"`
	GitHubRepositoryID *int64 `json:"github_repository_id"`
}

// parseQueryInteger returns the fallback when the query parameter is absent.
func parseQueryInteger(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}

	return strconv.Atoi(raw)
}
