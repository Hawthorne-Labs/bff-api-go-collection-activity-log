package notifications

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

func Channel(tenantID, userEmail string) string {
	scope := strings.ToLower(strings.TrimSpace(tenantID)) + "\x00" + strings.ToLower(strings.TrimSpace(userEmail))
	sum := sha256.Sum256([]byte(scope))
	return "notifications:" + hex.EncodeToString(sum[:])
}

func EventFrame(event map[string]any) (int, string, bool) {
	raw := pickField(event, "event_id", "EventID")
	if raw == nil {
		return 0, "", false
	}
	eventID, err := strconv.Atoi(fmt.Sprint(raw))
	if err != nil || eventID < 0 {
		return 0, "", false
	}
	return eventID, fmt.Sprintf("id: %d\nevent: notification\ndata: {\"event_id\":%d}\n\n", eventID, eventID), true
}

type RecoverFunc func(afterEventID int) ([]map[string]any, error)

type SSE struct {
	redis          *goredis.Client
	tenantID       string
	userEmail      string
	recovered      []map[string]any
	lastEventID    int
	recoverAfter   RecoverFunc
	maxLifetime    time.Duration
}

func NewSSE(
	redisClient *goredis.Client,
	tenantID, userEmail string,
	recovered []map[string]any,
	lastEventID int,
	recoverAfter RecoverFunc,
	maxLifetime time.Duration,
) *SSE {
	if maxLifetime < 15*time.Second {
		maxLifetime = 300 * time.Second
	}
	return &SSE{
		redis:        redisClient,
		tenantID:     tenantID,
		userEmail:    userEmail,
		recovered:    recovered,
		lastEventID:  lastEventID,
		recoverAfter: recoverAfter,
		maxLifetime:  maxLifetime,
	}
}

func (s *SSE) writeFrames(w io.Writer, events []map[string]any) error {
	for _, event := range events {
		eventID, frame, ok := EventFrame(event)
		if !ok || eventID <= s.lastEventID {
			continue
		}
		s.lastEventID = eventID
		if _, err := io.WriteString(w, frame); err != nil {
			return err
		}
	}
	return nil
}

func (s *SSE) Serve(ctx context.Context, w io.Writer, flush func()) error {
	if s.redis == nil {
		if err := s.writeFrames(w, s.recovered); err != nil {
			return err
		}
		_, err := io.WriteString(w, "event: retry\ndata: {\"reason\":\"stream_unavailable\"}\n\n")
		return err
	}

	pubsub := s.redis.Subscribe(ctx, Channel(s.tenantID, s.userEmail))
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		if err := s.writeFrames(w, s.recovered); err != nil {
			return err
		}
		_, err = io.WriteString(w, "event: retry\ndata: {\"reason\":\"stream_unavailable\"}\n\n")
		return err
	}

	if err := s.writeFrames(w, s.recovered); err != nil {
		return err
	}
	if s.recoverAfter != nil {
		events, err := s.recoverAfter(s.lastEventID)
		if err != nil {
			_, werr := io.WriteString(w, "event: retry\ndata: {\"reason\":\"recovery_unavailable\"}\n\n")
			if werr != nil {
				return werr
			}
			return err
		}
		if err := s.writeFrames(w, events); err != nil {
			return err
		}
	}

	// anti-regresion: BUG-0353 ver handoffs/regressions.md (no revertir sin leer)
	if _, err := io.WriteString(w, ": connected\n\n"); err != nil {
		return err
	}
	if flush != nil {
		flush()
	}

	expiresAt := time.Now().Add(s.maxLifetime)
	for {
		remaining := time.Until(expiresAt)
		if remaining <= 0 {
			return nil
		}
		timeout := remaining
		if timeout > 15*time.Second {
			timeout = 15 * time.Second
		}

		msg, err := pubsub.ReceiveTimeout(ctx, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if s.recoverAfter != nil {
				events, recErr := s.recoverAfter(s.lastEventID)
				if recErr != nil {
					_, werr := io.WriteString(w, "event: retry\ndata: {\"reason\":\"recovery_unavailable\"}\n\n")
					if werr != nil {
						return werr
					}
					return recErr
				}
				if err := s.writeFrames(w, events); err != nil {
					return err
				}
			}
			if _, err := io.WriteString(w, ": keep-alive\n\n"); err != nil {
				return err
			}
			if flush != nil {
				flush()
			}
			continue
		}

		redisMsg, ok := msg.(*goredis.Message)
		if !ok {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(redisMsg.Payload), &event); err != nil {
			continue
		}
		eventID, frame, ok := EventFrame(event)
		if !ok || eventID <= s.lastEventID {
			continue
		}
		s.lastEventID = eventID
		if _, err := io.WriteString(w, frame); err != nil {
			return err
		}
		if flush != nil {
			flush()
		}
	}
}

func pickField(item map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := item[key]
		if ok && value != nil {
			return value
		}
	}
	return nil
}
