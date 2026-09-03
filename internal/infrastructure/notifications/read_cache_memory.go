package notifications

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// MemoryReadCache is an in-process stand-in for ReadCache (unit tests / no Redis).
type MemoryReadCache struct {
	mu     sync.Mutex
	ver    map[string]int64
	unread map[string]int
	lists  map[string]map[string]any
}

func NewMemoryReadCache() *MemoryReadCache {
	return &MemoryReadCache{
		ver:    map[string]int64{},
		unread: map[string]int{},
		lists:  map[string]map[string]any{},
	}
}

func (c *MemoryReadCache) userKey(tenantID, userEmail string) string {
	return strings.TrimSpace(tenantID) + "|" + normalizeEmail(userEmail)
}

func (c *MemoryReadCache) versionLocked(tenantID, userEmail string) int64 {
	return c.ver[c.userKey(tenantID, userEmail)]
}

func (c *MemoryReadCache) GetUnreadCount(_ context.Context, tenantID, userEmail string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d|%s|unread", c.versionLocked(tenantID, userEmail), c.userKey(tenantID, userEmail))
	count, ok := c.unread[key]
	return count, ok
}

func (c *MemoryReadCache) SetUnreadCount(_ context.Context, tenantID, userEmail string, count int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d|%s|unread", c.versionLocked(tenantID, userEmail), c.userKey(tenantID, userEmail))
	c.unread[key] = count
}

func (c *MemoryReadCache) GetList(_ context.Context, tenantID, userEmail, state, severity string, limit int) (map[string]any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d|%s|list|%s|%s|%d", c.versionLocked(tenantID, userEmail), c.userKey(tenantID, userEmail), state, severity, limit)
	payload, ok := c.lists[key]
	if !ok {
		return nil, false
	}
	return payload, true
}

func (c *MemoryReadCache) SetList(_ context.Context, tenantID, userEmail, state, severity string, limit int, payload map[string]any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fmt.Sprintf("%d|%s|list|%s|%s|%d", c.versionLocked(tenantID, userEmail), c.userKey(tenantID, userEmail), state, severity, limit)
	c.lists[key] = payload
}

func (c *MemoryReadCache) InvalidateUser(_ context.Context, tenantID, userEmail string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ver[c.userKey(tenantID, userEmail)]++
}
