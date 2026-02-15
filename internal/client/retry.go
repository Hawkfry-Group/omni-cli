package client

import (
	"errors"
	"net/http"
	"strconv"
	"time"
)

type retryTransport struct {
	base       http.RoundTripper
	maxRetries int
}

func newRetryTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &retryTransport{base: base, maxRetries: 3}
}

func (t *retryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !isRetryableMethod(req.Method) {
		return t.base.RoundTrip(req)
	}

	var lastErr error
	for attempt := 0; attempt <= t.maxRetries; attempt++ {
		resp, err := t.base.RoundTrip(req)
		if err != nil {
			lastErr = err
			if attempt == t.maxRetries || !isRetryableError(err) {
				return nil, err
			}
			sleepBackoff(attempt, 0, req)
			continue
		}

		if !isRetryableStatus(resp.StatusCode) || attempt == t.maxRetries {
			return resp, nil
		}

		_ = resp.Body.Close()
		retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
		sleepBackoff(attempt, retryAfter, req)
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("retry transport exhausted attempts")
}

func isRetryableMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return true
	default:
		return false
	}
}

func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

func isRetryableError(err error) bool {
	// Network-layer errors are generally safe to retry for idempotent requests.
	return err != nil
}

func sleepBackoff(attempt int, retryAfter time.Duration, req *http.Request) {
	if retryAfter > 0 {
		select {
		case <-req.Context().Done():
			return
		case <-time.After(retryAfter):
			return
		}
	}
	base := 200 * time.Millisecond
	d := base * time.Duration(1<<attempt)
	if d > 2*time.Second {
		d = 2 * time.Second
	}
	select {
	case <-req.Context().Done():
		return
	case <-time.After(d):
	}
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if ts, err := http.ParseTime(v); err == nil {
		d := time.Until(ts)
		if d > 0 {
			return d
		}
	}
	return 0
}
