package config

import (
	"encoding/json"
)

const (
	// DefaultSessionTTL is the default TTL before a Session closes
	DefaultSessionTTL = 15 * 60 // 15 minutes in seconds
	// DefaultMaxLoginAttempts is the number of times failed login attempts are allowed
	DefaultMaxLoginAttempts = 10
)

// Config holds global settings used by the app
// These may end up just being global constants.
type Config struct {
	TTL                 int
	FailedLoginAttempts int
	MaxLoginAttempts    int
}

// NewConfig creates a new global config struct with default settings.
// This is primarily used for initializing a new data store
func NewConfig() Config {
	return Config{
		TTL:                 DefaultSessionTTL,
		FailedLoginAttempts: 0,
		MaxLoginAttempts:    DefaultMaxLoginAttempts,
	}
}

// ToJSON is a helper method for GlobalConfig
func (g Config) ToJSON() []byte {
	b, _ := json.Marshal(g)
	return b
}
