package notifications

import "testing"

func TestEventFrameAcceptsPascalCase(t *testing.T) {
	id, frame, ok := EventFrame(map[string]any{"EventID": float64(7)})
	if !ok || id != 7 || frame == "" {
		t.Fatalf("unexpected frame: ok=%v id=%d frame=%q", ok, id, frame)
	}
}

func TestChannelIsDeterministic(t *testing.T) {
	a := Channel("COGASA", "User@Example.com")
	b := Channel("cogasa", "user@example.com")
	if a != b {
		t.Fatalf("expected stable channel, got %q vs %q", a, b)
	}
}
