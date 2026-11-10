// Package auth provides non-TUI access to the Discord token.
//
// The TUI stores the token in the OS keyring; the CLI reads the same entry so
// that logging in once in the TUI is enough for both.
package auth

import (
	"errors"
	"os"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/zalando/go-keyring"
)

const (
	// Service is the keyring service the TUI writes to.
	Service = consts.Name
	// User is the keyring entry the TUI writes to.
	User = "token"
	// EnvVar overrides the keyring. A token from the environment is never
	// persisted.
	EnvVar = "DISCORDO_TOKEN"
)

// ErrNoToken is returned when no token is available.
var ErrNoToken = errors.New("no Discord token found: run `discordo` and log in, or set " + EnvVar)

// Token returns the token from the environment, falling back to the keyring.
func Token() (string, error) {
	if token := os.Getenv(EnvVar); token != "" {
		return token, nil
	}

	token, err := keyring.Get(Service, User)
	if err != nil || token == "" {
		return "", ErrNoToken
	}

	return token, nil
}

// FromEnv reports whether the token comes from the environment.
func FromEnv() bool { return os.Getenv(EnvVar) != "" }

// SetToken stores the token in the keyring. Tokens provided through the
// environment are not persisted, since the environment is the source of truth
// for those.
func SetToken(token string) error {
	if FromEnv() || token == "" {
		return nil
	}

	return keyring.Set(Service, User, token)
}

// DeleteToken removes the stored token.
func DeleteToken() error {
	return keyring.Delete(Service, User)
}
