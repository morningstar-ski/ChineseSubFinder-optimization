package random_auth_key

import (
	"strings"
	"testing"
)

func TestRandomAuthKey_GetAuthKeyRejectsShortOrEmptyKeys(t *testing.T) {
	t.Run("empty base key", func(t *testing.T) {
		_, err := NewRandomAuthKey(5, AuthKey{}).GetAuthKey()
		if err == nil {
			t.Fatal("expected error for empty auth key")
		}
		if !strings.Contains(err.Error(), "base key is not set") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("short aes key", func(t *testing.T) {
		_, err := NewRandomAuthKey(5, AuthKey{
			BaseKey:  BaseKey,
			AESKey16: "short",
			AESIv16:  AESIv16,
		}).GetAuthKey()
		if err == nil {
			t.Fatal("expected error for short AES key")
		}
		if !strings.Contains(err.Error(), "AESKey16 is not set") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
