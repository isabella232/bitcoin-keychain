//go:build !integration
// +build !integration

package keystore

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/ledgerhq/bitcoin-keychain/pb/bitcoin"
	"github.com/ledgerhq/bitcoin-keychain/pkg/chaincfg"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type mockBitcoinClient struct{}

func (c mockBitcoinClient) ValidateAddress(
	ctx context.Context,
	in *bitcoin.ValidateAddressRequest,
	opts ...grpc.CallOption,
) (*bitcoin.ValidateAddressResponse, error) {
	return &bitcoin.ValidateAddressResponse{
		Address: in.Address,
		IsValid: true,
	}, nil
}

func (c mockBitcoinClient) DeriveExtendedKey(
	ctx context.Context,
	in *bitcoin.DeriveExtendedKeyRequest,
	opts ...grpc.CallOption,
) (*bitcoin.DeriveExtendedKeyResponse, error) {
	extendedKey := in.ExtendedKey
	publicKey := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	chainCode := []byte{0xCA, 0xFE, 0xBA, 0xBE}

	for _, index := range in.Derivation {
		extendedKey += "->" + strconv.Itoa(int(index))
		publicKey = append(publicKey, byte(index))
		chainCode = append(chainCode, byte(index))
	}

	return &bitcoin.DeriveExtendedKeyResponse{
		ExtendedKey: extendedKey,
		PublicKey:   publicKey,
		ChainCode:   chainCode,
	}, nil
}

func (c mockBitcoinClient) GetAccountExtendedKey(
	ctx context.Context,
	in *bitcoin.GetAccountExtendedKeyRequest,
	opts ...grpc.CallOption,
) (*bitcoin.GetAccountExtendedKeyResponse, error) {
	return &bitcoin.GetAccountExtendedKeyResponse{
		ExtendedKey: "xpub1111",
	}, nil
}

func (c mockBitcoinClient) EncodeAddress(
	ctx context.Context,
	in *bitcoin.EncodeAddressRequest,
	opts ...grpc.CallOption,
) (*bitcoin.EncodeAddressResponse, error) {
	net, err := networkFromChainParams(in.ChainParams)
	if err != nil {
		panic(err)
	}

	scheme, err := schemeFromProtoEncoding(in.Encoding)
	if err != nil {
		panic(err)
	}

	addr := fmt.Sprintf("%s-%s-%s",
		hex.EncodeToString(in.PublicKey), scheme, net)

	return &bitcoin.EncodeAddressResponse{
		Address: addr,
	}, nil
}

func NewMockInMemoryKeystore() Keystore {
	return &InMemoryKeystore{
		db:     schema{},
		client: mockBitcoinClient{},
	}
}

func TestInMemoryKeystore_UUID(t *testing.T) {
	test := struct {
		name          string
		extendedKey   string
		fromChainCode *FromChainCode
		scheme        Scheme
		network       chaincfg.Network
		index         uint32
		info          string
	}{
		name:        "native segwit",
		extendedKey: "xpub1111",
		scheme:      BIP84,
		network:     chaincfg.BitcoinMainnet,
		index:       1,
		info:        "",
	}

	keystore := NewMockInMemoryKeystore()

	info1, err := keystore.Create(
		test.extendedKey, test.fromChainCode, test.scheme, test.network,
		DefaultLookaheadSize, test.index, test.info,
	)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	info2, err := keystore.Create(
		test.extendedKey, test.fromChainCode, test.scheme, test.network,
		DefaultLookaheadSize, test.index, test.info,
	)
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if !reflect.DeepEqual(info1, info2) {
		t.Fatalf("UUID must be the same")
	}

	info3, err := keystore.Create(
		"xpub2222", test.fromChainCode, test.scheme, test.network,
		DefaultLookaheadSize, test.index, test.info,
	)

	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}

	if reflect.DeepEqual(info1, info3) {
		t.Fatalf("UUID must be different")
	}
}

