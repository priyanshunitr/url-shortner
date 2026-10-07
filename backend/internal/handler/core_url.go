package controllers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
)

type CreateURLRequest struct {
	URL       string     `json:"url" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// CreatePublicURL godoc
// @Summary Create a short URL
// @Description Creates an anonymous URL; optional expires_at must be in the future.
// @Tags urls
// @Accept json
// @Produce json
// @Param request body CreateURLRequest true "URL to shorten"
// @Success 201 {object} models.CoreURLResponse
// @Failure 400 {object} models.HTTPError
// @Failure 429 {object} models.HTTPError
// @Router /api/urls [post]
func (uc *URLController) CreatePublicURL(c *gin.Context) {
	var req CreateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provide a valid JSON URL request"})
		return
	}
	parsed, err := url.ParseRequestURI(req.URL)
	if err != nil || parsed.Hostname() == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(req.URL) > 8192 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url must be an absolute HTTP or HTTPS URL (maximum 8192 bytes)"})
		return
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expires_at must be in the future"})
		return
	}
	result, err := uc.Service.CreatePublicURL(c.Request.Context(), req.URL, req.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create short URL"})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// GetPublicStats godoc
// @Summary Get statistics for a public short URL
// @Tags urls
// @Produce json
// @Param code path string true "Short code"
// @Success 200 {object} models.URLStats
// @Failure 404 {object} models.HTTPError
// @Router /api/urls/{code}/stats [get]
func (uc *URLController) GetPublicStats(c *gin.Context) {
	stats, err := uc.Service.GetPublicStats(c.Request.Context(), c.Param("code"))
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read URL statistics"})
		}
		return
	}
	c.JSON(http.StatusOK, stats)
}
