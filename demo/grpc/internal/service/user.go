package service

import (
	"context"

	userv1 "github.com/Maxim-Ba/debugviz/demo/grpc/gen/pb/user/v1"
)

type UserService struct{}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) GetUser(ctx context.Context, _ *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	_ = ctx
	return &userv1.GetUserResponse{}, nil
}

func (s *UserService) ListUsers(ctx context.Context, _ *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	_ = ctx
	return &userv1.ListUsersResponse{}, nil
}

func (s *UserService) DeleteUser(ctx context.Context, _ *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	_ = ctx
	return &userv1.DeleteUserResponse{}, nil
}
