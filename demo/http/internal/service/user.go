package service

import (
	"context"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/repository"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	return s.repo.List(ctx)
}

func (s *UserService) Create(ctx context.Context, user model.User) (*model.User, error) {
	return s.repo.Create(ctx, user)
}
