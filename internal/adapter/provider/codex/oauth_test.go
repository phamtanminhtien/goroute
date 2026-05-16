package codex

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/phamtanminhtien/goroute/internal/domain/connection"
)

func TestGenerateOAuthURLIncludesCodexOAuthParameters(t *testing.T) {
	authURL, err := GenerateOAuthURL(connection.Record{ProviderID: "cx"})
	if err != nil {
		t.Fatalf("GenerateOAuthURL returned error: %v", err)
	}

	if !strings.Contains(authURL, "scope=openid%20profile%20email%20offline_access") {
		t.Fatalf("expected scope to use %%20 encoding, got %q", authURL)
	}

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("parse auth url: %v", err)
	}

	query := parsedURL.Query()
	if got := parsedURL.Scheme + "://" + parsedURL.Host + parsedURL.Path; got != authorizeURL {
		t.Fatalf("unexpected authorize url: %q", got)
	}
	if got := query.Get("response_type"); got != "code" {
		t.Fatalf("unexpected response_type: %q", got)
	}
	if got := query.Get("client_id"); got != clientID {
		t.Fatalf("unexpected client_id: %q", got)
	}
	if got := query.Get("redirect_uri"); got != redirectURI {
		t.Fatalf("unexpected redirect_uri: %q", got)
	}
	if got := query.Get("code_challenge_method"); got != codeChallengeMethod {
		t.Fatalf("unexpected code_challenge_method: %q", got)
	}
	if got := query.Get("id_token_add_organizations"); got != idTokenAddOrganizations {
		t.Fatalf("unexpected id_token_add_organizations: %q", got)
	}
	if got := query.Get("codex_cli_simplified_flow"); got != codexCLISimplifiedFlow {
		t.Fatalf("unexpected codex_cli_simplified_flow: %q", got)
	}
	if got := query.Get("originator"); got != codexCLIOriginator {
		t.Fatalf("unexpected originator: %q", got)
	}
	if got := query.Get("state"); got == "" {
		t.Fatal("expected non-empty state")
	}
	if got := query.Get("code_challenge"); got == "" {
		t.Fatal("expected non-empty code_challenge")
	}
}

func TestCompleteCodexOAuthFromCallbackURL(t *testing.T) {
	t.Cleanup(func() {
		oauthHTTPClient = http.DefaultClient
	})

	var capturedBody string
	oauthHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost {
				t.Fatalf("unexpected method: %s", req.Method)
			}
			if got := req.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
				t.Fatalf("unexpected content type: %q", got)
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			capturedBody = string(body)

			return jsonResponse(http.StatusOK, `{
				"access_token":"access-123",
				"refresh_token":"refresh-456",
				"token_type":"Bearer",
				"expires_in":3600,
				"id_token":"`+testJWT(`{"email":"shaylee11@jabmag.com","https://api.openai.com/auth":{"chatgpt_account_id":"acct_123","chatgpt_plan_type":"plus"}}`)+`"
			}`), nil
		}),
	}

	result, err := CompleteCodexOAuthFromCallbackURL(
		PendingCodexOAuth{
			CodeVerifier: "verifier-123",
			State:        "state-456",
			RedirectURI:  redirectURI,
		},
		"http://localhost:1455/auth/callback?code=code-789&state=state-456",
	)
	if err != nil {
		t.Fatalf("CompleteCodexOAuthFromCallbackURL returned error: %v", err)
	}

	if result.AccessToken != "access-123" {
		t.Fatalf("unexpected access token: %q", result.AccessToken)
	}
	if result.RefreshToken != "refresh-456" {
		t.Fatalf("unexpected refresh token: %q", result.RefreshToken)
	}
	if result.TokenType != "Bearer" {
		t.Fatalf("unexpected token type: %q", result.TokenType)
	}
	if result.ExpiresIn != 3600 {
		t.Fatalf("unexpected expires_in: %d", result.ExpiresIn)
	}
	if result.Email != "shaylee11@jabmag.com" {
		t.Fatalf("unexpected email: %q", result.Email)
	}

	values, err := url.ParseQuery(capturedBody)
	if err != nil {
		t.Fatalf("parse form body: %v", err)
	}
	if got := values.Get("grant_type"); got != "authorization_code" {
		t.Fatalf("unexpected grant_type: %q", got)
	}
	if got := values.Get("client_id"); got != clientID {
		t.Fatalf("unexpected client_id: %q", got)
	}
	if got := values.Get("code"); got != "code-789" {
		t.Fatalf("unexpected code: %q", got)
	}
	if got := values.Get("redirect_uri"); got != redirectURI {
		t.Fatalf("unexpected redirect_uri: %q", got)
	}
	if got := values.Get("code_verifier"); got != "verifier-123" {
		t.Fatalf("unexpected code_verifier: %q", got)
	}
}

