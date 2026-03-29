// Package fbsd_node_exporter provides collectors for FreeBSD-specific metrics.
package fbsd_node_exporter

import (
	"bufio"
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

// DiskUsageCollector collects metrics about filesystem usage on FreeBSD.
// It uses the getfsstat(2) system call and mount(8) command to gather data.
type DiskUsageCollector struct {
	fsSize       *prometheus.Desc
	fsFree       *prometheus.Desc
	fsAvail      *prometheus.Desc
	fsUsed       *prometheus.Desc
	fsPercent    *prometheus.Desc
	fsInodes     *prometheus.Desc
	fsInodesFree *prometheus.Desc
	fsInodesUsed *prometheus.Desc
	fsInodesPct  *prometheus.Desc

	// Optional I/O stats (best-effort)
	fsReadsCompleted  *prometheus.Desc
	fsWritesCompleted *prometheus.Desc
	fsReadBytes       *prometheus.Desc
	fsWrittenBytes    *prometheus.Desc
}

// NewDiskUsageCollector creates a new DiskUsageCollector with pre-defined Prometheus descriptors.
func NewDiskUsageCollector() *DiskUsageCollector {
	labels := []string{"mountpoint", "fstype"}

	return &DiskUsageCollector{
		fsSize: prometheus.NewDesc(
			"bsd_disk_filesystem_size_bytes",
			"Total filesystem size in bytes.",
			labels, nil,
		),
		fsFree: prometheus.NewDesc(
			"bsd_disk_filesystem_free_bytes",
			"Free bytes including reserved blocks.",
			labels, nil,
		),
		fsAvail: prometheus.NewDesc(
			"bsd_disk_filesystem_avail_bytes",
			"Available bytes for unprivileged users.",
			labels, nil,
		),
		fsUsed: prometheus.NewDesc(
			"bsd_disk_filesystem_used_bytes",
			"Used bytes.",
			labels, nil,
		),
		fsPercent: prometheus.NewDesc(
			"bsd_disk_usage_percent",
			"Percentage of filesystem used.",
			labels, nil,
		),
		fsInodes: prometheus.NewDesc(
			"bsd_disk_inodes_total",
			"Total inodes.",
			labels, nil,
		),
		fsInodesFree: prometheus.NewDesc(
			"bsd_disk_inodes_free",
			"Free inodes.",
			labels, nil,
		),
		fsInodesUsed: prometheus.NewDesc(
			"bsd_disk_inodes_used",
			"Used inodes.",
			labels, nil,
		),
		fsInodesPct: prometheus.NewDesc(
			"bsd_disk_inodes_usage_percent",
			"Percentage of inodes used.",
			labels, nil,
		),
		fsReadsCompleted: prometheus.NewDesc(
			"bsd_disk_reads_completed_total",
			"Total number of reads completed for this filesystem (best-effort).",
			labels, nil,
		),
		fsWritesCompleted: prometheus.NewDesc(
			"bsd_disk_writes_completed_total",
			"Total number of writes completed for this filesystem (best-effort).",
			labels, nil,
		),
		fsReadBytes: prometheus.NewDesc(
			"bsd_disk_read_bytes_total",
			"Total number of bytes read for this filesystem (best-effort).",
			labels, nil,
		),
		fsWrittenBytes: prometheus.NewDesc(
			"bsd_disk_written_bytes_total",
			"Total number of bytes written for this filesystem (best-effort).",
			labels, nil,
		),
	}
}

// Describe sends the descriptors of each metric over to the provided channel.
// It is used by Prometheus to register the collector.
func (c *DiskUsageCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.fsSize
	ch <- c.fsFree
	ch <- c.fsAvail
	ch <- c.fsUsed
	ch <- c.fsPercent
	ch <- c.fsInodes
	ch <- c.fsInodesFree
	ch <- c.fsInodesUsed
	ch <- c.fsInodesPct
	ch <- c.fsReadsCompleted
	ch <- c.fsWritesCompleted
	ch <- c.fsReadBytes
	ch <- c.fsWrittenBytes
}

// Collect fetches the filesystem statistics and sends them as Prometheus metrics.
// It filters out pseudo-filesystems and handles jail-specific constraints.
func (c *DiskUsageCollector) Collect(ch chan<- prometheus.Metric) {
	mounts, err := getMounts()
	if err != nil {
		return
	}

	jailed := isJailed()

	// Skip pseudo filesystems (node_exporter-style)
	skipFSTypes := map[string]bool{
		"devfs":     true,
		"fdescfs":   true,
		"linprocfs": true,
		"linsysfs":  true,
		"procfs":    true,
		"tmpfs":     true,
		"autofs":    true,
	}

	// Deduplicate mountpoints (and collapse nullfs)
	seen := make(map[string]bool)

	for _, m := range mounts {
		mp := filepath.Clean(m.Mountpoint)
		fsType := m.Fstype

		// Jail-aware: inside a jail, you may want to skip some host-only mounts
		if jailed && skipInJail(mp, fsType) {
			continue
		}

		// Skip pseudo filesystems
		if skipFSTypes[fsType] {
			continue
		}

		// Collapse nullfs: prefer the "real" mount if we see duplicates
		key := mp
		if seen[key] {
			continue
		}
		seen[key] = true

		var st unix.Statfs_t
		if err := unix.Statfs(mp, &st); err != nil {
			continue
		}

		bsize := float64(st.Bsize)
		total := float64(st.Blocks) * bsize
		free := float64(st.Bfree) * bsize
		avail := float64(st.Bavail) * bsize
		used := total - free

		percent := 0.0
		if total > 0 {
			percent = (used / total) * 100.0
		}

		// Inodes
		inodesTotal := float64(st.Files)
		inodesFree := float64(st.Ffree)
		inodesUsed := inodesTotal - inodesFree
		inodesPct := 0.0
		if inodesTotal > 0 {
			inodesPct = (inodesUsed / inodesTotal) * 100.0
		}

		ch <- prometheus.MustNewConstMetric(c.fsSize, prometheus.GaugeValue, total, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsFree, prometheus.GaugeValue, free, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsAvail, prometheus.GaugeValue, avail, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsUsed, prometheus.GaugeValue, used, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsPercent, prometheus.GaugeValue, percent, mp, fsType)

		ch <- prometheus.MustNewConstMetric(c.fsInodes, prometheus.GaugeValue, inodesTotal, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsInodesFree, prometheus.GaugeValue, inodesFree, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsInodesUsed, prometheus.GaugeValue, inodesUsed, mp, fsType)
		ch <- prometheus.MustNewConstMetric(c.fsInodesPct, prometheus.GaugeValue, inodesPct, mp, fsType)

		// Best-effort I/O stats (per-fstype, not per-mount; still useful)
		if ioStats, ok := getFSIOStats(fsType); ok {
			ch <- prometheus.MustNewConstMetric(c.fsReadsCompleted, prometheus.CounterValue, ioStats.Reads, mp, fsType)
			ch <- prometheus.MustNewConstMetric(c.fsWritesCompleted, prometheus.CounterValue, ioStats.Writes, mp, fsType)
			ch <- prometheus.MustNewConstMetric(c.fsReadBytes, prometheus.CounterValue, ioStats.ReadBytes, mp, fsType)
			ch <- prometheus.MustNewConstMetric(c.fsWrittenBytes, prometheus.CounterValue, ioStats.WrittenBytes, mp, fsType)
		}
	}
}

// MountInfo holds basic information about a mountpoint.
type MountInfo struct {
	Mountpoint string
	Fstype     string
}

// getMounts returns mountpoint + fstype using `mount -p`
func getMounts() ([]MountInfo, error) {
	cmd := exec.Command("mount", "-p")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var mounts []MountInfo
	sc := bufio.NewScanner(bytes.NewReader(out))

	for sc.Scan() {
		// Format: <fs> <mountpoint> <type> <opts>
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}

		mounts = append(mounts, MountInfo{
			Mountpoint: fields[1],
			Fstype:     fields[2],
		})
	}

	return mounts, nil
}

