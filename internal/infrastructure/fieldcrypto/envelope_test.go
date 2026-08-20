package fieldcrypto

import "testing"

func TestEnvelopeFormatAndParseRoundtrip(t *testing.T) {
	e := Envelope{
		KID:        "k1",
		Nonce:      make([]byte, 12),
		Ciphertext: []byte("abc"),
		Tag:        bytesRepeat(16, 1),
	}
	parsed, err := ParseEnvelope(e.ToString())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.KID != e.KID || string(parsed.Ciphertext) != string(e.Ciphertext) {
		t.Fatalf("unexpected parsed envelope: %+v", parsed)
	}
}

func TestEnvelopeRejectsMalformed(t *testing.T) {
	cases := []any{
		"enc:v1:k1:AAAA:BBBB",
		"enc:v1:k1:AAAA:BBBB:CCCC:DD",
		"x:v1:k1:AAAA:BBBB:CCCC",
		"enc:v1::AAAA:BBBB:CCCC",
		123,
	}
	for _, bad := range cases {
		if _, err := ParseEnvelope(bad); err == nil {
			t.Fatalf("expected error for %#v", bad)
		}
	}
}

func TestIsEnvelope(t *testing.T) {
	if !IsEnvelope("enc:v1:k1:a:b:c") {
		t.Fatal("expected envelope")
	}
	if IsEnvelope("plain") || IsEnvelope(123) {
		t.Fatal("expected non-envelope")
	}
}

func bytesRepeat(n int, b byte) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = b
	}
	return out
}
