package applications

import "time"

// Application represents an application in the domain.
type Application struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	RepositoryURL string    `json:"repository_url"`
	Branch        string    `json:"branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ApplicationModel maps an application to the database schema.
type ApplicationModel struct {
	ID            int64     `gorm:"primaryKey;column:id"`
	Name          string    `gorm:"column:name"`
	RepositoryURL string    `gorm:"column:repository_url"`
	Branch        string    `gorm:"column:branch"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

// TableName specifies the existing database table.
func (ApplicationModel) TableName() string {
	return "applications"
}
