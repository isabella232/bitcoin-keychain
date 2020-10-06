package grpc

import (
	"context"

	"github.com/ledgerhq/bitcoin-keychain-svc/log"

	pb "github.com/ledgerhq/bitcoin-keychain-svc/pb/keychain"
	"github.com/ledgerhq/bitcoin-keychain-svc/pkg/keystore"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Controller is a type that implements the pb.KeychainServiceServer
// interface.
type Controller struct {
	store keystore.Keystore
}

func (c Controller) CreateKeychain(
	ctx context.Context, request *pb.CreateKeychainRequest,
) (*pb.KeychainInfo, error) {
	net, err := Network(request.Network)
	if err != nil {
		return nil, err
	}

	scheme, err := Scheme(request.Scheme)
	if err != nil {
		return nil, err
	}

	lookaheadSize := uint32(keystore.DefaultLookaheadSize) // default lookahead size
	if s := request.GetLookaheadSize(); s != 0 {
		lookaheadSize = s
	}

	r, err := c.store.Create(
		request.ExtendedPublicKey, scheme, net, lookaheadSize)
	if err != nil {
		return nil, err
	}

	return KeychainInfo(r)
}

func (c Controller) DeleteKeychain(
	ctx context.Context, request *pb.DeleteKeychainRequest,
) (*emptypb.Empty, error) {
	id, err := KeychainID(request.KeychainId)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, c.store.Delete(id)
}

func (c Controller) GetKeychainInfo(
	ctx context.Context, request *pb.GetKeychainInfoRequest,
) (*pb.KeychainInfo, error) {
	id, err := KeychainID(request.KeychainId)
	if err != nil {
		return nil, err
	}

	r, err := c.store.Get(id)
	if err != nil {
		return nil, err
	}

	return KeychainInfo(r)
}

func (c Controller) GetFreshAddresses(
	ctx context.Context, request *pb.GetFreshAddressesRequest,
) (*pb.GetFreshAddressesResponse, error) {
	id, err := KeychainID(request.KeychainId)
	if err != nil {
		return nil, err
	}

	change, err := Change(request.Change)
	if err != nil {
		return nil, err
	}

	addrs, err := c.store.GetFreshAddresses(id, change, request.BatchSize)
	if err != nil {
		return nil, err
	}

	return &pb.GetFreshAddressesResponse{Addresses: addrs}, nil
}

func (c Controller) MarkAddressesAsUsed(
	ctx context.Context, request *pb.MarkAddressesAsUsedRequest,
) (*emptypb.Empty, error) {
	id, err := KeychainID(request.KeychainId)
	if err != nil {
		return nil, err
	}

	for _, addr := range request.Addresses {
		if err := c.store.MarkAddressAsUsed(id, addr); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (c Controller) GetAllObservableAddresses(
	ctx context.Context, request *pb.GetAllObservableAddressesRequest,
) (*pb.GetAllObservableAddressesResponse, error) {
	id, err := KeychainID(request.KeychainId)
	if err != nil {
		log.WithFields(log.Fields{
			"id":    request.KeychainId,
			"error": err,
		}).Error("[grpc] GetAllObservableAddresses: invalid KeychainID")

		return nil, err
	}

	var changeList []keystore.Change
	if request.Change == pb.Change_CHANGE_UNSPECIFIED {
		changeList = []keystore.Change{keystore.External, keystore.Internal}
	} else {
		change, err := Change(request.Change)
		if err != nil {
			log.WithFields(log.Fields{
				"id":     id.String(),
				"change": request.Change.String(),
				"error":  err,
			}).Error("[grpc] GetAllObservableAddresses: invalid Change")

			return nil, err
		}

		changeList = []keystore.Change{change}
	}

	// If the toIndex field is left out in the request payload, we substitute
	// it with a large value so that the max observable range is used instead.
	var to uint32
	if request.GetToIndex() == 0 {
		to = (1 << 31) - 1 // uint32 max
	} else {
		to = request.ToIndex
	}

	var addrs []keystore.AddressInfo

	for _, change := range changeList {
		log.WithFields(log.Fields{
			"id":     id,
			"change": change,
			"range":  []uint32{request.FromIndex, to},
		}).Info("[grpc] GetAllObservableAddresses: get from keystore")

		changeAddrs, err := c.store.GetAllObservableAddresses(
			id, change, request.FromIndex, to)
		if err != nil {
			log.WithFields(log.Fields{
				"id":     id.String(),
				"change": request.Change.String(),
				"error":  err,
			}).Error("[grpc] GetAllObservableAddresses: failed to fetch from keystore")

			return nil, err
		}

		addrs = append(addrs, changeAddrs...)
	}

	var addrInfoList []*pb.AddressInfo

	for _, addrInfo := range addrs {
		addrInfoProto, err := AddressInfoProto(addrInfo)
		if err != nil {
			log.WithFields(log.Fields{
				"id":    id.String(),
				"addr":  addrInfo.Address,
				"error": err,
			}).Error("[grpc] GetAllObservableAddresses: invalid AddressInfo")

			return nil, err
		}

		addrInfoList = append(addrInfoList, addrInfoProto)
	}

	log.WithFields(log.Fields{
		"id":     id.String(),
		"num":    len(addrInfoList),
		"change": changeList,
	}).Info("[grpc] GetAllObservableAddresses: successful")

	return &pb.GetAllObservableAddressesResponse{Addresses: addrInfoList}, nil
}

// NewKeychainController returns a new instance of a Controller struct that
// implements the pb.KeychainServiceServer interface.
func NewKeychainController() *Controller {
	return &Controller{
		store: keystore.NewInMemoryKeystore(),
	}
}
