// Package main implements a Kubernetes operator that syncs secrets from 1Password into Kubernetes secrets.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackweinbender/k8s-secret-sync/pkg/config"
	"github.com/jackweinbender/k8s-secret-sync/pkg/sync"

	// Import necessary packages for context, logging, Kubernetes client, and 1Password integration.

	"k8s.io/klog/v2"
)

// Annotation keys and default values used for identifying and processing secrets.
var (
	annotationPrefix       = "k8s-secret-sync.weinbender.io/" // Base prefix for all annotations used by this operator
	annotationKeyProvider  = "provider"                       // Annotation to specify the secret provider (e.g., "op" for 1Password)
	annotationKeyRef       = "ref"                            // Annotation to specify the reference or ID of the secret in the provider
	annotationKeySecretKey = "secret-key"                     // Annotation to specify the key in the secret data to update
	defaultSecretDataKey   = "value"                          // Default key in the secret data if annotation is not set
)

func main() {
	// Initialize klog flags. Flags will be parsed by config's client init.
	klog.InitFlags(nil)
	defer klog.Flush()

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.New()
	klog.InfoS("starting", "component", "k8s-secret-sync", "pollInterval", cfg.PollInterval)

	if err := sync.Run(ctx, cfg); err != nil {
		klog.ErrorS(err, "sync exited with error")
	}

	<-ctx.Done()
	klog.InfoS("shutting down")

	// Log the starting of the operator
	log.Println("Starting k8s-secret-sync operator...")
}
