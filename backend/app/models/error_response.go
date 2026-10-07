package models

// HTTPError represents a standard API error response
type HTTPError struct {
	Error string `json:"error" example:"invalid credentials"`
}

// HTTPValidationError represents field-specific validation errors
type HTTPValidationError struct {
	Error map[string]string `json:"error" example:"long_url:give valid URL"`
}

// HTTPDetailsError represents a standard error accompanied by system details
type HTTPDetailsError struct {
	Error   string `json:"error" example:"failed to fetch analytics"`
	Details string `json:"details" example:"connection refused"`
}

// HTTPResponseErr represents an error response keyed under 'err'
type HTTPResponseErr struct {
	Err string `json:"err" example:"not found"`
}
