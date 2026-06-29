package service

import (
	"context"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/repository"
)

type ItemService struct {
	repo *repository.ItemRepository
}

func NewItemService(repo *repository.ItemRepository) *ItemService {
	return &ItemService{repo: repo}
}

func (s *ItemService) GetByID(ctx context.Context, id int) (*model.Item, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *ItemService) List(ctx context.Context) ([]model.Item, error) {
	return s.repo.List(ctx)
}
