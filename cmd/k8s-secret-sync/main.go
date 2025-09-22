// Package main implements a Kubernetes operator that syncs secrets from 1Password into Kubernetes secrets.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackweinbender/k8s-secret-sync/pkg/config"
	"github.com/jackweinbender/k8s-secret-sync/pkg/sync"
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

	// Load configuration from environment variables and initialize Kubernetes client
	klog.InfoS("Loading configuration...")
	cfg := config.New()

	// Start the sync process
	klog.InfoS("Starting sync process...")
	if err := sync.Run(ctx, cfg); err != nil {
		klog.ErrorS(err, "Sync exited with error")
	}

	// Wait for shutdown signal
	<-ctx.Done()
	klog.InfoS("Shutting down")
}
