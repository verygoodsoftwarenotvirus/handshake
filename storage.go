package handshake

import (
	"errors"

	"github.com/nomasters/hashmap"
)

// StorageEngine type for enum
type StorageEngine int

const (
	// BoltEngine is the default storage engine for device storage
	BoltEngine StorageEngine = iota
	// HashmapEngine is the default Rendezvous storage type
	HashmapEngine
	// IPFSEngine is the default message storage type
	IPFSEngine
)

const (
	// DefaultStorageEngine is used to set the storage engine if none is set in
	// storage options
	defaultStorageEngine = BoltEngine
	// DefaultBoltFilePath is the default path and file name for BoltDB storage
	defaultBoltFilePath = "handshake.boltdb"
	// DefaultTLB is the name of the top level bucket for BoltDB
	defaultTLB = "handshake"
	// GlobalConfigKey is the key string for where global-config is stored
	globalConfigKey      = "global-config"
	maxIPFSRead          = 3000000 // ~3MB
	defaultRendezvousURL = "https://prototype.hashmap.sh"
)

type signatureType int

const (
	// ed25519
	ed25519 signatureType = iota
)

// consensusRule is a datatype to capture basic rules around how consensus with multiple nodes should
// work for storage such as IPFS and Hashmap if multiple endpoints are configured.
type consensusRule int

const (
	// firstSuccess dictates that if any node returns a success, success is returned
	firstSuccess consensusRule = iota
	// redundantPairSuccess dictates that if any two nodes return a success, success is returned
	redundantPairSuccess
	// majoritySuccess dictates that if a simple majority of nodes returns success, a sucess is returned
	majoritySuccess
	// unanimousSuccess dictates that all nodes must return a success to return a sucess
	unanimousSuccess
)

const (
	defaultConsensusRule  = firstSuccess
	defaultHashmapSigType = ed25519
)

// Storage is the primary interface for interacting with the KV store in handshake
type storage interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) (string, error)
	Delete(key string) error
	List(path string) ([]string, error)
	Close() error
	export() (storageConfig, error)
	share() (peerStorage, error)
}

func newDefaultRendezvous() *hashmapStorage {
	privateKey := hashmap.GenerateKey()
	publicKey := privateKey[32:]
	n := node{
		URL: defaultRendezvousURL,
	}
	sig := signatureAlgorithm{
		Type:       ed25519,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return &hashmapStorage{
		WriteNodes: []node{n},
		Signatures: []signatureAlgorithm{sig},
		WriteRule:  defaultConsensusRule,
	}
}

func newDefaultMessageStorage() ipfsStorage {
	settings := make(map[string]string)
	settings["query_type"] = "api"

	n := node{
		URL:      "https://ipfs.infura.io:5001/",
		Settings: settings,
	}

	return ipfsStorage{
		WriteNodes: []node{n},
		WriteRule:  defaultConsensusRule,
	}
}

// peerStorage is a set of aggregate settings used for sharing and storing storage settings
type peerStorage struct {
	Type       StorageEngine `json:"type"`
	ReadNodes  []node        `json:"read_nodes,omitempty"`
	WriteNodes []node        `json:"write_nodes,omitempty"`
	ReadRule   consensusRule `json:"read_rule,omitempty"`
	WriteRule  consensusRule `json:"write_rule,omitempty"`
}

// storageConfig is a set of settings used to in storage interface gob storage
type storageConfig struct {
	Type       StorageEngine
	ReadNodes  []node
	WriteNodes []node
	ReadRule   consensusRule
	WriteRule  consensusRule
	Signatures []signatureAlgorithm
	Latest     int64
}

type node struct {
	URL      string            `json:"url,omitempty"`
	Header   map[string]string `json:"header,omitempty"`
	Settings map[string]string `json:"settings,omitempty"`
}

// StorageOptions are used to pass in initialization settings
type StorageOptions struct {
	Engine     StorageEngine
	FilePath   string
	Signatures []signatureAlgorithm
	ReadNodes  []node
	WriteNodes []node
	ReadRule   consensusRule
	WriteRule  consensusRule
}

// NewStorage initiates a new storage Interface
func newStorage(cfg Config, opts StorageOptions) (storage, error) {
	switch opts.Engine {
	case BoltEngine:
		return newBoltStorage(cfg, opts)
	default:
		return nil, errors.New("invalid engine type")
	}
}

func newStorageFromPeer(s peerStorage) (storage, error) {
	switch s.Type {
	case IPFSEngine:
		return ipfsStorage{
			ReadNodes: s.ReadNodes,
			ReadRule:  s.ReadRule,
		}, nil
	case HashmapEngine:
		return &hashmapStorage{
			ReadNodes: s.ReadNodes,
			ReadRule:  s.ReadRule,
		}, nil
	default:
		return nil, errors.New("invalid storage engine type")
	}
}

func newStorageFromConfig(s storageConfig) (storage, error) {
	switch s.Type {
	case IPFSEngine:
		return ipfsStorage{
			ReadNodes:  s.ReadNodes,
			ReadRule:   s.ReadRule,
			WriteNodes: s.WriteNodes,
			WriteRule:  s.WriteRule,
		}, nil
	case HashmapEngine:
		return &hashmapStorage{
			ReadNodes:  s.ReadNodes,
			ReadRule:   s.ReadRule,
			WriteNodes: s.WriteNodes,
			WriteRule:  s.WriteRule,
			Signatures: s.Signatures,
			Latest:     s.Latest,
		}, nil
	default:
		return nil, errors.New("invalid storage engine type")
	}
}
