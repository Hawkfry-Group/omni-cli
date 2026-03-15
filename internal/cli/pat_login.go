package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	goruntime "runtime"
	"strings"
	"time"
)

type PATLoginFunc func(baseURL string) (string, error)

type oauthProtectedResourceMetadata struct {
	AuthorizationServers []string `json:"authorization_servers"`
	Resource             string   `json:"resource"`
	ScopesSupported      []string `json:"scopes_supported"`
}

type oauthAuthorizationServerMetadata struct {
	AuthorizationEndpoint       string   `json:"authorization_endpoint"`
	CodeChallengeMethodsSupport []string `json:"code_challenge_methods_supported"`
	GrantTypesSupported         []string `json:"grant_types_supported"`
	Issuer                      string   `json:"issuer"`
	RegistrationEndpoint        string   `json:"registration_endpoint"`
	ResponseTypesSupported      []string `json:"response_types_supported"`
	ScopesSupported             []string `json:"scopes_supported"`
	TokenEndpoint               string   `json:"token_endpoint"`
}

type oauthDynamicClientRegistrationResponse struct {
	ClientID     string   `json:"client_id"`
	RedirectURIs []string `json:"redirect_uris"`
}

type oauthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

type oauthCallbackResult struct {
	Code  string
	State string
	Err   string
}

func defaultPATLogin(baseURL string) (string, error) {
	const callbackHost = "https://callbacks.omniapp.co"

	httpClient := &http.Client{Timeout: 30 * time.Second}

	resourceMeta, err := fetchProtectedResourceMetadata(httpClient, callbackHost+"/.well-known/oauth-protected-resource")
	if err != nil {
		return "", fmt.Errorf("fetch protected-resource metadata: %w", err)
	}
	authServerURL := callbackHost
	if len(resourceMeta.AuthorizationServers) > 0 && strings.TrimSpace(resourceMeta.AuthorizationServers[0]) != "" {
		authServerURL = strings.TrimSpace(resourceMeta.AuthorizationServers[0])
	}
	authMeta, err := fetchAuthorizationServerMetadata(httpClient, strings.TrimRight(authServerURL, "/")+"/.well-known/oauth-authorization-server")
	if err != nil {
		return "", fmt.Errorf("fetch authorization-server metadata: %w", err)
	}

	redirectURI, listener, err := startLoopbackListener()
	if err != nil {
		return "", fmt.Errorf("start loopback callback listener: %w", err)
	}
	defer listener.Close()

	clientReg, err := registerOAuthClient(httpClient, authMeta.RegistrationEndpoint, redirectURI)
	if err != nil {
		return "", fmt.Errorf("register OAuth client: %w", err)
	}

	codeVerifier, err := randomBase64URL(32)
	if err != nil {
		return "", fmt.Errorf("generate PKCE code verifier: %w", err)
	}
	state, err := randomBase64URL(24)
	if err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}

	callbackCh := make(chan oauthCallbackResult, 1)
	serverErrCh := make(chan error, 1)
	srv := startOAuthCallbackServer(listener, callbackCh, serverErrCh)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	authURL, err := buildAuthorizationURL(authMeta.AuthorizationEndpoint, resourceMeta.Resource, clientReg.ClientID, redirectURI, state, codeVerifier)
	if err != nil {
		return "", fmt.Errorf("build authorization URL: %w", err)
	}
	if err := openBrowser(authURL); err != nil {
		return "", fmt.Errorf("open browser for PAT login: %w", err)
	}

	callbackCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var callback oauthCallbackResult
	select {
	case callback = <-callbackCh:
	case err := <-serverErrCh:
		return "", fmt.Errorf("callback server failed: %w", err)
	case <-callbackCtx.Done():
		return "", fmt.Errorf("timed out waiting for browser login callback")
	}
	if callback.Err != "" {
		return "", fmt.Errorf("authorization failed: %s", callback.Err)
	}
	if callback.State != state {
		return "", fmt.Errorf("authorization state mismatch")
	}
	if strings.TrimSpace(callback.Code) == "" {
		return "", fmt.Errorf("authorization callback did not include a code")
	}

	tokenResp, err := exchangeAuthorizationCode(httpClient, authMeta.TokenEndpoint, resourceMeta.Resource, clientReg.ClientID, redirectURI, callback.Code, codeVerifier)
	if err != nil {
		return "", fmt.Errorf("exchange authorization code: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(tokenResp.TokenType), "bearer") {
		return "", fmt.Errorf("unexpected token type %q", tokenResp.TokenType)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return "", fmt.Errorf("token response did not include an access token")
	}
	_ = baseURL
	return strings.TrimSpace(tokenResp.AccessToken), nil
}

func fetchProtectedResourceMetadata(httpClient *http.Client, metadataURL string) (oauthProtectedResourceMetadata, error) {
	var meta oauthProtectedResourceMetadata
	if err := getJSON(httpClient, metadataURL, &meta); err != nil {
		return meta, err
	}
	if strings.TrimSpace(meta.Resource) == "" {
		return meta, fmt.Errorf("missing resource in protected-resource metadata")
	}
	return meta, nil
}

