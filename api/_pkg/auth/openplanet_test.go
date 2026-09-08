package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"rmpc-server/api/_pkg/config"
)

// withUpstream points ValidateOpenplanetToken at a stub Openplanet and restores
// the previous configuration afterwards.
func withUpstream(t *testing.T, h http.HandlerFunc) {
	t.Helper()

	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)

	prevURL := config.Env.OpenplanetAuthURL
	prevSecret := config.Env.OpenplanetPluginSecret
	config.Env.OpenplanetAuthURL = srv.URL
	config.Env.OpenplanetPluginSecret = "test-secret"
	t.Cleanup(func() {
		config.Env.OpenplanetAuthURL = prevURL
		config.Env.OpenplanetPluginSecret = prevSecret
	})
}

func TestValidateClassifiesUpstreamFailures(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		wantErr error
	}{
		// Openplanet is broken: nothing is known about the token, so the
		// player must not be told their token is bad.
		{"500 internal error", http.StatusInternalServerError, `{"error":"boom"}`, ErrUpstream},
		{"502 bad gateway", http.StatusBadGateway, "<html>bad gateway</html>", ErrUpstream},
		{"503 unavailable", http.StatusServiceUnavailable, "maintenance", ErrUpstream},
		{"200 but not JSON", http.StatusOK, "<html>captive portal</html>", ErrUpstream},
		{"unexpected 302", http.StatusFound, "", ErrUpstream},

		// Throttling is transient. Telling players to reauthenticate here
		// would send them straight back into the same rate limit.
		{"429 too many requests", http.StatusTooManyRequests, `{"error":"slow down"}`, ErrUpstream},

		// Openplanet answered and refused.
		{"400 bad request", http.StatusBadRequest, `{"error":"bad secret"}`, ErrInvalidToken},
		{"401 unauthorized", http.StatusUnauthorized, `{"error":"invalid token"}`, ErrInvalidToken},
		{"403 forbidden", http.StatusForbidden, `{"error":"nope"}`, ErrInvalidToken},
		{"200 with error field", http.StatusOK, `{"error":"token expired"}`, ErrInvalidToken},
		{"200 with no account_id", http.StatusOK, `{}`, ErrInvalidToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			user, err := ValidateOpenplanetToken("some-token")
			if user != nil {
				t.Fatalf("expected no user, got %+v", user)
			}
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got error %v, want it to match %v", err, tt.wantErr)
			}
			// Only a real rejection may send the player back to log in again.
			if tt.wantErr != ErrInvalidToken && errors.Is(err, ErrInvalidToken) {
				t.Fatalf("error %v must not be reported as an invalid token", err)
			}
		})
	}
}

func TestValidateSuccess(t *testing.T) {
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_, _ = w.Write([]byte(`{"account_id":"abc-123","display_name":"AlicE"}`))
	})

	user, err := ValidateOpenplanetToken("good-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.AccountID != "abc-123" || user.DisplayName != "AlicE" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

// An unreachable Openplanet is an outage, not a bad token.
func TestValidateUnreachableUpstreamIsUpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	url := srv.URL
	srv.Close() // nothing is listening now

	prevURL := config.Env.OpenplanetAuthURL
	prevSecret := config.Env.OpenplanetPluginSecret
	config.Env.OpenplanetAuthURL = url
	config.Env.OpenplanetPluginSecret = "test-secret"
	defer func() {
		config.Env.OpenplanetAuthURL = prevURL
		config.Env.OpenplanetPluginSecret = prevSecret
	}()

	_, err := ValidateOpenplanetToken("some-token")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("got %v, want ErrUpstream", err)
	}
	if errors.Is(err, ErrInvalidToken) {
		t.Fatal("an unreachable upstream must never be reported as an invalid token")
	}
}

// A hanging Openplanet must time out as an outage, not as a rejection.
func TestValidateTimeoutIsUpstreamError(t *testing.T) {
	release := make(chan struct{})
	withUpstream(t, func(w http.ResponseWriter, r *http.Request) {
		<-release // hold the request open past the client timeout
	})

	prev := openplanetClient
	openplanetClient = &http.Client{Timeout: 150 * time.Millisecond}
	defer func() {
		openplanetClient = prev
		close(release)
	}()

	_, err := ValidateOpenplanetToken("some-token")
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("got %v, want ErrUpstream", err)
	}
}

// A missing plugin secret is our deployment being wrong, not the player's token.
func TestValidateMissingSecretIsMisconfiguration(t *testing.T) {
	prev := config.Env.OpenplanetPluginSecret
	config.Env.OpenplanetPluginSecret = ""
	defer func() { config.Env.OpenplanetPluginSecret = prev }()

	_, err := ValidateOpenplanetToken("some-token")
	if !errors.Is(err, ErrMisconfigured) {
		t.Fatalf("got %v, want ErrMisconfigured", err)
	}
	if errors.Is(err, ErrInvalidToken) {
		t.Fatal("a misconfigured server must not be reported as an invalid token")
	}
}