func TestInMemoryKeystore_GetCreate(t *testing.T) {
	tests := []struct {
		name          string
		extendedKey   string
		fromChainCode *FromChainCode
		scheme        Scheme
		network       chaincfg.Network
		index         uint32
		info          string
		want          KeychainInfo
		wantErr       error
	}{
		{
			name:        "native segwit",
			extendedKey: "xpub1111",
			scheme:      BIP84,
			network:     chaincfg.BitcoinMainnet,
			index:       1,
			info:        "",
			want: KeychainInfo{
				ExternalDescriptor:          "wpkh(xpub1111/0/*)",
				InternalDescriptor:          "wpkh(xpub1111/1/*)",
				ExtendedPublicKey:           "xpub1111",
				SLIP32ExtendedPublicKey:     "xpub1111",
				ExternalXPub:                "xpub1111->0",
				MaxConsecutiveExternalIndex: 0,
				InternalXPub:                "xpub1111->1",
				MaxConsecutiveInternalIndex: 0,
				LookaheadSize:               20,
				Scheme:                      "BIP84",
				Network:                     chaincfg.BitcoinMainnet,
				AccountIndex:                1,
				Metadata:                    "",
			},
		},
		{
			name:          "native segwit (from chain code)",
			fromChainCode: &FromChainCode{},
			scheme:        BIP84,
			network:       chaincfg.BitcoinMainnet,
			index:         2,
			info:          "random info",
			want: KeychainInfo{
				ExternalDescriptor:          "wpkh(xpub1111/0/*)",
				InternalDescriptor:          "wpkh(xpub1111/1/*)",
				ExtendedPublicKey:           "xpub1111",
				SLIP32ExtendedPublicKey:     "xpub1111",
				ExternalXPub:                "xpub1111->0",
				MaxConsecutiveExternalIndex: 0,
				InternalXPub:                "xpub1111->1",
				MaxConsecutiveInternalIndex: 0,
				LookaheadSize:               20,
				Scheme:                      "BIP84",
				Network:                     chaincfg.BitcoinMainnet,
				AccountIndex:                2,
				Metadata:                    "random info",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keystore := NewMockInMemoryKeystore()

			gotInfo, err := keystore.Create(
				tt.extendedKey, tt.fromChainCode, tt.scheme, tt.network,
				DefaultLookaheadSize, tt.index, tt.info,
			)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("Create() unexpected error: %v", err)
			}

			if err == nil && tt.wantErr != nil {
				t.Fatalf("Create() gotInfo no error, want '%v'",
					tt.wantErr)
			}

			if err != nil && errors.Cause(err) != tt.wantErr {
				t.Fatalf("Create() gotInfo error '%v', want '%v'",
					err, tt.wantErr)
			}

			// Do not compare UUIDs, since it random.
			tt.want.ID = gotInfo.ID

			if !reflect.DeepEqual(gotInfo, tt.want) {
				t.Fatalf("Create() gotInfo = '%v', want = '%v'",
					gotInfo, tt.want)
			}

			gotDB, dbErr := keystore.Get(gotInfo.ID)
			if dbErr != nil && err == nil {
				t.Fatalf("Get() unexpected error: %v", dbErr)
			}

			if dbErr == nil && err != nil {
				t.Fatalf("Get() gotInfo no error, want '%v'",
					ErrKeychainNotFound)
			}

			if dbErr != nil && errors.Cause(dbErr) != ErrKeychainNotFound {
				t.Fatalf("Get() gotInfo error '%v', want '%v'",
					dbErr, ErrKeychainNotFound)
			}

			if !reflect.DeepEqual(gotDB, tt.want) {
				t.Fatalf("Get() gotInfo = '%v', want = '%v'",
					gotDB, tt.want)
			}
		})
	}
}

