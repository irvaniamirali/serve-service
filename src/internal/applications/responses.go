package applications

// ListApplicationsResponse represents a paginated application list response.
type ListApplicationsResponse struct {
	Applications []Application `json:"applications"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}
