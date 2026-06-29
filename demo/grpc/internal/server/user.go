package server

import (
	"context"

	userv1 "github.com/Maxim-Ba/debugviz/demo/grpc/gen/pb/user/v1"
)

type UserServer struct {
	userv1.UnimplementedUserServiceServer
}

func NewUserServer() *UserServer {
	return &UserServer{}
}

func (s *UserServer) GetUser(_ context.Context, _ *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return &userv1.GetUserResponse{}, nil
}

func (s *UserServer) ListUsers(_ context.Context, _ *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	return &userv1.ListUsersResponse{}, nil
}

func (s *UserServer) DeleteUser(_ context.Context, _ *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	return &userv1.DeleteUserResponse{}, nil
}
