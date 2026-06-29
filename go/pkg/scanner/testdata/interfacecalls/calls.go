package interfacecalls

import "context"

type Store interface {
	FindByID(ctx context.Context, id int) (*Item, error)
}

type Item struct {
	ID int
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context, id int) (*Item, error) {
	return s.store.FindByID(ctx, id)
}

type MemoryStore struct{}

func (MemoryStore) FindByID(ctx context.Context, id int) (*Item, error) {
	return &Item{ID: id}, nil
}
