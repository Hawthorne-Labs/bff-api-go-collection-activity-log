package handler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/auth/keycloak"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/config"
	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/infrastructure/session"
)

// AuthHandler handles BFF cookie-based OIDC authentication.
type AuthHandler struct {
	cfg        *config.Config
	oidc       *keycloak.KeycloakOIDCClient
	sessions   *session.Store
	flowStates *session.FlowStateStore
}

// NewAuthHandler wires auth dependencies.
func NewAuthHandler(
	cfg *config.Config,
	oidc *keycloak.KeycloakOIDCClient,
	sessions *session.Store,
	flowStates *session.FlowStateStore,
) *AuthHandler {
	return &AuthHandler{
		cfg:        cfg,
		oidc:       oidc,
		sessions:   sessions,
		flowStates: flowStates,
	}
}

type devLoginRequest struct {
	Username string `json:"username" binding:"required,min=1,max=256"`
	Password string `json:"password" binding:"required,min=1,max=512"`
}

// Login initiates Authorization Code + PKCE against Keycloak.
func (h *AuthHandler) Login(c *gin.Context) {
	returnTo := strings.TrimSpace(c.Query("return_to"))
	if returnTo == "" {
		returnTo = h.cfg.BFFPostLoginURL
	}
	if len(returnTo) > 2048 {
		returnTo = h.cfg.BFFPostLoginURL
	}

	codeVerifier, codeChallenge, err := keycloak.MakePKCEPair()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 5000, "message": "No se pudo iniciar sesión."}})
		return
	}
	state, err := session.NewStateID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 5000, "message": "No se pudo iniciar sesión."}})
		return
	}
	nonce, err := session.NewStateID()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 5000, "message": "No se pudo iniciar sesión."}})
		return
	}

	_ = h.flowStates.Put(state, session.FlowState{
		CodeVerifier: codeVerifier,
		Nonce:        nonce,
		ReturnTo:     returnTo,
		CreatedAt:    time.Now().Unix(),
	}, 300)

	url := h.oidc.BuildAuthorizeURL(h.cfg.BFFOIDCRedirectURI, state, codeChallenge, nonce, "")
	c.Redirect(http.StatusFound, url)
}

// Callback exchanges the authorization code and sets the session cookie.
func (h *AuthHandler) Callback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": map[string]any{"code": 90013, "message": "Solicitud invalida."}})
		return
	}
	if len(code) > 2048 || len(state) > 512 {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": 20003, "message": "Solicitud invalida."}})
		return
	}

	flow := h.flowStates.Pop(state)
	if flow == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": map[string]any{"code": 20003, "message": "Solicitud invalida."}})
		return
	}

	tokens, err := h.oidc.ExchangeCode(code, flow.CodeVerifier, h.cfg.BFFOIDCRedirectURI)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": 5071, "message": err.Error()}})
		return
	}

	sessionID, maxAge, err := h.saveSessionFromTokens(tokens, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 5000, "message": "No se pudo crear la sesión."}})
		return
	}

	setSessionCookie(c, h.cfg, sessionID, maxAge)
	c.Redirect(http.StatusFound, flow.ReturnTo)
}

// Logout revokes the refresh token and clears the session cookie.
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(session.CookieName())
	revoked := false
	if sessionID != "" {
		payload := h.sessions.Get(sessionID)
		if payload != nil && payload.RefreshToken != "" {
			h.oidc.Logout(payload.RefreshToken)
		}
		h.sessions.Delete(sessionID)
		revoked = true
	}
	clearSessionCookie(c, h.cfg)
	c.JSON(http.StatusOK, gin.H{"revoked": revoked})
}

