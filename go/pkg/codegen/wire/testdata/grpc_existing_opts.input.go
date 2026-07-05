package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer(grpc.MaxRecvMsgSize(1024))
	log.Println("listening")
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
