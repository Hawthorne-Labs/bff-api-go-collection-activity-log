package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// SessionPayload holds server-side OAuth session state keyed by the opaque cookie id.
type SessionPayload struct {
	Sub              string  `json:"sub"`
	AccessToken      string  `json:"access_token"`
	AccessExpiresAt  int64   `json:"access_expires_at"`
	RefreshToken     string  `json:"refresh_token"`
	RefreshExpiresAt int64   `json:"refresh_expires_at"`
	CSRFToken        string  `json:"csrf_token"`
	IDToken          *string `json:"id_token,omitempty"`
}

// FlowState holds pre-login OIDC state keyed by the OAuth state parameter.
type FlowState struct {
	CodeVerifier string `json:"code_verifier"`
	Nonce        string `json:"nonce"`
	ReturnTo     string `json:"return_to"`
	CreatedAt    int64  `json:"created_at"`
}

// Store persists session payloads in Redis or memory.
type Store struct {
	client *goredis.Client
	memory *memorySessionStore
}

// FlowStateStore persists OIDC flow state in Redis or memory.
type FlowStateStore struct {
	client *goredis.Client
	memory *memoryFlowStateStore
}

type memorySessionStore struct {
	mu      sync.Mutex
	entries map[string]memoryEntry
}

type memoryFlowStateStore struct {
	mu      sync.Mutex
	entries map[string]memoryFlowEntry
}

type memoryEntry struct {
	payload   SessionPayload
	expiresAt int64
}

type memoryFlowEntry struct {
	state     FlowState
	expiresAt int64
}

func sessionKey(sessionID string) string {
	return "bff:session:" + sessionID
}

func flowKey(state string) string {
	return "bff:flow:" + state
}

// NewStore creates a Redis-backed session store.
func NewStore(client *goredis.Client) *Store {
	if client == nil {
		return NewMemoryStore()
	}
	return &Store{client: client}
}

// NewMemoryStore creates an in-memory session store.
func NewMemoryStore() *Store {
	return &Store{memory: &memorySessionStore{entries: make(map[string]memoryEntry)}}
}

// NewFlowStateStore creates a Redis-backed OIDC flow store.
func NewFlowStateStore(client *goredis.Client) *FlowStateStore {
	if client == nil {
		return NewMemoryFlowStateStore()
	}
	return &FlowStateStore{client: client}
}

// NewMemoryFlowStateStore creates an in-memory OIDC flow store.
func NewMemoryFlowStateStore() *FlowStateStore {
	return &FlowStateStore{memory: &memoryFlowStateStore{entries: make(map[string]memoryFlowEntry)}}
}

func CookieName() string {
	return "bff_session"
}

func NewCSRFToken() (string, error) {
	return tokenURLSafe(32)
}

func NewSessionID() (string, error) {
	return tokenURLSafe(32)
}

func NewStateID() (string, error) {
	return tokenURLSafe(24)
}

func tokenURLSafe(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.TrimRight(base64.URLEncoding.EncodeToString(buf), "="), nil
}

func (p SessionPayload) toJSON() (string, error) {
	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func sessionPayloadFromJSON(raw string) (*SessionPayload, error) {
	var payload SessionPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

// Put stores a session payload with the given TTL.
func (s *Store) Put(sessionID string, payload SessionPayload, ttlSeconds int) error {
	if s == nil || sessionID == "" {
		return nil
	}
	if s.memory != nil {
		s.memory.put(sessionID, payload, ttlSeconds)
		return nil
	}
	raw, err := payload.toJSON()
	if err != nil {
		return err
	}
	return s.client.Set(context.Background(), sessionKey(sessionID), raw, time.Duration(ttlSeconds)*time.Second).Err()
}

// Get loads a session payload by id.
func (s *Store) Get(sessionID string) *SessionPayload {
	if s == nil || sessionID == "" {
		return nil
	}
	if s.memory != nil {
		return s.memory.get(sessionID)
	}
	raw, err := s.client.Get(context.Background(), sessionKey(sessionID)).Result()
	if err != nil || raw == "" {
		return nil
	}
	payload, err := sessionPayloadFromJSON(raw)
	if err != nil {
		s.client.Del(context.Background(), sessionKey(sessionID))
		return nil
	}
	return payload
}

// Delete removes a session by id.
func (s *Store) Delete(sessionID string) {
	if s == nil || sessionID == "" {
		return
	}
	if s.memory != nil {
		s.memory.delete(sessionID)
		return
	}
	s.client.Del(context.Background(), sessionKey(sessionID))
}

func (m *memorySessionStore) put(sessionID string, payload SessionPayload, ttlSeconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[sessionID] = memoryEntry{
		payload:   payload,
		expiresAt: time.Now().Unix() + int64(ttlSeconds),
	}
}

func (m *memorySessionStore) get(sessionID string) *SessionPayload {
	m.mu.Lock()
	defer m.mu.Unlock()
	entry, ok := m.entries[sessionID]
	if !ok {
		return nil
	}
	if entry.expiresAt <= time.Now().Unix() {
		delete(m.entries, sessionID)
		return nil
	}
	payload := entry.payload
	return &payload
}

func (m *memorySessionStore) delete(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, sessionID)
}

// Put stores OIDC flow state with the given TTL.
func (f *FlowStateStore) Put(state string, value FlowState, ttlSeconds int) error {
	if f == nil || state == "" {
		return nil
	}
	if f.memory != nil {
		f.memory.put(state, value, ttlSeconds)
		return nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return f.client.Set(context.Background(), flowKey(state), raw, time.Duration(ttlSeconds)*time.Second).Err()
}

// Pop atomically retrieves and deletes OIDC flow state.
func (f *FlowStateStore) Pop(state string) *FlowState {
	if f == nil || state == "" {
		return nil
	}
	if f.memory != nil {
		return f.memory.pop(state)
	}
	ctx := context.Background()
	key := flowKey(state)
	pipe := f.client.Pipeline()
	getCmd := pipe.Get(ctx, key)
	pipe.Del(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil && err != goredis.Nil {
		return nil
	}
	raw, err := getCmd.Result()
	if err != nil || raw == "" {
		return nil
	}
	var flow FlowState
	if err := json.Unmarshal([]byte(raw), &flow); err != nil {
		return nil
	}
	return &flow
}

func (m *memoryFlowStateStore) put(state string, value FlowState, ttlSeconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[state] = memoryFlowEntry{
		state:     value,
		expiresAt: time.Now().Unix() + int64(ttlSeconds),
	}
}

func (m *memoryFlowStateStore) pop(state string) *FlowState {
	m.mu.Lock()
	entry, ok := m.entries[state]
	if ok {
		delete(m.entries, state)
	}
	m.mu.Unlock()
	if !ok {
		return nil
	}
	if entry.expiresAt <= time.Now().Unix() {
		return nil
	}
	flow := entry.state
	return &flow
}

func ConstantTimeEqual(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := 0; i < len(a); i++ {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
