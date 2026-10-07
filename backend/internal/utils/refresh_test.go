package utils

import (
	"testing"
)

func TestGenerateRefreshToken(t *testing.T) {
	token, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(token) == 0 {
		t.Error("expected non-empty token")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-refresh-token"
	hashed := HashToken(token)
	if len(hashed) != 64 {
		t.Errorf("expected SHA-256 hex string of length 64, got %d", len(hashed))
	}
}

func BenchmarkGenerateRefreshToken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateRefreshToken()
	}
}

func BenchmarkHashToken(b *testing.B) {
	token := "test-refresh-token"
	for i := 0; i < b.N; i++ {
		_ = HashToken(token)
	}
}
