package applications

import "time"

// Application represents a deployable application.
type Application struct {
	ID                 int64     `json:"id"`
	Name               string    `json:"name"`
	RepositoryURL      string    `json:"repository_url"`
	Branch             string    `json:"branch"`
	GitHubRepositoryID *int64    `json:"github_repository_id,omitempty"`
	WebhookSecret      string    `json:"-"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}
