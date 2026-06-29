package pb

import (
	"context"

	"google.golang.org/grpc"
)

type UserServiceServer interface {
	mustEmbedUnimplementedUserServiceServer()
	GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
	CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error)
}

type UnimplementedUserServiceServer struct{}

func (UnimplementedUserServiceServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return nil, nil
}

func (UnimplementedUserServiceServer) CreateUser(context.Context, *CreateUserRequest) (*CreateUserResponse, error) {
	return nil, nil
}

func (UnimplementedUserServiceServer) mustEmbedUnimplementedUserServiceServer() {}

type GetUserRequest struct{}
type GetUserResponse struct{}
type CreateUserRequest struct{}
type CreateUserResponse struct{}

var UserService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "demo.v1.UserService",
	Methods: []grpc.MethodDesc{
		{MethodName: "GetUser"},
		{MethodName: "CreateUser"},
	},
}

func RegisterUserServiceServer(s grpc.ServiceRegistrar, srv UserServiceServer) {
	s.RegisterService(&UserService_ServiceDesc, srv)
}
