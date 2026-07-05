package service

import (
	"context"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
	"github.com/Maxim-Ba/debugviz/demo/http/internal/repository"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

type ItemService struct {
	repo *repository.ItemRepository
}

func NewItemService(repo *repository.ItemRepository) *ItemService {
	return &ItemService{repo: repo}
}

func (s *ItemService) GetByID(ctx context.Context, id int) (*model.Item, error) {
	var __dv_end func()
	ctx, __dv_end = debugviz.StartSpan(ctx, "service.ItemService.GetByID")
	defer __dv_end()
	return s.repo.FindByID(ctx, id)
}

func (s *ItemService) List(ctx context.Context) ([]model.Item, error) {
	var __dv_end func()
	ctx, __dv_end = debugviz.StartSpan(ctx, "service.ItemService.List")
	defer __dv_end()
	return s.repo.List(ctx)
}
