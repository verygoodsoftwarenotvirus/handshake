package handshake

import (
	"testing"
)

func TestGetFromIPFS(t *testing.T) {
	settings := make(map[string]string)
	settings["query_type"] = "api"

	happyNodes := []node{
		// node{
		// 	URL:      "http://127.0.0.1:5001",
		// 	Settings: settings,
		// },
		node{
			URL:      "https://ipfs.infura.io:5001/",
			Settings: settings,
		},
		node{
			URL: "https://cloudflare-ipfs.com",
		},
	}
	hash := "QmZULkCELmmk5XNfCgTnCyFgAVxBRBXyDHGGMVoLFLiXEN"
	for _, n := range happyNodes {
		resp, err := getFromIPFS(n, hash)
		if err != nil {
			t.Error(err)
		}
		t.Log(string(resp))
	}
}

func TestPostToIPFS(t *testing.T) {
	settings := make(map[string]string)

	settings["query_type"] = "api"

	happyNodes := []node{
		// node{
		// 	URL:      "http://127.0.0.1:5001",
		// 	Settings: settings,
		// },
		node{
			URL:      "https://ipfs.infura.io:5001/",
			Settings: settings,
		},
		node{
			URL: "https://hardbin.com",
		},
	}
	body := []byte("hello, world")
	for _, n := range happyNodes {
		resp, err := postToIPFS(n, body)
		if err != nil {
			t.Error(err)
		}
		t.Log(string(resp))
	}
}
