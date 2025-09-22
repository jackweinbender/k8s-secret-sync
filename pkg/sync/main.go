package sync

import (
	"context"
	"encoding/json"
	"maps"
	"time"

	"github.com/jackweinbender/k8s-secret-sync/pkg/config"
	"github.com/jackweinbender/k8s-secret-sync/pkg/op"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
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
				klog.ErrorS(nil, "Failed to cast object to Secret on add event, skipping")
				return
			}

			// Check for required provider annotation
			providerName, exists := secret.Annotations[cfg.Annotations.ProviderName]
			klog.InfoS("Processing secret with provider", "namespace", secret.Namespace, "name", secret.Name, "provider", providerName)
			if !exists || providerName == "" {
				klog.InfoS("Ignoring secret as it does not have the required provider annotation", "namespace", secret.Namespace, "name", secret.Name)
				return
			}

			// Check for required ref annotation
			secretID, exists := secret.Annotations[cfg.Annotations.ProviderRef]
			if !exists || secretID == "" {
				klog.InfoS("Ignoring secret as it does not have the required ref annotation", "namespace", secret.Namespace, "name", secret.Name)
				return
			}

			// Check for last-synced annotation
			if _, synced := secret.Annotations["last-synced"]; synced {
				klog.InfoS("Secret has already been synced (last-synced annotation present)", "namespace", secret.Namespace, "name", secret.Name)
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
				klog.ErrorS(err, "Failed to initialize provider", "provider", providerName)
				return
			}

			value, err := provider.GetSecretValue(ctx, secretID)
			if err != nil {
				klog.ErrorS(err, "Failed to resolve secret URI", "secretID", secretID)
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
				klog.ErrorS(err, "Failed to marshal patch data")
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
				klog.ErrorS(err, "Failed to update Kubernetes Secret", "namespace", secret.Namespace, "name", secret.Name)
				return
			}
			klog.InfoS("Successfully updated Kubernetes Secret with provider value and set last-synced annotation", "namespace", secret.Namespace, "name", secret.Name)
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
