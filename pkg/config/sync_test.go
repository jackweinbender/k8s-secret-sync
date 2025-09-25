package config

import (
	"testing"

	"k8s.io/client-go/kubernetes"
)

func TestNewDefaults(t *testing.T) {
	cs := &kubernetes.Clientset{}
	cfg := New(cs)
	if cfg == nil {
		t.Fatalf("expected config, got nil")
	}
	if cfg.Clientset != cs {
		t.Errorf("expected provided clientset to be set")
	}

	cases := []struct{ field, got, want string }{
		{"ProviderName", cfg.Annotations.ProviderName, "k8s-secret-sync.weinbender.io/provider-name"},
		{"ProviderRef", cfg.Annotations.ProviderRef, "k8s-secret-sync.weinbender.io/provider-ref"},
		{"SecretKey", cfg.Annotations.SecretKey, "k8s-secret-sync.weinbender.io/secret-key"},
		{"DefaultSecretDataKey", cfg.DefaultSecretDataKey, "value"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %s, want %s", c.field, c.got, c.want)
		}
	}
	if cfg.PollInterval != 300 {
		t.Errorf("PollInterval = %d, want 300", cfg.PollInterval)
	}
}

func TestNewOverrides(t *testing.T) {
	// Set overrides
	t.Setenv("KSS_SECRET_ANNOTATION_KEY_PROVIDER_NAME", "custom/provider")
	t.Setenv("KSS_SECRET_ANNOTATION_KEY_PROVIDER_REF", "custom/ref")
	t.Setenv("KSS_SECRET_ANNOTATION_KEY_SECRET_KEY", "custom/key")
	t.Setenv("KSS_DEFAULT_SECRET_DATA_KEY", "customval")
	t.Setenv("KSS_POLL_INTERVAL", "123")

	cfg := New(&kubernetes.Clientset{})
	if cfg.Annotations.ProviderName != "custom/provider" {
		t.Errorf("ProviderName = %s", cfg.Annotations.ProviderName)
	}
	if cfg.Annotations.ProviderRef != "custom/ref" {
		t.Errorf("ProviderRef = %s", cfg.Annotations.ProviderRef)
	}
	if cfg.Annotations.SecretKey != "custom/key" {
		t.Errorf("SecretKey = %s", cfg.Annotations.SecretKey)
	}
	if cfg.DefaultSecretDataKey != "customval" {
		t.Errorf("DefaultSecretDataKey = %s", cfg.DefaultSecretDataKey)
	}
	if cfg.PollInterval != 123 {
		t.Errorf("PollInterval = %d", cfg.PollInterval)
	}
}

func TestNewInvalidPollInterval(t *testing.T) {
	t.Setenv("KSS_POLL_INTERVAL", "not-an-int")
	cfg := New(&kubernetes.Clientset{})
	if cfg.PollInterval != 300 {
		t.Errorf("PollInterval = %d, want 300 on invalid input", cfg.PollInterval)
	}
}
