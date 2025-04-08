package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jackweinbender/secrets-operator/op"
	"github.com/jackweinbender/secrets-operator/shared"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	poviderAnnotation       = "k8s-secret-sync.weinbender.io/provider"
	refAnnotation           = "k8s-secret-sync.weinbender.io/ref"
	secretDataKeyAnnotation = "k8s-secret-sync.weinbender.io/secret-key"
	defaultSecretDataKey    = "value"
)

func main() {
	// Define the context for the application
	ctx := context.Background()

	// Initialize Kubernetes clientset
	clientset, err := kubernetesClientsetInit()
	if err != nil {
		log.Fatalf("Error initializing Kubernetes clientset: %v", err)
	}

	// Initialize 1Password SDK
	opClient, err := op.NewProvider()
	if err != nil {
		log.Fatalf("Error initializing 1Password SDK: %v", err)
	}

	providers := map[string]shared.SecretProvider{
		"op": opClient,
	}

	// Set up informer to watch secrets
	secretInformer := informers.NewSharedInformerFactory(
		clientset, 5*time.Second).Core().V1().Secrets().Informer()

	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, _ any) {
			oldSecret, ok := oldObj.(*v1.Secret)

			if !ok {
				// If the old object is not a Secret, skip processing
				log.Printf("Failed to cast old object to Secret, skipping.")
				return
			}

			// Check to see if the secret has a provider annotation
			providerName, exists := oldSecret.Annotations[poviderAnnotation]
			if !exists || providerName != "1password" {
				// If the annotation is missing or empty, skip processing
				log.Printf("Ignoring %s/%s as it does not have the required `provider` annotation", oldSecret.Namespace, oldSecret.Name)
				return
			}

			secretID, exits := oldSecret.Annotations[refAnnotation]
			if !exits || secretID == "" {
				// If the annotation is missing or empty, skip processing
				log.Printf("Ignoring %s/%s as it does not have the required `ref` annotation", oldSecret.Namespace, oldSecret.Name)
				return
			}

			// Determine the secret data key to use, use default otherwise
			secretDataKey := defaultSecretDataKey
			secretKeyAnnotationValue, exists := oldSecret.Annotations[secretDataKeyAnnotation]

			if exists && secretKeyAnnotationValue != "" {
				secretDataKey = secretKeyAnnotationValue
			}

			// fetch the value of the "op" key from the old secret to ensure it is a valid 1Password item reference
			value, err := providers[providerName].GetSecretValue(ctx, secretID)

			if err != nil {
				log.Printf("Failed to resolve 1Password secret URI %s: %v", secretID, err)
				return
			}

			patchData := v1.Secret{
				Data: map[string][]byte{
					secretDataKey: []byte(value),
				},
			}
			payloadBytes, err := json.Marshal(patchData)
			if err != nil {
				panic(err)
			}

			_, err = clientset.CoreV1().Secrets(oldSecret.Namespace).Patch(
				ctx,
				oldSecret.Name,
				types.StrategicMergePatchType,
				payloadBytes,
				metav1.PatchOptions{})

			// Update the Kubernetes secret in the cluster
			if err != nil {
				// Handle error updating the secret
				log.Printf("Failed to update Kubernetes Secret %s/%s: %v", oldSecret.Namespace, oldSecret.Name, err)
				return
			}
			// Log successful update
			log.Printf("Successfully updated Kubernetes Secret %s/%s with 1Password value", oldSecret.Namespace, oldSecret.Name)
		},
	})

	// Start informer
	stop := make(chan struct{})
	defer close(stop)
	secretInformer.Run(stop)

	// Keep the main function running
	select {}
}

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
