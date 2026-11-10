package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/auth"
)

// Exit codes used by the CLI.
const (
	exitFailure = 1
	exitUsage   = 2
	exitAuth    = 3
	exitHTTP    = 4
)

const (
	reasonUsage        = "usage"
	reasonNoToken      = "no_token"
	reasonUnauthorized = "unauthorized"
	reasonForbidden    = "forbidden"
	reasonNotFound     = "not_found"
	reasonRateLimited  = "rate_limited"
	reasonHTTP         = "http_error"
	reasonFailure      = "failure"
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// ExitCode reports the exit code of err, if it is a CLI error.
func ExitCode(err error) (int, bool) {
	var e *exitError
	if errors.As(err, &e) {
		return e.code, true
	}

	return 0, false
}

type errorBody struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Error  string `json:"error"`
	Code   int    `json:"code,omitempty"`
}

// writeOK writes a successful response. Every response carries a single data
// field under key, plus the status, so the shape stays stable for agents.
func writeOK(key string, payload any, count int) error {
	return writeOKWith(key, payload, count, nil)
}

// writeOKWith writes a successful response with additional top-level fields,
// such as pagination cursors.
func writeOKWith(key string, payload any, count int, extra map[string]any) error {
	wrapper := map[string]any{"status": "ok", key: payload}
	if count >= 0 {
		wrapper["count"] = count
	}
	for k, v := range extra {
		wrapper[k] = v
	}

	return writeJSON(os.Stdout, wrapper)
}

func writeJSON(f *os.File, v any) error {
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

func writeError(err error) {
	body := errorBody{Status: "error", Reason: reasonFailure}

	switch {
	case errors.Is(err, auth.ErrNoToken):
		body.Reason = reasonNoToken
	case isUsage(err):
		body.Reason = reasonUsage
	}

	var httpErr *httputil.HTTPError
	if errors.As(err, &httpErr) {
		body.Reason = httpReason(httpErr.Status)
		body.Code = httpErr.Status
	} else {
		var httpVal httputil.HTTPError
		if errors.As(err, &httpVal) {
			body.Reason = httpReason(httpVal.Status)
			body.Code = httpVal.Status
		}
	}

	body.Error = err.Error()
	_ = writeJSON(os.Stderr, body)
}

func httpReason(status int) string {
	switch status {
	case 401:
		return reasonUnauthorized
	case 403:
		return reasonForbidden
	case 404:
		return reasonNotFound
	case 429:
		return reasonRateLimited
	default:
		return reasonHTTP
	}
}

func httpExitCode(status int) int {
	if status == 401 {
		return exitAuth
	}
	return exitHTTP
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usagef(format string, args ...any) error {
	return &exitError{code: exitUsage, err: &usageError{msg: fmt.Sprintf(format, args...)}}
}

func isUsage(err error) bool {
	var e *usageError
	return errors.As(err, &e)
}

// fail wraps an error so that it is reported with the exit code that matches
// its cause.
func fail(err error) error {
	if err == nil {
		return nil
	}

	var e *exitError
	if errors.As(err, &e) {
		return err
	}

	code := exitFailure
	if errors.Is(err, auth.ErrNoToken) {
		code = exitAuth
	}

	var httpErr *httputil.HTTPError
	if errors.As(err, &httpErr) {
		code = httpExitCode(httpErr.Status)
	}

	return &exitError{code: code, err: err}
}
