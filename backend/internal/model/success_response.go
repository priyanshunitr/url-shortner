package models

// AuthRegisterSuccess represents the success response for user registration
type AuthRegisterSuccess struct {
	Token string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// AuthLoginSuccess represents the success response for user login
type AuthLoginSuccess struct {
	Message string `json:"message" example:"user logged in successfully"`
	Token   string `json:"token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// URLShortenSuccess represents the success response for URL shortening
type URLShortenSuccess struct {
	Success string `json:"success" example:"url inserted to db"`
}

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Total      int `json:"total" example:"150"`
	Page       int `json:"page" example:"1"`
	Limit      int `json:"limit" example:"50"`
	TotalPages int `json:"total_pages" example:"3"`
}

// PaginatedURLResponse represents the paginated response of URLs
type PaginatedURLResponse struct {
	Data []URL          `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginatedUserResponse represents the paginated response of users (Admin only)
type PaginatedUserResponse struct {
	Data []User         `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// MessageSuccess represents a generic success response with a message
type MessageSuccess struct {
	Message string `json:"message" example:"operation successful"`
}

// Session represents a user's active device session
type Session struct {
	ID              int64  `json:"id"`
	Device          string `json:"device"`
	Browser         string `json:"browser"`
	Location        string `json:"location"`
	IPAddress       string `json:"ip_address"`
	LastActive      string `json:"last_active"`
	IsCurrentDevice bool   `json:"is_current_device"`
}

// SessionListResponse represents the response containing a list of sessions
type SessionListResponse struct {
	Sessions []Session `json:"sessions"`
}
