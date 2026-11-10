package http

import (
	"bytes"
	"errors"

	"github.com/ayn2op/arikawa/v3/utils/httputil"
)

// IsCaptchaRequired reports whether Discord rejected a login with an hCaptcha
// challenge. The challenge cannot be solved from the terminal.
func IsCaptchaRequired(err error) bool {
	if err == nil {
		return false
	}

	var httpErr *httputil.HTTPError
	if errors.As(err, &httpErr) {
		return bytes.Contains(httpErr.Body, []byte("captcha-required"))
	}

	var httpVal httputil.HTTPError
	if errors.As(err, &httpVal) {
		return bytes.Contains(httpVal.Body, []byte("captcha-required"))
	}

	return false
}
