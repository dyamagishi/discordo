package chat

import (
	"log/slog"

	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/auth"
)

// persistRefreshedToken keeps the CLI's shared keyring entry current when
// Discord rotates the token. The TUI only needs to register this handler.
func persistRefreshedToken(event gateway.Event) {
	ready, ok := event.(*gateway.ReadyEvent)
	if !ok || ready.AuthToken == "" {
		return
	}
	if err := auth.SetToken(ready.AuthToken); err != nil {
		slog.Error("failed to persist refreshed token", "err", err)
	}
}
