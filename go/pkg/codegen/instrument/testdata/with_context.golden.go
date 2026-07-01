package service

import (
	"context"
	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
)

type TechService struct {
	repo TechRepository
}

type TechRepository interface {
	FindByID(ctx context.Context, id int) (*Tech, error)
}

type Tech struct {
	ID int
}

func (s *TechService) GetByID(ctx context.Context, id int) (*Tech, error) {
	ctx, __dv_end := debugviz.StartSpan(ctx, "service.TechService.GetByID")
	defer __dv_end()
	return s.repo.FindByID(ctx, id)
}
