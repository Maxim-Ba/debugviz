package v1

import (
	"context"

	"google.golang.org/grpc"
)

type UserServiceServer interface {
	mustEmbedUnimplementedUserServiceServer()
	GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error)
	ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error)
	DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error)
}

type UnimplementedUserServiceServer struct{}

func (UnimplementedUserServiceServer) GetUser(context.Context, *GetUserRequest) (*GetUserResponse, error) {
	return nil, nil
}

func (UnimplementedUserServiceServer) ListUsers(context.Context, *ListUsersRequest) (*ListUsersResponse, error) {
	return nil, nil
}

func (UnimplementedUserServiceServer) DeleteUser(context.Context, *DeleteUserRequest) (*DeleteUserResponse, error) {
	return nil, nil
}

func (UnimplementedUserServiceServer) mustEmbedUnimplementedUserServiceServer() {}

type GetUserRequest struct{}
type GetUserResponse struct{}
type ListUsersRequest struct{}
type ListUsersResponse struct{}
type DeleteUserRequest struct{}
type DeleteUserResponse struct{}

var UserService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "user.v1.UserService",
	HandlerType: (*UserServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetUser"},
		{MethodName: "ListUsers"},
		{MethodName: "DeleteUser"},
	},
	Streams: []grpc.StreamDesc{},
}

func RegisterUserServiceServer(s grpc.ServiceRegistrar, srv UserServiceServer) {
	s.RegisterService(&UserService_ServiceDesc, srv)
}
