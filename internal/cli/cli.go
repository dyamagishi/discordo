// Package cli implements the read-only command line interface of discordo.
//
// The CLI shares the token stored by the TUI in the OS keyring and only uses
// the REST API, so it never opens a gateway connection and never conflicts
// with a running TUI session.
package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/auth"
	dhttp "github.com/ayn2op/discordo/internal/http"
)

// commands are the arguments that the CLI handles instead of the TUI.
var commands = []string{
	"guild",
	"channel",
	"msg",
	"search",
	"thread",
	"member",
	"pin",
	"dm",
	"user",
	"auth",
	"help",
}

// IsCommand reports whether name is a CLI subcommand.
func IsCommand(name string) bool {
	for _, command := range commands {
		if command == name {
			return true
		}
	}

	return false
}

// Run executes the CLI. It returns an error whose exit code can be read with
// ExitCode.
func Run(args []string) error {
	command, rest := args[0], args[1:]

	var err error
	switch command {
	case "help":
		usage()
	case "auth":
		err = runAuth(rest)
	case "guild":
		err = runGuild(rest)
	case "channel":
		err = runChannel(rest)
	case "msg":
		err = runMsg(rest)
	case "search":
		err = runSearch(rest)
	case "thread":
		err = runThread(rest)
	case "member":
		err = runMember(rest)
	case "pin":
		err = runPin(rest)
	case "dm":
		err = runDM(rest)
	case "user":
		err = runUser(rest)
	default:
		err = usagef("unknown command %q", command)
	}

	if err != nil {
		writeError(err)
		return fail(err)
	}

	return nil
}

func usage() {
	fmt.Fprint(os.Stderr, `discordo - Discord terminal client

Run without arguments to start the TUI. The commands below are read-only and
print JSON on stdout; errors are printed as JSON on stderr with a non-zero exit
code.

Usage:
  discordo                          start the TUI
  discordo -version                 print the version

  discordo auth status              show the logged in account

  discordo guild list [-limit N]
  discordo guild get <guild-id>

  discordo channel list <guild-id>
  discordo channel get <channel-id>

  discordo msg list <channel-id> [-limit N] [-before ID] [-after ID]
  discordo msg get <channel-id> <message-id>

  discordo search <guild-id> [-query TEXT] [-channel ID] [-author ID]
                            [-offset N] [-sort asc|desc]

  discordo thread active <guild-id>
  discordo thread archived <channel-id> [-limit N]

  discordo member list <guild-id> [-limit N]
  discordo member get <guild-id> <user-id>

  discordo pin list <channel-id>

  discordo dm list

  discordo user me
  discordo user get <user-id>

Exit codes:
  0 success  1 failure  2 usage  3 authentication  4 Discord API error
`)
}

// newClient returns a REST client that uses the same token, browser headers and
// TLS fingerprint as the TUI.
func newClient() (*api.Client, error) {
	token, err := auth.Token()
	if err != nil {
		return nil, err
	}

	client := dhttp.NewClient(token)
	client.OnRequest = append(client.OnRequest, httputil.WithHeaders(dhttp.Headers()))

	return client, nil
}

// newFlagSet returns a flag set that does not print its own errors, since the
// CLI reports everything as JSON.
func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet("discordo "+name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {}

	return fs
}

// parseFlags parses args, allowing positional arguments and flags to be mixed.
// The flag package stops at the first positional argument, which is surprising
// for an agent-driven CLI, so flags are separated first.
func parseFlags(fs *flag.FlagSet, args []string) ([]string, error) {
	var (
		flags      []string
		positional []string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if len(arg) > 1 && arg[0] == '-' {
			flags = append(flags, arg)

			name := strings.TrimLeft(arg, "-")
			if eq := strings.IndexByte(name, '='); eq >= 0 {
				continue
			}

			f := fs.Lookup(name)
			if f == nil || !isBoolFlag(f) {
				if i+1 < len(args) {
					i++
					flags = append(flags, args[i])
				}
			}

			continue
		}

		positional = append(positional, arg)
	}

	if err := fs.Parse(flags); err != nil {
		return nil, usagef("%s: %v", fs.Name(), err)
	}

	return positional, nil
}

func isBoolFlag(f *flag.Flag) bool {
	type boolFlag interface{ IsBoolFlag() bool }

	bf, ok := f.Value.(boolFlag)
	return ok && bf.IsBoolFlag()
}

// requireArgs checks that exactly one positional argument is present.
func requireArgs(fs *flag.FlagSet, args []string, usage string) (string, error) {
	if len(args) == 0 {
		return "", usagef("%s: missing %s", fs.Name(), usage)
	}
	if len(args) > 1 {
		return "", usagef("%s: unexpected argument %q", fs.Name(), args[1])
	}

	return args[0], nil
}

// authFromEnv reports whether the token comes from the environment.
func authFromEnv() bool { return auth.FromEnv() }

func noArgs(fs *flag.FlagSet, args []string) error {
	if len(args) > 0 {
		return usagef("%s: unexpected argument %q", fs.Name(), args[0])
	}

	return nil
}
