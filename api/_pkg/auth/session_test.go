package auth

import (
	"testing"
)

func TestGenerateSessionToken(t *testing.T) {
	plaintext, hash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() error: %v", err)
	}

	if plaintext == "" {
		t.Fatal("plaintext token should not be empty")
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if plaintext == hash {
		t.Fatal("plaintext and hash should differ")
	}
}

func TestGenerateSessionTokenUniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		plaintext, _, err := GenerateSessionToken()
		if err != nil {
			t.Fatalf("GenerateSessionToken() error: %v", err)
		}
		if tokens[plaintext] {
			t.Fatalf("duplicate token generated on iteration %d", i)
		}
		tokens[plaintext] = true
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	token := "test-token-value"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Fatalf("HashToken should be deterministic: %q != %q", hash1, hash2)
	}
}

func TestHashTokenDifferentInputs(t *testing.T) {
	hash1 := HashToken("token-a")
	hash2 := HashToken("token-b")

	if hash1 == hash2 {
		t.Fatal("different tokens should produce different hashes")
	}
}

func TestHashTokenRoundtrip(t *testing.T) {
	plaintext, expectedHash, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken() error: %v", err)
	}

	actualHash := HashToken(plaintext)
	if actualHash != expectedHash {
		t.Fatalf("hash mismatch: HashToken(plaintext) = %q, expected %q", actualHash, expectedHash)
	}
}
