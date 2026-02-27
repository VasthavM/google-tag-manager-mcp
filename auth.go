package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GTM API scopes.
var gtmScopes = []string{
	"https://www.googleapis.com/auth/tagmanager.delete.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	"https://www.googleapis.com/auth/tagmanager.publish",
}

// configDir returns the config directory, creating it if needed.
// Respects $GTM_MCP_CONFIG_DIR override.
func configDir() (string, error) {
	dir := os.Getenv("GTM_MCP_CONFIG_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".config", "google-tag-manager-mcp")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config dir %s: %w", dir, err)
	}
	return dir, nil
}

// credentialsPath returns the path to credentials.json.
func credentialsPath() (string, error) {
	if p := os.Getenv("GTM_MCP_CREDENTIALS"); p != "" {
		return p, nil
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "credentials.json"), nil
}

// tokenPath returns the path to token.json.
func tokenPath() (string, error) {
	if p := os.Getenv("GTM_MCP_TOKEN"); p != "" {
		return p, nil
	}
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "token.json"), nil
}

// installedCredentials is the shape of a Google "Desktop" credentials.json.
type installedCredentials struct {
	Installed struct {
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
		RedirectURIs []string `json:"redirect_uris"`
	} `json:"installed"`
}

// loadOAuthConfig reads a Google Desktop credentials.json and returns an oauth2.Config.
func loadOAuthConfig(path string) (*oauth2.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read credentials file %s: %w\n"+
			"Download it from Google Cloud Console → APIs & Services → Credentials → OAuth 2.0 Client (Desktop app)", path, err)
	}

	var creds installedCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("invalid credentials.json format: %w", err)
	}
	if creds.Installed.ClientID == "" || creds.Installed.ClientSecret == "" {
		return nil, fmt.Errorf("credentials.json missing client_id or client_secret (expected 'installed' key for Desktop app)")
	}

	return &oauth2.Config{
		ClientID:     creds.Installed.ClientID,
		ClientSecret: creds.Installed.ClientSecret,
		Scopes:       gtmScopes,
		Endpoint:     google.Endpoint,
	}, nil
}

// loadToken reads a saved OAuth2 token from disk.
func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("invalid token.json: %w", err)
	}
	return &tok, nil
}

// saveToken writes an OAuth2 token to disk with restricted permissions.
func saveToken(path string, tok *oauth2.Token) error {
	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal token: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// browserAuthFlow runs the one-time interactive OAuth2 consent flow.
// It opens a local callback server, prints the auth URL, and waits for the code.
func browserAuthFlow(ctx context.Context, cfg *oauth2.Config) (*oauth2.Token, error) {
	// Listen on a random available port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("cannot start local callback server: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	redirectURL := fmt.Sprintf("http://127.0.0.1:%d/callback", port)
	cfg.RedirectURL = redirectURL

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			fmt.Fprintf(w, "<html><body><h2>Authentication failed: %s</h2><p>You may close this tab.</p></body></html>", errParam)
			errCh <- fmt.Errorf("oauth error: %s", errParam)
			return
		}
		fmt.Fprintf(w, "<html><body><h2>Authentication successful!</h2><p>You may close this tab and return to the terminal.</p></body></html>")
		codeCh <- code
	})
	srv.Handler = mux

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer srv.Close()

	authURL := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Fprintf(os.Stderr, "\nOpen this URL in your browser to authenticate with Google:\n%s\n\n", authURL)
	// Attempt to auto-open in browser (WSL/Linux)
	for _, cmd := range []string{"xdg-open", "wslview", "open"} {
		if err := exec.CommandContext(ctx, cmd, authURL).Start(); err == nil {
			break
		}
	}

	select {
	case code := <-codeCh:
		tok, err := cfg.Exchange(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("token exchange failed: %w", err)
		}
		return tok, nil
	case err := <-errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// persistingTokenSource wraps an oauth2.TokenSource and saves refreshed tokens to disk.
type persistingTokenSource struct {
	mu       sync.Mutex
	inner    oauth2.TokenSource
	savePath string
	current  *oauth2.Token
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tok, err := p.inner.Token()
	if err != nil {
		return nil, err
	}
	// Save if the token changed (i.e. was refreshed)
	if p.current == nil || tok.AccessToken != p.current.AccessToken {
		p.current = tok
		if saveErr := saveToken(p.savePath, tok); saveErr != nil {
			slog.Warn("failed to persist refreshed token", "error", saveErr)
		}
	}
	return tok, nil
}

// getTokenSource is the main entry point for OAuth2.
// It loads credentials, then either reuses a saved token or runs the browser flow.
func getTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	credPath, err := credentialsPath()
	if err != nil {
		return nil, err
	}
	tokPath, err := tokenPath()
	if err != nil {
		return nil, err
	}

	cfg, err := loadOAuthConfig(credPath)
	if err != nil {
		return nil, err
	}

	var tok *oauth2.Token
	saved, loadErr := loadToken(tokPath)
	if loadErr == nil && saved.RefreshToken != "" {
		slog.Info("using saved OAuth2 token", "token_path", tokPath)
		tok = saved
	} else {
		slog.Info("no saved token found, starting browser auth flow")
		tok, err = browserAuthFlow(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		if err := saveToken(tokPath, tok); err != nil {
			slog.Warn("failed to save token", "error", err)
		} else {
			slog.Info("token saved", "path", tokPath)
		}
	}

	inner := cfg.TokenSource(ctx, tok)
	return &persistingTokenSource{
		inner:    inner,
		savePath: tokPath,
		current:  tok,
	}, nil
}