func TestCheckAndRefreshTokenRefreshesExpiringToken(t *testing.T) {
	t.Cleanup(func() {
		oauthHTTPClient = http.DefaultClient
		timeNow = time.Now
	})

	timeNow = func() time.Time {
		return time.Unix(1710000000, 0)
	}

	oauthHTTPClient = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read refresh body: %v", err)
			}
			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("parse refresh body: %v", err)
			}
			if got := values.Get("grant_type"); got != "refresh_token" {
				t.Fatalf("unexpected grant_type: %q", got)
			}
			if got := values.Get("refresh_token"); got != "refresh-123" {
				t.Fatalf("unexpected refresh token: %q", got)
			}

			return jsonResponse(http.StatusOK, `{"access_token":"access-456","refresh_token":"refresh-789","expires_in":3600,"token_type":"Bearer"}`), nil
		}),
	}

	token, err := checkAndRefreshToken(connection.Record{
		ProviderID:           "cx",
		AccessToken:          "access-123",
		RefreshToken:         "refresh-123",
		AccessTokenExpiresAt: timeNow().Add(2 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("checkAndRefreshToken returned error: %v", err)
	}
	if token != "access-456" {
		t.Fatalf("unexpected refreshed token: %q", token)
	}
}

func TestCompleteCodexOAuthFromCallbackURLStateMismatch(t *testing.T) {
	_, err := CompleteCodexOAuthFromCallbackURL(
		PendingCodexOAuth{
			CodeVerifier: "verifier-123",
			State:        "expected-state",
			RedirectURI:  redirectURI,
		},
		"http://localhost:1455/auth/callback?code=code-789&state=other-state",
	)
	if err == nil || !strings.Contains(err.Error(), "state mismatch") {
		t.Fatalf("expected state mismatch error, got %v", err)
	}
}

func TestCompleteCodexOAuthFromCallbackURLUnexpectedCallbackEndpoint(t *testing.T) {
	_, err := CompleteCodexOAuthFromCallbackURL(
		PendingCodexOAuth{
			CodeVerifier: "verifier-123",
			State:        "state-456",
			RedirectURI:  redirectURI,
		},
		"http://localhost:9999/auth/callback?code=code-789&state=state-456",
	)
	if err == nil || !strings.Contains(err.Error(), "unexpected callback host") {
		t.Fatalf("expected unexpected callback host error, got %v", err)
	}
}

func TestCompleteCodexOAuthFromCallbackURLOAuthCancelled(t *testing.T) {
	_, err := CompleteCodexOAuthFromCallbackURL(
		PendingCodexOAuth{
			CodeVerifier: "verifier-123",
			State:        "state-456",
			RedirectURI:  redirectURI,
		},
		"http://localhost:1455/auth/callback?error=access_denied&error_description=user%20cancelled",
	)
	if err == nil || !strings.Contains(err.Error(), "oauth was cancelled by the user") {
		t.Fatalf("expected oauth cancelled error, got %v", err)
	}
}

func jsonResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Status:     http.StatusText(statusCode),
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func testJWT(payload string) string {
	return "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." + base64URL(payload) + ".signature"
}

func base64URL(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}
