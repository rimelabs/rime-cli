package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestGenerateState(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("generateState failed: %v", err)
	}

	// 16 bytes → 32 hex chars
	if len(state) != 32 {
		t.Errorf("expected state length 32, got %d", len(state))
	}

	for _, c := range state {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("state contains non-hex character: %c", c)
		}
	}
}

func TestGenerateStateUniqueness(t *testing.T) {
	states := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		s, err := generateState()
		if err != nil {
			t.Fatalf("generateState failed on iteration %d: %v", i, err)
		}
		if states[s] {
			t.Fatalf("duplicate state generated: %s", s)
		}
		states[s] = true
	}
}

// makeCallbackHandler replicates the /callback handler from login() so we can
// unit-test its logic directly via httptest without running a full server.
func makeCallbackHandler(state string, tokenCh chan string, errCh chan error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

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
	}
}

func TestCallbackHandler_Success(t *testing.T) {
	const state = "abc123deadbeef00"
	const token = "sk-test-token"

	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	handler := makeCallbackHandler(state, tokenCh, errCh)

	req := httptest.NewRequest("GET", fmt.Sprintf("/callback?state=%s&token=%s", state, token), nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Authentication successful") {
		t.Error("expected success HTML in response body")
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}

	select {
	case got := <-tokenCh:
		if got != token {
			t.Errorf("expected token %q, got %q", token, got)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for token")
	}
}

func TestCallbackHandler_StateMismatch(t *testing.T) {
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	handler := makeCallbackHandler("correct-state", tokenCh, errCh)

	req := httptest.NewRequest("GET", "/callback?state=wrong-state&token=sk-test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Authentication failed") {
		t.Error("expected error HTML in response body")
	}

	select {
	case err := <-errCh:
		if !strings.Contains(err.Error(), "state mismatch") {
			t.Errorf("expected state mismatch error, got: %v", err)
		}
	case <-tokenCh:
		t.Fatal("expected no token on state mismatch")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

func TestCallbackHandler_MissingToken(t *testing.T) {
	const state = "abc123deadbeef00"
	tokenCh := make(chan string, 1)
	errCh := make(chan error, 1)
	handler := makeCallbackHandler(state, tokenCh, errCh)

	req := httptest.NewRequest("GET", fmt.Sprintf("/callback?state=%s", state), nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}

	select {
	case err := <-errCh:
		if !strings.Contains(err.Error(), "no token") {
			t.Errorf("expected 'no token' error, got: %v", err)
		}
	case <-tokenCh:
		t.Fatal("expected no token when token param is missing")
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for error")
	}
}

// TestLogin_EndToEnd tests the full login flow without opening a real browser.
// The mock opener inspects the auth URL, extracts the redirect_uri and state,
// and immediately hits the CLI's local callback server as the web app would.
func TestLogin_EndToEnd(t *testing.T) {
	const fakeToken = "sk-integration-test-token"

	mockOpener := func(authURL string) error {
		u, err := url.Parse(authURL)
		if err != nil {
			return fmt.Errorf("mock opener: bad URL: %w", err)
		}

		redirectURI := u.Query().Get("redirect_uri")
		state := u.Query().Get("state")
		if redirectURI == "" || state == "" {
			return fmt.Errorf("mock opener: missing redirect_uri or state")
		}

		// Simulate the browser hitting the CLI's callback after auth.
		go func() {
			callbackURL := fmt.Sprintf("%s?token=%s&state=%s", redirectURI, fakeToken, url.QueryEscape(state))
			resp, err := http.Get(callbackURL) //nolint:noctx
			if err == nil {
				resp.Body.Close()
			}
		}()

		return nil
	}

	token, err := login("https://fake.rime.ai", mockOpener)
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}
	if token != fakeToken {
		t.Errorf("expected token %q, got %q", fakeToken, token)
	}
}

// TestLogin_StateMismatch_EndToEnd verifies that a CSRF attempt (wrong state
// in the callback) causes login() to return an error.
func TestLogin_StateMismatch_EndToEnd(t *testing.T) {
	mockOpener := func(authURL string) error {
		u, _ := url.Parse(authURL)
		redirectURI := u.Query().Get("redirect_uri")

		go func() {
			// Send a callback with a tampered state value.
			callbackURL := fmt.Sprintf("%s?token=evil-token&state=tampered", redirectURI)
			resp, err := http.Get(callbackURL) //nolint:noctx
			if err == nil {
				resp.Body.Close()
			}
		}()

		return nil
	}

	_, err := login("https://fake.rime.ai", mockOpener)
	if err == nil {
		t.Fatal("expected error on state mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("expected state mismatch error, got: %v", err)
	}
}
