package services

import (
	"context"

	"github.com/gottatouchsomegrass/url/app/models"
	"github.com/gottatouchsomegrass/url/app/repositories"
)

type AdminService struct {
	UserRepo *repositories.UserQuery
}

func NewAdminService(uq *repositories.UserQuery) *AdminService {
	return &AdminService{
		UserRepo: uq,
	}
}

func (s *AdminService) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, int, error) {
	users, err := s.UserRepo.GetAllUsers(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.UserRepo.CountAllUsers(ctx)
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
