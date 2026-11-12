package chat

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/auth"
	"github.com/zalando/go-keyring"
)

func TestPersistRefreshedToken(t *testing.T) {
	keyring.MockInit()
	t.Setenv(auth.EnvVar, "")
	if err := auth.SetToken("original-token"); err != nil {
		t.Fatal(err)
	}
	persistRefreshedToken(&gateway.ReadyEvent{AuthToken: "refreshed-token"})
	if got, err := auth.Token(); err != nil || got != "refreshed-token" {
		t.Fatalf("CLI shared token = %q, %v", got, err)
	}
	persistRefreshedToken(&gateway.ReadyEvent{})
	persistRefreshedToken(&gateway.ResumedEvent{})
	if got, _ := auth.Token(); got != "refreshed-token" {
		t.Fatal("unrelated or empty event overwrote the token")
	}
	t.Setenv(auth.EnvVar, "environment-token")
	persistRefreshedToken(&gateway.ReadyEvent{AuthToken: "ignored-token"})
	t.Setenv(auth.EnvVar, "")
	if got, _ := auth.Token(); got != "refreshed-token" {
		t.Fatal("environment session overwrote the keyring")
	}
}
