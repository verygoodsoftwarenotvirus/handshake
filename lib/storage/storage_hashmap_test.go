package storage

import (
	"encoding/base64"
	"testing"
)

func TestHashmapSet(t *testing.T) {
	// this needs to be refactored to use a mock server
	privateKeyString := "6zjTWCoDkKESjroDj26qrw0/xSU0B14Co/lIZZHhbHUFFt6rMcqyLt21y1PmoPJbokhXrvO4p+zauvk+GuujzA=="
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyString)
	if err != nil {
		t.Errorf("base64 failed to decode: %v\n", err)
	}
	publicKey := privateKey[32:]

	n := Node{
		URL: "https://prototype.hashmap.sh",
	}

	sig := SignatureAlgorithm{
		Type:       ED25519,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	opts := StorageOptions{
		WriteNodes: []Node{n},
		Signatures: []SignatureAlgorithm{sig},
		WriteRule:  defaultConsensusRule,
	}
	hms, err := NewHashmapStorage(opts)
	if err != nil {
		t.Errorf("new hashmapStore failed: %v\n", err)
	}

	if _, err := hms.Set("", []byte("this was generated from a go test. right now for real")); err != nil {
		t.Errorf("set failed: %v\n", err)
	}
}

func TestHashmapStorageGet(t *testing.T) {
	n := Node{
		URL: "https://prototype.hashmap.sh/2DrjgbL8QfKRvxU9KtFYFdNiPZrQijyxkvWXH17QnvNmzB3apR",
		//    2Drjgb5DseoVAvRLngcVmd4YfJAi3J1145kiNFV3CL32Hs6vzb
	}
	opts := StorageOptions{
		ReadNodes: []Node{n},
		ReadRule:  defaultConsensusRule,
	}
	hms, err := NewHashmapStorage(opts)
	if err != nil {
		t.Errorf("new hashmapStore failed: %v\n", err)
	}
	response, err := hms.Get("")
	if err != nil {
		t.Errorf("response failed: %v\n", err)
	}
	t.Log(string(response))
}
