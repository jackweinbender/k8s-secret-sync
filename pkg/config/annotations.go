package config

// config.Annotations holds the configuration for how KSS
// reads and interprets Kubernetes Secret Annotations. In most
// cases, the default values should be used. However, in some cases where
// you may not be able to control the Annotations applied to a Secret, it may
// be prefereable to change the keys used to read the Annotations.
type Annotations struct {
	// Key for the annotation that specifies the secret provider.
	// Used to specify which secret provider to use to fetch the secret value.
	ProviderName string // default: "k8s-secret-sync.weinbender.io/provider"

	// Key for the annotation that specifies the secret reference for the provider.
	// Used to specify the identifier or path of the secret for a given provider.
	ProviderRef string // default: "k8s-secret-sync.weinbender.io/provider-ref"

	// Key for the annotation that specifies where to store the fetched data.
	// Used to specify which key in the Kubernetes Secret to update with the fetched secret value.
	SecretKey string // default: "k8s-secret-sync.weinbender.io/secret-key"
}
