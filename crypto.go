package handshake

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

const (
	// secretBoxDefaultChunkSize is the default size of an encrypted chunk of data
	secretBoxDefaultChunkSize = 16000
	// secretBoxDecryptionOffset is the additional offset of bytes needed to offset
	// for the nonce and authentication bytes
	secretBoxDecryptionOffset = 40
	// secretBoxNonceLength is the length in bytes required for the nonce
	secretBoxNonceLength = 24
	// secretBoxKeyLength is the length in bytes required for the key
	secretBoxKeyLength = 32
	blake2b256code     = uint64(45600)
	blake2b256length   = 32
	blake2b256name     = "blake2b-256"
	lookupHashLength   = 24
)

// NonceType is used for type enumeration for Ciphers Nonces
type NonceType int

// CipherType is used for type enumeration of Ciphers
type CipherType int

const (
	// RandomNonce is the NonceType used for pure crypto/rand generated nonces
	RandomNonce NonceType = iota
	// TimeSeriesNonce is the NonceType used for 4 byte unix time prefixed crypto/rand generated nonces
	TimeSeriesNonce
)

const (
	// SecretBox is a CipherType
	SecretBox CipherType = iota
)

// Cipher is an interface used for encrypting and decrypting byte slices.
type cipher interface {
	Encrypt(data []byte, key []byte) ([]byte, error)
	Decrypt(data []byte, key []byte) ([]byte, error)
	share() (peerCipher, error)
	export() (cipherConfig, error)
}

// peerCipher is a struct used to share cipher settings to a peer in handshake
type peerCipher struct {
	Type      CipherType `json:"type"`
	ChunkSize int        `json:"chunk_size,omitempty"`
}

// cipherConfig is a struct used to share cipher settings to a peer in handshake
type cipherConfig struct {
	Type      CipherType
	ChunkSize int
}

// genRandBytes takes a length of l and returns a byte slice of random data
func genRandBytes(l int) []byte {
	b := make([]byte, l)
	rand.Read(b)
	return b
}

// genLookups takes a pepper and entropy []byte, a CipherType, and a count and returns a map[string][]byte for lookup hashes
func genLookups(pepper [64]byte, entropy [96]byte, cipherType CipherType, count int) (lookup, error) {
	lookups := make(map[string][]byte)
	if count < 1 {
		return lookups, errors.New("count must be greater than or equal to 1")
	}
	p, e1, e2, e3 := pepper[:], entropy[:32], entropy[32:64], entropy[64:]
	var keyLength int
	switch cipherType {
	case SecretBox:
		keyLength = secretBoxKeyLength
	default:
		return lookups, fmt.Errorf("cipher type %v is not implemented for lookup generation", cipherType)
	}
	lookupBytes := argon2.IDKey(p, e2, 1, 64*1024, 4, uint32(count*lookupHashLength))
	keyBytes := argon2.IDKey(e1, e3, 1, 64*1024, 4, uint32(count*keyLength))

	for i := 1; i < count; i++ {
		lookupStart := (i - 1) * lookupHashLength
		lookupEnd := i * lookupHashLength
		keyStart := (i - 1) * keyLength
		keyEnd := i * keyLength
		k := base64.StdEncoding.EncodeToString(lookupBytes[lookupStart:lookupEnd])
		v := keyBytes[keyStart:keyEnd]
		lookups[k] = v
	}
	return lookups, nil
}

// genTimeStampNonce takes an int for the nonce size and returns a byte slice of length size.
// A byte slice is created for the nonce and filled with random data from `crypto/rand`, then the
// first 4 bytes of the nonce are overwritten with LittleEndian encoding of `time.Now().Unix()`
// The purpose of this function is to avoid an unlikely collision in randomly generating nonces
// by prefixing the nonce with time series data.
func genTimeStampNonce(l int) []byte {
	nonce := genRandBytes(l)
	timeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBytes, uint64(time.Now().Unix()))
	copy(nonce, timeBytes[:4])
	return nonce
}

// DeriveKey takes a password and salt and applies a set of fixed parameters
// to the argon2 IDKey algorithm.
func deriveKey(pw, salt []byte) []byte {
	return argon2.IDKey(pw, salt, 1, 64*1024, 4, secretBoxKeyLength)
}

// SecretBoxCipher is a struct and method set that conforms to the Cipher interface. This is the primary cipher used
// for all blob encryption and decryption for handshake
type SecretBoxCipher struct {
	Nonce     NonceType
	ChunkSize int
}

