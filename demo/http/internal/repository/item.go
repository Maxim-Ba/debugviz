package repository

import (
	"context"
	"errors"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

var ErrItemNotFound = errors.New("item not found")

type ItemRepository struct {
	items map[int]model.Item
}

func NewItemRepository() *ItemRepository {
	return &ItemRepository{
		items: map[int]model.Item{
			1: {ID: 1, Name: "Widget", Description: "A useful widget", Price: 9.99},
			2: {ID: 2, Name: "Gadget", Description: "A handy gadget", Price: 19.99},
		},
	}
}

func (r *ItemRepository) FindByID(ctx context.Context, id int) (*model.Item, error) {
	_, __dv_end := debugviz.StartSpan(ctx, "repository.ItemRepository.FindByID")
	defer __dv_end()
	item, ok := r.items[id]
	if !ok {
		return nil, ErrItemNotFound
	}
	return &item, nil
}

func (r *ItemRepository) List(ctx context.Context) ([]model.Item, error) {
	_, __dv_end := debugviz.StartSpan(ctx, "repository.ItemRepository.List")
	defer __dv_end()
	result := make([]model.Item, 0, len(r.items))
	for _, item := range r.items {
		result = append(result, item)
	}
	return result, nil
}
