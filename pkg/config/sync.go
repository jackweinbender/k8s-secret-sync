package config

import (
	"log"

	"k8s.io/client-go/kubernetes"
)

type Sync struct {
	Clientset            *kubernetes.Clientset
	Annotations          Annotations
	DefaultSecretDataKey string // Default key in the secret data to store fetched calues if annotation is not set
	PollInterval         int    // Sync interval in seconds
}

func New() *Sync {
	log.Println("Intializing configuration...")

	// Set up the Kubernetes clientset for interacting with the cluster
	clientset, err := initClientSet()
	if err != nil {
		log.Fatalf("Error initializing Kubernetes clientset: %v", err)
	}

	// Read in configuration from environment variables with defaults
	log.Println("Loading configuration from environment variables...")
	return &Sync{
		Clientset: clientset,
		Annotations: Annotations{
			ProviderName: env("KSS_SECRET_ANNOTATION_KEY_PROVIDER_NAME", "k8s-secret-sync.weinbender.io/provider-name"),
			ProviderRef:  env("KSS_SECRET_ANNOTATION_KEY_PROVIDER_REF", "k8s-secret-sync.weinbender.io/provider-ref"),
			SecretKey:    env("KSS_SECRET_ANNOTATION_KEY_SECRET_KEY", "k8s-secret-sync.weinbender.io/secret-key"),
		},
		DefaultSecretDataKey: env("KSS_DEFAULT_SECRET_DATA_KEY", "value"),
		PollInterval:         env("KSS_POLL_INTERVAL", 300),
	}
}
