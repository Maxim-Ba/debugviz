package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	userv1 "github.com/Maxim-Ba/debugviz/demo/grpc/gen/pb/user/v1"
	"github.com/Maxim-Ba/debugviz/demo/grpc/internal/server"
)

//debugviz:app name=demo-grpc
//go:generate go run ../../go/cmd/debugviz wire --config debugviz.yaml --write .

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, server.NewUserServer())
	reflection.Register(srv)

	log.Println("demo/grpc listening on :9090")
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
