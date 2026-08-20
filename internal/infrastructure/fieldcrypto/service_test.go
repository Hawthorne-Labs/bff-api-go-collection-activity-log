package fieldcrypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

const (
	testActiveKID = "k1"
	testOldKID    = "k0"
)

func testProvider(t *testing.T) *EnvKeyProvider {
	t.Helper()
	provider, err := NewEnvKeyProvider(map[string][]byte{
		testOldKID: bytesRepeat(32, 2),
		testActiveKID: bytesRepeat(32, 1),
	}, testActiveKID)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	return provider
}

func testContext(kid string) CryptoContext {
	return CryptoContext{KID: kid, Endpoint: "/t", Method: "POST", Direction: "encrypt"}
}

func TestValueRoundtripPreservesType(t *testing.T) {
	service := NewFieldCryptoService(testProvider(t))
	ctx := testContext(testActiveKID)
	values := []any{"Jose", "", float64(0), float64(15000), 1.5, true, false, nil}
	for _, value := range values {
		sealed, err := service.EncryptValue(value, ctx)
		if err != nil || !strings.HasPrefix(sealed, "enc:v1:") {
			t.Fatalf("encrypt %#v: %v %q", value, err, sealed)
		}
		restored, err := service.DecryptValue(sealed, ctx)
		if err != nil {
			t.Fatalf("decrypt %#v: %v", value, err)
		}
		if restored != value {
			t.Fatalf("restored %#v got %#v", value, restored)
		}
	}
}

func TestNestedObjectStructureAndTypes(t *testing.T) {
	service := NewFieldCryptoService(testProvider(t))
	ctx := testContext(testActiveKID)
	payload := map[string]any{
		"name":   "Jose",
		"amount": float64(15000),
		"active": true,
		"address": map[string]any{
			"city": "TGU",
			"zip":  nil,
		},
	}
	sealed, err := service.EncryptJSON(payload, ctx)
	if err != nil {
		t.Fatalf("encrypt json: %v", err)
	}
	sealedMap := sealed.(map[string]any)
	if _, ok := sealedMap["name"].(string); !ok || !strings.HasPrefix(sealedMap["name"].(string), "enc:v1:") {
		t.Fatal("expected encrypted scalar")
	}
	plain, err := service.DecryptJSON(sealed, ctx)
	if err != nil {
		t.Fatalf("decrypt json: %v", err)
	}
	plainMap := plain.(map[string]any)
	if plainMap["name"] != "Jose" {
		t.Fatalf("unexpected plain: %#v", plainMap)
	}
}

func TestDecryptJSONRejectsPlaintextScalar(t *testing.T) {
	service := NewFieldCryptoService(testProvider(t))
	ctx := testContext(testActiveKID)
	if _, err := service.DecryptJSON(map[string]any{"name": "Jose"}, ctx); err == nil {
		t.Fatal("expected plaintext rejected")
	}
}

func TestKeyRotationDecryptsOldKid(t *testing.T) {
	service := NewFieldCryptoService(testProvider(t))
	oldCtx := testContext(testOldKID)
	sealed, err := service.EncryptValue("previous", oldCtx)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !strings.HasPrefix(sealed, "enc:v1:k0:") {
		t.Fatalf("unexpected kid in envelope: %q", sealed)
	}
	restored, err := service.DecryptValue(sealed, oldCtx)
	if err != nil || restored != "previous" {
		t.Fatalf("decrypt old kid: %v %#v", err, restored)
	}
}

func TestValueCodecRoundtrip(t *testing.T) {
	values := []any{"Jose", float64(0), float64(15000), true, false, nil}
	for _, value := range values {
		raw, err := EncodeValue(value)
		if err != nil {
			t.Fatalf("encode %#v: %v", value, err)
		}
		restored, err := DecodeValue(raw)
		if err != nil {
			t.Fatalf("decode %#v: %v", value, err)
		}
		if restored != value {
			t.Fatalf("roundtrip %#v got %#v", value, restored)
		}
	}
}

func TestAADUsesEncV1Kid(t *testing.T) {
	got := string(aad("kid-1"))
	want := "enc:v1:kid-1"
	if got != want {
		t.Fatalf("aad=%q want %q", got, want)
	}
}

func TestEnvKeyProviderFromEnvRoundtrip(t *testing.T) {
	t.Setenv("CRYPTO_KEYS", "k1:"+base64.RawURLEncoding.EncodeToString(bytesRepeat(32, 9)))
	t.Setenv("CRYPTO_ACTIVE_KID", "k1")
	provider, err := EnvKeyProviderFromEnv()
	if err != nil {
		t.Fatalf("from env: %v", err)
	}
	if provider.ActiveKID() != "k1" {
		t.Fatal("unexpected active kid")
	}
}
