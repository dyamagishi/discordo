package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/discordo/internal/auth"
	"github.com/zalando/go-keyring"
)

// No parallel tests: endpoint globals and standard streams belong to the
// process. Use the real production HTTP client and API decoding with a local
// server, so dependency updates exercise the CLI's actual REST contract.
func localAPI(t *testing.T, status int, body string) <-chan *http.Request {
	t.Helper()
	return localAPIHandler(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	})
}

func localAPIHandler(t *testing.T, handler http.HandlerFunc) <-chan *http.Request {
	t.Helper()
	requests := make(chan *http.Request, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r
		w.Header().Set("Content-Type", "application/json")
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	for endpoint, suffix := range map[*string]string{
		&api.EndpointMe: "/users/@me", &api.EndpointUsers: "/users/",
		&api.EndpointGuilds: "/guilds/", &api.EndpointChannels: "/channels/",
	} {
		previous := *endpoint
		*endpoint = server.URL + suffix
		t.Cleanup(func() { *endpoint = previous })
	}
	keyring.MockInit()
	t.Setenv(auth.EnvVar, "test-token")
	return requests
}

func runCaptured(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	errOut, err := os.CreateTemp(t.TempDir(), "stderr")
	if err != nil {
		t.Fatal(err)
	}
	defer errOut.Close()
	stdout, stderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = out, errOut
	defer func() { os.Stdout, os.Stderr = stdout, stderr }()
	runErr := Run(args)
	read := func(f *os.File) string {
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(f)
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	return read(out), read(errOut), runErr
}

// Assert only the public fields relevant to each command, allowing additive
// fields while detecting changes in types, array order and pagination.
func assertJSONContains(t *testing.T, got, want string) {
	t.Helper()
	var actual, expected any
	if err := json.Unmarshal([]byte(got), &actual); err != nil {
		t.Fatalf("invalid JSON %q: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &expected); err != nil {
		t.Fatal(err)
	}
	var check func(any, any)
	check = func(got, want any) {
		t.Helper()
		switch want := want.(type) {
		case map[string]any:
			object, ok := got.(map[string]any)
			if !ok {
				t.Fatalf("expected object, got %#v", got)
			}
			for key, value := range want {
				check(object[key], value)
			}
		case []any:
			array, ok := got.([]any)
			if !ok || len(array) != len(want) {
				t.Fatalf("expected array %#v, got %#v", want, got)
			}
			for i := range want {
				check(array[i], want[i])
			}
		default:
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("JSON value = %#v, want %#v; response: %#v", got, want, actual)
			}
		}
	}
	check(actual, expected)
}

func TestCommandRESTContracts(t *testing.T) {
	const user = `{"id":"123","username":"alice"}`
	const channel = `{"id":"456","type":0,"name":"general"}`
	const messages = `[{"id":"789","channel_id":"456","content":"newer"},{"id":"788","channel_id":"456","content":"older"}]`
	tests := []struct {
		name, args, target, response, want string
	}{
		{"account", "user me", "/users/@me", user, `{"status":"ok","user":{"id":"123","username":"alice"}}`},
		{"user", "user get 123", "/users/123", user, `{"user":{"id":"123"}}`},
		{"auth", "auth status", "/users/@me", user, `{"account":{"id":"123"},"token_source":"env"}`},
		{"guilds", "guild list -limit 2", "/users/@me/guilds?limit=2&with_counts=false", `[{"id":"123","name":"guild"}]`, `{"count":1,"guilds":[{"id":"123","name":"guild"}]}`},
		{"guild", "guild get 123", "/guilds/123?with_counts=true", `{"id":"123","approximate_member_count":42,"approximate_presence_count":7}`, `{"guild":{"id":"123","member_count":42,"online_count":7}}`},
		{"channels", "channel list 123", "/guilds/123/channels", "[" + channel + "]", `{"count":1,"channels":[{"id":"456","type_name":"guild_text"}]}`},
		{"channel", "channel get 456", "/channels/456", channel, `{"channel":{"id":"456","name":"general"}}`},
		{"messages", "msg list 456 -limit 2", "/channels/456/messages?limit=2", messages, `{"count":2,"messages":[{"id":"788","content":"older"},{"id":"789","content":"newer"}],"next_before":"788"}`},
		{"before", "msg list 456 -limit 2 -before 790", "/channels/456/messages?limit=2&before=790", "[]", `{"count":0,"messages":[]}`},
		{"after", "msg list 456 -limit 2 -after 787", "/channels/456/messages?limit=2&after=787", "[]", `{"count":0,"messages":[]}`},
		{"message uses user endpoint", "msg get 456 789", "/channels/456/messages?limit=5&around=789", messages, `{"message":{"id":"789","content":"newer"}}`},
		{"search", "search 123 -query hello -channel 456 -author 321 -offset 2 -sort asc", "/guilds/123/messages/search?content=hello&channel_id=456&author_id=321&offset=2&sort_order=asc", `{"messages":[` + messages + `],"total_results":9}`, `{"count":2,"offset":2,"total_results":9,"messages":[{"id":"788"},{"id":"789"}]}`},
		{"threads use user endpoint", "thread list 456 -query topic -archived -limit 2", "/channels/456/threads/search?name=topic&archived=true&limit=2&sort_by=last_message_time", `{"threads":[],"has_more":true,"total_results":5}`, `{"threads":[],"count":0,"has_more":true,"total_results":5}`},
		{"archived threads", "thread archived 456 -limit 2", "/channels/456/threads/archived/public?limit=2", `{"threads":[],"has_more":false}`, `{"threads":[],"count":0,"has_more":false}`},
		{"member", "member get 123 321", "/guilds/123/members/321", `{"user":{"id":"321","username":"bob"},"nick":"B"}`, `{"member":{"user":{"id":"321"},"nick":"B"}}`},
		{"pins", "pin list 456", "/channels/456/pins", messages, `{"count":2,"messages":[{"id":"788"},{"id":"789"}]}`},
		{"DMs", "dm list", "/users/@me/channels", `[{"id":"456","type":1,"recipients":[` + user + `]}]`, `{"channels":[{"id":"456","type_name":"dm","recipients":[{"id":"123"}]}],"count":1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := localAPI(t, http.StatusOK, tt.response)
			out, stderr, err := runCaptured(t, strings.Fields(tt.args)...)
			if err != nil || stderr != "" {
				t.Fatalf("Run: %v, stderr: %s", err, stderr)
			}
			assertJSONContains(t, out, tt.want)
			if len(requests) != 1 {
				t.Fatalf("expected one REST request, got %d", len(requests))
			}
			r := <-requests
			want, err := url.ParseRequestURI(tt.target)
			if err != nil {
				t.Fatal(err)
			}
			if r.Method != http.MethodGet || r.URL.Path != want.Path || !reflect.DeepEqual(r.URL.Query(), want.Query()) {
				t.Fatalf("request = %s %s, want GET %s", r.Method, r.URL, tt.target)
			}
			if r.Header.Get("Authorization") != "test-token" || r.Header.Get("User-Agent") == "" || r.Header.Get("X-Super-Properties") == "" {
				t.Fatal("missing shared authentication or browser headers")
			}
		})
	}
}

func TestCommandErrors(t *testing.T) {
	for _, tt := range []struct {
		name, args, body, reason string
		status, exit             int
	}{
		{"usage", "msg list 456 -limit 101", `{}`, "usage", 200, 2},
		{"unauthorized", "user me", `{"message":"Unauthorized","code":0}`, "unauthorized", 401, 3},
		{"forbidden", "user me", `{"message":"Missing Access","code":50001}`, "forbidden", 403, 4},
		{"missing", "user me", `{"message":"Unknown User","code":10013}`, "not_found", 404, 4},
		{"rate limited", "user me", `{"message":"Rate limited","retry_after":0,"global":false}`, "rate_limited", 429, 4},
		{"message absent from window", "msg get 456 789", `[]`, "failure", 200, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := localAPI(t, tt.status, tt.body)
			out, stderr, err := runCaptured(t, strings.Fields(tt.args)...)
			if code, ok := ExitCode(err); !ok || code != tt.exit || out != "" {
				t.Fatalf("exit = %d (%v), stdout = %q, err = %v", code, ok, out, err)
			}
			assertJSONContains(t, stderr, `{"status":"error","reason":"`+tt.reason+`"}`)
			if tt.name == "usage" && len(requests) != 0 {
				t.Fatal("invalid arguments made an API request")
			}
		})
	}
}

func TestGuildListFetchesAllPagesByDefault(t *testing.T) {
	requests := localAPIHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("after") != "" {
			_, _ = io.WriteString(w, `[{"id":"201","name":"last"}]`)
			return
		}
		_, _ = io.WriteString(w, "[")
		for id := 1; id <= 200; id++ {
			if id > 1 {
				_, _ = io.WriteString(w, ",")
			}
			_, _ = fmt.Fprintf(w, `{"id":"%d","name":"guild"}`, id)
		}
		_, _ = io.WriteString(w, "]")
	})
	out, stderr, err := runCaptured(t, "guild", "list")
	if err != nil || stderr != "" {
		t.Fatalf("Run: %v, %s", err, stderr)
	}
	var result struct {
		Count  int         `json:"count"`
		Guilds []guildJSON `json:"guilds"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.Count != 201 || len(result.Guilds) != 201 || result.Guilds[0].ID != "1" || result.Guilds[200].ID != "201" {
		t.Fatal("default guild list truncated or reordered the pages")
	}
	if len(requests) != 2 {
		t.Fatalf("expected two pages, got %d requests", len(requests))
	}
	first, second := <-requests, <-requests
	if first.URL.Path != "/users/@me/guilds" || second.URL.Path != first.URL.Path || first.URL.Query().Get("after") != "" || second.URL.Query().Get("after") != "200" {
		t.Fatalf("unexpected page cursors: %s, %s", first.URL, second.URL)
	}
}

func TestCLISharesKeyring(t *testing.T) {
	requests := localAPI(t, 200, `{"id":"123","username":"alice"}`)
	t.Setenv(auth.EnvVar, "")
	if err := auth.SetToken("test-token"); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := runCaptured(t, "auth", "status")
	if err != nil || stderr != "" {
		t.Fatalf("Run: %v, %s", err, stderr)
	}
	assertJSONContains(t, out, `{"token_source":"keyring","account":{"id":"123"}}`)
	if len(requests) != 1 || (<-requests).Header.Get("Authorization") != "test-token" {
		t.Fatal("CLI did not use the shared token")
	}
	if err := auth.DeleteToken(); err != nil {
		t.Fatal(err)
	}
	out, stderr, err = runCaptured(t, "user", "me")
	if code, ok := ExitCode(err); !ok || code != 3 || out != "" || len(requests) != 0 {
		t.Fatalf("missing token: %v, stdout=%q, requests=%d", err, out, len(requests))
	}
	assertJSONContains(t, stderr, `{"reason":"no_token"}`)
}

func TestAuthLoginValidatesBeforeReplacingSharedToken(t *testing.T) {
	for _, tt := range []struct {
		name, input, response, wantToken string
		status, exit                     int
	}{
		{"valid", "  pasted-token\n", `{"id":"123","username":"alice"}`, "pasted-token", 200, 0},
		{"rejected", "pasted-token\n", `{"message":"Unauthorized","code":0}`, "original-token", 401, 3},
		{"empty", "\n", `{}`, "original-token", 200, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			requests := localAPI(t, tt.status, tt.response)
			t.Setenv(auth.EnvVar, "")
			if err := auth.SetToken("original-token"); err != nil {
				t.Fatal(err)
			}
			stdin, err := os.CreateTemp(t.TempDir(), "stdin")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = stdin.Close() })
			if _, err := io.WriteString(stdin, tt.input); err != nil {
				t.Fatal(err)
			}
			if _, err := stdin.Seek(0, io.SeekStart); err != nil {
				t.Fatal(err)
			}
			previous := os.Stdin
			os.Stdin = stdin
			t.Cleanup(func() { os.Stdin = previous })
			out, stderr, err := runCaptured(t, "auth", "login")
			code, _ := ExitCode(err)
			if code != tt.exit || (tt.exit == 0 && err != nil) {
				t.Fatalf("exit = %d, err = %v, stderr = %s", code, err, stderr)
			}
			if got, err := auth.Token(); err != nil || got != tt.wantToken {
				t.Fatalf("stored token = %q, %v", got, err)
			}
			if tt.exit == 0 {
				assertJSONContains(t, out, `{"status":"ok","token_source":"keyring","account":{"id":"123"}}`)
				if stderr != "" {
					t.Fatalf("unexpected stderr: %s", stderr)
				}
			} else if out != "" || !json.Valid([]byte(stderr)) {
				t.Fatalf("expected only a JSON error: stdout=%q stderr=%q", out, stderr)
			}
			if tt.name == "empty" {
				if len(requests) != 0 {
					t.Fatal("empty input sent an API request")
				}
			} else if len(requests) != 1 || (<-requests).Header.Get("Authorization") != "pasted-token" {
				t.Fatal("login did not validate the pasted token")
			}
			if strings.Contains(out+stderr, "pasted-token") || strings.Contains(out+stderr, "original-token") {
				t.Fatal("token leaked into CLI output")
			}
		})
	}
}
