// Package fbsd_node_exporter provides collectors for FreeBSD metrics.
package fbsd_node_exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/mem"
)

// MemoryCollector implements the prometheus.Collector interface to export system memory metrics.
// It uses gopsutil to fetch virtual memory statistics on FreeBSD.
type MemoryCollector struct {
	// Descriptors (Desc) define the metric names, help strings, and labels for Prometheus.
	// We pre-calculate them for efficiency during the scraping process.
	total     *prometheus.Desc
	used      *prometheus.Desc
	available *prometheus.Desc
	usedPct   *prometheus.Desc
}

// NewMemoryCollector creates and returns a new instance of MemoryCollector.
// The provided namespace prefixes all metric names exported by this collector.
func NewMemoryCollector(namespace string) *MemoryCollector {
	return &MemoryCollector{
		// BuildFQName creates names in the format: "namespace_subsystem_name".
		total: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "memory", "total_bytes"),
			"Total system memory in bytes.",
			nil, nil,
		),
		used: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "memory", "used_bytes"),
			"Used system memory in bytes.",
			nil, nil,
		),
		available: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "memory", "available_bytes"),
			"Available system memory in bytes.",
			nil, nil,
		),
		usedPct: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "memory", "used_percent"),
			"Used system memory as a percentage.",
			nil, nil,
		),
	}
}

// Describe sends the descriptors of all metrics this collector produces to the provided channel.
// This is required by the prometheus.Collector interface and is used during registration.
func (m *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.total
	ch <- m.used
	ch <- m.available
	ch <- m.usedPct
}

// Collect is called by the Prometheus registry whenever a scrape request is received.
// It fetches live memory data and sends it as metrics to the provided channel.
func (m *MemoryCollector) Collect(ch chan<- prometheus.Metric) {
	// VirtualMemory fetches virtual memory usage stats.
	// On FreeBSD, gopsutil uses sysctl to read memory information from the kernel.
	vm, err := mem.VirtualMemory()
	if err != nil {
		// If we encounter an error, we silently return. In a production system,
		// you might want to increment an internal error counter for monitoring.
		return
	}

	// MustNewConstMetric creates a one-time metric (Gauge) based on the current state.
	// We convert the uint64 values from gopsutil to float64 for Prometheus compatibility.

	// Total system memory.
	ch <- prometheus.MustNewConstMetric(
		m.total,
		prometheus.GaugeValue,
		float64(vm.Total),
	)
	// Currently used memory.
	ch <- prometheus.MustNewConstMetric(
		m.used,
		prometheus.GaugeValue,
		float64(vm.Used),
	)
	// Memory available for new processes.
	ch <- prometheus.MustNewConstMetric(
		m.available,
		prometheus.GaugeValue,
		float64(vm.Available),
	)
	// Memory usage percentage.
	ch <- prometheus.MustNewConstMetric(
		m.usedPct,
		prometheus.GaugeValue,
		vm.UsedPercent,
	)
}
