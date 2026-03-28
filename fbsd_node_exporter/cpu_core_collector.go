package fbsd_node_exporter

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
)

type CPUCollector struct {
	usageDesc *prometheus.Desc
}

func NewCPUCollector() *CPUCollector {
	return &CPUCollector{
		usageDesc: prometheus.NewDesc(
			"node_cpu_usage_percent",
			"CPU usage percent per core",
			[]string{"core"}, // label
			nil,
		),
	}
}

func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
}

func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	// true = per-core usage
	usages, err := cpu.Percent(0, true)
	if err != nil {
		return
	}

	for i, u := range usages {
		coreLabel := fmt.Sprintf("%d", i)

		ch <- prometheus.MustNewConstMetric(
			c.usageDesc,
			prometheus.GaugeValue,
			u,
			coreLabel,
		)
	}
}
