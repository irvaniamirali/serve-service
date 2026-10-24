package controllers

import "github.com/irvaniamirali/serve-service/internal/domains/applications"

// listResponse represents a paginated application list response.
type listResponse struct {
	Applications []applications.Application `json:"applications"`
	Limit        int                        `json:"limit"`
	Offset       int                        `json:"offset"`
}