func fetchAuthorizationServerMetadata(httpClient *http.Client, metadataURL string) (oauthAuthorizationServerMetadata, error) {
	var meta oauthAuthorizationServerMetadata
	if err := getJSON(httpClient, metadataURL, &meta); err != nil {
		return meta, err
	}
	if strings.TrimSpace(meta.AuthorizationEndpoint) == "" || strings.TrimSpace(meta.TokenEndpoint) == "" || strings.TrimSpace(meta.RegistrationEndpoint) == "" {
		return meta, fmt.Errorf("incomplete authorization-server metadata")
	}
	return meta, nil
}

func registerOAuthClient(httpClient *http.Client, registrationEndpoint string, redirectURI string) (oauthDynamicClientRegistrationResponse, error) {
	body := map[string]any{
		"client_name":                "omni-cli",
		"application_type":           "native",
		"grant_types":                []string{"authorization_code"},
		"response_types":             []string{"code"},
		"redirect_uris":              []string{redirectURI},
		"token_endpoint_auth_method": "none",
	}

	var resp oauthDynamicClientRegistrationResponse
	if err := postJSON(httpClient, registrationEndpoint, body, &resp); err != nil {
		return resp, err
	}
	if strings.TrimSpace(resp.ClientID) == "" {
		return resp, fmt.Errorf("registration response missing client_id")
	}
	return resp, nil
}

func buildAuthorizationURL(authorizationEndpoint, resource, clientID, redirectURI, state, codeVerifier string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(authorizationEndpoint))
	if err != nil {
		return "", err
	}
	codeChallenge := pkceS256(codeVerifier)
	q := u.Query()
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "mcp:access")
	q.Set("state", state)
	q.Set("code_challenge", codeChallenge)
	q.Set("code_challenge_method", "S256")
	if strings.TrimSpace(resource) != "" {
		q.Set("resource", strings.TrimSpace(resource))
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func exchangeAuthorizationCode(httpClient *http.Client, tokenEndpoint, resource, clientID, redirectURI, code, codeVerifier string) (oauthTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("redirect_uri", redirectURI)
	form.Set("code", code)
	form.Set("code_verifier", codeVerifier)
	if strings.TrimSpace(resource) != "" {
		form.Set("resource", strings.TrimSpace(resource))
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(tokenEndpoint), strings.NewReader(form.Encode()))
	if err != nil {
		return oauthTokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return oauthTokenResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthTokenResponse{}, fmt.Errorf("token endpoint returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var tokenResp oauthTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return oauthTokenResponse{}, fmt.Errorf("parse token response: %w", err)
	}
	return tokenResp, nil
}

func startLoopbackListener() (string, net.Listener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		listener.Close()
		return "", nil, fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", addr.Port)
	return redirectURI, listener, nil
}

func startOAuthCallbackServer(listener net.Listener, callbackCh chan<- oauthCallbackResult, serverErrCh chan<- error) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		result := oauthCallbackResult{
			Code:  strings.TrimSpace(r.URL.Query().Get("code")),
			State: strings.TrimSpace(r.URL.Query().Get("state")),
			Err:   firstNonEmpty(r.URL.Query().Get("error"), r.URL.Query().Get("error_description")),
		}
		select {
		case callbackCh <- result:
		default:
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if result.Err != "" {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = io.WriteString(w, "<html><body><h1>Omni login failed</h1><p>You can close this window and return to the CLI.</p></body></html>")
			return
		}
		_, _ = io.WriteString(w, "<html><body><h1>Omni login complete</h1><p>You can close this window and return to the CLI.</p></body></html>")
	})

	srv := &http.Server{Handler: mux}
	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
	}()
	return srv
}

func getJSON(httpClient *http.Client, rawURL string, out any) error {
	resp, err := httpClient.Get(strings.TrimSpace(rawURL))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s: %s", rawURL, resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("parse %s response: %w", rawURL, err)
	}
	return nil
}

func postJSON(httpClient *http.Client, rawURL string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, strings.TrimSpace(rawURL), strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s returned %s: %s", rawURL, resp.Status, strings.TrimSpace(string(respBody)))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("parse %s response: %w", rawURL, err)
	}
	return nil
}

func pkceS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func randomBase64URL(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func openBrowser(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func obtainPAT(rt *runtime, baseURL string) (string, error) {
	if strings.TrimSpace(baseURL) == "" {
		return "", fmt.Errorf("missing Omni URL for PAT login")
	}
	if rt == nil || rt.PATLogin == nil {
		return "", fmt.Errorf("PAT login flow is unavailable")
	}
	token, err := rt.PATLogin(baseURL)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", fmt.Errorf("PAT login returned an empty token")
	}
	return strings.TrimSpace(token), nil
}
