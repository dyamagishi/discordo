package auth

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// The in-memory provider keeps tests away from the user's real keyring.
func TestSharedTokenStore(t *testing.T) {
	keyring.MockInit()
	t.Setenv(EnvVar, "")
	if _, err := Token(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("missing token: %v", err)
	}
	if err := SetToken("stored-token"); err != nil {
		t.Fatal(err)
	}
	if got, err := keyring.Get(Service, User); err != nil || got != "stored-token" {
		t.Fatalf("shared keyring entry = %q, %v", got, err)
	}
	t.Setenv(EnvVar, "environment-token")
	if got, err := Token(); err != nil || got != "environment-token" {
		t.Fatalf("environment override = %q, %v", got, err)
	}
	if err := SetToken("rotated-token"); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvVar, "")
	if got, err := Token(); err != nil || got != "stored-token" {
		t.Fatalf("environment must not overwrite keyring: %q, %v", got, err)
	}
	if err := SetToken(""); err != nil {
		t.Fatal(err)
	}
	if got, _ := Token(); got != "stored-token" {
		t.Fatal("empty token overwrote the shared entry")
	}
	if err := DeleteToken(); err != nil {
		t.Fatal(err)
	}
	if _, err := Token(); !errors.Is(err, ErrNoToken) {
		t.Fatalf("deleted token: %v", err)
	}
}