// Me returns the current session authentication state.
func (h *AuthHandler) Me(c *gin.Context) {
	sessionID, _ := c.Cookie(session.CookieName())
	if sessionID == "" {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	payload := h.sessions.Get(sessionID)
	if payload == nil {
		c.JSON(http.StatusOK, gin.H{"authenticated": false})
		return
	}
	body := gin.H{"authenticated": true}
	if payload.Sub != "" {
		body["sub"] = payload.Sub
	}
	if payload.CSRFToken != "" {
		body["csrf_token"] = payload.CSRFToken
	}
	c.JSON(http.StatusOK, body)
}

// DevLogin performs a password grant in non-production environments.
func (h *AuthHandler) DevLogin(c *gin.Context) {
	if !h.cfg.IsDevLoginEnabled() {
		c.JSON(http.StatusNotFound, gin.H{"error": map[string]any{"code": 4040, "message": "not found"}})
		return
	}

	var req devLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": map[string]any{"code": 90013, "message": "Solicitud invalida."}})
		return
	}

	tokens, err := h.oidc.PasswordGrant(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": map[string]any{"code": 5071, "message": "invalid credentials"}})
		return
	}

	sub := extractSub(tokens.AccessToken)
	sessionID, maxAge, err := h.saveSessionFromTokens(tokens, sub)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": map[string]any{"code": 5000, "message": "No se pudo crear la sesión."}})
		return
	}

	saved := h.sessions.Get(sessionID)
	csrf := ""
	if saved != nil {
		csrf = saved.CSRFToken
	}

	setSessionCookie(c, h.cfg, sessionID, maxAge)
	c.JSON(http.StatusOK, gin.H{
		"authenticated": true,
		"sub":           sub,
		"csrf_token":    csrf,
		"access_token":  tokens.AccessToken,
	})
}

func (h *AuthHandler) saveSessionFromTokens(tokens *keycloak.TokenSet, subHint string) (string, int, error) {
	now := time.Now().Unix()
	sub := subHint
	if sub == "" {
		sub = extractSub(tokens.AccessToken)
	}
	csrf, err := session.NewCSRFToken()
	if err != nil {
		return "", 0, err
	}
	sessionID, err := session.NewSessionID()
	if err != nil {
		return "", 0, err
	}

	var idToken *string
	if tokens.IDToken != "" {
		idToken = &tokens.IDToken
	}

	payload := session.SessionPayload{
		Sub:              sub,
		AccessToken:      tokens.AccessToken,
		AccessExpiresAt:  now + int64(tokens.ExpiresIn),
		RefreshToken:     tokens.RefreshToken,
		RefreshExpiresAt: now + int64(tokens.RefreshExpiresIn),
		CSRFToken:        csrf,
		IDToken:          idToken,
	}

	ttl := tokens.RefreshExpiresIn
	if ttl < 60 {
		ttl = 60
	}
	if err := h.sessions.Put(sessionID, payload, ttl); err != nil {
		return "", 0, err
	}
	return sessionID, ttl, nil
}

func extractSub(jwtToken string) string {
	parts := strings.Split(jwtToken, ".")
	if len(parts) < 2 {
		return ""
	}
	padLen := (4 - len(parts[1])%4) % 4
	padded := parts[1] + strings.Repeat("=", padLen)
	raw, err := base64.URLEncoding.DecodeString(padded)
	if err != nil {
		raw, err = base64.RawURLEncoding.DecodeString(parts[1])
		if err != nil {
			return ""
		}
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		return ""
	}
	sub, _ := claims["sub"].(string)
	return sub
}

func setSessionCookie(c *gin.Context, cfg *config.Config, sessionID string, maxAge int) {
	c.SetSameSite(cfg.CookieSameSite())
	c.SetCookie(
		session.CookieName(),
		sessionID,
		maxAge,
		"/",
		"",
		cfg.BFFCookieSecure,
		true,
	)
}

func clearSessionCookie(c *gin.Context, cfg *config.Config) {
	c.SetSameSite(cfg.CookieSameSite())
	c.SetCookie(
		session.CookieName(),
		"",
		-1,
		"/",
		"",
		cfg.BFFCookieSecure,
		true,
	)
}
