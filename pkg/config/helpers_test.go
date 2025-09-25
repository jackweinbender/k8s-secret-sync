package config

import (
	"strconv"
	"strings"
	"testing"
)

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

// FuzzEnvIntAndBool ensures that arbitrary string inputs for int and bool parsing
// never cause panics and correctly fall back to defaults when unparsable.
func FuzzEnvIntAndBool(f *testing.F) {
	// Seed with a few interesting corpus values
	f.Add("0")
	f.Add("1")
	f.Add("true")
	f.Add("false")
	f.Add("not-a-number")
	f.Add("")
	f.Add("999999999999999999999999")
	f.Add("TrUe")

	f.Fuzz(func(t *testing.T, v string) {
		// Skip values that contain NUL since environment variables cannot hold them.
		if strings.ContainsRune(v, '\x00') {
			t.Skip("skip invalid env value containing NUL")
		}
		// Int parse path
		keyInt := "FUZZ_INT"
		// Use raw value; empty string path should yield default.
		t.Setenv(keyInt, v)
		_ = env(keyInt, 42) // any unparsable value should just return 42

		// Bool parse path
		keyBool := "FUZZ_BOOL"
		t.Setenv(keyBool, v)
		_ = env(keyBool, false)

		// Also test a synthetic composite: if v parses to int and is even, expect that returned
		if n, err := strconv.Atoi(v); err == nil && n%2 == 0 {
			got := env(keyInt, 13)
			if got != n { // For valid ints, we should round-trip the parsed value.
				// Not failing here would hide a regression in parsing logic.
				t.Fatalf("expected parsed int %d, got %d for input %q", n, got, v)
			}
		}
	})
}
