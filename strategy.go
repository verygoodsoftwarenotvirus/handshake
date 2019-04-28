package handshake

import (
	"encoding/json"

	"github.com/nomasters/handshake/lib/storage"
)

type strategy struct {
	Rendezvous storage.Storage
	Storage    storage.Storage
	Cipher     cipher
}

// strategyPeerConfig is a a struct that encapsulates the shared strategy settings for handshake
type strategyPeerConfig struct {
	Rendezvous storage.PeerStorage `json:"rendezvous"`
	Storage    storage.PeerStorage `json:"storage"`
	Cipher     peerCipher          `json:"cipher"`
}

// strategyConfig is a struct that encapsulates internal chat strategy settings
type strategyConfig struct {
	Rendezvous storage.Config
	Storage    storage.Config
	Cipher     cipherConfig
}

// Share returns the strategyPeerConfig for the strategy
func (s strategy) Share() (config strategyPeerConfig, err error) {
	if config.Rendezvous, err = s.Rendezvous.Share(); err != nil {
		return
	}
	if config.Storage, err = s.Storage.Share(); err != nil {
		return
	}
	if config.Cipher, err = s.Cipher.share(); err != nil {
		return
	}
	return
}

// Share returns the strategyPeerConfig for the strategy
func (s strategy) Export() (config strategyConfig, err error) {
	if config.Rendezvous, err = s.Rendezvous.Export(); err != nil {
		return
	}
	if config.Storage, err = s.Storage.Export(); err != nil {
		return
	}
	if config.Cipher, err = s.Cipher.export(); err != nil {
		return
	}
	return
}

func strategyFromPeerConfig(config strategyPeerConfig) (s strategy, err error) {
	if s.Rendezvous, err = storage.NewStorageFromPeer(config.Rendezvous); err != nil {
		return
	}
	if s.Storage, err = storage.NewStorageFromPeer(config.Storage); err != nil {
		return
	}
	if s.Cipher, err = newCipherFromPeer(config.Cipher); err != nil {
		return
	}
	return
}

func strategyFromConfig(config strategyConfig) (s strategy, err error) {
	if s.Rendezvous, err = storage.NewStorageFromConfig(config.Rendezvous); err != nil {
		return
	}
	if s.Storage, err = storage.NewStorageFromConfig(config.Storage); err != nil {
		return
	}
	if s.Cipher, err = newCipherFromConfig(config.Cipher); err != nil {
		return
	}
	return
}

// ConfigJSONBytes marshall's the strategyPeerConfig as a json file
func (s strategy) ShareJSONBytes() ([]byte, error) {
	config, err := s.Share()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(config)
}

func newDefaultStrategy() strategy {
	return strategy{
		Rendezvous: storage.NewDefaultRendezvous(),
		Storage:    storage.NewDefaultMessageStorage(),
		Cipher:     newDefaultCipher(),
	}
}
