package main

import (
	"log"
	"net"

	"github.com/Maxim-Ba/debugviz/go/lib/debugviz"
	"google.golang.org/grpc"
)

func main() {
	if err := debugviz.ConfigureFromEnv(); err != nil {
		log.Fatalf("debugviz: %v", err)
	}
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(debugviz.UnaryServerInterceptor()), grpc.ChainStreamInterceptor(debugviz.StreamServerInterceptor()), grpc.MaxRecvMsgSize(1024))
	log.Println("listening")
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
