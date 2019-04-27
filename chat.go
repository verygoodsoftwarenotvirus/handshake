package handshake

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

const (
	maxMessageSize = 250000 // ~2 Megabytes
	defaultChatTTL = 604800 // 7 days in seconds
)

type lookup map[string][]byte

// ChatLog represents a log of chat messages
type ChatLog map[string]ChatLogEntry

// ChatLogEntry represents an entry in a chat log
type ChatLogEntry struct {
	ID       string   `json:"id,omitempty"`
	Sender   string   `json:"sender,omitempty"`
	Sent     int64    `json:"sent,omitempty"`
	Received int64    `json:"received,omitempty"`
	TTL      int64    `json:"ttl,omitempty"`
	Data     chatData `json:"data"`
}

// SortedJSON sorts the chat log and renders it to a JSON representation
func (cl ChatLog) SortedJSON() ([]byte, error) {
	return json.Marshal(cl.Sorted())
}

// Sorted sorts the ChatLog's messages by key and returns just the messages
func (cl ChatLog) Sorted() []ChatLogEntry {
	var entries []ChatLogEntry
	var keys []string

	for k := range cl {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, v := range keys {
		entries = append(entries, cl[v])
	}
	return entries
}

// AddEntry adds an entry to the ChatLog
func (cl *ChatLog) AddEntry(entry ChatLogEntry) error {
	if (entry.Sent == 0) && (entry.Received == 0) {
		return errors.New("no valid timestamp found")
	}
	timestamp := entry.Sent
	if timestamp == 0 {
		timestamp = entry.Received
	}

	key := fmt.Sprintf("%v-%v", timestamp, entry.ID)
	(*cl)[key] = entry
	return nil
}

// HashInLog retursn whether or not a given hash is contained in the ChatLog
func (cl ChatLog) HashInLog(hash string) bool {
	for entry := range cl {
		if strings.Contains(entry, hash) {
			return true
		}
	}
	return false
}

type chatData struct {
	Parent    string   `json:"parent,omitempty"`
	Timestamp int64    `json:"timestamp,omitempty"`
	Media     []string `json:"media,omitempty"`
	Message   string   `json:"message,omitempty"`
	TTL       int64    `json:"ttl,omitempty"`
}

type chat struct {
	ID       string
	PeerID   string
	LastSent string
	Peers    map[string]chatPeer
	Settings chatSettings
}

// a chatConfig allows safe encoding of a chat
type chatConfig struct {
	ID       string
	PeerID   string
	LastSent string
	Peers    map[string]chatPeerConfig
	Settings chatSettings
}

type chatSettings struct {
	MaxTTL int64
}

// uniqueChatIDsFromPaths takes a lists of paths from and a profile ID and strips out unique ChatID
// entries and returns a slice of strings
func uniqueChatIDsFromPaths(list []string, profileID string) (ids []string) {
	idMap := make(map[string]struct{})
	for _, l := range list {
		if strings.Contains(l, profileID) {
			s := strings.Split(l, "/")
			idMap[s[1]] = struct{}{}
		}
	}
	for id := range idMap {
		ids = append(ids, id)
	}
	return
}

func newLookupFromGob(b []byte) (lookup, error) {
	l := make(lookup)
	var buffer bytes.Buffer
	buffer.Write(b)
	err := gob.NewDecoder(&buffer).Decode(&l)
	return l, err
}

func newChatFromGob(b []byte) (chat, error) {
	var config chatConfig
	var buffer bytes.Buffer
	buffer.Write(b)
	err := gob.NewDecoder(&buffer).Decode(&config)
	if err != nil {
		return chat{}, err
	}
	return config.Chat()
}

func newChatLogFromGob(b []byte) (ChatLog, error) {
	var cl ChatLog
	var buffer bytes.Buffer
	buffer.Write(b)
	err := gob.NewDecoder(&buffer).Decode(&cl)
	if err != nil {
		return ChatLog{}, err
	}
	return cl, nil
}

func (l lookup) get(key string) []byte {
	return l[key]
}

func (l *lookup) popKey(key string) []byte {
	v := l.get(key)
	delete(*l, key)
	return v
}

func (l *lookup) popRandom() (string, []byte) {
	k, v := l.getRandom()
	delete(*l, k)
	return k, v
}

func (l lookup) getRandom() (string, []byte) {
	x, _ := rand.Int(rand.Reader, big.NewInt(int64(len(l))))
	i := x.Int64()
	c := int64(0)
	for k, v := range l {
		if c == i {
			return k, v
		}
		c++
	}
	return "", []byte{}
}

// TODO:
// - get ChatLog
// - retrieve Chatmessages (this should query all peer's endpoints)
// -- this should be recursive and query chats until all either a hash match
// -- or the lookup hash doesn't exist
// -- or no parent exists
// - postToChat (this should post a message to chat)
// -

func (config chatConfig) Chat() (chat, error) {
	c := chat{
		ID:       config.ID,
		PeerID:   config.PeerID,
		LastSent: config.LastSent,
		Peers:    make(map[string]chatPeer),
		Settings: config.Settings,
	}
	for _, peerConfig := range config.Peers {
		peer, err := peerConfig.Peer()
		if err != nil {
			return chat{}, err
		}
		c.Peers[peer.ID] = peer
	}
	return c, nil
}

func (c chat) TTL() int64 {
	if c.Settings.MaxTTL <= 0 {
		return defaultChatTTL
	}
	return c.Settings.MaxTTL
}

func (c chat) Config() (chatConfig, error) {
	config := chatConfig{
		ID:       c.ID,
		PeerID:   c.PeerID,
		LastSent: c.LastSent,
		Peers:    make(map[string]chatPeerConfig),
		Settings: c.Settings,
	}

	for _, peer := range c.Peers {
		peerConfig, err := peer.Config()
		if err != nil {
			return chatConfig{}, err
		}
		config.Peers[peer.ID] = peerConfig
	}
	return config, nil
}

type chatPeer struct {
	ID       string
	Alias    string
	Strategy strategy
}

type chatPeerConfig struct {
	ID       string
	Alias    string
	Strategy strategyConfig
}

// Peer converts a chatPeerConfig into a chatPeer
func (config chatPeerConfig) Peer() (chatPeer, error) {
	peer := chatPeer{
		ID:    config.ID,
		Alias: config.Alias,
	}
	s, err := strategyFromConfig(config.Strategy)
	if err != nil {
		return chatPeer{}, err
	}
	peer.Strategy = s
	return peer, nil
}

// Config returns a storage-safe chatPeerConfig and an error
func (c chatPeer) Config() (chatPeerConfig, error) {
	config := chatPeerConfig{
		ID:    c.ID,
		Alias: c.Alias,
	}
	s, err := c.Strategy.Export()
	config.Strategy = s
	return config, err
}
