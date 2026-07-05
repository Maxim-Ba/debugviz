package service

import (
	"context"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/repository"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

type UserService struct {
	repo *repository.UserRepository
}

func NewUserService(repo *repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id int) (*model.User, error) {
	var __dv_end func()
	ctx, __dv_end = debugviz.StartSpan(ctx, "service.UserService.GetByID")
	defer __dv_end()
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]model.User, error) {
	var __dv_end func()
	ctx, __dv_end = debugviz.StartSpan(ctx, "service.UserService.List")
	defer __dv_end()
	return s.repo.List(ctx)
}

func (s *UserService) Create(ctx context.Context, user model.User) (*model.User, error) {
	var __dv_end func()
	ctx, __dv_end = debugviz.StartSpan(ctx, "service.UserService.Create")
	defer __dv_end()
	return s.repo.Create(ctx, user)
}