func TestInMemoryKeystore_GetFreshAddress(t *testing.T) {
	tests := []struct {
		name        string
		extendedKey string
		scheme      Scheme
		change      Change
		network     chaincfg.Network
		want        *AddressInfo
		wantErr     error
	}{
		{
			name:        "p2pkh mainnet",
			extendedKey: "xpub1111",
			scheme:      BIP84,
			change:      External,
			network:     chaincfg.BitcoinMainnet,
			want:        &AddressInfo{Address: "deadbeef00-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 0}, Change: External},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keystore := NewMockInMemoryKeystore()
			info, err := keystore.Create(
				tt.extendedKey, nil, tt.scheme, tt.network, DefaultLookaheadSize,
				1, "",
			)
			if err != nil {
				panic(err)
			}

			got, err := keystore.GetFreshAddress(info.ID, tt.change)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("GetFreshAddress() unexpected error: %v", err)
			}

			if err == nil && tt.wantErr != nil {
				t.Fatalf("GetFreshAddress() got no error, want '%v'",
					tt.wantErr)
			}

			if err != nil && errors.Cause(err) != tt.wantErr {
				t.Fatalf("GetFreshAddress() got error '%v', want '%v'",
					err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetFreshAddress() got = '%v', want = '%v'",
					got, tt.want)
			}
		})
	}
}

func TestInMemoryKeystore_GetFreshAddresses(t *testing.T) {
	tests := []struct {
		name        string
		extendedKey string
		change      Change
		scheme      Scheme
		network     chaincfg.Network
		size        uint32
		want        []AddressInfo
		wantErr     error
	}{
		{
			name:        "empty",
			extendedKey: "xpub1111",
			scheme:      BIP84,
			change:      External,
			network:     chaincfg.BitcoinMainnet,
			size:        0,
			want:        []AddressInfo{},
		},
		{
			name:        "p2pkh mainnet multi",
			extendedKey: "xpub1111",
			scheme:      BIP84,
			change:      External,
			network:     chaincfg.BitcoinMainnet,
			size:        5,
			want: []AddressInfo{
				{Address: "deadbeef00-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 0}, Change: External},
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External},
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 2}, Change: External},
				{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 3}, Change: External},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 4}, Change: External},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keystore := NewMockInMemoryKeystore()
			info, err := keystore.Create(
				tt.extendedKey, nil, tt.scheme, tt.network, DefaultLookaheadSize,
				1, "",
			)
			if err != nil {
				panic(err)
			}

			got, err := keystore.GetFreshAddresses(info.ID, tt.change, tt.size)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("GetFreshAddresses() unexpected error: %v", err)
			}

			if err == nil && tt.wantErr != nil {
				t.Fatalf("GetFreshAddresses() got no error, want '%v'",
					tt.wantErr)
			}

			if err != nil && errors.Cause(err) != tt.wantErr {
				t.Fatalf("GetFreshAddresses() got error '%v', want '%v'",
					err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetFreshAddresses() got = '%v', want = '%v'",
					got, tt.want)
			}
		})
	}
}

