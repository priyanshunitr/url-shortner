package controllers

import (
	"errors"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/services"
	"github.com/gottatouchsomegrass/url/pkg/utils"
	"github.com/mssola/user_agent"
)

type AuthController struct {
	Service *services.AuthService
}

func NewAuthController(
	s *services.AuthService,
) *AuthController {
	return &AuthController{
		Service: s,
	}
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body RegisterRequest true "Registration Info"
// @Success      201  {object}  models.AuthRegisterSuccess
// @Failure      400  {object}  models.HTTPError
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/register [post]
func (auc *AuthController) Register(c *gin.Context) {
	ctx := c.Request.Context()
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.HTTPError{Error: err.Error()})
		return
	}

	token, refresh, err := auc.Service.RegisterUser(ctx, req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		c.JSON(500, models.HTTPError{Error: err.Error()})
		return
	}

	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(
		c.Writer,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    refresh,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Path:     "/api/v1/auth/refresh",
			MaxAge:   7 * 24 * 3600,
		},
	)

	c.JSON(201, models.AuthRegisterSuccess{
		Token: token,
	})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate user and return a JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginRequest true "Login Info"
// @Success      200  {object}  models.AuthLoginSuccess
// @Failure      400  {object}  models.HTTPError
// @Failure      401  {object}  models.HTTPError
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/login [post]
func (auc *AuthController) Login(c *gin.Context) {
	ctx := c.Request.Context()
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, models.HTTPError{Error: err.Error()})
		return
	}

	token, refresh, err := auc.Service.LoginUser(ctx, req.Email, req.Password, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if errors.Is(err, utils.ErrRateLimitExceeded) {
			c.JSON(429, models.HTTPError{Error: "Too many login attempts. Please try again in 15 minutes."})
			return
		}

		// In a real app, map errors to correct status codes (401 vs 500)
		if err.Error() == "invalid credentials" {
			c.JSON(401, models.HTTPError{Error: err.Error()})
		} else {
			c.JSON(500, models.HTTPError{Error: err.Error()})
		}
		return
	}

	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(
		c.Writer,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    refresh,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Path:     "/api/v1/auth/refresh",
			MaxAge:   7 * 24 * 3600,
		},
	)

	c.JSON(200, models.AuthLoginSuccess{
		Message: "user logged in successfully",
		Token:   token,
	})
}

// Logout logs out the user by invalidating their token in the Redis Blacklist.
// @Summary      Logout user
// @Description  Invalidate the user's JWT token
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.MessageSuccess
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/logout [post]
func (auc *AuthController) Logout(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	tokenString := authHeader[len("Bearer "):]

	refreshToken, _ := c.Cookie("refresh_token")

	err := auc.Service.LogoutUser(c.Request.Context(), tokenString, refreshToken)
	if err != nil {
		c.JSON(500, models.HTTPError{Error: "failed to logout"})
		return
	}

	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(
		c.Writer,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    "",
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Path:     "/api/v1/auth/refresh",
			MaxAge:   -1,
		},
	)

	c.JSON(200, models.MessageSuccess{Message: "user logged out successfully"})
}

// Me returns the user's information.
// Me godoc
// @Summary      Get current user info
// @Description  Get current authenticated user info
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.User
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/me [get]
func (auc *AuthController) Me(
	c *gin.Context,
) {
	userID := c.MustGet(
		"userID",
	).(int64)

	user, err := auc.Service.UserRepo.GetUserByID(
		c.Request.Context(),
		userID,
	)

	if err != nil {
		c.JSON(500, models.HTTPError{
			Error: err.Error(),
		})
		return
	}

	c.JSON(200, user)
}

// DeleteAccount godoc
// @Summary      Delete user account
// @Description  Completely delete the user's account and all associated URLs and analytics
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.MessageSuccess
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/me [delete]
func (auc *AuthController) DeleteAccount(c *gin.Context) {
	userID := c.MustGet("userID").(int64)

	err := auc.Service.UserRepo.DeleteUserAccount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(500, models.HTTPError{
			Error: "Failed to delete account: " + err.Error(),
		})
		return
	}

	c.JSON(200, models.MessageSuccess{
		Message: "account deleted successfully",
	})
}

