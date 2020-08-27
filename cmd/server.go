package main

import (
	"fmt"
	"net"

	controllers "github.com/ledgerhq/bitcoin-keychain-svc/grpc"
	"github.com/ledgerhq/bitcoin-keychain-svc/log"
	"github.com/ledgerhq/bitcoin-keychain-svc/pb/v1"
	"google.golang.org/grpc"
)

func serve() {
	addr := fmt.Sprintf(":%d", 50052)

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Cannot listen to address %s", addr)
	}

	s := grpc.NewServer()
	keychainController := controllers.NewKeychainController()
	pb.RegisterKeychainServiceServer(s, keychainController)

	if err := s.Serve(conn); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

func main() {
	serve()
}
