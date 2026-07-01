package repository

import (
	"context"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

type ItemRepository struct{}

func (r *ItemRepository) FindByID(ctx context.Context, id int) error {
	ctx, __dv_end := debugviz.StartSpan(ctx, "repository.ItemRepository.FindByID")
	defer __dv_end()
	return nil
}
