package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	DiskHealthStatus *prometheus.GaugeVec
	DiskTemperature  *prometheus.GaugeVec
	RaidArrayStatus  *prometheus.GaugeVec
	DiskSectorErrors *prometheus.GaugeVec
	DiskPowerOnHours *prometheus.GaugeVec
	ExporterUp       prometheus.Gauge
}

// New creates and registers all metrics
func New() *Metrics {
	m := &Metrics{
		DiskHealthStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_health_status",
				Help: "Disk health status (0=unknown, 1=ok, 2=warning, 3=critical)",
			},
			[]string{"device", "type", "serial", "model", "location"},
		),
		DiskTemperature: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_temperature_celsius",
				Help: "Disk temperature in Celsius",
			},
			[]string{"device", "serial", "model"},
		),
		RaidArrayStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "raid_array_status",
				Help: "RAID array status (0=unknown, 1=ok, 2=degraded, 3=failed)",
			},
			[]string{"array_id", "raid_level", "state"},
		),
		DiskSectorErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_sector_errors_total",
				Help: "Total number of disk sector errors",
			},
			[]string{"device", "serial", "model", "error_type"},
		),
		DiskPowerOnHours: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "disk_power_on_hours_total",
				Help: "Total power-on hours for the disk",
			},
			[]string{"device", "serial", "model"},
		),
		ExporterUp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "disk_health_exporter_up",
				Help: "Whether the disk health exporter is up and running",
			},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.DiskHealthStatus,
		m.DiskTemperature,
		m.RaidArrayStatus,
		m.DiskSectorErrors,
		m.DiskPowerOnHours,
		m.ExporterUp,
	)

	return m
}

// Reset clears all metrics
func (m *Metrics) Reset() {
	m.DiskHealthStatus.Reset()
	m.DiskTemperature.Reset()
	m.RaidArrayStatus.Reset()
	m.DiskSectorErrors.Reset()
	m.DiskPowerOnHours.Reset()
}
