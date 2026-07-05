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

	srv := grpc.NewServer()
	log.Println("listening on :9090")
	if err := srv.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
