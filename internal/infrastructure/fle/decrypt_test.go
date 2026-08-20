package fle

import "testing"

func TestContactsPolicy(t *testing.T) {
	policy, ok := PolicyFor("POST", "/api/v1/contacts")
	if !ok {
		t.Fatal("expected contacts policy")
	}
	if len(policy.RequiredEncryptedFields) != 3 {
		t.Fatalf("expected 3 required fields, got %d", len(policy.RequiredEncryptedFields))
	}
}

func TestDecryptRequestBodyRejectsUnknownFields(t *testing.T) {
	policy, _ := PolicyFor("POST", "/api/v1/contacts")
	_, err := DecryptRequestBody(
		map[string]any{"name": "enc", "email": "enc", "message": "enc", "extra": "x"},
		policy,
		func(fields map[string]string) (map[string]string, error) {
			out := make(map[string]string, len(fields))
			for k, v := range fields {
				out[k] = "plain-" + v
			}
			return out, nil
		},
	)
	if err == nil {
		t.Fatal("expected unknown field error")
	}
}
