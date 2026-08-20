package middleware

import "testing"

func TestCryptoSessionIsNotPublic(t *testing.T) {
	if isPublicPath("/api/v1/collections/crypto-session") {
		t.Fatal("crypto-session handshake must authenticate like Python require_scope")
	}
}

func TestHealthIsPublic(t *testing.T) {
	if !isPublicPath("/health") {
		t.Fatal("health must stay public")
	}
}
