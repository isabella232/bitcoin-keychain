package grpc

import (
	"context"
	"github.com/ledgerhq/bitcoin-keychain-svc/pb/v1"
	"github.com/ledgerhq/bitcoin-keychain-svc/pkg/keystore"
	"google.golang.org/protobuf/types/known/emptypb"
)

type controller struct {
	store keystore.Keystore
}

func (c controller) CreateKeychain(
	ctx context.Context, request *pb.CreateKeychainRequest,
) (*pb.KeychainInfo, error) {
	panic("implement me")
}

func (c controller) DeleteKeychain(
	ctx context.Context, request *pb.DeleteKeychainRequest,
) (*emptypb.Empty, error) {
	panic("implement me")
}

func (c controller) GetKeychainInfo(
	ctx context.Context, request *pb.GetKeychainInfoRequest,
) (*pb.KeychainInfo, error) {
	panic("implement me")
}

func NewKeychainController() *controller {
	return &controller{
		store: keystore.NewInMemoryKeystore(),
	}
}
