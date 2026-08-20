package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
)

var errInvalidNotificationCursor = errors.New("invalid notification cursor")

type notificationCursorCodec struct {
	secret []byte
}

func newNotificationCursorCodec(secret string) *notificationCursorCodec {
	if secret == "" {
		secret = "dev-notification-cursor-secret"
	}
	return &notificationCursorCodec{secret: []byte(secret)}
}

func (c *notificationCursorCodec) encode(createdAt, notificationID, recipient, state, severity, fromDate, toDate string) string {
	payload, _ := json.Marshal(map[string]string{
		"created_at":      createdAt,
		"notification_id": notificationID,
		"scope":           notificationCursorScope(recipient, state, severity, fromDate, toDate),
	})
	encoded := notificationCursorB64(payload)
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(encoded))
	return encoded + "." + notificationCursorB64(mac.Sum(nil))
}

func (c *notificationCursorCodec) decode(cursor, recipient, state, severity, fromDate, toDate string) (string, string, error) {
	encoded, signature, ok := strings.Cut(cursor, ".")
	if !ok || encoded == "" || signature == "" {
		return "", "", errInvalidNotificationCursor
	}
	supplied, err := notificationCursorDecodeB64(signature)
	if err != nil {
		return "", "", errInvalidNotificationCursor
	}
	mac := hmac.New(sha256.New, c.secret)
	_, _ = mac.Write([]byte(encoded))
	if !hmac.Equal(mac.Sum(nil), supplied) {
		return "", "", errInvalidNotificationCursor
	}
	raw, err := notificationCursorDecodeB64(encoded)
	if err != nil {
		return "", "", errInvalidNotificationCursor
	}
	var payload map[string]string
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", "", errInvalidNotificationCursor
	}
	if payload["scope"] != notificationCursorScope(recipient, state, severity, fromDate, toDate) {
		return "", "", errInvalidNotificationCursor
	}
	createdAt := payload["created_at"]
	notificationID := payload["notification_id"]
	if createdAt == "" || notificationID == "" {
		return "", "", errInvalidNotificationCursor
	}
	return createdAt, notificationID, nil
}

func notificationCursorScope(recipient, state, severity, fromDate, toDate string) string {
	value := strings.ToLower(strings.TrimSpace(recipient)) + "\x00" + state + "\x00" + severity + "\x00" + fromDate + "\x00" + toDate
	sum := sha256.Sum256([]byte(value))
	return hexEncode(sum[:])
}

func notificationCursorB64(value []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(value), "=")
}

func notificationCursorDecodeB64(value string) ([]byte, error) {
	switch len(value) % 4 {
	case 2:
		value += "=="
	case 3:
		value += "="
	}
	return base64.URLEncoding.DecodeString(value)
}

func hexEncode(value []byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, len(value)*2)
	for i, b := range value {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out)
}
