package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func pickNotificationField(item map[string]any, keys ...string) any {
	for _, key := range keys {
		value, ok := item[key]
		if !ok || value == nil {
			continue
		}
		if text, isString := value.(string); isString && text == "" {
			continue
		}
		return value
	}
	return nil
}

func notificationString(item map[string]any, keys ...string) string {
	value := pickNotificationField(item, keys...)
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func notificationStringOrNil(item map[string]any, keys ...string) any {
	value := notificationString(item, keys...)
	if value == "" {
		return nil
	}
	return value
}

func notificationInt(item map[string]any, keys ...string) int {
	value := pickNotificationField(item, keys...)
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	case json.Number:
		n, _ := typed.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(typed)
		return n
	default:
		return 0
	}
}

func notificationMetadata(item map[string]any) map[string]any {
	raw := pickNotificationField(item, "metadata", "Metadata")
	if mapped, ok := raw.(map[string]any); ok && mapped != nil {
		return mapped
	}
	return map[string]any{}
}

func mapNotificationItem(item map[string]any) map[string]any {
	readAt := notificationStringOrNil(item, "read_at", "ReadAt")
	return map[string]any{
		"id":          notificationString(item, "id", "ID"),
		"event_id":    notificationInt(item, "event_id", "EventID"),
		"type":        notificationString(item, "type", "NotificationType", "notification_type"),
		"severity":    notificationString(item, "severity", "Severity"),
		"destination": notificationString(item, "destination", "Destination"),
		"metadata":    notificationMetadata(item),
		"created_at":  notificationString(item, "created_at", "CreatedAt"),
		"read_at":     readAt,
	}
}

func mapNotificationDetail(item map[string]any) map[string]any {
	return map[string]any{
		"id":             notificationString(item, "id", "ID"),
		"type":           notificationString(item, "type", "NotificationType", "notification_type"),
		"severity":       notificationString(item, "severity", "Severity"),
		"destination":    notificationString(item, "destination", "Destination"),
		"actor_name":     notificationStringOrNil(item, "actor_name", "ActorName"),
		"actor_role":     notificationStringOrNil(item, "actor_role", "ActorRole"),
		"client_name":    notificationStringOrNil(item, "client_name", "ClientName"),
		"loan_reference": notificationStringOrNil(item, "loan_reference", "LoanReference"),
		"description":    notificationStringOrNil(item, "description", "Description"),
		"result":         notificationStringOrNil(item, "result", "Result"),
		"status":         notificationStringOrNil(item, "status", "Status"),
		"created_at":     notificationString(item, "created_at", "CreatedAt"),
		"read_at":        notificationStringOrNil(item, "read_at", "ReadAt"),
	}
}

func asObjectMap(value any) map[string]any {
	if mapped, ok := value.(map[string]any); ok {
		return mapped
	}
	return nil
}

func asObjectSlice(value any) []map[string]any {
	switch raw := value.(type) {
	case []map[string]any:
		return raw
	case []any:
		items := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if mapped := asObjectMap(item); mapped != nil {
				items = append(items, mapped)
			}
		}
		return items
	default:
		return nil
	}
}

func notificationItems(result map[string]any) []map[string]any {
	if result == nil {
		return nil
	}
	if items, ok := result["items"]; ok {
		return asObjectSlice(items)
	}
	return asObjectSlice(result["Items"])
}

func notificationCount(value map[string]any) int {
	if value == nil {
		return 0
	}
	if nested := asObjectMap(value["data"]); nested != nil {
		if _, ok := nested["count"]; ok {
			return notificationInt(nested, "count", "Count")
		}
		if _, ok := nested["Count"]; ok {
			return notificationInt(nested, "count", "Count")
		}
	}
	return notificationInt(value, "count", "Count")
}
