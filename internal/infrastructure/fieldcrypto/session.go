// Package fieldcrypto implements ECDHE(P-256) + HKDF-SHA256 crypto-session
// handshake compatible with the Python BFF's CryptoSessionManager (enc:v1).
package fieldcrypto

import (
	"crypto/ecdh"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"
)

const (
	info       = "hawthorne-fieldcrypto-v1"
	audience   = "crypto-session"
	defaultTTL = 900
)

func b64uEnc(raw []byte) string {
	return base64.RawURLEncoding.EncodeToString(raw)
}

func b64uDec(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// SessionStore is an in-memory session-id -> key map with TTL.
type SessionStore struct {
	ttl  time.Duration
	mu   sync.RWMutex
	keys map[string]sessionEntry
}

type sessionEntry struct {
	key    []byte
	expiry time.Time
}

// NewSessionStore creates a store with the given TTL.
func NewSessionStore(ttlSeconds int) *SessionStore {
	if ttlSeconds <= 0 {
		ttlSeconds = defaultTTL
	}
	return &SessionStore{
		ttl:  time.Duration(ttlSeconds) * time.Second,
		keys: make(map[string]sessionEntry),
	}
}

// Put stores a session key.
func (s *SessionStore) Put(sessionID string, key []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[sessionID] = sessionEntry{key: key, expiry: time.Now().Add(s.ttl)}
}

// Get retrieves a session key. Returns nil if expired or missing.
func (s *SessionStore) Get(sessionID string) []byte {
	s.mu.RLock()
	entry, ok := s.keys[sessionID]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiry) {
		s.mu.Lock()
		delete(s.keys, sessionID)
		s.mu.Unlock()
		return nil
	}
	return entry.key
}

// HandshakeResult is the JSON response returned to the frontend.
type HandshakeResult struct {
	ServerPublicKey   string `json:"serverPublicKey"`
	Salt              string `json:"salt"`
	CryptoSessionID   string `json:"cryptoSessionId"`
	CryptoAccessToken string `json:"cryptoAccessToken"`
	ExpiresIn         int    `json:"expiresIn"`
}

// SessionManager performs the local P-256 ECDH handshake.
type SessionManager struct {
	store         *SessionStore
	signingSecret []byte
	issuer        string
	ttlSeconds    int
}

// NewSessionManager creates a manager.
func NewSessionManager(store *SessionStore, signingSecret string, issuer string, ttlSeconds int) *SessionManager {
	if ttlSeconds <= 0 {
		ttlSeconds = defaultTTL
	}
	return &SessionManager{
		store:         store,
		signingSecret: []byte(signingSecret),
		issuer:        issuer,
		ttlSeconds:    ttlSeconds,
	}
}

// Handshake performs the ECDHE(P-256) + HKDF-SHA256 handshake.
func (m *SessionManager) Handshake(clientPublicB64 string, subject string, scope string) (*HandshakeResult, error) {
	clientPubBytes, err := b64uDec(clientPublicB64)
	if err != nil {
		return nil, fmt.Errorf("decode client public key: %w", err)
	}

	clientPub, err := ecdh.P256().NewPublicKey(clientPubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse client public key: %w", err)
	}

	serverPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate server key: %w", err)
	}

	shared, err := serverPriv.ECDH(clientPub)
	if err != nil {
		return nil, fmt.Errorf("ECDH derive: %w", err)
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	sessionKey := hkdfSHA256(shared, salt, []byte(info), 32)

	sessionID := randomHex(16)
	m.store.Put(sessionID, sessionKey)

	serverPubBytes := serverPriv.PublicKey().Bytes()

	now := time.Now()
	token, err := m.signJWT(subject, scope, sessionID, now)
	if err != nil {
		return nil, fmt.Errorf("sign JWT: %w", err)
	}

	return &HandshakeResult{
		ServerPublicKey:   b64uEnc(serverPubBytes),
		Salt:              b64uEnc(salt),
		CryptoSessionID:   sessionID,
		CryptoAccessToken: token,
		ExpiresIn:         m.ttlSeconds,
	}, nil
}

// SessionKey returns the key for a session ID, or nil if not found/expired.
func (m *SessionManager) SessionKey(sessionID string) []byte {
	return m.store.Get(sessionID)
}

func (m *SessionManager) signJWT(subject, scope, sessionID string, now time.Time) (string, error) {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":               m.issuer,
		"aud":               audience,
		"sub":               subject,
		"scope":             scope,
		"crypto_session_id": sessionID,
		"iat":               now.Unix(),
		"exp":               now.Add(time.Duration(m.ttlSeconds) * time.Second).Unix(),
	}

	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := b64uEnc(headerJSON)
	claimsB64 := b64uEnc(claimsJSON)
	signingInput := headerB64 + "." + claimsB64

	mac := hmac.New(sha256.New, m.signingSecret)
	mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)

	return signingInput + "." + b64uEnc(sig), nil
}

func hkdfSHA256(secret, salt, info []byte, length int) []byte {
	mac := hmac.New(sha256.New, salt)
	mac.Write(secret)
	prk := mac.Sum(nil)

	mac2 := hmac.New(sha256.New, prk)
	mac2.Write(info)
	mac2.Write([]byte{1})
	return mac2.Sum(nil)[:length]
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return strconv.FormatInt(time.Now().UnixNano(), 36) + fmt.Sprintf("%x", b)[:n]
}
