# FreeBSD Node Exporter

A modular Prometheus exporter for FreeBSD metrics, built with Go. This project provides a lightweight alternative to the full Prometheus `node_exporter`, focusing on key system metrics like CPU and Memory usage on FreeBSD systems.

## Project Structure

- `main.go`: The entry point that sets up the HTTP server and registers collectors.
- `fbsd_node_exporter/collector.go`: Defines the base `NodeCollector` interface that all collectors must implement.
- `fbsd_node_exporter/cpu_collector.go`: Implements the aggregate CPU usage collector using `gopsutil`.
- `fbsd_node_exporter/cpu_core_collector.go`: Implements the per-core CPU usage collector using `gopsutil`.
- `fbsd_node_exporter/mem_collector.go`: Implements the Memory usage collector using `gopsutil`.

## Configuration

The exporter can be configured via environment variables:

- `EXPORTER_NAMESPACE`: The prefix for all exported metrics (default: `mini_node`).
- `EXPORTER_ADDR`: The address and port to listen on (default: `:91001`).

## How to Implement More Collectors

The project is designed to be easily extensible. To add a new collector (e.g., Disk, Network), follow these steps:

### 1. Create a New Collector File
Create a new Go file (e.g., `disk_collector.go`) in the `fbsd_node_exporter/` directory and define a struct for your collector.

```go
package fbsd_node_exporter

import (
    "github.com/prometheus/client_golang/prometheus"
)

type DiskCollector struct {
    readBytes *prometheus.Desc
    // Add more descriptors as needed
}
```

### 2. Implement the Constructor
Create a `NewDiskCollector(namespace string)` function that initializes the Prometheus descriptors.

```go
func NewDiskCollector(namespace string) *DiskCollector {
    return &DiskCollector{
        readBytes: prometheus.NewDesc(
            prometheus.BuildFQName(namespace, "disk", "read_bytes_total"),
            "Total bytes read from disk.",
            nil, nil,
        ),
    }
}
```

### 3. Implement the `Describe` Method
This method is part of the `prometheus.Collector` interface. It sends your descriptors to the provided channel.

```go
func (c *DiskCollector) Describe(ch chan<- *prometheus.Desc) {
    ch <- c.readBytes
}
```

### 4. Implement the `Collect` Method
This method is where you fetch the actual system metrics and send them to Prometheus.

```go
func (c *DiskCollector) Collect(ch chan<- prometheus.Metric) {
    // 1. Fetch data from the system (e.g., using gopsutil or sysctl)
    // 2. Create a metric and send it
    ch <- prometheus.MustNewConstMetric(
        c.readBytes,
        prometheus.CounterValue,
        float64(your_value_here),
    )
}
```

### 5. Register the New Collector
Open `main.go`, initialize your new collector, and register it with the Prometheus registry.

```go
// In main.go:
diskCollector := fbsd_node_exporter.NewDiskCollector(namespace)
reg.MustRegister(diskCollector)
```

## Running the Exporter

To build and run the exporter:

```bash
go build -o fbsd_node_exporter
./fbsd_node_exporter
```

You can then view the metrics at `http://localhost:9100/metrics`.
