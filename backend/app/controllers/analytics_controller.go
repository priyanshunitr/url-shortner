package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/services"
)

type AnalyticsController struct {
	Service *services.AnalyticsService
}

func NewAnalyticsController(s *services.AnalyticsService) *AnalyticsController {
	return &AnalyticsController{
		Service: s,
	}
}

func parseURLID(c *gin.Context) (int64, error) {
	urlID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return urlID, nil
}

func handleServiceError(c *gin.Context, err error) {
	switch err.Error() {
	case "url not found":
		c.JSON(http.StatusNotFound, models.HTTPError{Error: err.Error()})
	case "forbidden":
		c.JSON(http.StatusForbidden, models.HTTPError{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, models.HTTPDetailsError{
			Error:   "internal server error",
			Details: err.Error(),
		})
	}
}

// GetAnalyticsOverview godoc
func (ac *AnalyticsController) GetAnalyticsOverview(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	urlID, err := parseURLID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.HTTPError{Error: "invalid url id"})
		return
	}

	overview, err := ac.Service.GetAnalyticsOverview(ctx, urlID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, overview)
}

// GetDailyClicks godoc
func (ac *AnalyticsController) GetDailyClicks(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	urlID, err := parseURLID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.HTTPError{Error: "invalid url id"})
		return
	}

	clicks, err := ac.Service.GetDailyClicks(ctx, urlID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, clicks)
}

// GetRecentVisits godoc
func (ac *AnalyticsController) GetRecentVisits(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	urlID, err := parseURLID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.HTTPError{Error: "invalid url id"})
		return
	}

	visits, err := ac.Service.GetRecentVisits(ctx, urlID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, visits)
}

// GetBrowserAnalytics godoc
func (ac *AnalyticsController) GetBrowserAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	urlID, err := parseURLID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.HTTPError{Error: "invalid url id"})
		return
	}

	browserStats, err := ac.Service.GetBrowserAnalytics(ctx, urlID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, browserStats)
}

// GetDeviceAnalytics godoc
func (ac *AnalyticsController) GetDeviceAnalytics(c *gin.Context) {
	ctx := c.Request.Context()
	userID := c.MustGet("userID").(int64)

	urlID, err := parseURLID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.HTTPError{Error: "invalid url id"})
		return
	}

	deviceStats, err := ac.Service.GetDeviceAnalytics(ctx, urlID, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, deviceStats)
}
