package service

import (
	"context"
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
	return s.repo.FindByID(ctx, id)
}
