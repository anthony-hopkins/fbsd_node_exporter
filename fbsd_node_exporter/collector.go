// Package fbsd_node_exporter provides a Prometheus exporter for FreeBSD metrics.
// It is designed to be modular and easy to extend with new collectors.
package fbsd_node_exporter

import "github.com/prometheus/client_golang/prometheus"

// NodeCollector is an interface that wraps the prometheus.Collector interface.
// It serves as a base for all hardware-specific collectors in this exporter.
// By using a common interface, we can easily register and manage multiple collectors.
// You can extend this later with common initialization hooks, configuration, or metadata.
type NodeCollector interface {
	prometheus.Collector
}
