package services

import (
	"context"
	"errors"
	"time"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/repositories"
	"github.com/gottatouchsomegrass/url/pkg/utils"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrRateLimitExceeded = errors.New("rate limit exceeded: too many login attempts")

type AuthService struct {
	UserRepo *repositories.UserQuery
}

func NewAuthService(ur *repositories.UserQuery) *AuthService {
	return &AuthService{
		UserRepo: ur,
	}
}

func (s *AuthService) RegisterUser(ctx context.Context, email, password, ip, userAgent string) (string, string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	tx, err := s.UserRepo.DB.Begin(ctx)
	if err != nil {
		return "", "", errors.New("failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	user := models.User{
		Email:        email,
		PasswordHash: string(hash),
	}

	qtx := s.UserRepo.WithTx(tx)

	err = qtx.CreateUser(ctx, &user)
	if err != nil {
		return "", "", err
	}

	refresh, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	rt := models.RefreshToken{
		UserID:      user.ID,
		RefreshHash: utils.HashToken(refresh),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		IP:          ip,
		UserAgent:   userAgent,
	}
	err = qtx.CreateRefreshToken(ctx, &rt)
	if err != nil {
		return "", "", errors.New("failed to store refresh token")
	}

	err = qtx.EnforceMaxRefreshTokens(ctx, user.ID, 10)
	if err != nil {
		return "", "", errors.New("failed to enforce max refresh sessions")
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", errors.New("failed to commit transaction")
	}

	token, err := utils.GenerateJWT(user.ID, user.Role, rt.ID)
	if err != nil {
		return "", "", err
	}

	// return jwt token, refresh token, error
	return token, refresh, nil
}

func (s *AuthService) LoginUser(ctx context.Context, email, password, ip, userAgent string) (string, string, error) {
	// --- 1. RATE LIMITING CHECK ---
	if err := utils.CheckLoginRateLimit(ctx, s.UserRepo.RDB, ip, email); err != nil {
		return "", "", err
	}

	user, err := s.UserRepo.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", errors.New("invalid credentials")
	}
	if err != nil {
		return "", "", errors.New("user login db error")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid credentials")
	}

	tx, err := s.UserRepo.DB.Begin(ctx)
	if err != nil {
		return "", "", errors.New("failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.UserRepo.WithTx(tx)

	refresh, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	rt := models.RefreshToken{
		UserID:      user.ID,
		RefreshHash: utils.HashToken(refresh),
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
		IP:          ip,
		UserAgent:   userAgent,
	}
	err = qtx.CreateRefreshToken(ctx, &rt)
	if err != nil {
		return "", "", errors.New("failed to store refresh token")
	}
	err = qtx.EnforceMaxRefreshTokens(ctx, user.ID, 10)
	if err != nil {
		return "", "", errors.New("failed to enforce max refresh sessions")
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", errors.New("failed to commit transaction")
	}

	token, err := utils.GenerateJWT(user.ID, user.Role, rt.ID)
	if err != nil {
		return "", "", err
	}

	// --- 2. CLEAR LIMITS ON SUCCESS ---
	utils.ResetLoginAttempts(ctx, s.UserRepo.RDB, ip, email)

	// return jwt token, refresh token, error
	return token, refresh, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken, ip, userAgent string) (string, string, error) {
	hash := utils.HashToken(refreshToken)
	rt, err := s.UserRepo.GetRefreshToken(ctx, hash)
	if err != nil {
		return "", "", errors.New("invalid refresh token")
	}
	if rt.RevokedAt != nil || rt.ExpiresAt.Before(time.Now()) {
		return "", "", errors.New("refresh token is expired or revoked")
	}

	user, err := s.UserRepo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return "", "", errors.New("failed to fetch user")
	}

	tx, err := s.UserRepo.DB.Begin(ctx)
	if err != nil {
		return "", "", errors.New("failed to begin transaction")
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := s.UserRepo.WithTx(tx)

	// --- REVOKE + INSERT METHOD (For highly secure strict session tracking) ---
	// err = s.UserRepo.RevokeRefreshTokenTx(ctx, tx, hash)
	// if err != nil {
	// 	return "", "", errors.New("failed to revoke old refresh token")
	// }
	// newRefresh, err := utils.GenerateRefreshToken()
	// if err != nil {
	// 	return "", "", err
	// }
	// newRt := models.RefreshToken{
	// 	UserID:      user.ID,
	// 	RefreshHash: utils.HashToken(newRefresh),
	// 	ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
	// 	IP:          ip,
	// 	UserAgent:   userAgent,
	// }
	// err = s.UserRepo.CreateRefreshTokenTx(ctx, tx, &newRt)
	// if err != nil {
	// 	return "", "", errors.New("failed to store new refresh token")
	// }
	// --- END REVOKE + INSERT METHOD ---

	// --- IN-PLACE ROTATION METHOD (Less DB bloat, easy active session tracking) ---
	newRefresh, err := utils.GenerateRefreshToken()
	if err != nil {
		return "", "", err
	}

	err = qtx.RotateRefreshToken(ctx, rt.ID, utils.HashToken(newRefresh), time.Now().Add(7*24*time.Hour), ip, userAgent)
	if err != nil {
		return "", "", errors.New("failed to rotate refresh token")
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", errors.New("failed to commit transaction")
	}

	// create jwt
	token, err := utils.GenerateJWT(user.ID, user.Role, rt.ID)
	if err != nil {
		return "", "", err
	}

	// return jwt token, refresh token, error
	return token, newRefresh, nil
}

func (s *AuthService) LogoutUser(ctx context.Context, tokenString, refreshToken string) error {
	// 1. Blacklist JWT
	err := s.UserRepo.InvalidateToken(ctx, tokenString, 24*time.Hour)
	if err != nil {
		return errors.New("failed to logout")
	}

	// 2. Revoke refresh token in DB if provided
	if refreshToken != "" {
		hash := utils.HashToken(refreshToken)
		_ = s.UserRepo.RevokeRefreshToken(ctx, hash)
	}

	return nil
}

func (s *AuthService) GetSessions(ctx context.Context, userID int64, currentSessionID int64) (models.SessionListResponse, error) {
	rts, err := s.UserRepo.GetUserSessions(ctx, userID)
	if err != nil {
		return models.SessionListResponse{}, err
	}

	var sessions []models.Session
	for _, rt := range rts {
		// Just passing the raw user agent. Parsing will happen in the controller if needed or can be added here.
		lastActive := rt.CreatedAt.Format(time.RFC3339)
		if rt.LastUsedAt != nil {
			lastActive = rt.LastUsedAt.Format(time.RFC3339)
		}

		sessions = append(sessions, models.Session{
			ID:              rt.ID,
			Device:          "Unknown Device",   // Placeholder, will parse later
			Browser:         rt.UserAgent,       // We will parse this in controller
			Location:        "Unknown Location", // Placeholder
			IPAddress:       rt.IP,
			LastActive:      lastActive,
			IsCurrentDevice: rt.ID == currentSessionID,
		})
	}

	return models.SessionListResponse{Sessions: sessions}, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, sessionID int64, userID int64) error {
	// Need to ensure the session belongs to the user
	query := `UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL`
	cmd, err := s.UserRepo.DB.Exec(ctx, query, sessionID, userID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return errors.New("session not found or not owned by user")
	}
	return nil
}

func (s *AuthService) RevokeAllOtherSessions(ctx context.Context, userID int64, currentSessionID int64) error {
	return s.UserRepo.RevokeAllOtherSessions(ctx, userID, currentSessionID)
}
