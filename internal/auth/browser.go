package auth

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

//go:embed assets/logo.svg
var logoSVG string

const callbackTimeout = 5 * time.Minute

const successHTMLTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Rime CLI</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #000;
      color: #fff;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      text-align: center;
    }
    .card {
      max-width: 400px;
      padding: 2rem;
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 1.75rem;
    }
    .check {
      width: 52px;
      height: 52px;
      background: #16a34a;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 1.5rem;
      line-height: 1;
    }
    h1 { font-size: 1.375rem; font-weight: 600; margin-bottom: 0.5rem; }
    p { color: #a1a1aa; line-height: 1.6; font-size: 0.9375rem; }
  </style>
</head>
<body>
  <div class="card">
    {{LOGO}}
    <div class="check">✓</div>
    <div>
      <h1>Authentication successful!</h1>
      <p>You're logged in. You can close this tab<br>and return to your terminal.</p>
    </div>
  </div>
</body>
</html>`

var successHTML = strings.ReplaceAll(successHTMLTemplate, "{{LOGO}}", logoSVG)

const errorHTML = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Rime CLI</title>
  <style>
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background: #000;
      color: #fff;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      display: flex;
      align-items: center;
      justify-content: center;
      min-height: 100vh;
      text-align: center;
    }
    .card { max-width: 400px; padding: 2rem; }
    .icon {
      width: 56px;
      height: 56px;
      background: #dc2626;
      border-radius: 50%;
      display: flex;
      align-items: center;
      justify-content: center;
      margin: 0 auto 1.5rem;
      font-size: 1.75rem;
      line-height: 1;
    }
    h1 { font-size: 1.5rem; font-weight: 600; margin-bottom: 0.75rem; }
    p { color: #a1a1aa; line-height: 1.6; }
  </style>
</head>
<body>
  <div class="card">
    <div class="icon">✗</div>
    <h1>Authentication failed</h1>
    <p>Something went wrong. Please close this tab and try <code>rime login</code> again.</p>
  </div>
</body>
</html>`

// Login performs a browser-based authentication flow and returns the API key.
// It binds a one-shot local HTTP server to a random port, opens the Rime
// dashboard in the user's browser, and waits for the callback carrying the token.
func Login(dashboardURL string) (string, error) {
	return login(dashboardURL, openBrowser)
}

// login is the testable core of Login. openBrowserFn is called with the full
// auth URL and is expected to direct the user's browser to that URL.
func login(dashboardURL string, openBrowserFn func(string) error) (string, error) {
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Bind to a random available port on localhost.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to start local server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)

	authURL := fmt.Sprintf(
		"%s/cli/auth?state=%s&redirect_uri=%s",
		dashboardURL,
		url.QueryEscape(state),
		url.QueryEscape(redirectURI),
	)

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		// Validate state to prevent CSRF.
		if q.Get("state") != state {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, errorHTML)
			select {
			case errCh <- fmt.Errorf("state mismatch: possible CSRF attack"):
			default:
			}
			return
		}

		token := q.Get("token")
		if token == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, errorHTML)
			select {
			case errCh <- fmt.Errorf("no token received from authentication server"):
			default:
			}
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, successHTML)
		select {
		case tokenCh <- token:
		default:
		}
	})

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			select {
			case errCh <- fmt.Errorf("local server error: %w", err):
			default:
			}
		}
	}()

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		server.Shutdown(ctx) //nolint:errcheck
	}()

	if err := openBrowserFn(authURL); err != nil {
		return "", fmt.Errorf("failed to open browser: %w", err)
	}

	select {
	case token := <-tokenCh:
		return token, nil
	case err := <-errCh:
		return "", err
	case <-time.After(callbackTimeout):
		return "", fmt.Errorf("authentication timed out after %v", callbackTimeout)
	}
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return cmd.Start()
}
