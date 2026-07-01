package repository

import (
	"context"
)

type ItemRepository struct{}

func (r *ItemRepository) FindByID(_ context.Context, id int) error {
	return nil
}
