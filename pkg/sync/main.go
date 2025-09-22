package sync

import (
	"context"
	"encoding/json"
	"log"
	"maps"
	"time"

	"github.com/jackweinbender/k8s-secret-sync/pkg/config"
	"github.com/jackweinbender/k8s-secret-sync/pkg/op"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type SecretProvider interface {
	GetSecretValue(ctx context.Context, secretID string) (string, error)
}

func Run(ctx context.Context, cfg *config.Sync) error {
	// Map of supported secret providers (currently only 1Password)
	providers := map[string]func() (SecretProvider, error){
		"op": func() (SecretProvider, error) {
			opClient, err := NewProvider()
			if err != nil {
				return nil, err
			}
			return opClient, nil
		},
	}

	// Set up a shared informer to watch for changes to Kubernetes secrets
	secretInformer := informers.NewSharedInformerFactory(
		cfg.Clientset, 10*time.Second).Core().V1().Secrets().Informer()

	// Register event handlers for secret add and update events
	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		// Handler for new secret creation events
		AddFunc: func(obj any) {
			secret, ok := obj.(*v1.Secret)
			if !ok {
				log.Printf("Failed to cast object to Secret on add event, skipping.")
				return
			}

			// Check for required provider annotation
			providerName, exists := secret.Annotations[cfg.Annotations.ProviderName]
			log.Printf("Processing %s/%s with provider %s", secret.Namespace, secret.Name, providerName)
			if !exists || providerName == "" {
				log.Printf("Ignoring %s/%s as it does not have the required `provider` annotation", secret.Namespace, secret.Name)
				return
			}

			// Check for required ref annotation
			secretID, exists := secret.Annotations[cfg.Annotations.ProviderRef]
			if !exists || secretID == "" {
				log.Printf("Ignoring %s/%s as it does not have the required `ref` annotation", secret.Namespace, secret.Name)
				return
			}

			// Check for last-synced annotation
			if _, synced := secret.Annotations["last-synced"]; synced {
				log.Printf("Secret %s/%s has already been synced (last-synced annotation present)", secret.Namespace, secret.Name)
				return
			}

			// Determine which key in the secret data to update
			secretDataKey := cfg.DefaultSecretDataKey
			if secretKeyAnnotationValue, exists := secret.Annotations[cfg.Annotations.SecretKey]; exists && secretKeyAnnotationValue != "" {
				secretDataKey = secretKeyAnnotationValue
			}

			// Fetch the secret value from the provider (e.g., 1Password)
			provider, err := providers[providerName]()
			if err != nil {
				log.Printf("Failed to initialize provider %s: %v", providerName, err)
				return
			}

			value, err := provider.GetSecretValue(ctx, secretID)
			if err != nil {
				log.Printf("Failed to resolve 1Password secret URI %s: %v", secretID, err)
				return
			}

			// Copy annotations and add last-synced
			annotations := make(map[string]string)
			maps.Copy(annotations, secret.Annotations)
			annotations["last-synced"] = time.Now().UTC().Format(time.RFC3339)

			// Prepare the patch data to update the Kubernetes secret
			patchData := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
				Data: map[string][]byte{
					secretDataKey: []byte(value),
				},
			}
			payloadBytes, err := json.Marshal(patchData)
			if err != nil {
				log.Printf("Failed to marshal patch data: %v", err)
				return
			}

			// Patch the secret in the Kubernetes cluster
			_, err = cfg.Clientset.CoreV1().Secrets(secret.Namespace).Patch(
				ctx,
				secret.Name,
				types.StrategicMergePatchType,
				payloadBytes,
				metav1.PatchOptions{})

			if err != nil {
				log.Printf("Failed to update Kubernetes Secret %s/%s: %v", secret.Namespace, secret.Name, err)
				return
			}
			log.Printf("Successfully updated Kubernetes Secret %s/%s with 1Password value and set last-synced annotation", secret.Namespace, secret.Name)
		},
	})

	// Start the informer to begin watching for secret events
	stop := make(chan struct{})
	defer close(stop)
	secretInformer.Run(stop)

	// Block forever to keep the operator running
	select {}
}

func NewProvider() (SecretProvider, error) {
	client, err := op.InitClient()
	if err != nil {
		return nil, err
	}

	return op.SecretProvider{
		Client: client,
	}, nil
}
