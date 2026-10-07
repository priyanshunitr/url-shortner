// Package controllers handles HTTP request logic
package controllers

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/services"
	"github.com/gottatouchsomegrass/url/pkg/utils"
	"github.com/redis/go-redis/v9"
)

type URLController struct {
	Service *services.URLService
	RDB     *redis.Client
}

type Request struct {
	LongURL    string     `json:"long_url" binding:"required,url"`
	CustomCode string     `json:"custom_code,omitempty" binding:"omitempty,shortcode"`
	Expiry     *time.Time `json:"expiry,omitempty"`
}

// NewURLController adds new url to db
func NewURLController(s *services.URLService, rdb *redis.Client) *URLController {
	return &URLController{
		Service: s,
		RDB:     rdb,
	}
}

// ShortenURL godoc
// ... (swagger docs omitted for brevity)
func (uc *URLController) ShortenURL(c *gin.Context) {
	ctx := c.Request.Context()

	var req Request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.HTTPValidationError{
			Error: utils.ValidateErrors(err),
		})
		return
	}

	userID := c.MustGet("userID").(int64)
	roleRaw, exists := c.Get("userRole")
	userRole := "free"
	if exists {
		userRole = roleRaw.(string)
	}

	_, err := uc.Service.ShortenURL(ctx, userID, userRole, req.LongURL, req.CustomCode)
	if err != nil {
		status := 500
		if strings.Contains(err.Error(), "limit") || strings.Contains(err.Error(), "require a premium") {
			status = 403
		} else if strings.Contains(err.Error(), "exists") {
			status = 400
		}
		c.JSON(status, models.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(201, models.URLShortenSuccess{
		Success: "url inserted to db",
	})
}

// RedirectURL godoc
// ...
func (uc *URLController) RedirectURL(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("shortCode")
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()
	referer := c.Request.Referer()

	longURL, err := uc.Service.HandleRedirect(ctx, code, ip, userAgent, referer)
	if err != nil {
		if err.Error() == "not found" {
			c.JSON(404, models.HTTPResponseErr{Err: "not found"})
			return
		}
		if err.Error() == "link expired" {
			c.JSON(410, models.HTTPResponseErr{Err: "link expired"})
			return
		}
		c.JSON(500, models.HTTPResponseErr{Err: err.Error()})
		return
	}

	c.Redirect(302, longURL)
}

// GetUserURLs godoc
// ...
func (uc *URLController) GetUserURLs(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	pageStr := c.Query("page")
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err == nil && page > 1 {
			offset = (page - 1) * limit
		}
	}

	URLs, total, err := uc.Service.GetUserURLs(ctx, userID, limit, offset)
	if err != nil {
		c.JSON(500, models.HTTPResponseErr{Err: err.Error()})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	currentPage := (offset / limit) + 1

	c.JSON(200, models.PaginatedURLResponse{
		Data: URLs,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       currentPage,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

type UpdateURLRequest struct {
	LongURL string `json:"long_url" binding:"required,url"`
}

// UpdateUserURL godoc
func (uc *URLController) UpdateUserURL(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	var req UpdateURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.HTTPValidationError{
			Error: utils.ValidateErrors(err),
		})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, models.HTTPError{Error: "invalid url id"})
		return
	}

	err = uc.Service.UpdateUserURL(ctx, id, userID, req.LongURL)
	if err != nil {
		status := 500
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "unauthorized") {
			status = 400
		}
		c.JSON(status, models.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(200, models.MessageSuccess{Message: "url updated successfully"})
}

// DeleteUserURL godoc
func (uc *URLController) DeleteUserURL(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, models.HTTPError{Error: "invalid url id"})
		return
	}

	if err := uc.Service.DeleteUserURL(ctx, id, userID); err != nil {
		c.JSON(500, models.HTTPResponseErr{Err: err.Error()})
		return
	}

	c.JSON(200, models.MessageSuccess{Message: "url deleted"})
}

type BulkDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required,min=1"`
}

// BulkDeleteUserURLs godoc
func (uc *URLController) BulkDeleteUserURLs(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	var req BulkDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.HTTPValidationError{
			Error: utils.ValidateErrors(err),
		})
		return
	}

	if err := uc.Service.BulkDeleteUserURLs(ctx, req.IDs, userID); err != nil {
		c.JSON(500, models.HTTPResponseErr{Err: err.Error()})
		return
	}

	c.JSON(200, models.MessageSuccess{
		Message: fmt.Sprintf("successfully deleted %d urls", len(req.IDs)),
	})
}
