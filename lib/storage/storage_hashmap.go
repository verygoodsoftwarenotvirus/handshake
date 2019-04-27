package storage

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nomasters/hashmap"
)

// HashmapStorage interacts with a hashmap server and
// conforms to the Storage interface
type HashmapStorage struct {
	ReadNodes  []Node
	WriteNodes []Node
	Signatures []SignatureAlgorithm
	ReadRule   consensusRule
	WriteRule  consensusRule
	Latest     int64
}

// SignatureAlgorithm describes the
type SignatureAlgorithm struct {
	Type       signatureType
	PrivateKey []byte
	PublicKey  []byte
}

// NewHashmapStorage builds a new Hashmap Storage instance
func NewHashmapStorage(opts StorageOptions) (*HashmapStorage, error) {
	return &HashmapStorage{
		Signatures: opts.Signatures,
		ReadNodes:  opts.ReadNodes,
		WriteNodes: opts.WriteNodes,
		ReadRule:   opts.ReadRule,
		WriteRule:  opts.WriteRule,
	}, nil
}

func (s *HashmapStorage) updateLatest(timeStamp int64) error {
	// check for timestamp set too far in the future
	if timeStamp > (time.Now().UnixNano() + (5 * 1000000000)) {
		return errors.New("invalid future timestamp")
	}
	// check for potential replay attack, which latest timestamp
	// detected newer than the one provided by the server
	if s.Latest > timeStamp {
		return errors.New("stale timestamp")
	}
	s.Latest = timeStamp
	return nil
}

// getHashFromPath takes a path string and returns the hash at the end of the path
func getHashFromPath(path string) string {
	lastIndex := strings.LastIndex(path, "/")
	if lastIndex == -1 {
		return path
	}
	return path[lastIndex+1:]
}

// getFirstSuccess loops through all ReadNodes in a hashmapStorage and attempts to resolve the data from a
// payload. There is an important set of steps that this goes through, including:
// - validating the MultiHash in the URL is supported
// - comparing the payload pubkey to the url hash, which must match.
// if all verification and validations are successful, it returns the data bytes from the payload
func (s *HashmapStorage) getFirstSuccess() ([]byte, error) {
	for _, node := range s.ReadNodes {
		u, err := url.Parse(node.URL)
		if err != nil {
			return []byte{}, fmt.Errorf("invalid url for: %v", node.URL)
		}
		urlHash := getHashFromPath(u.Path)
		if !isHashmapMultihash(urlHash) {
			return []byte{}, fmt.Errorf("invalid hashmap endpoint for: %v", node.URL)
		}

		resp, err := http.Get(node.URL)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		payload, err := hashmap.NewPayloadFromReader(resp.Body)
		if err != nil {
			continue
		}

		pubkey, err := payload.PubKeyBytes()
		if err != nil {
			return []byte{}, fmt.Errorf("invalid pubkey in payload for: %v", node.URL)
		}

		if urlHash != base58Multihash(pubkey) {
			return []byte{}, fmt.Errorf("payload and endpoint hash mismatch for: %v", node.URL)
		}

		data, err := payload.GetData()
		if err != nil {
			return []byte{}, err
		}
		if err := s.updateLatest(data.Timestamp); err != nil {
			return []byte{}, err
		}
		return data.MessageBytes()
	}
	return []byte{}, errors.New("no servers available")
}

// Get fetches an item from storage for a given key
func (s *HashmapStorage) Get(key string) ([]byte, error) {
	if len(s.ReadNodes) < 1 {
		return []byte{}, errors.New("no read nodes configured")
	}
	switch s.ReadRule {
	case firstSuccess:
		return s.getFirstSuccess()
	default:
		return []byte{}, errors.New("This readRule is not yet implemented")
	}

}

func (s *HashmapStorage) setFirstSuccess(payload []byte) error {
	for _, node := range s.WriteNodes {
		resp, err := http.Post(node.URL, "application/json", bytes.NewReader(payload))
		if err != nil {
			continue
		}
		if resp.StatusCode > 399 {
			continue
		}
		return nil
	}
	return errors.New("no servers available")
}

// Set blah
func (s *HashmapStorage) Set(key string, value []byte) (string, error) {
	if len(s.WriteNodes) < 1 {
		return key, errors.New("no write nodes configured")
	}

	opts := hashmap.GeneratePayloadOptions{Message: string(value)}
	// TODO: currently we only support one signature, but this will change
	payload, err := hashmap.GeneratePayload(opts, s.Signatures[0].PrivateKey)
	if err != nil {
		return key, err
	}

	switch s.WriteRule {
	case firstSuccess:
		return key, s.setFirstSuccess(payload)
	default:
		return key, errors.New("This writeRule is not yet implemented")
	}
}

// Delete is used to remove references from hashmap. Not currently implemented.
// TODO : a delete could be accomplished by writing a blank dataset to each endpoint
func (s HashmapStorage) Delete(key string) (e error) { return }

// List is not implemented for hashmapStorage, returns "", nil
func (s HashmapStorage) List(path string) ([]string, error) {
	return []string{}, errors.New("no implemented")
}

// Close is not used in hashmap, returns nil
func (s HashmapStorage) Close() (e error) { return }

// Share returns a PeerStorage and error, it generates read nodes from the write nodes + pubkey
// it also returns ReadRules based on the WriteRules
func (s HashmapStorage) Share() (PeerStorage, error) {
	readNodes, err := s.genReadFromWriteNodes()
	if err != nil {
		return PeerStorage{}, err
	}

	return PeerStorage{
		Type:      HashmapEngine,
		ReadNodes: readNodes,
		ReadRule:  s.WriteRule,
	}, nil
}

// Export returns a storage configuration based on the storage instance
// TODO: configure Export settings for this
func (s HashmapStorage) Export() (StorageConfig, error) {
	return StorageConfig{
		Type:       HashmapEngine,
		ReadNodes:  s.ReadNodes,
		WriteNodes: s.WriteNodes,
		ReadRule:   s.ReadRule,
		WriteRule:  s.WriteRule,
		Signatures: s.Signatures,
		Latest:     s.Latest,
	}, nil
}

// genReadFromWriteNodes creates a set of read nodes based on all signature
// files times the number of write urls and returns a list of nodes and and error
func (s HashmapStorage) genReadFromWriteNodes() ([]Node, error) {
	var readNodes []Node
	var endpoints []string
	for _, sig := range s.Signatures {
		endpoints = append(endpoints, base58Multihash(sig.PublicKey))
	}
	for _, writeNode := range s.WriteNodes {
		for _, endpoint := range endpoints {
			u, err := url.Parse(writeNode.URL)
			if err != nil {
				return readNodes, err
			}
			u.Path = endpoint
			readNodes = append(readNodes, Node{URL: u.String()})
		}
	}
	return readNodes, nil
}
