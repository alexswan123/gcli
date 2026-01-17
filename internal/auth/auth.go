package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alexandraswan/gcli/internal/config"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/gmail/v1"
)

// Scopes required for Gmail and Calendar access
var Scopes = []string{
	gmail.GmailReadonlyScope,
	gmail.GmailComposeScope,
	gmail.GmailSendScope,
	gmail.GmailModifyScope,
	calendar.CalendarReadonlyScope,
	calendar.CalendarEventsScope,
}

// GetOAuthConfig creates an OAuth2 config for the given account
func GetOAuthConfig(account config.AccountConfig) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     account.ClientID,
		ClientSecret: account.ClientSecret,
		RedirectURL:  "http://localhost:8085/callback",
		Scopes:       Scopes,
		Endpoint:     google.Endpoint,
	}
}

// LoadToken loads the token for the specified account
func LoadToken(accountName string) (*oauth2.Token, error) {
	tokenPath, err := config.GetTokenPath(accountName)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no token found for account '%s' - run 'gcli auth add %s' first", accountName, accountName)
		}
		return nil, fmt.Errorf("failed to read token: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return &token, nil
}

// SaveToken saves the token for the specified account
func SaveToken(accountName string, token *oauth2.Token) error {
	if err := config.EnsureConfigDir(); err != nil {
		return err
	}

	tokenPath, err := config.GetTokenPath(accountName)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	if err := os.WriteFile(tokenPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write token: %w", err)
	}

	return nil
}

// TokenExists checks if a token exists for the account
func TokenExists(accountName string) bool {
	tokenPath, err := config.GetTokenPath(accountName)
	if err != nil {
		return false
	}
	_, err = os.Stat(tokenPath)
	return err == nil
}

// GetClient returns an authenticated HTTP client for the specified account
func GetClient(ctx context.Context, accountName string, account config.AccountConfig) (*http.Client, error) {
	oauthConfig := GetOAuthConfig(account)

	token, err := LoadToken(accountName)
	if err != nil {
		return nil, err
	}

	// Create a token source that will auto-refresh
	tokenSource := oauthConfig.TokenSource(ctx, token)

	// Get a potentially refreshed token
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	// If token was refreshed, save it
	if newToken.AccessToken != token.AccessToken {
		if err := SaveToken(accountName, newToken); err != nil {
			// Log but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to save refreshed token: %v\n", err)
		}
	}

	return oauthConfig.Client(ctx, newToken), nil
}

// AuthenticateAccount performs the OAuth flow for a new account
func AuthenticateAccount(accountName string, account config.AccountConfig) error {
	oauthConfig := GetOAuthConfig(account)

	// Create a channel to receive the auth code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start local server to handle callback
	server := &http.Server{Addr: ":8085"}
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, "<html><body><h1>Authentication failed</h1><p>No authorization code received.</p></body></html>")
			return
		}

		codeChan <- code
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html><body><h1>Authentication successful!</h1><p>You can close this window and return to the terminal.</p><script>setTimeout(function(){window.close();}, 2000);</script></body></html>")
	})

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- fmt.Errorf("failed to start callback server: %w", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Generate the auth URL
	authURL := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("\n🔐 Opening browser for authentication...")
	fmt.Println("If browser doesn't open, visit this URL:")
	fmt.Println(authURL)
	fmt.Println()

	// Try to open browser
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't open browser: %v\n", err)
	}

	fmt.Println("⏳ Waiting for authentication...")

	// Wait for code or error
	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		server.Close()
		return err
	case <-time.After(5 * time.Minute):
		server.Close()
		return fmt.Errorf("authentication timeout")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)

	// Exchange code for token
	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return fmt.Errorf("failed to exchange code for token: %w", err)
	}

	// Save the token
	if err := SaveToken(accountName, token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\n✅ Authentication successful!")
	return nil
}

// RemoveToken removes the token file for an account
func RemoveToken(accountName string) error {
	tokenPath, err := config.GetTokenPath(accountName)
	if err != nil {
		return err
	}
	return os.Remove(tokenPath)
}
