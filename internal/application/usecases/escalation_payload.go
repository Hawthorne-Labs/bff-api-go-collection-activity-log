package usecases

import "strings"

var escalationStatusAliases = map[string]string{
	"pending":           "PENDING",
	"pendiente":         "PENDING",
	"reviewed":          "REVIEWED",
	"in_progress":       "REVIEWED",
	"en_proceso":        "REVIEWED",
	"closed":            "CLOSED",
	"resolved":          "CLOSED",
	"resuelto":          "CLOSED",
	"approved":          "APPROVED",
	"approve":           "APPROVED",
	"aprobado":          "APPROVED",
	"rejected":          "REJECTED",
	"reject":            "REJECTED",
	"rechazado":         "REJECTED",
	"changes_requested": "CHANGES_REQUESTED",
	"request_changes":   "CHANGES_REQUESTED",
	"pedir_cambios":     "CHANGES_REQUESTED",
}

func mapCreateEscalationPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return payload
	}
	out := copyAnyMap(payload)
	if notes, ok := stringField(out, "notes"); ok && notes != "" {
		if _, hasReason := stringField(out, "reason"); !hasReason {
			out["reason"] = notes
		}
	}
	delete(out, "notes")
	return out
}

func mapEscalationStatusPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return payload
	}
	out := copyAnyMap(payload)
	raw, _ := stringField(out, "escalation_status")
	if raw == "" {
		raw, _ = stringField(out, "status")
	}
	if raw != "" {
		out["status"] = normalizeEscalationStatus(raw)
	}
	delete(out, "escalation_status")
	return out
}

func mapEscalationDecisionPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return payload
	}
	out := copyAnyMap(payload)
	if raw, ok := stringField(out, "status"); ok {
		out["status"] = normalizeEscalationStatus(raw)
	}
	if reason, ok := stringField(out, "reason"); ok {
		trimmed := strings.TrimSpace(reason)
		if trimmed == "" {
			out["reason"] = nil
		} else {
			out["reason"] = trimmed
		}
	}
	return out
}

func normalizeEscalationStatus(value string) string {
	key := strings.ToLower(strings.TrimSpace(value))
	key = strings.ReplaceAll(key, "-", "_")
	key = strings.ReplaceAll(key, " ", "_")
	if mapped, ok := escalationStatusAliases[key]; ok {
		return mapped
	}
	return strings.ToUpper(strings.TrimSpace(value))
}

func copyAnyMap(payload map[string]any) map[string]any {
	out := make(map[string]any, len(payload))
	for key, value := range payload {
		out[key] = value
	}
	return out
}

func stringField(payload map[string]any, key string) (string, bool) {
	value, ok := payload[key]
	if !ok || value == nil {
		return "", false
	}
	text, ok := value.(string)
	if !ok {
		return "", false
	}
	return text, true
}
