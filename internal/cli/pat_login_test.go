package cli

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFetchProtectedResourceMetadataRequiresResource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"authorization_servers":["https://callbacks.omniapp.co"]}`)
	}))
	defer server.Close()

	_, err := fetchProtectedResourceMetadata(server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "missing resource") {
		t.Fatalf("expected missing resource error, got %v", err)
	}
}

func TestFetchAuthorizationServerMetadataRequiresEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"authorization_endpoint":"https://auth.example/authorize"}`)
	}))
	defer server.Close()

	_, err := fetchAuthorizationServerMetadata(server.Client(), server.URL)
	if err == nil || !strings.Contains(err.Error(), "incomplete authorization-server metadata") {
		t.Fatalf("expected incomplete metadata error, got %v", err)
	}
}

func TestRegisterOAuthClientRequiresClientID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected JSON content type, got %q", got)
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	_, err := registerOAuthClient(server.Client(), server.URL, "http://127.0.0.1:1234/callback")
	if err == nil || !strings.Contains(err.Error(), "missing client_id") {
		t.Fatalf("expected missing client_id error, got %v", err)
	}
}

func TestBuildAuthorizationURLIncludesPKCEAndResource(t *testing.T) {
	rawURL, err := buildAuthorizationURL(
		"https://callbacks.omniapp.co/callback/mcp/oauth/authorize",
		"https://callbacks.omniapp.co/callback/mcp",
		"client-123",
		"http://127.0.0.1:9999/callback",
		"state-xyz",
		"verifier-abc",
	)
	if err != nil {
		t.Fatalf("buildAuthorizationURL returned error: %v", err)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse authorization URL: %v", err)
	}
	query := parsed.Query()
	if query.Get("client_id") != "client-123" {
		t.Fatalf("expected client_id, got %q", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "http://127.0.0.1:9999/callback" {
		t.Fatalf("expected redirect_uri, got %q", query.Get("redirect_uri"))
	}
	if query.Get("resource") != "https://callbacks.omniapp.co/callback/mcp" {
		t.Fatalf("expected resource, got %q", query.Get("resource"))
	}
	if query.Get("code_challenge") != pkceS256("verifier-abc") {
		t.Fatalf("expected PKCE code challenge, got %q", query.Get("code_challenge"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Fatalf("expected S256 code challenge method, got %q", query.Get("code_challenge_method"))
	}
}

func TestExchangeAuthorizationCodeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
			t.Fatalf("expected form content type, got %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Fatalf("expected auth code grant, got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("resource") != "https://callbacks.omniapp.co/callback/mcp" {
			t.Fatalf("expected resource, got %q", r.Form.Get("resource"))
		}
		_, _ = io.WriteString(w, `{"access_token":"pat-123","token_type":"Bearer","scope":"mcp:access"}`)
	}))
	defer server.Close()

	tokenResp, err := exchangeAuthorizationCode(
		server.Client(),
		server.URL,
		"https://callbacks.omniapp.co/callback/mcp",
		"client-123",
		"http://127.0.0.1:9999/callback",
		"auth-code",
		"code-verifier",
	)
	if err != nil {
		t.Fatalf("exchangeAuthorizationCode returned error: %v", err)
	}
	if tokenResp.AccessToken != "pat-123" || tokenResp.TokenType != "Bearer" {
		t.Fatalf("unexpected token response: %#v", tokenResp)
	}
}

func TestExchangeAuthorizationCodeReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadRequest)
	}))
	defer server.Close()

	_, err := exchangeAuthorizationCode(server.Client(), server.URL, "", "client-123", "http://127.0.0.1:9999/callback", "auth-code", "code-verifier")
	if err == nil || !strings.Contains(err.Error(), "400 Bad Request") {
		t.Fatalf("expected token endpoint HTTP error, got %v", err)
	}
}

func TestStartLoopbackListenerAndCallbackServer(t *testing.T) {
	redirectURI, listener, err := startLoopbackListener()
	if err != nil {
		t.Fatalf("startLoopbackListener returned error: %v", err)
	}
	defer listener.Close()

	callbackCh := make(chan oauthCallbackResult, 1)
	serverErrCh := make(chan error, 1)
	srv := startOAuthCallbackServer(listener, callbackCh, serverErrCh)
	defer srv.Close()

	resp, err := http.Get(redirectURI + "?code=auth-code&state=state-123")
	if err != nil {
		t.Fatalf("GET callback failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read callback response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "Omni login complete") {
		t.Fatalf("expected success body, got %q", string(body))
	}

	select {
	case result := <-callbackCh:
		if result.Code != "auth-code" || result.State != "state-123" || result.Err != "" {
			t.Fatalf("unexpected callback result: %#v", result)
		}
	case err := <-serverErrCh:
		t.Fatalf("callback server returned error: %v", err)
	}
}

func TestObtainPATValidatesInputsAndTrimsToken(t *testing.T) {
	if _, err := obtainPAT(nil, "https://acme.omniapp.co"); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("expected unavailable PAT login error, got %v", err)
	}
	if _, err := obtainPAT(&runtime{PATLogin: func(baseURL string) (string, error) { return "token", nil }}, " "); err == nil || !strings.Contains(err.Error(), "missing Omni URL") {
		t.Fatalf("expected missing URL error, got %v", err)
	}

	token, err := obtainPAT(&runtime{
		PATLogin: func(baseURL string) (string, error) {
			if baseURL != "https://acme.omniapp.co" {
				t.Fatalf("expected base URL to be passed through, got %q", baseURL)
			}
			return " pat-token ", nil
		},
	}, "https://acme.omniapp.co")
	if err != nil {
		t.Fatalf("obtainPAT returned error: %v", err)
	}
	if token != "pat-token" {
		t.Fatalf("expected trimmed token, got %q", token)
	}
}
