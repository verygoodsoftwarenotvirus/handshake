package storage

import (
	"errors"

	"github.com/nomasters/hashmap"

	"github.com/nomasters/handshake/lib/config"
)

// Engine type for enum
type Engine int

const (
	// BoltEngine is the default Storage engine for device Storage
	BoltEngine Engine = iota
	// HashmapEngine is the default Rendezvous Storage type
	HashmapEngine
	// IPFSEngine is the default message Storage type
	IPFSEngine
)

const (
	// DefaultStorageEngine is used to set the Storage engine if none is set in
	// Storage options
	DefaultStorageEngine = BoltEngine
	// DefaultBoltFilePath is the default path and file name for BoltDB Storage
	DefaultBoltFilePath = "handshake.boltdb"
	// DefaultTLB is the name of the top level bucket for BoltDB
	defaultTLB = "handshake"
	// GlobalConfigKey is the key string for where global-config is stored
	globalConfigKey      = "global-config"
	maxIPFSRead          = 3000000 // ~3MB
	defaultRendezvousURL = "https://prototype.hashmap.sh"
)

type signatureType int

const (
	// ED25519 is the primary signature type
	ED25519 signatureType = iota
)

// consensusRule is a datatype to capture basic rules around how consensus with multiple nodes should
// work for Storage such as IPFS and Hashmap if multiple endpoints are configured.
type consensusRule int

const (
	// firstSuccess dictates that if any Node returns a success, success is returned
	firstSuccess consensusRule = iota
	// redundantPairSuccess dictates that if any two nodes return a success, success is returned
	redundantPairSuccess
	// majoritySuccess dictates that if a simple majority of nodes returns success, a sucess is returned
	majoritySuccess
	// unanimousSuccess dictates that all nodes must return a success to return a sucess
	unanimousSuccess
)

const (
	// DefaultConsensusRule is the default consensusRule
	DefaultConsensusRule  = firstSuccess
	defaultHashmapSigType = ED25519
)

// Storage is the primary interface for interacting with the KV store in handshake
type Storage interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) (string, error)
	Delete(key string) error
	List(path string) ([]string, error)
	Close() error
	Export() (Config, error)
	Share() (PeerStorage, error)
}

// NewDefaultRendezvous provides the default rendezvous storage location
func NewDefaultRendezvous() *HashmapStorage {
	privateKey := hashmap.GenerateKey()
	publicKey := privateKey[32:]
	n := Node{
		URL: defaultRendezvousURL,
	}
	sig := SignatureAlgorithm{
		Type:       ED25519,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}
	return &HashmapStorage{
		WriteNodes: []Node{n},
		Signatures: []SignatureAlgorithm{sig},
		WriteRule:  DefaultConsensusRule,
	}
}

// NewDefaultMessageStorage provides the default long-term storage location
func NewDefaultMessageStorage() Storage {
	settings := make(map[string]string)
	settings["query_type"] = "api"

	n := Node{
		URL:      "https://ipfs.infura.io:5001/",
		Settings: settings,
	}

	return IPFSStorage{
		WriteNodes: []Node{n},
		WriteRule:  DefaultConsensusRule,
	}
}

// PeerStorage is a set of aggregate settings used for sharing and storing Storage settings
type PeerStorage struct {
	Type       Engine        `json:"type"`
	ReadNodes  []Node        `json:"read_nodes,omitempty"`
	WriteNodes []Node        `json:"write_nodes,omitempty"`
	ReadRule   consensusRule `json:"read_rule,omitempty"`
	WriteRule  consensusRule `json:"write_rule,omitempty"`
}

// Config is a set of settings used to in Storage interface gob Storage
type Config struct {
	Type       Engine
	ReadNodes  []Node
	WriteNodes []Node
	ReadRule   consensusRule
	WriteRule  consensusRule
	Signatures []SignatureAlgorithm
	Latest     int64
}

// Node represents DOCUMENTME
type Node struct {
	URL      string            `json:"url,omitempty"`
	Header   map[string]string `json:"header,omitempty"`
	Settings map[string]string `json:"settings,omitempty"`
}

// Options are used to pass in initialization settings
type Options struct {
	Engine     Engine
	FilePath   string
	Signatures []SignatureAlgorithm
	ReadNodes  []Node
	WriteNodes []Node
	ReadRule   consensusRule
	WriteRule  consensusRule
}

// NewStorage initiates a new Storage Interface
func NewStorage(cfg config.Config, opts Options) (Storage, error) {
	switch opts.Engine {
	case BoltEngine:
		return newBoltStorage(cfg, opts)
	default:
		return nil, errors.New("invalid engine type")
	}
}

// NewStorageFromPeer creates a new Storage from a PeerStorage
func NewStorageFromPeer(s PeerStorage) (Storage, error) {
	switch s.Type {
	case IPFSEngine:
		return IPFSStorage{
			ReadNodes: s.ReadNodes,
			ReadRule:  s.ReadRule,
		}, nil
	case HashmapEngine:
		return &HashmapStorage{
			ReadNodes: s.ReadNodes,
			ReadRule:  s.ReadRule,
		}, nil
	default:
		return nil, errors.New("invalid Storage engine type")
	}
}

// NewStorageFromConfig creates a new Storage from a Config
func NewStorageFromConfig(cfg Config) (Storage, error) {
	switch cfg.Type {
	case IPFSEngine:
		return IPFSStorage{
			ReadNodes:  cfg.ReadNodes,
			ReadRule:   cfg.ReadRule,
			WriteNodes: cfg.WriteNodes,
			WriteRule:  cfg.WriteRule,
		}, nil
	case HashmapEngine:
		return &HashmapStorage{
			ReadNodes:  cfg.ReadNodes,
			ReadRule:   cfg.ReadRule,
			WriteNodes: cfg.WriteNodes,
			WriteRule:  cfg.WriteRule,
			Signatures: cfg.Signatures,
			Latest:     cfg.Latest,
		}, nil
	default:
		return nil, errors.New("invalid Storage engine type")
	}
}
