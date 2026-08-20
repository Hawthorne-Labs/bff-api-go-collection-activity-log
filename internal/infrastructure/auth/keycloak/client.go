package keycloak

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// KeycloakOIDCError indicates a Keycloak token or connectivity failure.
type KeycloakOIDCError struct {
	Message string
}

func (e *KeycloakOIDCError) Error() string {
	return e.Message
}

// TokenSet holds OAuth tokens returned by Keycloak.
type TokenSet struct {
	AccessToken      string
	RefreshToken     string
	ExpiresIn        int
	RefreshExpiresIn int
	IDToken          string
	Raw              map[string]any
}

// KeycloakOIDCClient performs Authorization Code + PKCE against Keycloak.
type KeycloakOIDCClient struct {
	baseURL   string
	publicURL string
	realm     string
	clientID  string
	http      *http.Client
}

// NewKeycloakOIDCClient builds a Keycloak OIDC client from configuration values.
func NewKeycloakOIDCClient(baseURL, publicURL, realm, clientID string, timeoutSeconds int) *KeycloakOIDCClient {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	public := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if public == "" {
		public = base
	}
	timeout := defaultTimeout
	if timeoutSeconds > 0 {
		timeout = time.Duration(timeoutSeconds) * time.Second
	}
	return &KeycloakOIDCClient{
		baseURL:   base,
		publicURL: public,
		realm:     realm,
		clientID:  clientID,
		http:      &http.Client{Timeout: timeout},
	}
}

// ClientID returns the configured OAuth client id.
func (c *KeycloakOIDCClient) ClientID() string {
	return c.clientID
}

func (c *KeycloakOIDCClient) authorizeURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/auth", c.publicURL, c.realm)
}

func (c *KeycloakOIDCClient) tokenURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", c.baseURL, c.realm)
}

func (c *KeycloakOIDCClient) logoutURL() string {
	return fmt.Sprintf("%s/realms/%s/protocol/openid-connect/logout", c.baseURL, c.realm)
}

// MakePKCEPair returns (code_verifier, code_challenge_S256).
func MakePKCEPair() (string, string, error) {
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	verifier := b64URL(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge := b64URL(sum[:])
	return verifier, challenge, nil
}

func b64URL(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// BuildAuthorizeURL constructs the browser redirect URL for the OIDC flow.
func (c *KeycloakOIDCClient) BuildAuthorizeURL(redirectURI, state, codeChallenge, nonce, scope string) string {
	if scope == "" {
		scope = "openid profile email"
	}
	params := url.Values{
		"client_id":             {c.clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {scope},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	return c.authorizeURL() + "?" + params.Encode()
}

func (c *KeycloakOIDCClient) postToken(form url.Values) (*TokenSet, error) {
	req, err := http.NewRequest(http.MethodPost, c.tokenURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, &KeycloakOIDCError{Message: fmt.Sprintf("keycloak unreachable: %v", err)}
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, &KeycloakOIDCError{Message: fmt.Sprintf("keycloak unreachable: %v", err)}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &KeycloakOIDCError{Message: fmt.Sprintf("keycloak unreachable: %v", err)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &KeycloakOIDCError{Message: fmt.Sprintf("keycloak token endpoint %d", resp.StatusCode)}
	}

	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, &KeycloakOIDCError{Message: "keycloak token endpoint invalid response"}
	}

	accessToken, _ := parsed["access_token"].(string)
	refreshToken, _ := parsed["refresh_token"].(string)
	idToken, _ := parsed["id_token"].(string)
	expiresIn := intNumber(parsed["expires_in"])
	refreshExpiresIn := intNumber(parsed["refresh_expires_in"])

	return &TokenSet{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		ExpiresIn:        expiresIn,
		RefreshExpiresIn: refreshExpiresIn,
		IDToken:          idToken,
		Raw:              parsed,
	}, nil
}

func intNumber(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// ExchangeCode redeems an authorization code using PKCE.
func (c *KeycloakOIDCClient) ExchangeCode(code, codeVerifier, redirectURI string) (*TokenSet, error) {
	return c.postToken(url.Values{
		"grant_type":    {"authorization_code"},
		"client_id":     {c.clientID},
		"code":          {code},
		"code_verifier": {codeVerifier},
		"redirect_uri":  {redirectURI},
	})
}

// Refresh obtains new tokens using a refresh token.
func (c *KeycloakOIDCClient) Refresh(refreshToken string) (*TokenSet, error) {
	return c.postToken(url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {c.clientID},
		"refresh_token": {refreshToken},
	})
}

// PasswordGrant performs the resource-owner password grant (dev-login only).
func (c *KeycloakOIDCClient) PasswordGrant(username, password string) (*TokenSet, error) {
	return c.postToken(url.Values{
		"grant_type": {"password"},
		"client_id":  {c.clientID},
		"username":   {username},
		"password":   {password},
		"scope":      {"openid profile email"},
	})
}

// Logout revokes the refresh token at Keycloak (best-effort).
func (c *KeycloakOIDCClient) Logout(refreshToken string) {
	if refreshToken == "" {
		return
	}
	form := url.Values{
		"client_id":     {c.clientID},
		"refresh_token": {refreshToken},
	}
	req, err := http.NewRequest(http.MethodPost, c.logoutURL(), strings.NewReader(form.Encode()))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
}
