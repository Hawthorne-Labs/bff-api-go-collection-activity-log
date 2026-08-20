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
