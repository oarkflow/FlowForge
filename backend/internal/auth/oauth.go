package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OAuthUser represents the user information returned by an OAuth provider.
type OAuthUser struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	Provider  string `json:"provider"`
}

// OAuthConfig holds the OAuth2 credentials for a provider.
type OAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// OAuthToken holds the token response from an OAuth2 exchange.
type OAuthToken struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

// OAuthProvider defines the interface for OAuth2 providers.
type OAuthProvider interface {
	// AuthURL returns the authorization URL to redirect the user to.
	AuthURL(state string) string
	// Exchange exchanges an authorization code for an access token.
	Exchange(ctx context.Context, code string) (*OAuthToken, error)
	// GetUser fetches the authenticated user's profile using the access token.
	GetUser(ctx context.Context, token *OAuthToken) (*OAuthUser, error)
}

// httpClient is the HTTP client used for OAuth requests. Can be overridden in tests.
var httpClient = &http.Client{Timeout: 10 * time.Second}

// -------------------------------------------------------------------
// GitHub OAuth Provider
// -------------------------------------------------------------------

// GitHubProvider implements OAuthProvider for GitHub.
type GitHubProvider struct {
	Config OAuthConfig
}

func NewGitHubProvider(cfg OAuthConfig) *GitHubProvider {
	return &GitHubProvider{Config: cfg}
}

func (g *GitHubProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":    {g.Config.ClientID},
		"redirect_uri": {g.Config.RedirectURL},
		"scope":        {"read:user user:email"},
		"state":        {state},
	}
	return "https://github.com/login/oauth/authorize?" + params.Encode()
}

func (g *GitHubProvider) Exchange(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":     {g.Config.ClientID},
		"client_secret": {g.Config.ClientSecret},
		"code":          {code},
		"redirect_uri":  {g.Config.RedirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://github.com/login/oauth/access_token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("github exchange: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github exchange: status %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("github exchange: decode: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("github exchange: empty access token")
	}
	return &token, nil
}

func (g *GitHubProvider) GetUser(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	// Fetch user profile
	user, err := githubAPIGet[githubUser](ctx, "https://api.github.com/user", token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("github get user: %w", err)
	}

	email := user.Email
	if email == "" {
		// If email is private, fetch from /user/emails
		emails, err := githubAPIGet[[]githubEmail](ctx, "https://api.github.com/user/emails", token.AccessToken)
		if err == nil && emails != nil {
			for _, e := range *emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
			// Fallback: first verified email
			if email == "" {
				for _, e := range *emails {
					if e.Verified {
						email = e.Email
						break
					}
				}
			}
		}
	}

	return &OAuthUser{
		ID:        fmt.Sprintf("%d", user.ID),
		Email:     email,
		Username:  user.Login,
		AvatarURL: user.AvatarURL,
		Provider:  "github",
	}, nil
}

type githubUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func githubAPIGet[T any](ctx context.Context, urlStr, accessToken string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// -------------------------------------------------------------------
// GitLab OAuth Provider
// -------------------------------------------------------------------

// GitLabProvider implements OAuthProvider for GitLab.
type GitLabProvider struct {
	Config  OAuthConfig
	BaseURL string // defaults to "https://gitlab.com" if empty
}

func NewGitLabProvider(cfg OAuthConfig, baseURL string) *GitLabProvider {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return &GitLabProvider{Config: cfg, BaseURL: strings.TrimRight(baseURL, "/")}
}

func (g *GitLabProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {g.Config.ClientID},
		"redirect_uri":  {g.Config.RedirectURL},
		"response_type": {"code"},
		"scope":         {"read_user"},
		"state":         {state},
	}
	return g.BaseURL + "/oauth/authorize?" + params.Encode()
}

func (g *GitLabProvider) Exchange(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":     {g.Config.ClientID},
		"client_secret": {g.Config.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {g.Config.RedirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.BaseURL+"/oauth/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("gitlab exchange: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab exchange: status %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("gitlab exchange: decode: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("gitlab exchange: empty access token")
	}
	return &token, nil
}

func (g *GitLabProvider) GetUser(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.BaseURL+"/api/v4/user", nil)
	if err != nil {
		return nil, fmt.Errorf("gitlab get user: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gitlab get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gitlab get user: status %d: %s", resp.StatusCode, string(body))
	}

	var user gitlabUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("gitlab get user: decode: %w", err)
	}

	return &OAuthUser{
		ID:        fmt.Sprintf("%d", user.ID),
		Email:     user.Email,
		Username:  user.Username,
		AvatarURL: user.AvatarURL,
		Provider:  "gitlab",
	}, nil
}

type gitlabUser struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

// -------------------------------------------------------------------
// Google OAuth Provider
// -------------------------------------------------------------------

// GoogleProvider implements OAuthProvider for Google.
type GoogleProvider struct {
	Config OAuthConfig
}

func NewGoogleProvider(cfg OAuthConfig) *GoogleProvider {
	return &GoogleProvider{Config: cfg}
}

func (g *GoogleProvider) AuthURL(state string) string {
	params := url.Values{
		"client_id":     {g.Config.ClientID},
		"redirect_uri":  {g.Config.RedirectURL},
		"response_type": {"code"},
		"scope":         {"openid email profile"},
		"state":         {state},
		"access_type":   {"offline"},
		"prompt":        {"consent"},
	}
	return "https://accounts.google.com/o/oauth2/v2/auth?" + params.Encode()
}

func (g *GoogleProvider) Exchange(ctx context.Context, code string) (*OAuthToken, error) {
	data := url.Values{
		"client_id":     {g.Config.ClientID},
		"client_secret": {g.Config.ClientSecret},
		"code":          {code},
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {g.Config.RedirectURL},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://oauth2.googleapis.com/token",
		strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("google exchange: create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google exchange: status %d: %s", resp.StatusCode, string(body))
	}

	var token OAuthToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("google exchange: decode: %w", err)
	}
	if token.AccessToken == "" {
		return nil, errors.New("google exchange: empty access token")
	}
	return &token, nil
}

func (g *GoogleProvider) GetUser(ctx context.Context, token *OAuthToken) (*OAuthUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("google get user: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google get user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("google get user: status %d: %s", resp.StatusCode, string(body))
	}

	var user googleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("google get user: decode: %w", err)
	}

	// Google doesn't have a username; derive one from the email prefix.
	username := user.Email
	if idx := strings.Index(user.Email, "@"); idx > 0 {
		username = user.Email[:idx]
	}

	return &OAuthUser{
		ID:        user.ID,
		Email:     user.Email,
		Username:  username,
		AvatarURL: user.Picture,
		Provider:  "google",
	}, nil
}

type googleUser struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// -------------------------------------------------------------------
// Provider Registry
// -------------------------------------------------------------------

// NewOAuthProvider creates an OAuthProvider by name.
// Supported providers: "github", "gitlab", "google".
func NewOAuthProvider(provider string, cfg OAuthConfig, gitlabBaseURL string) (OAuthProvider, error) {
	switch strings.ToLower(provider) {
	case "github":
		return NewGitHubProvider(cfg), nil
	case "gitlab":
		return NewGitLabProvider(cfg, gitlabBaseURL), nil
	case "google":
		return NewGoogleProvider(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
	}
}
