package service

import (
	"testing"
)

// TestHashAndVerifyPassword: Hash then verify succeeds.
func TestHashAndVerifyPassword(t *testing.T) {
	password := "correct-horse-battery-staple"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty hash")
	}

	if !VerifyPassword(password, hash) {
		t.Error("VerifyPassword returned false for correct password")
	}
}

// TestVerifyPassword_WrongPassword: Wrong password fails.
func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if VerifyPassword("wrong-password", hash) {
		t.Error("VerifyPassword returned true for wrong password")
	}
}

// TestGenerateToken: Token is 64 hex chars.
func TestGenerateToken(t *testing.T) {
	token, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	// tokenBytes = 32, hex-encoded → 64 characters
	if len(token) != 64 {
		t.Errorf("expected token length 64, got %d", len(token))
	}
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains non-hex character: %q", c)
		}
	}
}

// TestGenerateToken_Unique: Two tokens are different.
func TestGenerateToken_Unique(t *testing.T) {
	t1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	t2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken returned error: %v", err)
	}
	if t1 == t2 {
		t.Error("two generated tokens are identical — not unique")
	}
}
