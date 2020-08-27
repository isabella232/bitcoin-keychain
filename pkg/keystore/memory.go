package keystore

import "errors"

// InMemoryKeystore implements the Keystore interface where the storage
// is an in-memory map. Useful for unit-tests.
type InMemoryKeystore struct {
	db KeystoreSchema
}

func NewInMemoryKeystore() Keystore {
	return &InMemoryKeystore{}
}

func (s *InMemoryKeystore) Get(descriptor string) (KeychainInfo, error) {
	document, ok := s.db[descriptor]
	if !ok {
		return KeychainInfo{}, errors.New("does not exist")
	}

	return document.Main, nil
}

func (s *InMemoryKeystore) Create(descriptor string) (KeychainInfo, error) {
	return KeychainInfo{
		Descriptor:              descriptor,
		XPub:                    "",
		SLIP32ExtendedPublicKey: "",
		ExternalPublicKey:       "",
		ExternalChainCode:       "",
		MaxExternalIndex:        0,
		InternalPublicKey:       "",
		InternalChainCode:       "",
		MaxInternalIndex:        0,
		LookaheadSize:           0,
		Scheme:                  "",
	}, nil
}
