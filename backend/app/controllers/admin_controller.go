package controllers

import (
	"math"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/services"
)

type AdminController struct {
	AdminService *services.AdminService
	URLService   *services.URLService
}

func NewAdminController(as *services.AdminService, us *services.URLService) *AdminController {
	return &AdminController{
		AdminService: as,
		URLService:   us,
	}
}

func parsePagination(c *gin.Context) (int, int) {
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
	return limit, offset
}

// GetAllUsers godoc
func (ac *AdminController) GetAllUsers(c *gin.Context) {
	ctx := c.Request.Context()
	limit, offset := parsePagination(c)

	users, total, err := ac.AdminService.GetAllUsers(ctx, limit, offset)
	if err != nil {
		c.JSON(500, models.HTTPResponseErr{
			Err: "internal server error",
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	currentPage := (offset / limit) + 1

	c.JSON(200, models.PaginatedUserResponse{
		Data: users,
		Meta: models.PaginationMeta{
			Total:      total,
			Page:       currentPage,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

// GetUserURLsAsAdmin godoc
func (ac *AdminController) GetUserURLsAsAdmin(c *gin.Context) {
	ctx := c.Request.Context()

	userIDStr := c.Param("userid")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(400, models.HTTPError{Error: "invalid user id"})
		return
	}

	limit, offset := parsePagination(c)

	URLs, total, err := ac.URLService.GetUserURLs(ctx, userID, limit, offset)
	if err != nil {
		c.JSON(500, models.HTTPResponseErr{
			Err: "internal server error",
		})
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
