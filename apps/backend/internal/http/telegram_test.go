package httpapi

import (
	"strings"
	"testing"

	"github.com/local/printforge/apps/backend/internal/config"
)

func TestTelegramTokenEncryptionRoundTrip(t *testing.T) {
	server := &Server{cfg: config.Config{JWTSecret: "a-test-secret-that-is-longer-than-thirty-two-characters"}}
	token := "123456789:AAabcdefghijklmnopqrstuvwxyz0123456789"
	encrypted, err := server.encryptTelegramToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encrypted), token) {
		t.Fatal("encrypted token contains plaintext")
	}
	decrypted, err := server.decryptTelegramToken(encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != token {
		t.Fatalf("decrypted token = %q, want %q", decrypted, token)
	}
}

func TestTrackingCodeIsTelegramFriendly(t *testing.T) {
	for range 100 {
		code, err := newTrackingCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 10 || strings.ContainsAny(code, "01IO") {
			t.Fatalf("tracking code %q is not user friendly", code)
		}
		if extracted := extractTrackingCode("/start " + strings.ToLower(code)); extracted != code {
			t.Fatalf("extractTrackingCode() = %q, want %q", extracted, code)
		}
	}
}
