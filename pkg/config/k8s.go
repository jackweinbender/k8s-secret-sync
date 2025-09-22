package config

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

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
