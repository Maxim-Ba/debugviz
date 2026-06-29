package repository

import (
	"context"
	"errors"

	"github.com/Maxim-Ba/debugviz/demo/http/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository struct {
	users map[int]model.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: map[int]model.User{
			1: {ID: 1, Name: "Alice", Email: "alice@example.com"},
			2: {ID: 2, Name: "Bob", Email: "bob@example.com"},
		},
	}
}

func (r *UserRepository) FindByID(_ context.Context, id int) (*model.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	return &user, nil
}

func (r *UserRepository) List(_ context.Context) ([]model.User, error) {
	result := make([]model.User, 0, len(r.users))
	for _, user := range r.users {
		result = append(result, user)
	}
	return result, nil
}

func (r *UserRepository) Create(_ context.Context, user model.User) (*model.User, error) {
	nextID := len(r.users) + 1
	for {
		if _, exists := r.users[nextID]; !exists {
			break
		}
		nextID++
	}
	user.ID = nextID
	r.users[nextID] = user
	return &user, nil
}
