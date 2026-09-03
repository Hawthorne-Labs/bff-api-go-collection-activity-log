package activities

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MemoryReadCache is an in-process stand-in for ReadCache (unit tests / no Redis).
type MemoryReadCache struct {
	mu     sync.Mutex
	loanV  map[string]int64
	clientV map[string]int64
	lists  map[string]map[string]any
}

func NewMemoryReadCache() *MemoryReadCache {
	return &MemoryReadCache{
		loanV:   map[string]int64{},
		clientV: map[string]int64{},
		lists:   map[string]map[string]any{},
	}
}

func (c *MemoryReadCache) key(tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	loanVer := c.loanV[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(loanID)]
	clientVer := c.clientV[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(clientID)]
	return fmt.Sprintf("%d|%d|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d",
		loanVer, clientVer, tenantID, normalizeEmail(userEmail), loanID, loanIDsFingerprint(loanIDs),
		clientID, agentID, agentName, activityType, limit, offset)
}

func (c *MemoryReadCache) GetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int) (map[string]any, bool) {
	if !CacheableList(loanID, loanIDs, clientID, offset) {
		return nil, false
	}
	key := c.key(tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset)
	c.mu.Lock()
	defer c.mu.Unlock()
	payload, ok := c.lists[key]
	return payload, ok
}

func (c *MemoryReadCache) SetList(ctx context.Context, tenantID, userEmail, loanID string, loanIDs []string, clientID, agentID, agentName, activityType string, limit, offset int, payload map[string]any) {
	if !CacheableList(loanID, loanIDs, clientID, offset) || payload == nil {
		return
	}
	key := c.key(tenantID, userEmail, loanID, loanIDs, clientID, agentID, agentName, activityType, limit, offset)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lists[key] = payload
}

func (c *MemoryReadCache) InvalidateScopes(_ context.Context, tenantID, loanID, clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.TrimSpace(loanID) != "" {
		c.loanV[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(loanID)]++
	}
	if strings.TrimSpace(clientID) != "" {
		c.clientV[strings.TrimSpace(tenantID)+"|"+strings.TrimSpace(clientID)]++
	}
}
