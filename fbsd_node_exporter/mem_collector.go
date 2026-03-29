package fbsd_node_exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/mem"
)

type MemoryCollector struct {
	totalPct *prometheus.Desc
	used     *prometheus.Desc
	free     *prometheus.Desc
}

func NewMemoryCollector() *MemoryCollector {
	return &MemoryCollector{
		totalPct: prometheus.NewDesc(
			"node_memory_usage_percent",
			"Total system memory usage percentage",
			nil, nil,
		),
		used: prometheus.NewDesc(
			"node_memory_used_bytes",
			"Used system memory in bytes",
			nil, nil,
		),
		free: prometheus.NewDesc(
			"node_memory_free_bytes",
			"Free system memory in bytes",
			nil, nil,
		),
	}
}

func (c *MemoryCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPct
	ch <- c.used
	ch <- c.free
}

func (c *MemoryCollector) Collect(ch chan<- prometheus.Metric) {
	vm, err := mem.VirtualMemory()
	if err != nil {
		return
	}

	// vm.UsedPercent is already a float64 (0–100)
	ch <- prometheus.MustNewConstMetric(
		c.totalPct,
		prometheus.GaugeValue,
		vm.UsedPercent,
	)

	ch <- prometheus.MustNewConstMetric(
		c.used,
		prometheus.GaugeValue,
		float64(vm.Used),
	)

	ch <- prometheus.MustNewConstMetric(
		c.free,
		prometheus.GaugeValue,
		float64(vm.Free),
	)
}
