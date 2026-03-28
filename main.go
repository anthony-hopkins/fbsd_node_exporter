// Package fbsd_node_exporter provides the main entry point for the FreeBSD Node Exporter.
package main

import (
	"fbsd_node_exporter/fbsd_node_exporter"
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// main is the entry point of the exporter application.
// It initializes collectors, registers them with a Prometheus registry,
// and starts an HTTP server to expose the metrics.
func main() {
	// Namespace is used as a prefix for all metrics (e.g., mini_node_cpu_usage_percent).
	// It can be overridden via the EXPORTER_NAMESPACE environment variable.
	namespace := envOrDefault("EXPORTER_NAMESPACE", "DHS_")

	// The address the HTTP server will bind to.
	// It can be overridden via the EXPORTER_ADDR environment variable.
	addr := envOrDefault("EXPORTER_ADDR", ":9100")

	// Create a new, non-global Prometheus registry.
	// Using a custom registry is often cleaner than the default global one.
	reg := prometheus.NewRegistry()

	// Initialize hardware-specific collectors.
	// These are modular; more can be added easily by following the NodeCollector interface.
	// cpuCollector now handles both per-core and aggregate metrics.
	cpuCollector := fbsd_node_exporter.NewCPUCollector()
	memCollector := fbsd_node_exporter.NewMemoryCollector(namespace)

	// Register the collectors with our custom registry.
	// MustRegister will panic if there's an error (e.g., duplicate metric names).
	reg.MustRegister(cpuCollector)
	reg.MustRegister(memCollector)

	// Set up the /metrics endpoint using the registry we created.
	// promhttp.HandlerFor provides an HTTP handler that scrapes our registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	// Inform the user that we are starting.
	log.Printf("Starting exporter on %s with namespace %q", addr, namespace)

	// Start the HTTP server. This is a blocking call.
	// It will serve requests until the process is terminated.
	if err := http.ListenAndServe(addr, nil); err != nil {
		// Log any fatal errors that prevent the server from starting.
		log.Fatalf("listen and serve: %v", err)
	}
}

// envOrDefault is a helper function to read an environment variable or return a default value.
func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
