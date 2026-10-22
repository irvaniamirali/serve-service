package applications

import (
	"net/url"
	"strings"
)

// Validate checks whether an application contains valid required fields.
func Validate(application Application) error {
	if strings.TrimSpace(application.Name) == "" ||
		strings.TrimSpace(application.RepositoryURL) == "" ||
		strings.TrimSpace(application.Branch) == "" {
		return ErrValidation
	}
	if application.GitHubRepositoryID != nil && *application.GitHubRepositoryID < 1 {
		return ErrValidation
	}

	parsedURL, err := url.ParseRequestURI(application.RepositoryURL)
	if err != nil || parsedURL.Host == "" ||
		(parsedURL.Scheme != "https" && parsedURL.Scheme != "http") {
		return ErrValidation
	}

	return nil
}
