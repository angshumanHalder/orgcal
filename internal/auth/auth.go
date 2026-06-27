package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	googlecalendar "google.golang.org/api/calendar/v3"
)

var tokenPath = filepath.Join(os.Getenv("HOME"), ".local", "share", "orgcal", "token.json")

func oauthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("ORGCAL_CLIENT_ID"),
		ClientSecret: os.Getenv("ORGCAL_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8765",
		Scopes:       []string{googlecalendar.CalendarScope},
		Endpoint:     google.Endpoint,
	}
}

func Authenticate() error {
	cfg := oauthConfig()
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return fmt.Errorf("ORGCAL_CLIENT_ID and ORGCAL_CLIENT_SECRET must be set")
	}

	url := cfg.AuthCodeURL("state", oauth2.AccessTypeOffline, oauth2.ApprovalForce)
	fmt.Printf("Opening browser...\nIf it doesn't open, visit:\n%s\n", url)
	_ = openBrowser(url)

	code := make(chan string, 1)
	srv := &http.Server{Addr: ":8765"}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		c := r.URL.Query().Get("code")
		if c != "" {
			fmt.Fprintln(w, "Authentication successful! You can close this tab.")
			code <- c
		}
	})

	go func() { _ = srv.ListenAndServe() }()

	select {
	case c := <-code:
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)

		tok, err := cfg.Exchange(ctx, c)
		if err != nil {
			return fmt.Errorf("token exchange failed: %w", err)
		}
		return saveToken(tok)
	case <-time.After(2 * time.Minute):
		return fmt.Errorf("authentication timed out")
	}
}

func GetToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("no token found, run: orgcal auth")
	}
	var tok oauth2.Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("invalid token file")
	}
	return &tok, nil
}

func GetHTTPClient(ctx context.Context) (*http.Client, error) {
	tok, err := GetToken()
	if err != nil {
		return nil, err
	}
	return oauthConfig().Client(ctx, tok), nil
}

func saveToken(tok *oauth2.Token) error {
	if err := os.MkdirAll(filepath.Dir(tokenPath), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(tok)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenPath, data, 0600)
}

func openBrowser(url string) error {
	// try macOS, Linux, Windows
	for _, cmd := range []string{
		"open " + url,
		"xdg-open " + url,
		"start " + url,
	} {
		if err := runShell(cmd); err == nil {
			return nil
		}
	}
	return fmt.Errorf("could not open browser")
}

func runShell(cmd string) error {
	return runCmd("sh", "-c", cmd)
}
