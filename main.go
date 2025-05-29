// Package main implements a Kubernetes operator that syncs secrets from 1Password into Kubernetes secrets.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/jackweinbender/secrets-operator/op"
	"github.com/jackweinbender/secrets-operator/shared" // Import necessary packages for context, logging, Kubernetes client, and 1Password integration.

	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

// Annotation keys and default values used for identifying and processing secrets.
var (
	providerAnnotation      = "k8s-secret-sync.weinbender.io/provider"   // Annotation to specify the secret provider (e.g., "op" for 1Password)
	refAnnotation           = "k8s-secret-sync.weinbender.io/ref"        // Annotation to specify the reference or ID of the secret in the provider
	secretDataKeyAnnotation = "k8s-secret-sync.weinbender.io/secret-key" // Annotation to specify the key in the secret data to update
	defaultSecretDataKey    = "value"                                    // Default key in the secret data if annotation is not set
)

func main() {
	// Create a context for API calls and background operations
	ctx := context.Background()

	// Initialize the Kubernetes clientset for interacting with the cluster
	clientset, err := kubernetesClientsetInit()
	if err != nil {
		log.Fatalf("Error initializing Kubernetes clientset: %v", err)
	}

	// Initialize the 1Password provider for fetching secrets
	opClient, err := op.NewProvider()
	if err != nil {
		log.Fatalf("Error initializing 1Password SDK: %v", err)
	}

	// Map of supported secret providers (currently only 1Password)
	providers := map[string]shared.SecretProvider{
		"op": opClient,
	}

	// Set up a shared informer to watch for changes to Kubernetes secrets
	secretInformer := informers.NewSharedInformerFactory(
		clientset, 10*time.Second).Core().V1().Secrets().Informer()

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
			providerName, exists := secret.Annotations[providerAnnotation]
			log.Printf("Processing %s/%s with provider %s", secret.Namespace, secret.Name, providerName)
			if !exists || providerName == "" {
				log.Printf("Ignoring %s/%s as it does not have the required `provider` annotation", secret.Namespace, secret.Name)
				return
			}

			// Check for required ref annotation
			secretID, exists := secret.Annotations[refAnnotation]
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
			secretDataKey := defaultSecretDataKey
			if secretKeyAnnotationValue, exists := secret.Annotations[secretDataKeyAnnotation]; exists && secretKeyAnnotationValue != "" {
				secretDataKey = secretKeyAnnotationValue
			}

			// Fetch the secret value from the provider (e.g., 1Password)
			value, err := providers[providerName].GetSecretValue(ctx, secretID)
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
			_, err = clientset.CoreV1().Secrets(secret.Namespace).Patch(
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

// kubernetesClientsetInit initializes and returns a Kubernetes clientset, using in-cluster config if available, or falling back to kubeconfig file.
func kubernetesClientsetInit() (*kubernetes.Clientset, error) {
	var kubeconfig *string
	if home := os.Getenv("HOME"); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	// Try to get in-cluster config first, fall back to .kube if not running in a cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating clientset: %v", err)
		return nil, err
	}

	log.Println("Successfully connected to Kubernetes cluster")
	return clientset, nil
}
