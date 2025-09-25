package config

import "testing"

func TestEnvBasicTypes(t *testing.T) {
	// String present
	t.Setenv("KSS_STR_VAL", "hello")
	if got := env("KSS_STR_VAL", "default"); got != "hello" {
		t.Fatalf("expected 'hello', got %q", got)
	}

	// Int present
	t.Setenv("KSS_INT_VAL", "42")
	if got := env("KSS_INT_VAL", 0); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}

	// Bool present
	t.Setenv("KSS_BOOL_VAL", "true")
	if got := env("KSS_BOOL_VAL", false); !got {
		t.Fatalf("expected true, got %v", got)
	}
}

func TestEnvFallbackAndParsing(t *testing.T) {
	if got := env("KSS_MISSING_STR", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	if got := env("KSS_MISSING_INT", 7); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
	if got := env("KSS_MISSING_BOOL", true); got != true {
		t.Fatalf("expected true, got %v", got)
	}

	// Bad parses
	t.Setenv("KSS_BAD_INT", "not-an-int")
	if got := env("KSS_BAD_INT", 9); got != 9 {
		t.Fatalf("expected 9 (default), got %d", got)
	}
	// Bool
	t.Setenv("KSS_BAD_BOOL", "not-bool")
	if got := env("KSS_BAD_BOOL", false); got != false {
		t.Fatalf("expected false (default), got %v", got)
	}
}

func TestEnvEmptyStringBehavior(t *testing.T) {
	// Empty string should act as unset (implementation ignores empty value)
	t.Setenv("KSS_EMPTY", "")
	if got := env("KSS_EMPTY", "default"); got != "default" {
		t.Fatalf("expected default for empty value, got %q", got)
	}
}
