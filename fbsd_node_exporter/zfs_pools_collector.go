// Package fbsd_node_exporter provides collectors for FreeBSD-specific metrics.
package fbsd_node_exporter

import (
	"bufio"
	"bytes"
	"os/exec"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// ZFSPoolCollector collects metrics about ZFS pools on FreeBSD.
// It uses the zpool(8) command to gather pool-level statistics.
type ZFSPoolCollector struct {
	health        *prometheus.Desc
	allocBytes    *prometheus.Desc
	freeBytes     *prometheus.Desc
	fragPercent   *prometheus.Desc
	dedupRatio    *prometheus.Desc
	capacityPct   *prometheus.Desc
	sizeBytes     *prometheus.Desc
	expandszBytes *prometheus.Desc
}

// NewZFSPoolCollector creates a new ZFSPoolCollector with pre-defined Prometheus descriptors.
func NewZFSPoolCollector() *ZFSPoolCollector {
	labels := []string{"pool"}

	return &ZFSPoolCollector{
		health: prometheus.NewDesc(
			"zfs_pool_health",
			"ZFS pool health (0=offline,1=degraded,2=online).",
			labels, nil,
		),
		allocBytes: prometheus.NewDesc(
			"zfs_pool_alloc_bytes",
			"ZFS pool allocated bytes.",
			labels, nil,
		),
		freeBytes: prometheus.NewDesc(
			"zfs_pool_free_bytes",
			"ZFS pool free bytes.",
			labels, nil,
		),
		fragPercent: prometheus.NewDesc(
			"zfs_pool_fragmentation_percent",
			"ZFS pool fragmentation percent.",
			labels, nil,
		),
		dedupRatio: prometheus.NewDesc(
			"zfs_pool_dedup_ratio",
			"ZFS pool deduplication ratio.",
			labels, nil,
		),
		capacityPct: prometheus.NewDesc(
			"zfs_pool_capacity_percent",
			"ZFS pool capacity used percent.",
			labels, nil,
		),
		sizeBytes: prometheus.NewDesc(
			"zfs_pool_size_bytes",
			"ZFS pool total size in bytes.",
			labels, nil,
		),
		expandszBytes: prometheus.NewDesc(
			"zfs_pool_expandsz_bytes",
			"ZFS pool expandable size in bytes.",
			labels, nil,
		),
	}
}

// Describe sends the descriptors of each ZFS pool metric to the provided channel.
func (c *ZFSPoolCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.health
	ch <- c.allocBytes
	ch <- c.freeBytes
	ch <- c.fragPercent
	ch <- c.dedupRatio
	ch <- c.capacityPct
	ch <- c.sizeBytes
	ch <- c.expandszBytes
}

// Collect fetches the ZFS pool statistics and sends them as Prometheus metrics.
func (c *ZFSPoolCollector) Collect(ch chan<- prometheus.Metric) {
	// zpool list -Hp -o name,size,alloc,free,cap,health,frag,dedup,expandsz
	cmd := exec.Command("zpool", "list", "-Hp",
		"-o", "name,size,alloc,free,cap,health,frag,dedup,expandsz")
	out, err := cmd.Output()
	if err != nil {
		return
	}

	sc := bufio.NewScanner(bytes.NewReader(out))

	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 9 {
			continue
		}

		name := fields[0]
		sizeStr := fields[1]
		allocStr := fields[2]
		freeStr := fields[3]
		capStr := fields[4]
		healthStr := fields[5]
		fragStr := fields[6]
		dedupStr := fields[7]
		expandszStr := fields[8]

		size, _ := strconv.ParseFloat(sizeStr, 64)
		alloc, _ := strconv.ParseFloat(allocStr, 64)
		free, _ := strconv.ParseFloat(freeStr, 64)
		capPct, _ := strconv.ParseFloat(capStr, 64)
		fragPct, _ := strconv.ParseFloat(fragStr, 64)
		expandsz, _ := strconv.ParseFloat(expandszStr, 64)

		dedupStr = strings.TrimSuffix(dedupStr, "x")
		dedup, _ := strconv.ParseFloat(dedupStr, 64)

		healthVal := mapZFSHealth(healthStr)

		ch <- prometheus.MustNewConstMetric(c.health, prometheus.GaugeValue, healthVal, name)
		ch <- prometheus.MustNewConstMetric(c.sizeBytes, prometheus.GaugeValue, size, name)
		ch <- prometheus.MustNewConstMetric(c.allocBytes, prometheus.GaugeValue, alloc, name)
		ch <- prometheus.MustNewConstMetric(c.freeBytes, prometheus.GaugeValue, free, name)
		ch <- prometheus.MustNewConstMetric(c.capacityPct, prometheus.GaugeValue, capPct, name)
		ch <- prometheus.MustNewConstMetric(c.fragPercent, prometheus.GaugeValue, fragPct, name)
		ch <- prometheus.MustNewConstMetric(c.dedupRatio, prometheus.GaugeValue, dedup, name)
		ch <- prometheus.MustNewConstMetric(c.expandszBytes, prometheus.GaugeValue, expandsz, name)
	}
}

// mapZFSHealth converts ZFS health strings into numeric values for Prometheus metrics.
func mapZFSHealth(h string) float64 {
	h = strings.ToLower(h)
	switch h {
	case "online":
		return 2
	case "degraded":
		return 1
	case "faulted", "offline", "removed", "unavail":
		return 0
	default:
		return 0
	}
}
