package handshake

import (
	"encoding/base64"
	"testing"

	"github.com/nomasters/handshake/lib/storage"
)

func TestExportStrategy(t *testing.T) {
	privateKeyString := "6zjTWCoDkKESjroDj26qrw0/xSU0B14Co/lIZZHhbHUFFt6rMcqyLt21y1PmoPJbokhXrvO4p+zauvk+GuujzA=="
	privateKey, err := base64.StdEncoding.DecodeString(privateKeyString)
	if err != nil {
		t.Errorf("base64 failed to decode: %v\n", err)
	}
	publicKey := privateKey[32:]

	n := storage.Node{
		URL: "https://prototype.hashmap.sh",
	}

	sig := storage.SignatureAlgorithm{
		Type:       storage.ED25519,
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}

	rOpts := storage.Options{
		WriteNodes: []storage.Node{n},
		Signatures: []storage.SignatureAlgorithm{sig},
		WriteRule:  storage.DefaultConsensusRule,
	}
	r, err := storage.NewHashmapStorage(rOpts)
	if err != nil {
		t.Errorf("new hashmapStore failed: %v\n", err)
	}

	settings := make(map[string]string)
	settings["query_type"] = "api"

	n2 := storage.Node{
		URL:      "https://ipfs.infura.io:5001/",
		Settings: settings,
	}
	sOpts := storage.Options{
		WriteNodes: []storage.Node{n2},
		WriteRule:  storage.DefaultConsensusRule,
	}
	s, err := storage.NewIPFSStorage(sOpts)
	if err != nil {
		t.Errorf("new IPFS storage failed: %v\n", err)
	}

	c := newDefaultSBCipher()

	strat := strategy{
		Rendezvous: r,
		Storage:    s,
		Cipher:     c,
	}

	t.Log(strat.Share())
	stratJSON, err := strat.ShareJSONBytes()
	if err != nil {
		t.Errorf("failed on json bytes %v", err)
	}
	t.Log(string(stratJSON))
}