func TestInMemoryKeystore_MarkPathAsUsed(t *testing.T) {
	keystore := NewMockInMemoryKeystore()

	info, err := keystore.Create(
		"xpub1111", nil, BIP84, chaincfg.BitcoinMainnet, DefaultLookaheadSize, 1, "")
	if err != nil {
		panic(err)
	}

	workflow := []struct {
		name               string
		path               DerivationPath
		change             Change
		size               uint32
		wantFreshAddresses []AddressInfo
		wantFreshAddress   *AddressInfo
		wantErr            error
	}{
		{
			name:   "mark 0/0 as used",
			path:   DerivationPath{0, 0},
			change: External,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External}, // should have no gaps
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 2}, Change: External},
				{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 3}, Change: External},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 4}, Change: External},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 5}, Change: External},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External},
		},
		{
			name:   "mark 0/2 as used",
			path:   DerivationPath{0, 2}, // introduce a gap
			change: External,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External}, // should detect the gap
				{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 3}, Change: External},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 4}, Change: External},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 5}, Change: External},
				{Address: "deadbeef06-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 6}, Change: External},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External},
		},
		{
			name:   "mark 0/1 as used",
			path:   DerivationPath{0, 1}, // fill the gap
			change: External,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 3}, Change: External}, // should have no gaps
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 4}, Change: External},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 5}, Change: External},
				{Address: "deadbeef06-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 6}, Change: External},
				{Address: "deadbeef07-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 7}, Change: External},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 3}, Change: External},
		},
		{
			// internal chain should be unaffected by previous mutations
			name:   "mark 1/0 as used",
			path:   DerivationPath{1, 0},
			change: Internal,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 2}, Change: Internal},
				{Address: "deadbeef03-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 3}, Change: Internal},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 4}, Change: Internal},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 5}, Change: Internal},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
		},
		{
			name:   "mark 1/3 as used",
			path:   DerivationPath{1, 3},
			change: Internal,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 2}, Change: Internal},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 4}, Change: Internal},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 5}, Change: Internal},
				{Address: "deadbeef06-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 6}, Change: Internal},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
		},
		{
			name:   "mark 1/6 as used",
			path:   DerivationPath{1, 6},
			change: Internal,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 2}, Change: Internal},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 4}, Change: Internal},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 5}, Change: Internal},
				{Address: "deadbeef07-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 7}, Change: Internal},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 1}, Change: Internal},
		},
		{
			name:   "mark 1/1 as used",
			path:   DerivationPath{1, 1},
			change: Internal,
			size:   5,
			wantFreshAddresses: []AddressInfo{
				{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 2}, Change: Internal},
				{Address: "deadbeef04-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 4}, Change: Internal},
				{Address: "deadbeef05-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 5}, Change: Internal},
				{Address: "deadbeef07-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 7}, Change: Internal},
				{Address: "deadbeef08-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 8}, Change: Internal},
			},
			wantFreshAddress: &AddressInfo{Address: "deadbeef02-BIP84-bitcoin_mainnet", Derivation: DerivationPath{1, 2}, Change: Internal},
		},
	}

	for _, tt := range workflow {
		t.Run(tt.name, func(t *testing.T) {
			if err := keystore.MarkPathAsUsed(info.ID, tt.path); err != nil {
				t.Fatalf("MarkPathAsUsed() unexpected error: %v", err)
			}

			gotBulk, err := keystore.GetFreshAddresses(info.ID, tt.change, tt.size)
			if err != nil {
				t.Fatalf("GetFreshAddresses() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(gotBulk, tt.wantFreshAddresses) {
				t.Fatalf("GetFreshAddresses() got = '%v', want = '%v'",
					gotBulk, tt.wantFreshAddresses)
			}

			got, err := keystore.GetFreshAddress(info.ID, tt.change)
			if err != nil {
				t.Fatalf("GetFreshAddress() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.wantFreshAddress) {
				t.Fatalf("GetFreshAddress() got = '%v', want = '%v'",
					got, tt.wantFreshAddress)
			}
		})
	}
}

func TestInMemoryKeystore_GetAddressesPublicKeys(t *testing.T) {
	tests := []struct {
		name        string
		extendedKey string
		change      Change
		scheme      Scheme
		network     chaincfg.Network
		size        uint32
		derivations []DerivationPath
		want        []string
		wantErr     error
	}{
		{
			name:        "p2pkh mainnet multi (change: external)",
			extendedKey: "xpub1111",
			change:      External,
			scheme:      BIP84,
			network:     chaincfg.BitcoinMainnet,
			size:        5,
			derivations: []DerivationPath{
				{0, 0},
				{0, 1},
				{0, 2},
				{0, 3},
				{0, 4},
			},
			want: []string{
				"deadbeef00",
				"deadbeef01",
				"deadbeef02",
				"deadbeef03",
				"deadbeef04",
			},
		},
		// internal chain should return the same public keys
		{
			name:        "p2pkh mainnet multi (change: internal)",
			extendedKey: "xpub1111",
			change:      Internal,
			scheme:      BIP84,
			network:     chaincfg.BitcoinMainnet,
			size:        5,
			derivations: []DerivationPath{
				{1, 0},
				{1, 1},
				{1, 2},
				{1, 3},
				{1, 4},
			},
			want: []string{
				"deadbeef00",
				"deadbeef01",
				"deadbeef02",
				"deadbeef03",
				"deadbeef04",
			},
		},
		{
			name:        "p2pkh mainnet multi (wrong given derivations)",
			extendedKey: "xpub1111",
			change:      Internal,
			scheme:      BIP84,
			network:     chaincfg.BitcoinMainnet,
			size:        5,
			derivations: []DerivationPath{
				{1, 0},
				{1, 6},
				{1, 7},
				{1, 8},
				{1, 9},
			},
			wantErr: ErrDerivationNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keystore := NewMockInMemoryKeystore()
			info, err := keystore.Create(
				tt.extendedKey, nil, tt.scheme, tt.network, DefaultLookaheadSize, 1, "")
			if err != nil {
				panic(err)
			}

			// Firstly, call get fresh addresses to derive addresses
			_, err = keystore.GetFreshAddresses(info.ID, tt.change, tt.size)
			if err != nil {
				t.Fatalf("GetFreshAddresses() unexpected error: %v", err)
			}

			got, err := keystore.GetAddressesPublicKeys(info.ID, tt.derivations)
			if err != nil && tt.wantErr == nil {
				t.Fatalf("GetAddressesPublicKeys() unexpected error: %v", err)
			}

			if err == nil && tt.wantErr != nil {
				t.Fatalf("GetAddressesPublicKeys() got no error, want '%v'",
					tt.wantErr)
			}

			if err != nil && errors.Cause(err) != tt.wantErr {
				t.Fatalf("GetAddressesPublicKeys() got error '%v', want '%v'",
					err, tt.wantErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("GetAddressesPublicKeys() got = '%v', want = '%v'",
					got, tt.want)
			}
		})
	}
}

func TestInMemoryKeystore_Reset(t *testing.T) {
	keystore := NewMockInMemoryKeystore()

	info, err := keystore.Create(
		"xpub1111", nil, BIP84, chaincfg.BitcoinMainnet, DefaultLookaheadSize, 1, "")
	if err != nil {
		panic(err)
	}

	workflow := []struct {
		name                        string
		path                        DerivationPath
		change                      Change
		wantFreshAddressBeforeReset *AddressInfo
		wantFreshAddressAfterReset  *AddressInfo
		wantErr                     error
	}{
		{
			name:                        "mark 0/0 as used then reset",
			path:                        DerivationPath{0, 0},
			change:                      External,
			wantFreshAddressBeforeReset: &AddressInfo{Address: "deadbeef01-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 1}, Change: External},
			wantFreshAddressAfterReset:  &AddressInfo{Address: "deadbeef00-BIP84-bitcoin_mainnet", Derivation: DerivationPath{0, 0}, Change: External},
		},
	}

	for _, tt := range workflow {
		t.Run(tt.name, func(t *testing.T) {
			if err := keystore.MarkPathAsUsed(info.ID, tt.path); err != nil {
				t.Fatalf("MarkPathAsUsed() unexpected error: %v", err)
			}

			got, err := keystore.GetFreshAddress(info.ID, tt.change)
			if err != nil {
				t.Fatalf("GetFreshAddress() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(got, tt.wantFreshAddressBeforeReset) {
				t.Fatalf("GetFreshAddress() got = '%v', want = '%v'",
					got, tt.wantFreshAddressBeforeReset)
			}

			err = keystore.Reset(info.ID)
			if err != nil {
				t.Fatalf("Reset() unexpected error: %v", err)
			}

			gotAfterReset, err := keystore.GetFreshAddress(info.ID, tt.change)
			if err != nil {
				t.Fatalf("GetFreshAddress() unexpected error: %v", err)
			}

			if !reflect.DeepEqual(gotAfterReset, tt.wantFreshAddressAfterReset) {
				t.Fatalf("GetFreshAddress() got = '%v', want = '%v'",
					gotAfterReset, tt.wantFreshAddressAfterReset)
			}
		})
	}
}
