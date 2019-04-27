package handshake

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// IPFSStorage interacts with an IPFS gateway and conforms to the Storage interface
type ipfsStorage struct {
	ReadNodes  []node
	WriteNodes []node
	ReadRule   consensusRule
	WriteRule  consensusRule
}

func newIPFSStorage(opts StorageOptions) (ipfsStorage, error) {
	return ipfsStorage{
		ReadNodes:  opts.ReadNodes,
		WriteNodes: opts.WriteNodes,
		ReadRule:   opts.ReadRule,
		WriteRule:  opts.WriteRule,
	}, nil
}

func (s ipfsStorage) Get(key string) ([]byte, error) {
	if len(s.ReadNodes) < 1 {
		return []byte{}, errors.New("no read nodes configured")
	}
	switch s.ReadRule {
	case firstSuccess:
		return s.getFirstSuccess(key)
	default:
		return []byte{}, errors.New("This readRule is not yet implemented")
	}
}

func (s *ipfsStorage) getFirstSuccess(hash string) ([]byte, error) {
	for _, node := range s.ReadNodes {
		resp, err := getFromIPFS(node, hash)
		if err != nil {
			continue
		}
		return resp, nil
	}
	return []byte{}, errors.New("no servers available")
}

func (s ipfsStorage) Set(key string, value []byte) (string, error) {
	if len(s.WriteNodes) < 1 {
		return "", errors.New("no write nodes configured")
	}
	switch s.WriteRule {
	case firstSuccess:
		return s.setFirstSuccess(value)
	default:
		return "", errors.New("This writeRule is not yet implemented")
	}
}

func (s ipfsStorage) setFirstSuccess(body []byte) (string, error) {
	for _, node := range s.WriteNodes {
		resp, err := postToIPFS(node, body)
		if err != nil {
			continue
		}
		return resp, nil
	}
	return "", errors.New("no servers available")
}

func (s ipfsStorage) Delete(key string) error            { return nil }
func (s ipfsStorage) List(path string) ([]string, error) { return []string{}, nil }
func (s ipfsStorage) Close() error                       { return nil }

func (s ipfsStorage) share() (peerStorage, error) {
	return peerStorage{
		Type:      IPFSEngine,
		ReadNodes: s.WriteNodes,
		ReadRule:  s.WriteRule,
	}, nil
}

// TODO: configure export settings for this
func (s ipfsStorage) export() (storageConfig, error) {
	return storageConfig{
		Type:       IPFSEngine,
		ReadNodes:  s.ReadNodes,
		ReadRule:   s.ReadRule,
		WriteNodes: s.WriteNodes,
		WriteRule:  s.WriteRule,
	}, nil
}

// TODO: these should prob be moved into their own lib.

// appendToPath this safely appends two url paths together by ensuring that leading and trailing
// slashes are trimmed before joining them together
func appendToPath(base, add string) string {
	if add == "" {
		return base
	}
	base = strings.TrimSuffix(base, "/")
	add = strings.TrimPrefix(add, "/")
	return fmt.Sprintf("%s/%s", base, add)
}

func getFromIPFS(n node, hash string) ([]byte, error) {
	client := http.DefaultClient
	u, err := url.Parse(n.URL)
	if err != nil {
		return []byte{}, err
	}
	switch n.Settings["query_type"] {
	case "api":
		endpoint := "api/v0/cat"
		values := u.Query()
		values.Set("arg", hash)
		u.RawQuery = values.Encode()
		u.Path = appendToPath(u.Path, endpoint)
	default:
		endpoint := fmt.Sprintf("ipfs/%s", hash)
		u.Path = appendToPath(u.Path, endpoint)
	}

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return []byte{}, err
	}
	if len(n.Header) > 0 {
		for k, v := range n.Header {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("error closing response body: %v\n", err)
		}
	}()

	limitedReader := &io.LimitedReader{R: resp.Body, N: maxIPFSRead}
	return ioutil.ReadAll(limitedReader)
}

func postToIPFS(n node, body []byte) (string, error) {
	client := http.DefaultClient
	u, err := url.Parse(n.URL)
	if err != nil {
		return "", err
	}
	switch n.Settings["query_type"] {
	case "api":
		endpoint := "api/v0/add"
		u.Path = appendToPath(u.Path, endpoint)
		bodyBuf := &bytes.Buffer{}
		bodyWriter := multipart.NewWriter(bodyBuf)
		fileWriter, err := bodyWriter.CreateFormFile("file", "file")
		if err != nil {
			return "", err
		}
		if _, err := fileWriter.Write(body); err != nil {
			return "", err
		}
		contentType := bodyWriter.FormDataContentType()
		bodyWriter.Close()
		req, err := http.NewRequest("POST", u.String(), bodyBuf)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", contentType)
		if len(n.Header) > 0 {
			for k, v := range n.Header {
				req.Header.Set(k, v)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		output := make(map[string]string)
		if err := json.Unmarshal(body, &output); err != nil {
			return "", err
		}
		return output["Hash"], nil
	default:
		endpoint := "ipfs/"
		u.Path = appendToPath(u.Path, endpoint)
		req, err := http.NewRequest("POST", u.String(), bytes.NewReader(body))
		if err != nil {
			return "", err
		}
		if len(n.Header) > 0 {
			for k, v := range n.Header {
				req.Header.Set(k, v)
			}
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		return resp.Header.Get("Ipfs-Hash"), nil
	}
}
