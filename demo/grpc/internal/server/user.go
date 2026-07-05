package server

import (
	"context"

	userv1 "github.com/Maxim-Ba/debugviz/demo/grpc/gen/pb/user/v1"
	"github.com/Maxim-Ba/debugviz/demo/grpc/internal/service"
)

type UserServer struct {
	userv1.UnimplementedUserServiceServer
	svc *service.UserService
}

func NewUserServer() *UserServer {
	return &UserServer{svc: service.NewUserService()}
}

func (s *UserServer) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return s.svc.GetUser(ctx, req)
}

func (s *UserServer) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	return s.svc.ListUsers(ctx, req)
}

func (s *UserServer) DeleteUser(ctx context.Context, req *userv1.DeleteUserRequest) (*userv1.DeleteUserResponse, error) {
	return s.svc.DeleteUser(ctx, req)
}