// newTimeSeriesSBCipher returns a timeSeriesNonce based SecretBoxCipher struct that conforms to the
// Cipher interface
func newTimeSeriesSBCipher() SecretBoxCipher {
	return SecretBoxCipher{Nonce: TimeSeriesNonce, ChunkSize: secretBoxDefaultChunkSize}
}

func newDefaultCipher() SecretBoxCipher {
	return newDefaultSBCipher()
}

// newDefaultSBCipher returns a RandomNonce based SecretBoxCipher struct that conforms to the
// Cipher interface
func newDefaultSBCipher() SecretBoxCipher {
	return SecretBoxCipher{Nonce: RandomNonce, ChunkSize: secretBoxDefaultChunkSize}
}

// Encrypt takes byte slices for data and a key and returns the ciphertext output for secretbox
func (s SecretBoxCipher) Encrypt(data []byte, key []byte) ([]byte, error) {
	var encryptedData []byte
	chunkSize := s.ChunkSize

	if len(key) != secretBoxKeyLength {
		return encryptedData, errors.New("invalid key length")
	}

	var k [secretBoxKeyLength]byte
	copy(k[:], key)

	for i := 0; i < len(data); i = i + chunkSize {
		var chunk []byte
		if len(data[i:]) >= chunkSize {
			chunk = data[i : i+chunkSize]
		} else {
			chunk = data[i:]
		}
		nonce := s.genNonce()

		var n [secretBoxNonceLength]byte
		copy(n[:], nonce)

		encryptedChunk := secretbox.Seal(n[:], chunk, &n, &k)
		encryptedData = append(encryptedData, encryptedChunk...)
	}
	return encryptedData, nil
}

// Decrypt takes byte slices for data and key and returns the clear text output for secretbox
func (s SecretBoxCipher) Decrypt(data []byte, key []byte) ([]byte, error) {
	var decryptedData []byte
	chunkSize := s.ChunkSize + secretBoxDecryptionOffset

	if len(key) != secretBoxKeyLength {
		return decryptedData, errors.New("invalid key length")
	}

	var k [secretBoxKeyLength]byte
	copy(k[:], key)

	for i := 0; i < len(data); i = i + chunkSize {
		var chunk []byte
		if len(data[i:]) >= chunkSize {
			chunk = data[i : i+chunkSize]
		} else {
			chunk = data[i:]
		}
		var n [secretBoxNonceLength]byte
		copy(n[:], chunk[:secretBoxNonceLength])

		decryptedChunk, ok := secretbox.Open(nil, chunk[secretBoxNonceLength:], &n, &k)
		if !ok {
			return nil, errors.New("decrypt failed")
		}
		decryptedData = append(decryptedData, decryptedChunk...)
	}
	return decryptedData, nil
}

// GenNonce returns a set of nonce bytes based on the NonceType configured in the struct
func (s SecretBoxCipher) genNonce() []byte {
	switch s.Nonce {
	case RandomNonce:
		return genRandBytes(secretBoxNonceLength)
	case TimeSeriesNonce:
		return genTimeStampNonce(secretBoxNonceLength)
	default:
		return genRandBytes(secretBoxNonceLength)
	}
}

// share is used to export settings shared with a peer
func (s SecretBoxCipher) share() (peerCipher, error) {
	return peerCipher{
		Type:      SecretBox,
		ChunkSize: secretBoxDefaultChunkSize,
	}, nil
}

// export is used to export settings shared with a peer
func (s SecretBoxCipher) export() (cipherConfig, error) {
	return cipherConfig{
		Type:      SecretBox,
		ChunkSize: secretBoxDefaultChunkSize,
	}, nil
}

func newCipherFromPeer(config peerCipher) (c cipher, err error) {
	switch config.Type {
	case SecretBox:
		return SecretBoxCipher{
			Nonce:     RandomNonce,
			ChunkSize: config.ChunkSize,
		}, nil
	default:
		return c, errors.New("cipher not implemented for config import")
	}
}

func newCipherFromConfig(config cipherConfig) (c cipher, err error) {
	switch config.Type {
	case SecretBox:
		return SecretBoxCipher{
			Nonce:     RandomNonce,
			ChunkSize: config.ChunkSize,
		}, nil
	default:
		return c, errors.New("cipher not implemented for config import")
	}
}
