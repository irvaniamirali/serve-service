package applications

import "time"

// Application represents a deployable application.
type Application struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	RepositoryURL string    `json:"repository_url"`
	Branch        string    `json:"branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
