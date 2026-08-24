package usecases

import (
	"context"
	"fmt"
	"strings"

	"github.com/hawthorne/bff-api-go-collection-activity-log/internal/domain"
)

const escalationRequiresEffectiveGestion = 5015

var nonManagementActivityTypes = map[string]struct{}{
	"escalation": {},
}

func (u *EscalationsUsecase) prepareCreatePayload(
	ctx context.Context,
	payload map[string]any,
	traceID, tenantID, requestID, correlationID, traceparent, userEmail string,
) (map[string]any, error) {
	out := mapCreateEscalationPayload(payload)
	if reason, ok := stringField(out, "reason"); ok && reason != "" {
		encrypted, err := u.crypto.EncryptFields(
			ctx,
			"POST",
			"/api/v1/collections/escalations",
			map[string]string{"notes": reason},
			tenantID,
			requestID,
			correlationID,
			traceparent,
		)
		if err != nil {
			return nil, fmt.Errorf("encrypt escalation notes: %w", err)
		}
		if cipher, ok := encrypted["notes"]; ok {
			out["reason"] = cipher
		}
	}
	if _, ok := stringField(out, "last_effective_activity_id"); ok {
		return out, nil
	}
	// anti-regresion: BUG-0002 ver handoffs/regressions.md (no revertir sin leer)
	activityID, err := u.requireLastEffectiveGestion(ctx, out, traceID, tenantID, userEmail)
	if err != nil {
		return nil, err
	}
	out["last_effective_activity_id"] = activityID
	return out, nil
}

func (u *EscalationsUsecase) requireLastEffectiveGestion(
	ctx context.Context,
	payload map[string]any,
	traceID, tenantID, userEmail string,
) (string, error) {
	loanID, ok := stringField(payload, "loan_id")
	if !ok || loanID == "" {
		return "", domain.NewBusinessError(domain.EscalationCreateFailed, "Se requiere loan_id para escalar.")
	}
	page, err := u.core.ListActivities(ctx, traceID, tenantID, loanID, nil, "", "", "", "", userEmail, 20, 0)
	if err != nil {
		return "", err
	}
	for _, item := range activityItems(page) {
		if !isManagementActivity(item) || !isEffectiveActivity(item) {
			continue
		}
		if id := activityID(item); id != "" {
			return id, nil
		}
	}
	return "", domain.NewBusinessError(
		escalationRequiresEffectiveGestion,
		"Debe existir una gestion efectiva antes de escalar el caso.",
	)
}

func activityItems(page map[string]any) []map[string]any {
	raw, ok := page["items"].([]any)
	if !ok {
		return nil
	}
	items := make([]map[string]any, 0, len(raw))
	for _, value := range raw {
		mapped, ok := value.(map[string]any)
		if ok {
			items = append(items, mapped)
		}
	}
	return items
}

func activityID(item map[string]any) string {
	if value, ok := stringField(item, "id"); ok {
		return value
	}
	return strings.TrimSpace(fmt.Sprint(item["ID"]))
}

func isManagementActivity(item map[string]any) bool {
	activityType := strings.ToLower(strings.TrimSpace(fmt.Sprint(
		pickActivityField(item, "activity_type", "activityType", "ActivityType"),
	)))
	agentName := strings.ToLower(strings.TrimSpace(fmt.Sprint(
		pickActivityField(item, "agent_name", "agentName", "AgentName"),
	)))
	if _, blocked := nonManagementActivityTypes[activityType]; blocked {
		return false
	}
	return agentName != "sistema"
}

func isEffectiveActivity(item map[string]any) bool {
	return isTruthyAnswered(pickActivityField(item, "answered", "Answered"))
}

func isTruthyAnswered(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		normalized := strings.ToLower(strings.TrimSpace(typed))
		return normalized == "true" || normalized == "1" || normalized == "t"
	case float64:
		return typed == 1
	case int:
		return typed == 1
	case int64:
		return typed == 1
	default:
		return false
	}
}

func pickActivityField(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok && value != nil {
			return value
		}
	}
	return nil
}
