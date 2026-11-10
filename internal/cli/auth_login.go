package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ayn2op/discordo/internal/auth"
	"golang.org/x/term"
)

func authLogin(args []string) error {
	fs := newFlagSet("auth login")
	positional, err := parseFlags(fs, args)
	if err != nil {
		return err
	}
	if err := noArgs(fs, positional); err != nil {
		return err
	}

	token, err := readToken(os.Stdin)
	if err != nil {
		return err
	}
	if token == "" {
		return usagef("auth login: empty token")
	}

	// Validate before storing, so a copy/paste mistake does not replace a
	// working token with a broken one.
	client := newClientWithToken(token)
	me, err := client.Me()
	if err != nil {
		return fail(err)
	}

	if err := auth.SetToken(token); err != nil {
		return fmt.Errorf("failed to store the token in the keyring: %w", err)
	}

	return writeOKWith("account", newUser(*me), -1, map[string]any{
		"token_source": "keyring",
	})
}

// readToken reads a token from stdin. Terminals are read without echo, so a
// pasted token does not stay on screen or in the scrollback.
func readToken(r io.Reader) (string, error) {
	if f, ok := r.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		fmt.Fprint(os.Stderr, "Paste the token and press Enter: ")
		raw, err := term.ReadPassword(int(f.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(raw)), nil
	}

	raw, err := bufio.NewReader(r).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	return strings.TrimSpace(raw), nil
}
