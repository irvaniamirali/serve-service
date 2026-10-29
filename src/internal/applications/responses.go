package applications

// listResponse represents a paginated application list response.
type listResponse struct {
	Applications []Application `json:"applications"`
	Limit        int           `json:"limit"`
	Offset       int           `json:"offset"`
}
