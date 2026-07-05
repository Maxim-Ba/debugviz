package main

//go:generate protoc --go_out=gen/pb --go_opt=paths=source_relative --go-grpc_out=gen/pb --go-grpc_opt=paths=source_relative --proto_path=proto proto/user/v1/user.proto
