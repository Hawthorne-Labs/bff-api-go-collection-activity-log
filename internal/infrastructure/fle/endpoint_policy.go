package fle

import "strings"

// EndpointFieldPolicy describes SP-FLE field encryption rules for a route.
type EndpointFieldPolicy struct {
	Method                   string
	Path                     string
	RequiredEncryptedFields  []string
	OptionalEncryptedFields  []string
	AllowUnknownFields       bool
}

var contactsPolicy = EndpointFieldPolicy{
	Method:                  "POST",
	Path:                    "/api/v1/contacts",
	RequiredEncryptedFields: []string{"name", "email", "message"},
	OptionalEncryptedFields: []string{},
	AllowUnknownFields:      false,
}

var endpointPolicies = map[string]EndpointFieldPolicy{
	"POST:/api/v1/contacts": contactsPolicy,
}

// PolicyFor returns the SP-FLE policy for a method/path pair.
func PolicyFor(method, path string) (EndpointFieldPolicy, bool) {
	key := strings.ToUpper(strings.TrimSpace(method)) + ":" + path
	policy, ok := endpointPolicies[key]
	return policy, ok
}
