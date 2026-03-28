package fbsd_node_exporter

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/cpu"
)

type CPUCoreCollector struct {
	mu        sync.Mutex
	prevTimes []cpu.TimesStat

	usageDesc *prometheus.Desc
}

func NewCPUCoreCollector(namespace string) *CPUCoreCollector {
	return &CPUCoreCollector{
		usageDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cpu_core", "usage_percent"),
			"CPU usage percentage per core.",
			[]string{"cpu"},
			nil,
		),
	}
}

func (c *CPUCoreCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.usageDesc
}

func (c *CPUCoreCollector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	times, err := cpu.Times(true)
	if err != nil {
		return
	}

	if c.prevTimes == nil {
		c.prevTimes = times
		return
	}

	for i := range times {
		prev := c.prevTimes[i]
		curr := times[i]

		prevTotal := totalCPU(prev)
		currTotal := totalCPU(curr)

		totalDelta := currTotal - prevTotal
		idleDelta := curr.Idle - prev.Idle

		if totalDelta <= 0 {
			continue
		}

		usage := (1 - (idleDelta / totalDelta)) * 100

		ch <- prometheus.MustNewConstMetric(
			c.usageDesc,
			prometheus.GaugeValue,
			usage,
			curr.CPU,
		)
	}

	c.prevTimes = times
}

func totalCPU(t cpu.TimesStat) float64 {
	return t.User + t.System + t.Idle + t.Nice + t.Iowait +
		t.Irq + t.Softirq + t.Steal + t.Guest + t.GuestNice
}
