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
			"CPU_core_usage",
			"CPU usage percent per core",
			[]string{"core"}, // ONLY core label
			nil,              // no constant labels
		),
	}
}

func (c *CPUCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
}

func (c *CPUCollector) Collect(ch chan<- prometheus.Metric) {
	usages, err := cpu.Percent(0, true)
	if err != nil {
		return
	}

	for i, u := range usages {
		coreLabel := fmt.Sprintf("%d", i)

		ch <- prometheus.MustNewConstMetric(
			c.usageDesc,
			prometheus.GaugeValue,
			u/100.0, // convert 0–100 → 0.0–1.0
			coreLabel,
		)
	}
}
