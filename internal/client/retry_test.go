package client

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

func TestRetryTransportRetriesGetOn503(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		maxRetries: 1,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return &http.Response{
					StatusCode: 503,
					Body:       io.NopCloser(strings.NewReader("{}")),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	req, _ := http.NewRequest(http.MethodGet, "https://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", calls)
	}
}

func TestRetryTransportDoesNotRetryPost(t *testing.T) {
	calls := 0
	rt := &retryTransport{
		maxRetries: 3,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: 503,
				Body:       io.NopCloser(strings.NewReader("{}")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	req, _ := http.NewRequest(http.MethodPost, "https://example.com", strings.NewReader("{}"))
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	if resp.StatusCode != 503 {
		t.Fatalf("expected status 503, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Fatalf("expected single attempt for POST, got %d", calls)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	d := parseRetryAfter("2")
	if d < 2_000_000_000 || d > 2_100_000_000 {
		t.Fatalf("expected around 2s, got %v", d)
	}
}

func TestRetryHelpers(t *testing.T) {
	if _, ok := newRetryTransport(nil).(*retryTransport); !ok {
		t.Fatal("expected newRetryTransport to return retryTransport")
	}
	if !isRetryableError(io.EOF) {
		t.Fatal("expected non-nil error to be retryable")
	}
	if isRetryableError(nil) {
		t.Fatal("expected nil error to be non-retryable")
	}

	future := time.Now().Add(1500 * time.Millisecond).UTC().Format(http.TimeFormat)
	if d := parseRetryAfter(future); d <= 0 {
		t.Fatalf("expected positive duration for HTTP date, got %v", d)
	}
	if d := parseRetryAfter("garbage"); d != 0 {
		t.Fatalf("expected zero duration for invalid retry-after, got %v", d)
	}
}

func TestSleepBackoffReturnsOnCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com", nil)

	start := time.Now()
	sleepBackoff(3, time.Second, req)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("expected canceled backoff to return quickly, took %v", elapsed)
	}
}
