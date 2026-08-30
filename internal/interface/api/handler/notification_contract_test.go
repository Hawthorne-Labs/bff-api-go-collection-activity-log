package handler

import "testing"

func TestMapNotificationItemAcceptsCorePascalCase(t *testing.T) {
	got := mapNotificationItem(map[string]any{
		"ID":               "n-1",
		"EventID":          float64(9),
		"NotificationType": "ESCALATION",
		"Severity":         "IMPORTANT",
		"Destination":      "user@dev.io",
		"CreatedAt":        "2026-08-19T00:00:00Z",
		"ReadAt":           nil,
	})
	if got["id"] != "n-1" || got["type"] != "ESCALATION" || got["event_id"] != 9 {
		t.Fatalf("unexpected mapped item: %#v", got)
	}
}

func TestMapNotificationDetailPassesClientName(t *testing.T) {
	got := mapNotificationDetail(map[string]any{
		"ID":         "n-2",
		"ClientName": "Cliente Demo",
		"Severity":   "INFO",
		"CreatedAt":  "2026-08-30T00:00:00Z",
	})
	if got["client_name"] != "Cliente Demo" {
		t.Fatalf("expected client_name from ClientName, got %#v", got["client_name"])
	}
	gotSnake := mapNotificationDetail(map[string]any{
		"id":          "n-3",
		"client_name": "Otro Cliente",
		"severity":    "INFO",
		"created_at":  "2026-08-30T00:00:00Z",
	})
	if gotSnake["client_name"] != "Otro Cliente" {
		t.Fatalf("expected client_name snake_case, got %#v", gotSnake["client_name"])
	}
}

func TestNotificationCursorRoundTrip(t *testing.T) {
	codec := newNotificationCursorCodec("dev-notification-cursor-secret")
	cursor := codec.encode("2026-08-19T00:00:00Z", "n-1", "user@dev.io", "all", "", "", "")
	createdAt, id, err := codec.decode(cursor, "user@dev.io", "all", "", "", "")
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if createdAt != "2026-08-19T00:00:00Z" || id != "n-1" {
		t.Fatalf("unexpected position %s %s", createdAt, id)
	}
}
