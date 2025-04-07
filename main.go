package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/1password/onepassword-sdk-go"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
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
	opClient, err := onePasswordInit()
	if err != nil {
		log.Fatalf("Error initializing 1Password SDK: %v", err)
	}

	// Set up informer to watch secrets
	secretInformer := informers.NewSharedInformerFactory(clientset, time.Second*30).Core().V1().Secrets().Informer()

	secretInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj any) {
			oldSecret, ok := oldObj.(*v1.Secret)

			if !ok {
				// If the old object is not a Secret, skip processing
				log.Printf("Failed to cast old object to Secret")
				return
			}

			secretURI, exist := oldSecret.Data["op"]
			if !exist {
				// If the old secret does not have the "op" key, skip processing
				log.Printf("Skipping old secret as it does not contain 'op' key")
				return
			}

			// fetch the value of the "op" key from the old secret to ensure it is a valid 1Password item reference
			value, err := opClient.Secrets().Resolve(ctx, string(secretURI))
			if err != nil {
				log.Printf("Failed to resolve 1Password secret URI %s: %v", string(secretURI), err)
				return
			}

			// Proceed to update the new object with the resolved value
			newSecret, ok := newObj.(*runtime.Object)
			if !ok {
				log.Printf("Failed to cast new object to runtime.Object")
				return
			}
			// Ensure the new object is a Secret
			secret, ok := (*newSecret).(*v1.Secret)
			if !ok {
				log.Printf("Failed to cast new object to Secret")
				return
			}

			// Update the new secret with the resolved value from 1Password
			secret.Data["op"] = secretURI
			secret.Data["value"] = []byte(value)
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

func onePasswordInit() (*onepassword.Client, error) {
	token := os.Getenv("OP_SERVICE_ACCOUNT_TOKEN")

	client, err := onepassword.NewClient(
		context.TODO(),
		onepassword.WithServiceAccountToken(token),
		onepassword.WithIntegrationInfo("My k8s secret sync operator", "v0"),
	)
	if err != nil {
		return nil, err
	}

	return client, nil
}
