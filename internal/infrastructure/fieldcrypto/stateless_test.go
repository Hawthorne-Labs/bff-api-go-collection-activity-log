package fieldcrypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

const (
	statelessSecret = "stateless-signing-secret-32-bytes-min!!"
	statelessIssuer = "hawthorne-bff-test"
	testTenant      = "tenant-digest-aabbccddeeff0011"
	testAuth        = "Bearer user-access-token-abc123xyz"
)

func testKekRing(t *testing.T) *KekRing {
	t.Helper()
	key := bytesRepeat(32, 'A')
	ring, err := NewKekRing(map[string][]byte{"kek-1": key}, "kek-1")
	if err != nil {
		t.Fatalf("kek ring: %v", err)
	}
	return ring
}

func testStatelessManager(t *testing.T) *StatelessCryptoSessionManager {
	t.Helper()
	return NewStatelessCryptoSessionManager(testKekRing(t), statelessSecret, statelessIssuer, "activity-log", 300)
}

func testClientPublicB64(t *testing.T) string {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("client key: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes())
}

func testHandshake(t *testing.T, mgr *StatelessCryptoSessionManager) *HandshakeResult {
	t.Helper()
	ath, err := HashAccessToken(testAuth)
	if err != nil {
		t.Fatalf("hash token: %v", err)
	}
	res, err := mgr.Handshake(testClientPublicB64(t), "user-1", "collections:read", testTenant, ath)
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	return res
}

func TestStatelessHandshakeAndResolveRoundtrip(t *testing.T) {
	mgrA := testStatelessManager(t)
	res := testHandshake(t, mgrA)
	mgrB := testStatelessManager(t)
	verified, err := mgrB.Resolve(res.CryptoAccessToken, res.CryptoSessionID, testTenant, testAuth, "activity-log")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(verified.SessionKey) != 32 {
		t.Fatalf("unexpected session key length: %d", len(verified.SessionKey))
	}

	provider, err := NewFixedSessionKeyProvider(verified.SessionID, verified.SessionKey)
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	service := NewFieldCryptoService(provider)
	ctx := CryptoContext{KID: verified.SessionID, Method: "POST", Endpoint: "/t"}
	sealed, err := service.EncryptValue("Jose", ctx)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plain, err := service.DecryptValue(sealed, ctx)
	if err != nil || plain != "Jose" {
		t.Fatalf("decrypt: %v %#v", err, plain)
	}
}

func TestStatelessResolveRejectsOtherTenantDigest(t *testing.T) {
	mgr := testStatelessManager(t)
	res := testHandshake(t, mgr)
	if _, err := mgr.Resolve(res.CryptoAccessToken, res.CryptoSessionID, "other-digest-0123456789abcd", testAuth, "activity-log"); err == nil {
		t.Fatal("expected session invalid")
	}
}

func TestHashAccessToken(t *testing.T) {
	hash, err := HashAccessToken(testAuth)
	if err != nil || len(hash) != 64 || strings.ContainsAny(hash, "ABCDEF") {
		t.Fatalf("unexpected hash: %q err=%v", hash, err)
	}
}

func TestFixedSessionKeyProviderRejectsWrongKid(t *testing.T) {
	provider, err := NewFixedSessionKeyProvider("sid", bytesRepeat(32, 3))
	if err != nil {
		t.Fatalf("provider: %v", err)
	}
	if _, err := provider.KeyFor("other"); err == nil {
		t.Fatal("expected unknown kid")
	}
}
