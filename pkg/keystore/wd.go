package keystore

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/ledgerhq/bitcoin-keychain/pb/bitcoin"
	"github.com/ledgerhq/bitcoin-keychain/pkg/chaincfg"
)

// WDKeystore implement the Keystore interface to write on "wallet daemon" redis
// database, also known as "user pref"

type WDKeystore struct {
	baseRedisKeystore
}

func NewWDKeystore(redisOpts *redis.Options) (*WDKeystore, error) {
	rdb := redis.NewClient(redisOpts)

	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		return nil, fmt.Errorf("Pinging redis failed: %w", err)
	}

	baseKeystore := baseRedisKeystore{
		db:     rdb,
		client: bitcoin.NewBitcoinClient(),
	}

	return &WDKeystore{baseKeystore}, nil
}

func (s *WDKeystore) Delete(id uuid.UUID) error {
	var meta Meta

	err := get(s.db, id.String(), &meta)

	if err != nil {
		return ErrKeychainNotFound
	}

	var addresses []AddressInfo

	for addr, path := range meta.Addresses {
		addrInfo := AddressInfo{
			Address:    addr,
			Derivation: path,
		}
		addresses = append(addresses, addrInfo)
	}

	redistx := newRedisTransaction(s.db)
	if err := s.deleteAddresses(redistx, meta.Main, addresses); err != nil {
		return err
	}

	if err := s.deleteState(redistx, meta.Main); err != nil {
		return err
	}

	if err := redistx.del(id.String()); err != nil {
		return err
	}

	return redistx.exec()
}

func (s *WDKeystore) Reset(id uuid.UUID) error {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return ErrKeychainNotFound
	}

	var addresses []AddressInfo

	for addr, path := range meta.Addresses {
		addrInfo := AddressInfo{
			Address:    addr,
			Derivation: path,
			Change:     External,
		}
		addresses = append(addresses, addrInfo)
	}

	redistx := newRedisTransaction(s.db)
	if err := s.deleteAddresses(redistx, meta.Main, addresses); err != nil {
		return err
	}
	if err := s.deleteState(redistx, meta.Main); err != nil {
		return err
	}

	meta.ResetKeychainMeta()

	if err := redistx.set(id.String(), meta); err != nil {
		return err
	}

	return redistx.exec()
}

func (s *WDKeystore) GetFreshAddress(id uuid.UUID, change Change) (*AddressInfo, error) {
	addrs, err := s.GetFreshAddresses(id, change, 1)
	if err != nil {
		return nil, err
	}
	return &addrs[0], err
}

func (s *WDKeystore) GetFreshAddresses(id uuid.UUID, change Change, size uint32) ([]AddressInfo, error) {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return []AddressInfo{}, ErrKeychainNotFound
	}

	addrs, err := meta.keystoreGetFreshAddresses(s.client, change, size)
	if err != nil {
		return addrs, err
	}

	redistx := newRedisTransaction(s.db)

	err = s.updateAddresses(redistx, meta.Main, addrs)
	if err != nil {
		return []AddressInfo{}, err
	}

	err = s.updateState(redistx, meta.Main)
	if err != nil {
		return []AddressInfo{}, err
	}

	if err := redistx.set(id.String(), meta); err != nil {
		return []AddressInfo{}, err
	}

	if err := redistx.exec(); err != nil {
		return []AddressInfo{}, err
	}

	return addrs, nil
}

func (s *WDKeystore) MarkPathAsUsed(id uuid.UUID, path DerivationPath) error {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return ErrKeychainNotFound
	}

	err = meta.keystoreMarkPathAsUsed(path)
	if err != nil {
		return err
	}

	redistx := newRedisTransaction(s.db)

	err = redistx.set(id.String(), meta)
	if err != nil {
		return err
	}

	err = s.updateState(redistx, meta.Main)
	if err != nil {
		return err
	}

	return redistx.exec()
}

func (s *WDKeystore) GetAllObservableAddresses(
	id uuid.UUID, change Change, fromIndex uint32, toIndex uint32,
) ([]AddressInfo, error) {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return nil, ErrKeychainNotFound
	}

	addrs, err := meta.keystoreGetAllObservableAddresses(
		s.client, change, fromIndex, toIndex,
	)
	if err != nil {
		return nil, err
	}

	redistx := newRedisTransaction(s.db)

	err = redistx.set(id.String(), meta)
	if err != nil {
		return nil, err
	}

	err = s.updateState(redistx, meta.Main)
	if err != nil {
		return nil, err
	}

	err = s.updateAddresses(redistx, meta.Main, addrs)
	if err != nil {
		return nil, err
	}

	err = redistx.exec()
	if err != nil {
		return nil, err
	}

	return addrs, nil
}

func (s *WDKeystore) MarkAddressAsUsed(id uuid.UUID, address string) error {
	return keystoreMarkAddressAsUsed(s, id, address)
}

func (s *WDKeystore) updateState(redistx *redisTransaction, keychainInfo KeychainInfo) error {
	wdkey, err := keychainInfoToWDKey(keychainInfo)
	if err != nil {
		return err
	}
	stateKey, stateValue, err := keychainInfoToStateKV(wdkey, keychainInfo)
	if err != nil {
		return err
	}

	return redistx.set(stateKey, stateValue)
}

func (s *WDKeystore) deleteState(redistx *redisTransaction, keychainInfo KeychainInfo) error {
	wdkey, err := keychainInfoToWDKey(keychainInfo)
	if err != nil {
		return err
	}
	stateKey, _, err := keychainInfoToStateKV(wdkey, keychainInfo)
	if err != nil {
		return err
	}

	return redistx.del(stateKey)
}

