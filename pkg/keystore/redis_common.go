package keystore

import (
	"context"
	"encoding/json"
	"fmt"

	"reflect"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/ledgerhq/bitcoin-keychain/log"
	"github.com/ledgerhq/bitcoin-keychain/pb/bitcoin"
	"github.com/ledgerhq/bitcoin-keychain/pkg/chaincfg"
)

type baseRedisKeystore struct {
	db     *redis.Client
	client bitcoin.CoinServiceClient
}

func (s *baseRedisKeystore) Get(id uuid.UUID) (KeychainInfo, error) {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return KeychainInfo{}, ErrKeychainNotFound
	}

	return meta.Main, nil
}

func (s *baseRedisKeystore) Delete(id uuid.UUID) error {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return ErrKeychainNotFound
	}

	s.db.Del(context.Background(), id.String())

	return nil
}

func (s *baseRedisKeystore) Reset(id uuid.UUID) error {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return ErrKeychainNotFound
	}

	meta.ResetKeychainMeta()

	return set(s.db, id.String(), meta)
}

func (s *baseRedisKeystore) Create(
	extendedPublicKey string, fromChainCode *FromChainCode, scheme Scheme,
	net chaincfg.Network, lookaheadSize uint32, index uint32, metadata string,
) (KeychainInfo, error) {
	meta, err := keystoreCreate(
		extendedPublicKey,
		fromChainCode,
		scheme,
		net,
		lookaheadSize,
		index,
		metadata,
		s.client,
	)

	if err != nil {
		return KeychainInfo{}, err
	}

	if err := set(s.db, meta.Main.ID.String(), meta); err != nil {
		return KeychainInfo{}, err
	}

	return meta.Main, nil
}

func (s *baseRedisKeystore) GetDerivationPath(id uuid.UUID, address string) (DerivationPath, error) {
	var meta Meta
	err := get(s.db, id.String(), &meta)
	if err != nil {
		return DerivationPath{}, ErrKeychainNotFound
	}

	return meta.keystoreGetDerivationPath(address)
}

func (s *baseRedisKeystore) GetAddressesPublicKeys(id uuid.UUID, derivations []DerivationPath) ([]string, error) {
	var meta Meta

	err := get(s.db, id.String(), &meta)
	if err != nil {
		return nil, ErrKeychainNotFound
	}

	return meta.keystoreGetAddressesPublicKeys(derivations)
}

func unmarshall(val string, dest interface{}) error {
	err := json.Unmarshal([]byte(val), dest)
	if err != nil {
		va := reflect.ValueOf(val)
		reflect.ValueOf(dest).Elem().Set(va)
	}
	return nil
}

func marshall(value interface{}) (string, error) {
	var redisValue string

	v, ok := value.(string)
	if ok {
		redisValue = v
	} else {
		p, err := json.Marshal(value)
		if err != nil {
			return "", err
		}
		redisValue = string(p)
	}
	return redisValue, nil
}

func set(c *redis.Client, key string, value interface{}) error {
	redisValue, err := marshall(value)

	if err != nil {
		return err
	}

	log.Debug(fmt.Sprintf("Setting redis key[%s]:[%s]", key, value))
	return c.Set(context.Background(), key, redisValue, 0).Err()
}

func get(c *redis.Client, key string, dest interface{}) error {
	p, err := c.Get(context.Background(), key).Result()
	if err != nil {
		return err
	}

	err = unmarshall(p, dest)
	log.Debug(fmt.Sprintf("Getting redis key[%s]:[%s]", key, dest))
	return err
}
