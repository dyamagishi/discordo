package password

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"

	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

type TokenMsg string

// inputMsg is the text and cursor that typing in an input left.
type inputMsg struct {
	text   string
	cursor int
}

type (
	loginMsg    inputMsg
	passwordMsg inputMsg
	focusMsg    focus
	hoverMsg    bool
	submitMsg   struct{}
)

func submitLogin(login, password string) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		// Record that the password tab (not QR or token) is what is sending a
		// login request, without writing the credentials to the log.
		slog.Info("submitting password login",
			"login_len", len(login),
			"login_has_at", strings.Contains(login, "@"),
			"password_len", len(password),
		)

		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")

		client := http.NewClient("")
		client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))

		loginResp, err := client.Login(login, password)
		if err != nil {
			return loginError(err)
		}

		if loginResp.Token != "" {
			return TokenMsg(loginResp.Token)
		}

		if loginResp.MFA {
			return errors.New("multi-factor authentication is required; use token or QR login")
		}

		return errors.New("login response did not include a token")
	})
}

// loginError turns Discord's captcha rejection into an actionable message.
// Discord requires hCaptcha for password logins it considers suspicious, and
// the challenge cannot be solved from the terminal.
func loginError(err error) error {
	var httpErr *httputil.HTTPError
	if errors.As(err, &httpErr) && bytes.Contains(httpErr.Body, []byte("captcha-required")) {
		return errors.New("Discord requires a captcha for password login; use QR or token login instead")
	}

	return err
}