func (s *WDKeystore) updateAddresses(redistx *redisTransaction, keychainInfo KeychainInfo, addrs []AddressInfo) error {
	wdkey, err := keychainInfoToWDKey(keychainInfo)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		kv, err := wdValues(wdkey, addr)
		if err != nil {
			return ErrKeychainNotFound
		}
		for k, v := range kv {
			if err := redistx.set(k, v); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *WDKeystore) deleteAddresses(redistx *redisTransaction, keychainInfo KeychainInfo, addrs []AddressInfo) error {
	wdkey, err := keychainInfoToWDKey(keychainInfo)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		kv, err := wdValues(wdkey, addr)
		if err != nil {
			return ErrKeychainNotFound
		}
		for k := range kv {
			if err := redistx.del(k); err != nil {
				return err
			}
		}
	}

	return nil
}

type WdKey struct {
	Prefix     string `json:"prefix"`
	Workspace  string `json:"workspace"`
	WalletType string `json:"wallet_type"`
	Index      uint32 `json:"index"`
}

func keychainInfoToWDKey(keychainInfo KeychainInfo) (WdKey, error) {
	prefix := ""
	workspace := ""
	if len(keychainInfo.Metadata) > 0 {
		split := strings.Split(keychainInfo.Metadata, ":")
		if len(split) != 2 {
			return WdKey{}, fmt.Errorf("cannot parse 'info' %s", keychainInfo.Metadata)
		}
		prefix = split[0]
		workspace = split[1]
	}

	walletType, err := keychainInfoToWalletType(keychainInfo)
	if err != nil {
		return WdKey{}, err
	}

	return WdKey{
		Prefix:     prefix,
		Workspace:  workspace,
		WalletType: walletType,
		Index:      keychainInfo.AccountIndex,
	}, nil
}

func wdValues(wdkey WdKey, addr AddressInfo) (map[string]string, error) {
	ret := make(map[string]string)
	ns := fmt.Sprintf("core:user-preferences:%s:%s:", wdkey.Prefix, wdkey.Workspace)

	prefix := fmt.Sprintf("poolwallet_%saccount_%d", wdkey.WalletType, wdkey.Index)
	addrKey := fmt.Sprintf("%saddress:%s", prefix, addr.Address)
	addrValue := fmt.Sprintf("%d/%d", addr.Derivation[0], addr.Derivation[1])
	encodedAddrKey := base64.StdEncoding.EncodeToString([]byte(addrKey))
	encodedAddrValue := base64.StdEncoding.EncodeToString([]byte(addrValue))

	ret[ns+encodedAddrKey] = encodedAddrValue

	pathKey := fmt.Sprintf("%spath:%d/%d", prefix, addr.Derivation[0], addr.Derivation[1])
	pathValue := addr.Address
	encodedPathKey := base64.StdEncoding.EncodeToString([]byte(pathKey))
	encodedPathValue := base64.StdEncoding.EncodeToString([]byte(pathValue))

	ret[ns+encodedPathKey] = encodedPathValue

	return ret, nil
}

func keychainInfoToStateKV(wdkey WdKey, keychainInfo KeychainInfo) (string, string, error) {
	ns := fmt.Sprintf("core:user-preferences:%s:%s:", wdkey.Prefix, wdkey.Workspace)
	prefix := fmt.Sprintf("poolwallet_%saccount_%d", wdkey.WalletType, wdkey.Index)
	stateKey := prefix + "state"
	encodedStateKey := base64.StdEncoding.EncodeToString([]byte(stateKey))
	encodedStateValue, err := EncodeKeychainState(keychainInfoToWdState(keychainInfo))
	if err != nil {
		return "", "", err
	}

	return ns + encodedStateKey, encodedStateValue, nil
}

func keychainInfoToWdState(keychainInfo KeychainInfo) WDKeychainState {
	maxConsecutiveChangeIndex := keychainInfo.MaxConsecutiveInternalIndex
	maxConsecutiveReceiveIndex := keychainInfo.MaxConsecutiveExternalIndex
	nonConsecutiveChangeIndexes := make(map[uint32]bool)
	for _, v := range keychainInfo.NonConsecutiveInternalIndexes {
		nonConsecutiveChangeIndexes[v] = true
	}
	nonConsecutiveReceiveIndexes := make(map[uint32]bool)
	for _, v := range keychainInfo.NonConsecutiveExternalIndexes {
		nonConsecutiveReceiveIndexes[v] = true
	}

	return WDKeychainState{
		maxConsecutiveChangeIndex:    maxConsecutiveChangeIndex,
		maxConsecutiveReceiveIndex:   maxConsecutiveReceiveIndex,
		nonConsecutiveChangeIndexes:  nonConsecutiveChangeIndexes,
		nonConsecutiveReceiveIndexes: nonConsecutiveReceiveIndexes,
		empty:                        false,
	}
}

func keychainInfoToWalletType(keychainInfo KeychainInfo) (string, error) {
	switch keychainInfo.Network {
	case chaincfg.LitecoinMainnet:
		return "litecoin", nil
	case chaincfg.BitcoinMainnet:
		switch keychainInfo.Scheme {
		case BIP44:
			return "bitcoin", nil
		case BIP49:
			// XXX: bip49 is not supported in vault, this value will never be
			// seen in redis, but it is useful for lama test suite
			return "bitcoin_segwit", nil
		case BIP84:
			return "bitcoin_native_segwit", nil
		}
	case chaincfg.BitcoinTestnet3:
		switch keychainInfo.Scheme {
		case BIP44:
			return "bitcoin_testnet", nil
		case BIP49:
			return "bitcoin_testnet_segwit", nil
		case BIP84:
			return "bitcoin_testnet_native_segwit", nil
		}
	}

	return "", fmt.Errorf("unknown network %s and scheme %s",
		keychainInfo.Network, keychainInfo.Scheme)
}
