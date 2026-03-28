package codex

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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	authorizationEndpoint = "https://auth.openai.com/oauth/authorize"
	tokenEndpoint         = "https://auth.openai.com/oauth/token"
	clientID              = "app_EMoamEEZ73f0CkXaXp7hrann"
	redirectURI           = "http://localhost:1455/auth/callback"
	callbackAddr          = ":1455"
)

// tokenData holds OAuth token information.
type tokenData struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// oauthClient manages OAuth PKCE authentication with OpenAI.
type oauthClient struct {
	httpClient *http.Client
	tokenPath  string
}

// newOAuthClient creates a new oauthClient.
func newOAuthClient() (*oauthClient, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine home directory: %w", err)
	}
	tokenPath := filepath.Join(home, ".config", "jbuntai", "codex_token.json")
	return &oauthClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		tokenPath:  tokenPath,
	}, nil
}

// GetValidToken returns a valid access token, refreshing or re-authorizing as needed.
func (o *oauthClient) GetValidToken(ctx context.Context) (string, error) {
	token, err := o.loadToken()
	if err == nil && time.Now().Before(token.ExpiresAt) {
		return token.AccessToken, nil
	}

	// Try refresh if we have a refresh token
	if err == nil && token.RefreshToken != "" {
		refreshed, refreshErr := o.refresh(ctx, token.RefreshToken)
		if refreshErr == nil {
			if saveErr := o.saveToken(refreshed); saveErr != nil {
				return "", fmt.Errorf("failed to save refreshed token: %w", saveErr)
			}
			return refreshed.AccessToken, nil
		}
	}

	// Fall back to full authorization
	newToken, err := o.authorize(ctx)
	if err != nil {
		return "", fmt.Errorf("authorization failed: %w", err)
	}
	if err := o.saveToken(newToken); err != nil {
		return "", fmt.Errorf("failed to save token: %w", err)
	}
	return newToken.AccessToken, nil
}

// authorize performs the OAuth PKCE flow.
func (o *oauthClient) authorize(ctx context.Context) (*tokenData, error) {
	// Generate PKCE verifier and challenge
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate code verifier: %w", err)
	}
	codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	challengeHash := sha256.Sum256([]byte(codeVerifier))
	codeChallenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])

	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Start callback server
	listener, err := net.Listen("tcp", callbackAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer listener.Close()

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			errCh <- fmt.Errorf("state mismatch")
			http.Error(w, "State mismatch", http.StatusBadRequest)
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("authorization error: %s", errMsg)
			http.Error(w, errMsg, http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("no authorization code received")
			http.Error(w, "No code", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "<html><body><h1>Authorization successful!</h1><p>You can close this window.</p></body></html>")
		codeCh <- code
	})

	server := &http.Server{Handler: mux}
	go func() {
		if serveErr := server.Serve(listener); serveErr != http.ErrServerClosed {
			errCh <- serveErr
		}
	}()
	defer server.Shutdown(context.Background())

	// Build authorization URL
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&code_challenge=%s&code_challenge_method=S256&state=%s",
		authorizationEndpoint,
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(codeChallenge),
		url.QueryEscape(state),
	)

	// Open browser
	fmt.Fprintf(os.Stderr, "Opening browser for authentication...\n")
	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open browser. Please open this URL manually:\n%s\n", authURL)
	}

	// Wait for callback
	select {
	case code := <-codeCh:
		return o.exchangeCode(ctx, code, codeVerifier)
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// exchangeCode exchanges the authorization code for tokens.
func (o *oauthClient) exchangeCode(ctx context.Context, code, codeVerifier string) (*tokenData, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"client_id":     {clientID},
		"redirect_uri":  {redirectURI},
	}

	return o.requestToken(ctx, data)
}

// refresh exchanges a refresh token for new tokens.
func (o *oauthClient) refresh(ctx context.Context, refreshToken string) (*tokenData, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	return o.requestToken(ctx, data)
}

// requestToken makes a token request to the OAuth token endpoint.
func (o *oauthClient) requestToken(ctx context.Context, data url.Values) (*tokenData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("token request returned %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenData{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}

// loadToken reads the saved token from disk.
func (o *oauthClient) loadToken() (*tokenData, error) {
	data, err := os.ReadFile(o.tokenPath)
	if err != nil {
		return nil, err
	}
	var token tokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// saveToken writes the token to disk with restricted permissions.
func (o *oauthClient) saveToken(token *tokenData) error {
	dir := filepath.Dir(o.tokenPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(o.tokenPath, data, 0o600)
}

// openBrowser opens the given URL in the default browser.
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}
