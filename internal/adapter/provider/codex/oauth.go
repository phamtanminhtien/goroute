package codex

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/config"
)

const (
	authorizeURL                    = "https://auth.openai.com/oauth/authorize"
	tokenURL                        = "https://auth.openai.com/oauth/token"
	clientID                        = "app_EMoamEEZ73f0CkXaXp7hrann"
	redirectURI                     = "http://localhost:1455/auth/callback"
	scope                           = "openid profile email offline_access"
	codeChallengeMethod             = "S256"
	idTokenAddOrganizations         = "true"
	codexCLISimplifiedFlow          = "true"
	codexCLIOriginator              = "codex_cli_rs"
	defaultCodeVerifierEntropyBytes = 32
	defaultStateEntropyBytes        = 32
	defaultTokenExchangeTimeout     = 30 * time.Second
	proactiveRefreshBuffer          = 5 * 24 * time.Hour
)

var oauthHTTPClient = http.DefaultClient
var timeNow = time.Now

func GetAccessToken(connection config.ConnectionConfig) (string, error) {
	return checkAndRefreshToken(connection)
}

func checkAndRefreshToken(connection config.ConnectionConfig) (string, error) {
	nextConnection, err := refreshConnectionToken(connection, false)
	if err != nil {
		return "", err
	}

	accessToken := strings.TrimSpace(nextConnection.AccessToken)
	if accessToken == "" {
		return "", errors.New("missing access_token")
	}

	return accessToken, nil
}

func GenerateOAuthURL(connection config.ConnectionConfig) (string, error) {
	session, err := StartOAuth(connection)
	if err != nil {
		return "", err
	}

	return session.AuthorizationURL, nil
}

