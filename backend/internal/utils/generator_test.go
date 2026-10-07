package utils

import (
	"testing"
)

func TestGenerateShortCode(t *testing.T) {
	code := GenerateShortCode()
	if len(code) != 8 {
		t.Errorf("expected code length 8, got %d", len(code))
	}
}

func BenchmarkGenerateShortCode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateShortCode()
	}
}