// isJailed checks if the current process is running inside a FreeBSD jail.
func isJailed() bool {
	out, err := exec.Command("sysctl", "-n", "security.jail.jailed").Output()
	if err != nil {
		return false
	}
	s := strings.TrimSpace(string(out))
	return s == "1"
}

// skipInJail determines if a mountpoint should be skipped when running inside a jail.
func skipInJail(mp, fstype string) bool {
	// Example: inside a jail, you might skip /dev, /proc, etc.
	if strings.HasPrefix(mp, "/dev") || strings.HasPrefix(mp, "/proc") {
		return true
	}
	return false
}

// FSIOStats contains statistics for filesystem-level I/O operations.
type FSIOStats struct {
	Reads        float64
	Writes       float64
	ReadBytes    float64
	WrittenBytes float64
}

// getFSIOStats: best-effort, per-fstype via sysctl vfs.<fstype>.stats
// This is intentionally simple; you can extend it per-fstype if you want.
func getFSIOStats(fstype string) (FSIOStats, bool) {
	// Example for UFS: vfs.ufs.stats
	// Example for ZFS: kstat.zfs.misc.zio.* (more complex)
	// Here we just stub a minimal UFS example; extend as needed.

	if fstype != "ufs" {
		return FSIOStats{}, false
	}

	out, err := exec.Command("sysctl", "-n", "vfs.ufs.stats").Output()
	if err != nil {
		return FSIOStats{}, false
	}

	// This is highly version-dependent; treat as placeholder.
	// You can parse fields as needed for your FreeBSD version.
	_ = out

	return FSIOStats{}, false
}
