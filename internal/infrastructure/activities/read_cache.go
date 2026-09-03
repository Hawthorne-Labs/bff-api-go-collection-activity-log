package activities

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// anti-regresion: BUG-1072 ver handoffs/regressions.md (no revertir sin leer)
const (
	activityScopeLoanVerFmt   = "activities:cache:ver:v1:%s:loan:%s"
	activityScopeClientVerFmt = "activities:cache:ver:v1:%s:client:%s"
	activityListKeyFmt        = "activities:list:v1:%s:%s:%s:%s:%s:%s:%s:%s:%s:%d:%d"
)

// ReadCache is a short-TTL cache-aside for scoped first-page activity lists.
// Nil receiver is a no-op so the BFF works without REDIS_URL.
type ReadCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewReadCache(client *goredis.Client) *ReadCache {
	if client == nil {
		return nil
	}
	return &ReadCache{client: client, ttl: 25 * time.Second}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func loanIDsFingerprint(loanIDs []string) string {
	if len(loanIDs) == 0 {
		return "-"
	}
	cp := append([]string(nil), loanIDs...)
	sort.Strings(cp)
	sum := sha1.Sum([]byte(strings.Join(cp, ",")))
	return hex.EncodeToString(sum[:8])
}

func (c *ReadCache) scopeVersion(ctx context.Context, key string) string {
	if c == nil || c.client == nil {
		return "0"
	}
	ver, err := c.client.Get(ctx, key).Result()
	if err != nil || ver == "" {
		return "0"
	}
	return ver
}

func (c *ReadCache) versions(ctx context.Context, tenantID, loanID, clientID string) (loanVer, clientVer string) {
	tenantID = strings.TrimSpace(tenantID)
	loanID = strings.TrimSpace(loanID)
	clientID = strings.TrimSpace(clientID)
	loanVer, clientVer = "0", "0"
	if loanID != "" {
		loanVer = c.scopeVersion(ctx, fmt.Sprintf(activityScopeLoanVerFmt, tenantID, loanID))
	}
	if clientID != "" {
		clientVer = c.scopeVersion(ctx, fmt.Sprintf(activityScopeClientVerFmt, tenantID, clientID))
	}
	return loanVer, clientVer
}

func (c *ReadCache) listKey(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) string {
	loanVer, clientVer := c.versions(ctx, tenantID, loanID, clientID)
	return fmt.Sprintf(
		activityListKeyFmt,
		loanVer,
		clientVer,
		strings.TrimSpace(tenantID),
		normalizeEmail(userEmail),
		strings.TrimSpace(loanID),
		loanIDsFingerprint(loanIDs),
		strings.TrimSpace(clientID),
		strings.TrimSpace(agentID),
		strings.TrimSpace(agentName)+"|"+strings.TrimSpace(activityType),
		limit,
		offset,
	)
}

// CacheableList is true for first-page lists scoped to one loan or client.
// anti-regresion: BUG-1076 — loan_ids-only batches lack per-loan versioning and must bypass cache.
func CacheableList(loanID string, _ []string, clientID string, offset int) bool {
	if offset != 0 {
		return false
	}
	if strings.TrimSpace(loanID) != "" || strings.TrimSpace(clientID) != "" {
		return true
	}
	return false
}

func (c *ReadCache) GetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) (map[string]any, bool) {
	if c == nil || c.client == nil || !CacheableList(loanID, loanIDs, clientID, offset) {
		return nil, false
	}
	raw, err := c.client.Get(ctx, c.listKey(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset)).Bytes()
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func (c *ReadCache) SetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, payload map[string]any) {
	if c == nil || c.client == nil || payload == nil || !CacheableList(loanID, loanIDs, clientID, offset) {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = c.client.Set(ctx, c.listKey(ctx, tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset), raw, c.ttl).Err()
}

// InvalidateScopes bumps loan/client versions so scoped lists miss immediately.
func (c *ReadCache) InvalidateScopes(ctx context.Context, tenantID, loanID, clientID string) {
	if c == nil || c.client == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	loanID = strings.TrimSpace(loanID)
	clientID = strings.TrimSpace(clientID)
	if tenantID == "" {
		return
	}
	bump := func(key string) {
		_ = c.client.Incr(ctx, key).Err()
		_ = c.client.Expire(ctx, key, 24*time.Hour).Err()
	}
	if loanID != "" {
		bump(fmt.Sprintf(activityScopeLoanVerFmt, tenantID, loanID))
	}
	if clientID != "" {
		bump(fmt.Sprintf(activityScopeClientVerFmt, tenantID, clientID))
	}
}

// ScopeIDsFromPayload extracts loan/client ids when present in plaintext create payloads.
func ScopeIDsFromPayload(payload map[string]any) (loanID, clientID string) {
	if payload == nil {
		return "", ""
	}
	loanID = stringField(payload, "loan_id", "loanId")
	clientID = stringField(payload, "client_id", "clientId")
	if loanIDs, ok := payload["loan_ids"].([]any); ok {
		for _, raw := range loanIDs {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				if loanID == "" {
					loanID = strings.TrimSpace(s)
				}
			}
		}
	}
	return loanID, clientID
}

func stringField(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if v, ok := payload[key].(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}
