package grpcdemo

import (
	"log"
	"net"

	"google.golang.org/grpc"

	userv1 "github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters/testdata/grpcdemo/pb"
	"github.com/Maxim-Ba/debugviz/go/pkg/scanner/adapters/testdata/grpcdemo/server"
)

func main() {
	lis, err := net.Listen("tcp", ":19090")
	if err != nil {
		log.Fatal(err)
	}

	srv := grpc.NewServer()
	userv1.RegisterUserServiceServer(srv, server.NewUserServer())
	log.Fatal(srv.Serve(lis))
}
