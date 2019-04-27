package storage

import (
	multihash "github.com/multiformats/go-multihash"
)

const (
	blake2b256code   = uint64(45600)
	blake2b256length = 32
	blake2b256name   = "blake2b-256"
)

// base58Multihash a set of bytes to an IPFS style blake2b-256 multihash in base58 encoding
func base58Multihash(b []byte) string {
	mh, _ := multihash.Sum(b, blake2b256code, blake2b256length)
	return mh.B58String()
}

// isHashmapMultihash takes a string encoded base58 multihash and checks to see if it is supported
// by handshake. Currently, handshake only supports
func isHashmapMultihash(hash string) bool {
	mh, err := multihash.FromB58String(hash)
	if err != nil {
		return false // return false if error decoding
	}
	decoded, err := multihash.Decode(mh)
	if err != nil {
		return false
	}
	switch decoded.Name {
	case blake2b256name:
		return true
	}
	return false
}
