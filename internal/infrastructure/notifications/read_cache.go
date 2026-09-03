package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// anti-regresion: BUG-1071 ver handoffs/regressions.md (no revertir sin leer)
const (
	notificationCacheVersionKeyFmt = "notifications:cache:ver:v1:%s:%s"
	notificationUnreadKeyFmt       = "notifications:unread:v1:%s:%s:%s"
	notificationListKeyFmt         = "notifications:list:v1:%s:%s:%s:%s:%s:%d"
)

// ReadCache is a short-TTL cache-aside for unread-count and first-page notification lists.
// Nil receiver is a no-op (always miss) so BFF works without REDIS_URL.
type ReadCache struct {
	client    *goredis.Client
	unreadTTL time.Duration
	listTTL   time.Duration
}

func NewReadCache(client *goredis.Client) *ReadCache {
	if client == nil {
		return nil
	}
	return &ReadCache{
		client:    client,
		unreadTTL: 45 * time.Second,
		listTTL:   30 * time.Second,
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (c *ReadCache) version(ctx context.Context, tenantID, userEmail string) string {
	if c == nil || c.client == nil {
		return "0"
	}
	key := fmt.Sprintf(notificationCacheVersionKeyFmt, strings.TrimSpace(tenantID), normalizeEmail(userEmail))
	ver, err := c.client.Get(ctx, key).Result()
	if err != nil || ver == "" {
		return "0"
	}
	return ver
}

func (c *ReadCache) GetUnreadCount(ctx context.Context, tenantID, userEmail string) (int, bool) {
	if c == nil || c.client == nil {
		return 0, false
	}
	tenantID = strings.TrimSpace(tenantID)
	email := normalizeEmail(userEmail)
	if tenantID == "" || email == "" {
		return 0, false
	}
	key := fmt.Sprintf(notificationUnreadKeyFmt, c.version(ctx, tenantID, email), tenantID, email)
	raw, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return 0, false
	}
	count, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return count, true
}

func (c *ReadCache) SetUnreadCount(ctx context.Context, tenantID, userEmail string, count int) {
	if c == nil || c.client == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	email := normalizeEmail(userEmail)
	if tenantID == "" || email == "" {
		return
	}
	key := fmt.Sprintf(notificationUnreadKeyFmt, c.version(ctx, tenantID, email), tenantID, email)
	_ = c.client.Set(ctx, key, strconv.Itoa(count), c.unreadTTL).Err()
}

func (c *ReadCache) GetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int) (map[string]any, bool) {
	if c == nil || c.client == nil {
		return nil, false
	}
	tenantID = strings.TrimSpace(tenantID)
	email := normalizeEmail(userEmail)
	if tenantID == "" || email == "" {
		return nil, false
	}
	key := fmt.Sprintf(
		notificationListKeyFmt,
		c.version(ctx, tenantID, email),
		tenantID,
		email,
		strings.TrimSpace(state),
		strings.TrimSpace(severity),
		limit,
	)
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil || len(raw) == 0 {
		return nil, false
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func (c *ReadCache) SetList(ctx context.Context, tenantID, userEmail, state, severity string, limit int, payload map[string]any) {
	if c == nil || c.client == nil || payload == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	email := normalizeEmail(userEmail)
	if tenantID == "" || email == "" {
		return
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	key := fmt.Sprintf(
		notificationListKeyFmt,
		c.version(ctx, tenantID, email),
		tenantID,
		email,
		strings.TrimSpace(state),
		strings.TrimSpace(severity),
		limit,
	)
	_ = c.client.Set(ctx, key, raw, c.listTTL).Err()
}

// InvalidateUser bumps the per-user cache version so unread/list keys miss immediately.
func (c *ReadCache) InvalidateUser(ctx context.Context, tenantID, userEmail string) {
	if c == nil || c.client == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	email := normalizeEmail(userEmail)
	if tenantID == "" || email == "" {
		return
	}
	key := fmt.Sprintf(notificationCacheVersionKeyFmt, tenantID, email)
	_ = c.client.Incr(ctx, key).Err()
	_ = c.client.Expire(ctx, key, 24*time.Hour).Err()
}

// CacheableListPage is true for first-page polls without date/cursor filters.
func CacheableListPage(state, severity, fromDate, toDate, beforeAt, beforeID string) bool {
	if strings.TrimSpace(beforeAt) != "" || strings.TrimSpace(beforeID) != "" {
		return false
	}
	if strings.TrimSpace(fromDate) != "" || strings.TrimSpace(toDate) != "" {
		return false
	}
	switch strings.TrimSpace(state) {
	case "", "all", "unread", "read":
		return true
	default:
		return false
	}
}