func (auc *AuthController) RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(401, models.HTTPError{
			Error: "Refresh token is missing",
		})
		return
	}

	token, newRefresh, err := auc.Service.RefreshToken(c.Request.Context(), refreshToken, c.ClientIP(), c.Request.UserAgent())
	if err != nil {
		if err.Error() == "invalid refresh token" || err.Error() == "refresh token is expired or revoked" {
			c.JSON(401, models.HTTPError{Error: err.Error()})
		} else {
			c.JSON(500, models.HTTPError{Error: err.Error()})
		}
		return
	}

	secure := os.Getenv("APP_ENV") == "production"
	http.SetCookie(
		c.Writer,
		&http.Cookie{
			Name:     "refresh_token",
			Value:    newRefresh,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
			Path:     "/api/v1/auth/refresh",
			MaxAge:   7 * 24 * 3600,
		},
	)

	c.JSON(200, models.AuthLoginSuccess{
		Message: "token refreshed successfully",
		Token:   token,
	})
}

// GetSessions returns all active device sessions for the user
// @Summary      Get active sessions
// @Description  Get all active refresh token sessions for the current user
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.SessionListResponse
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/sessions [get]
func (auc *AuthController) GetSessions(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	sessionID := c.MustGet("sessionID").(int64)

	res, err := auc.Service.GetSessions(c.Request.Context(), userID, sessionID)
	if err != nil {
		c.JSON(500, models.HTTPError{Error: "failed to retrieve sessions"})
		return
	}

	for i, session := range res.Sessions {
		ua := user_agent.New(session.Browser)
		browserName, browserVersion := ua.Browser()
		os := ua.OS()

		res.Sessions[i].Browser = browserName + " " + browserVersion
		res.Sessions[i].Device = os
		if ua.Mobile() {
			res.Sessions[i].Device += " (Mobile)"
		}
	}

	c.JSON(200, res)
}

// RevokeSession logs out a specific device session
// @Summary      Revoke a session
// @Description  Revokes a specific refresh token session for the current user
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Param        id   path      int  true  "Session ID"
// @Success      200  {object}  models.MessageSuccess
// @Failure      400  {object}  models.HTTPError
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/sessions/{id} [delete]
func (auc *AuthController) RevokeSession(c *gin.Context) {
	userID := c.MustGet("userID").(int64)

	sessionIDStr := c.Param("id")
	sessionID, err := strconv.ParseInt(sessionIDStr, 10, 64)
	if err != nil {
		c.JSON(400, models.HTTPError{Error: "invalid session ID format"})
		return
	}

	err = auc.Service.RevokeSession(c.Request.Context(), sessionID, userID)
	if err != nil {
		c.JSON(500, models.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(200, models.MessageSuccess{Message: "session revoked successfully"})
}

// RevokeAllOtherSessions logs out all devices except the current one
// @Summary      Revoke all other sessions
// @Description  Revokes all refresh token sessions for the current user except the one making this request
// @Tags         auth
// @Produce      json
// @Security     Bearer
// @Success      200  {object}  models.MessageSuccess
// @Failure      400  {object}  models.HTTPError
// @Failure      500  {object}  models.HTTPError
// @Router       /auth/sessions/others [delete]
func (auc *AuthController) RevokeAllOtherSessions(c *gin.Context) {
	userID := c.MustGet("userID").(int64)
	currentSessionIDStr := c.MustGet("sessionID").(string) // from jwt claims

	currentSessionID, err := strconv.ParseInt(currentSessionIDStr, 10, 64)
	if err != nil {
		c.JSON(400, models.HTTPError{Error: "invalid current session ID format"})
		return
	}

	err = auc.Service.RevokeAllOtherSessions(c.Request.Context(), userID, currentSessionID)
	if err != nil {
		c.JSON(500, models.HTTPError{Error: err.Error()})
		return
	}

	c.JSON(200, models.MessageSuccess{Message: "all other sessions revoked successfully"})
}
