// Package fbsd_node_exporter provides collectors for FreeBSD metrics.
package fbsd_node_exporter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
)

// CPUCoreCollector implements the prometheus.Collector interface to export CPU usage metrics
// on a per-core basis. It uses gopsutil to fetch system-level CPU statistics on FreeBSD.
// Unlike CPUCollector, which provides an average, this collector exposes metrics for each CPU core.
type CPUCoreCollector struct {
	// mu is a mutex to ensure thread-safe access to prevTimes during scraping.
	mu sync.Mutex
	// prevTimes stores the CPU times from the previous scrape to calculate deltas.
	// Since per-core usage is derived from cumulative counters, we need two points in time.
	prevTimes []cpu.TimesStat

	// usageDesc is a Prometheus descriptor (Desc) for the per-core CPU usage metric.
	// It includes a "cpu" label to distinguish between different cores.
	usageDesc *prometheus.Desc
}

// NewCPUCoreCollector creates and returns a new instance of CPUCoreCollector.
// It requires a namespace which will be prefixed to all its exported metrics.
func NewCPUCoreCollector(namespace string) *CPUCoreCollector {
	return &CPUCoreCollector{
		// BuildFQName creates a fully qualified name: "namespace_subsystem_name".
		// Here it becomes "namespace_cpu_core_usage_percent".
		usageDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cpu_core", "usage_percent"),
			"CPU usage percentage per core.",
			[]string{"cpu"}, // Label to identify the specific CPU core.
			nil,
		),
	}
}

// Describe sends the descriptors of all metrics this collector produces to the provided channel.
// This is part of the prometheus.Collector interface and is called during registration.
func (c *CPUCoreCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
}

// Collect is called by the Prometheus registry during each scrape.
// It calculates per-core CPU usage by comparing current and previous CPU times.
func (c *CPUCoreCollector) Collect(ch chan<- prometheus.Metric) {
	// Lock the collector to ensure only one scrape can access/modify prevTimes at a time.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Fetch current CPU times for all cores.
	// The true parameter indicates that we want statistics for each individual CPU.
	times, err := cpu.Times(true)
	if err != nil {
		// If we failed to get metrics, we return without sending anything.
		return
	}

	// If this is the first scrape, we store the current times and return.
	// We need at least two sets of measurements to calculate the delta and percentage.
	if c.prevTimes == nil {
		c.prevTimes = times
		return
	}

	// Iterate through each core's statistics.
	for i := range times {
		prev := c.prevTimes[i]
		curr := times[i]

		// Calculate total CPU time for previous and current states.
		prevTotal := totalCPU(prev)
		currTotal := totalCPU(curr)

		// Calculate the difference in total time and idle time.
		totalDelta := currTotal - prevTotal
		idleDelta := curr.Idle - prev.Idle

		// Prevent division by zero if no time has passed or counters haven't incremented.
		if totalDelta <= 0 {
			continue
		}

		// Calculate usage percentage: (Total Time - Idle Time) / Total Time * 100.
		usage := (1 - (idleDelta / totalDelta)) * 100

		// Create and send a constant metric (Gauge) for the per-core CPU usage percentage.
		// Includes the CPU identifier (e.g., "cpu0", "cpu1") as a label value.
		ch <- prometheus.MustNewConstMetric(
			c.usageDesc,
			prometheus.GaugeValue,
			usage,
			curr.CPU,
		)
	}

	// Update the previous times with the current ones for the next scrape cycle.
	c.prevTimes = times
}

// totalCPU is a helper function that sums all CPU time states to get the total time.
// This includes User, System, Idle, and other specific states reported by gopsutil.
func totalCPU(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Iowait +
		t.Irq + t.Softirq + t.Steal + t.Guest + t.GuestNice
}
