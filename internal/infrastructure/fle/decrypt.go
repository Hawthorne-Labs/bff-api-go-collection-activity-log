package fle

import (
	"encoding/json"
	"fmt"
)

// DecryptRequestBody decrypts SP-FLE encrypted fields via crypto-bff and returns plaintext payload.
func DecryptRequestBody(
	body map[string]any,
	policy EndpointFieldPolicy,
	decrypt func(fields map[string]string) (map[string]string, error),
) (map[string]any, error) {
	encryptedPaths := append([]string{}, policy.RequiredEncryptedFields...)
	encryptedPaths = append(encryptedPaths, policy.OptionalEncryptedFields...)

	encryptedFields := make(map[string]string, len(encryptedPaths))
	for _, fieldPath := range encryptedPaths {
		value, ok := body[fieldPath]
		if !ok || value == nil {
			if contains(policy.RequiredEncryptedFields, fieldPath) {
				return nil, fmt.Errorf("missing required encrypted field %s", fieldPath)
			}
			continue
		}
		text, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("field %s is not a string", fieldPath)
		}
		encryptedFields[fieldPath] = text
	}

	if !policy.AllowUnknownFields {
		allowed := make(map[string]struct{}, len(encryptedPaths))
		for _, fieldPath := range encryptedPaths {
			allowed[fieldPath] = struct{}{}
		}
		for key := range body {
			if _, ok := allowed[key]; !ok {
				return nil, fmt.Errorf("unexpected fields in body")
			}
		}
	}

	plaintextFlat, err := decrypt(encryptedFields)
	if err != nil {
		return nil, err
	}

	result := cloneMap(body)
	for fieldPath, value := range plaintextFlat {
		result[fieldPath] = value
	}
	return result, nil
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func cloneMap(input map[string]any) map[string]any {
	raw, _ := json.Marshal(input)
	out := make(map[string]any)
	_ = json.Unmarshal(raw, &out)
	return out
}
