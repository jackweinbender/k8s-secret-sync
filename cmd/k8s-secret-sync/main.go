// Package main implements a Kubernetes operator that syncs secrets from 1Password into Kubernetes secrets.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jackweinbender/k8s-secret-sync/pkg/config"
	"github.com/jackweinbender/k8s-secret-sync/pkg/sync"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

func main() {
	// Initialize klog flags. Flags will be parsed by config's client init.
	klog.InitFlags(nil)
	defer klog.Flush()

	// Giddy up!
	klog.InfoS("Starting k8s-secret-sync operator...")

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Set up the Kubernetes clientset for interacting with the cluster
	klog.InfoS("Initializing Kubernetes clientset...")
	clientset, err := initClientSet()
	if err != nil {
		klog.ErrorS(err, "Failed to initialize Kubernetes clientset")
		return
	}

	// Load configuration from environment variables and initialize Kubernetes client
	klog.InfoS("Loading configuration...")
	cfg := config.New(clientset)

	// Start the sync process
	klog.InfoS("Starting sync process...")
	if err := sync.Run(ctx, cfg); err != nil {
		klog.ErrorS(err, "Sync exited with error")
	}

	// Wait for shutdown signal
	<-ctx.Done()
	klog.InfoS("Shutting down")
}

// initClientSet initializes and returns a Kubernetes clientset for cluster interaction.
// It attempts to create a connection using in-cluster configuration first. If that fails,
// it falls back to using the local kubeconfig file, typically found in ~/.kube/config.
// The kubeconfig path can be overridden using the -kubeconfig flag.
//
// Returns:
//   - *kubernetes.Clientset: The initialized Kubernetes client
//   - error: Any error encountered during initialization
func initClientSet() (*kubernetes.Clientset, error) {
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
		klog.ErrorS(err, "Error creating clientset")
		return nil, err
	}
	klog.InfoS("Successfully connected to Kubernetes cluster")
	return clientset, nil
}
