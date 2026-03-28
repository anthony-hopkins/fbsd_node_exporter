// Package fbsd_node_exporter provides collectors for FreeBSD metrics.
package fbsd_node_exporter

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
)

// CPUCollector implements the prometheus.Collector interface to export CPU usage metrics.
// It uses gopsutil to fetch system-level CPU statistics on FreeBSD.
type CPUCollector struct {
	// usage is a Prometheus descriptor (Desc) for the CPU usage metric.
	// It defines the metric name, help string, and labels (none in this case).
	usage *prometheus.Desc
}

// NewCPUCollector creates and returns a new instance of CPUCollector.
// It requires a namespace which will be prefixed to all its exported metrics.
func NewCPUCollector(namespace string) *CPUCollector {
	return &CPUCollector{
		// BuildFQName creates a fully qualified name: "namespace_subsystem_name".
		// Here it becomes "namespace_cpu_usage_percent".
		usage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cpu", "usage_percent"),
			"Total CPU usage percentage (averaged across all cores).",
			nil, nil,
		),
	}
}

// Describe sends the descriptors of all metrics this collector produces to the provided channel.
// This is part of the prometheus.Collector interface and is called during registration.
func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usage
}

// Collect is called by the Prometheus registry during each scrape.
// It fetches the latest CPU metrics and sends them to the metric channel.
func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	// Set a context timeout to ensure slow calls to gopsutil don't block scraping forever.
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// PercentWithContext retrieves CPU usage percentage.
	// Using a small interval (200ms) allows us to get a current snapshot of usage.
	// If interval is 0, it would return usage since the last call, which is less precise for scrapes.
	// The third parameter 'false' means we want the average across all CPUs, not per-core.
	percentages, err := cpu.PercentWithContext(ctx, 200*time.Millisecond, false)
	if err != nil || len(percentages) == 0 {
		// If we failed to get metrics, we return without sending anything.
		// In a production scenario, you might want to log this or increment an error counter.
		return
	}

	// Create and send a constant metric (Gauge) for the CPU usage percentage.
	// MustNewConstMetric creates a metric using the previously defined descriptor.
	ch <- prometheus.MustNewConstMetric(
		c.usage,
		prometheus.GaugeValue,
		percentages[0],
	)
}