func StartOAuth(config.ConnectionConfig) (*PendingCodexOAuth, error) {
	codeVerifier, err := randomBase64URL(defaultCodeVerifierEntropyBytes)
	if err != nil {
		return nil, err
	}

	state, err := randomBase64URL(defaultStateEntropyBytes)
	if err != nil {
		return nil, err
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	return &PendingCodexOAuth{
		CodeVerifier:     codeVerifier,
		State:            state,
		RedirectURI:      redirectURI,
		AuthorizationURL: buildCodexAuthURL(redirectURI, state, codeChallenge),
	}, nil
}

func randomBase64URL(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generateCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func strictQueryEscape(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func buildCodexAuthURL(redirectURIValue, state, codeChallenge string) string {
	params := []struct {
		Key   string
		Value string
	}{
		{Key: "response_type", Value: "code"},
		{Key: "client_id", Value: clientID},
		{Key: "redirect_uri", Value: redirectURIValue},
		{Key: "scope", Value: scope},
		{Key: "code_challenge", Value: codeChallenge},
		{Key: "code_challenge_method", Value: codeChallengeMethod},
		{Key: "id_token_add_organizations", Value: idTokenAddOrganizations},
		{Key: "codex_cli_simplified_flow", Value: codexCLISimplifiedFlow},
		{Key: "originator", Value: codexCLIOriginator},
		{Key: "state", Value: state},
	}

	parts := make([]string, 0, len(params))
	for _, param := range params {
		parts = append(parts, strictQueryEscape(param.Key)+"="+strictQueryEscape(param.Value))
	}

	return authorizeURL + "?" + strings.Join(parts, "&")
}

type PendingCodexOAuth struct {
	CodeVerifier     string
	State            string
	RedirectURI      string
	AuthorizationURL string
}

type CompletedCodexOAuth struct {
	AccessToken  string
	RefreshToken string
	TokenType    string
	ExpiresIn    int
	Email        string
}

type OAuthCallbackData struct {
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

type codexTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

func CompleteCodexOAuthFromCallbackURL(pending PendingCodexOAuth, callbackURL string) (*CompletedCodexOAuth, error) {
	callbackData, err := parseOAuthCallbackURL(callbackURL)
	if err != nil {
		return nil, err
	}

	redirectURIValue := strings.TrimSpace(pending.RedirectURI)
	if redirectURIValue == "" {
		redirectURIValue = redirectURI
	}

	if err := validateCallbackEndpoint(callbackURL, redirectURIValue); err != nil {
		return nil, err
	}
	if err := validateOAuthState(pending.State, callbackData.State); err != nil {
		return nil, err
	}
	if strings.TrimSpace(pending.CodeVerifier) == "" {
		return nil, errors.New("missing code verifier")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTokenExchangeTimeout)
	defer cancel()

	tokenResponse, err := exchangeCodexCode(ctx, callbackData.Code, pending.CodeVerifier, redirectURIValue)
	if err != nil {
		return nil, err
	}

	email, err := emailFromIDToken(tokenResponse.IDToken)
	if err != nil {
		return nil, err
	}

	return &CompletedCodexOAuth{
		AccessToken:  tokenResponse.AccessToken,
		RefreshToken: tokenResponse.RefreshToken,
		TokenType:    tokenResponse.TokenType,
		ExpiresIn:    tokenResponse.ExpiresIn,
		Email:        email,
	}, nil
}

func parseOAuthCallbackURL(raw string) (*OAuthCallbackData, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid callback url: %w", err)
	}

	q := u.Query()
	data := &OAuthCallbackData{
		Code:             strings.TrimSpace(q.Get("code")),
		State:            strings.TrimSpace(q.Get("state")),
		Error:            strings.TrimSpace(q.Get("error")),
		ErrorDescription: strings.TrimSpace(q.Get("error_description")),
	}

	if data.Error != "" {
		if data.Error == "access_denied" {
			if data.ErrorDescription != "" {
				return nil, fmt.Errorf("oauth was cancelled by the user: %s", data.ErrorDescription)
			}
			return nil, errors.New("oauth was cancelled by the user")
		}
		if data.ErrorDescription != "" {
			return nil, fmt.Errorf("oauth error: %s (%s)", data.Error, data.ErrorDescription)
		}
		return nil, fmt.Errorf("oauth error: %s", data.Error)
	}
	if data.Code == "" {
		return nil, errors.New("missing authorization code")
	}
	if data.State == "" {
		return nil, errors.New("missing state")
	}

	return data, nil
}

func validateOAuthState(expectedState, returnedState string) error {
	expectedState = strings.TrimSpace(expectedState)
	if expectedState == "" {
		return errors.New("missing expected state")
	}
	if strings.TrimSpace(returnedState) != expectedState {
		return errors.New("state mismatch")
	}
	return nil
}

func refreshConnectionToken(connection config.ConnectionConfig, force bool) (config.ConnectionConfig, error) {
	connection.AccessToken = strings.TrimSpace(connection.AccessToken)
	connection.RefreshToken = strings.TrimSpace(connection.RefreshToken)
	connection.TokenType = strings.TrimSpace(connection.TokenType)

	if !force && !shouldProactivelyRefresh(connection) {
		return connection, nil
	}
	if connection.RefreshToken == "" {
		if connection.AccessToken != "" {
			return connection, nil
		}
		return connection, errors.New("missing access_token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTokenExchangeTimeout)
	defer cancel()

	tokenResponse, err := refreshCodexToken(ctx, connection.RefreshToken)
	if err != nil {
		return connection, err
	}

	connection.AccessToken = strings.TrimSpace(tokenResponse.AccessToken)
	if refreshToken := strings.TrimSpace(tokenResponse.RefreshToken); refreshToken != "" {
		connection.RefreshToken = refreshToken
	}
	if tokenType := strings.TrimSpace(tokenResponse.TokenType); tokenType != "" {
		connection.TokenType = tokenType
	}
	connection.ExpiresIn = tokenResponse.ExpiresIn
	connection.AccessTokenExpiresAt = calculateAccessTokenExpiresAt(tokenResponse.ExpiresIn)

	return connection, nil
}

func shouldProactivelyRefresh(connection config.ConnectionConfig) bool {
	if strings.TrimSpace(connection.RefreshToken) == "" {
		return false
	}
	if connection.AccessTokenExpiresAt == 0 {
		return false
	}

	return time.Unix(connection.AccessTokenExpiresAt, 0).Before(timeNow().Add(proactiveRefreshBuffer))
}

func calculateAccessTokenExpiresAt(expiresIn int) int64 {
	if expiresIn <= 0 {
		return 0
	}

	return timeNow().Add(time.Duration(expiresIn) * time.Second).Unix()
}

func validateCallbackEndpoint(raw, expectedRedirectURI string) error {
	callbackURL, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid callback url: %w", err)
	}
	redirectURL, err := url.Parse(strings.TrimSpace(expectedRedirectURI))
	if err != nil {
		return fmt.Errorf("invalid redirect uri: %w", err)
	}

	if !strings.EqualFold(callbackURL.Scheme, redirectURL.Scheme) {
		return fmt.Errorf("unexpected callback scheme: %s", callbackURL.Scheme)
	}
	if !strings.EqualFold(callbackURL.Host, redirectURL.Host) {
		return fmt.Errorf("unexpected callback host: %s", callbackURL.Host)
	}
	if callbackURL.Path != redirectURL.Path {
		return fmt.Errorf("unexpected callback path: %s", callbackURL.Path)
	}

	return nil
}

func exchangeCodexCode(ctx context.Context, code, codeVerifier, redirectURIValue string) (*codexTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", strings.TrimSpace(code))
	form.Set("redirect_uri", strings.TrimSpace(redirectURIValue))
	form.Set("code_verifier", strings.TrimSpace(codeVerifier))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		if message := strings.TrimSpace(string(body)); message != "" {
			return nil, fmt.Errorf("token exchange failed: %s: %s", resp.Status, message)
		}
		return nil, fmt.Errorf("token exchange failed: %s", resp.Status)
	}

	var out codexTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, errors.New("token exchange failed: missing access token")
	}

	return &out, nil
}

func refreshCodexToken(ctx context.Context, refreshToken string) (*codexTokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(refreshToken))
	form.Set("client_id", clientID)
	form.Set("scope", scope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		if message := strings.TrimSpace(string(body)); message != "" {
			return nil, fmt.Errorf("token refresh failed: %s: %s", resp.Status, message)
		}
		return nil, fmt.Errorf("token refresh failed: %s", resp.Status)
	}

	var out codexTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode refresh response: %w", err)
	}
	if strings.TrimSpace(out.AccessToken) == "" {
		return nil, errors.New("token refresh failed: missing access token")
	}

	return &out, nil
}

func emailFromIDToken(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("missing id token")
	}

	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid id token")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("decode id token payload: %w", err)
	}

	var claims struct {
		Email      string `json:"email"`
		OpenAIAuth struct {
			ChatGPTAccountID string `json:"chatgpt_account_id"`
			ChatGPTPlanType  string `json:"chatgpt_plan_type"`
		} `json:"https://api.openai.com/auth"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("decode id token claims: %w", err)
	}

	email := strings.TrimSpace(claims.Email)
	if email == "" {
		return "", errors.New("missing email in id token")
	}

	return email, nil
}
