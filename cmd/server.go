package main

import (
	"fmt"
	"net"

	"github.com/ledgerhq/bitcoin-keychain/config"
	controllers "github.com/ledgerhq/bitcoin-keychain/grpc"
	"github.com/ledgerhq/bitcoin-keychain/log"
	pb "github.com/ledgerhq/bitcoin-keychain/pb/keychain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func serve(addr string) {
	conn, err := net.Listen("tcp", addr)
	if err != nil {
		log.WithFields(log.Fields{
			"addr": addr,
		}).Fatal("cannot listen to address")
	}

	s := grpc.NewServer()
	keychainController := controllers.NewKeychainController()
	pb.RegisterKeychainServiceServer(s, keychainController)

	reflection.Register(s)

	if err := s.Serve(conn); err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Fatal("failed to serve")
	}
}

func main() {
	configProvider := config.LoadProvider("bitcoin_keychain")

	var (
		host string
		port int32 = 50052
	)

	host = configProvider.GetString("host")

	if val := configProvider.GetInt32("port"); val != 0 {
		port = val
	}

	addr := fmt.Sprintf("%s:%d", host, port)

	serve(addr)
}
