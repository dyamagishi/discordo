package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/auth"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

type tokenMsg string

func useToken(token string) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		return tokenMsg(token)
	})
}

type loginMsg struct{}

func getToken() tview.Cmd {
	return cmd.Task(func() tview.Msg {
		token, err := auth.Token()
		if err != nil {
			slog.Info("failed to retrieve token", "err", err)
			return loginMsg{}
		}
		return tokenMsg(token)
	})
}

func setToken(token string) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if err := auth.SetToken(token); err != nil {
			slog.Error("failed to set token to keyring", "err", err)
		}
		return nil
	})
}

func deleteToken() tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if err := auth.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
		}
		return nil
	})
}
