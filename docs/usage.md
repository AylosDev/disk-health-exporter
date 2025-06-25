# Usage Guide

This guide covers how to use the Disk Health Exporter with Prometheus and create effective monitoring setups.

## Basic Usage

### Starting the Exporter

```bash
# Run directly
./disk-health-exporter

# With custom port
PORT=9101 ./disk-health-exporter

# As a service (after installation)
sudo systemctl start disk-health-exporter
```

### Accessing Metrics

The exporter provides metrics on HTTP endpoint:

```bash
# View all metrics
curl http://localhost:9100/metrics

# View specific metric families
curl -s http://localhost:9100/metrics | grep disk_health_status
curl -s http://localhost:9100/metrics | grep raid_array
curl -s http://localhost:9100/metrics | grep tool_available
```

## Prometheus Integration

### Prometheus Configuration

Add the following to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'disk-health'
    static_configs:
      - targets: ['localhost:9100']
    scrape_interval: 30s
    scrape_timeout: 10s
    metrics_path: /metrics
```

### Service Discovery

For multiple hosts, use service discovery:

```yaml
scrape_configs:
  - job_name: 'disk-health'
    dns_sd_configs:
      - names:
          - 'disk-health-exporters.example.com'
        type: 'A'
        port: 9100
    scrape_interval: 30s
```

## Monitoring Strategies

### Health Monitoring

#### Critical Disk Health

```promql
# Alert on critical disk health
disk_health_status == 3

# Alert on unknown disk health
disk_health_status == 0

# Alert on SMART failures
disk_smart_healthy == 0
```

#### Temperature Monitoring

```promql
# High temperature alert
disk_temperature_celsius > 60

# Temperature trend monitoring
increase(disk_temperature_max_celsius[1h]) > 10
```

### Error Monitoring

#### Sector Errors

```promql
# Reallocated sectors increasing
increase(disk_reallocated_sectors_total[1h]) > 0

# Pending sectors present
disk_pending_sectors_total > 0

# Uncorrectable errors increasing
increase(disk_uncorrectable_errors_total[1h]) > 0
```

#### Media Errors (NVMe)

```promql
# NVMe media errors increasing
increase(disk_media_errors_total[1h]) > 0

# Critical warnings present
disk_critical_warning > 0
```

### Wear and Endurance Monitoring

#### SSD Wear Leveling

```promql
# High wear leveling
disk_wear_leveling_percentage > 80

# Rapid wear increase
increase(disk_wear_leveling_percentage[24h]) > 5
```

#### NVMe Endurance

```promql
# High endurance usage
disk_percentage_used > 80

# Low available spare
disk_available_spare_percentage < 20
```

### RAID Monitoring

#### Array Health

```promql
# RAID array not optimal
raid_array_status != 1

# Failed drives in array
raid_array_failed_drives > 0

# Array rebuild in progress
raid_array_rebuild_progress_percentage < 100 and raid_array_rebuild_progress_percentage > 0
```

#### Software RAID

```promql
# Software RAID not clean
software_raid_array_status != 1

# Software RAID sync in progress
software_raid_sync_progress_percentage < 100 and software_raid_sync_progress_percentage > 0
```

### Tool Availability Monitoring

```promql
# Critical tools not available
disk_monitoring_tool_available{tool="smartctl"} == 0
disk_monitoring_tool_available{tool="megacli"} == 0
```

## Alerting Rules

### Prometheus Alerting Rules

Create a file `disk-health-alerts.yml`:

```yaml
groups:
  - name: disk_health
    rules:
      # Critical disk health
      - alert: DiskHealthCritical
        expr: disk_health_status == 3
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "Disk health critical on {{ $labels.device }}"
          description: "Disk {{ $labels.device }} ({{ $labels.model }}) has critical health status"

      # SMART failure
      - alert: DiskSmartFailed
        expr: disk_smart_healthy == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "SMART health check failed for {{ $labels.device }}"
          description: "SMART health check failed for disk {{ $labels.device }} ({{ $labels.model }})"

      # High temperature
      - alert: DiskTemperatureHigh
        expr: disk_temperature_celsius > 60
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High disk temperature on {{ $labels.device }}"
          description: "Disk {{ $labels.device }} temperature is {{ $value }}°C"

      # Reallocated sectors
      - alert: DiskReallocatedSectors
        expr: increase(disk_reallocated_sectors_total[1h]) > 0
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "Reallocated sectors detected on {{ $labels.device }}"
          description: "Disk {{ $labels.device }} has {{ $value }} new reallocated sectors"

      # NVMe endurance
      - alert: NVMeEnduranceHigh
        expr: disk_percentage_used > 80
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "High NVMe endurance usage on {{ $labels.device }}"
          description: "NVMe {{ $labels.device }} endurance usage is {{ $value }}%"

      # RAID array degraded
      - alert: RAIDArrayDegraded
        expr: raid_array_status != 1
        for: 0m
        labels:
          severity: critical
        annotations:
          summary: "RAID array {{ $labels.array_id }} degraded"
          description: "RAID array {{ $labels.array_id }} is in {{ $labels.state }} state"

      # Tool not available
      - alert: MonitoringToolMissing
        expr: disk_monitoring_tool_available{tool=~"smartctl|megacli"} == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk monitoring tool {{ $labels.tool }} not available"
          description: "Critical disk monitoring tool {{ $labels.tool }} is not available"
```

## Grafana Dashboards

### Dashboard Panels

#### Disk Health Overview

```json
{
  "targets": [
    {
      "expr": "disk_health_status",
      "legendFormat": "{{ device }} ({{ model }})"
    }
  ],
  "type": "stat",
  "title": "Disk Health Status"
}
```

#### Temperature Monitoring

```json
{
  "targets": [
    {
      "expr": "disk_temperature_celsius",
      "legendFormat": "{{ device }}"
    }
  ],
  "type": "graph",
  "title": "Disk Temperatures"
}
```

#### RAID Array Status

```json
{
  "targets": [
    {
      "expr": "raid_array_status",
      "legendFormat": "Array {{ array_id }} ({{ raid_level }})"
    }
  ],
  "type": "stat",
  "title": "RAID Array Status"
}
```

### Sample Dashboard Queries

#### Top 10 Hottest Disks

```promql
topk(10, disk_temperature_celsius)
```

#### Disks with Errors

```promql
disk_reallocated_sectors_total > 0 or disk_pending_sectors_total > 0 or disk_uncorrectable_errors_total > 0
```

#### NVMe Endurance Usage

```promql
disk_percentage_used
```

#### Tool Availability Summary

```promql
sum by (tool) (disk_monitoring_tool_available)
```

## Advanced Usage

### Custom Metrics Collection

You can filter metrics based on labels:

```promql
# Only NVMe disks
disk_health_status{interface="NVMe"}

# Only RAID disks
disk_health_status{type="raid"}

# Specific disk models
disk_health_status{model=~".*Samsung.*"}
```

### Trend Analysis

```promql
# Temperature trend over 24 hours
increase(disk_temperature_max_celsius[24h])

# Power-on hours growth
increase(disk_power_on_hours_total[24h])

# Wear leveling progression
increase(disk_wear_leveling_percentage[7d])
```

### Capacity Planning

```promql
# Available spare trend (NVMe)
predict_linear(disk_available_spare_percentage[7d], 86400 * 30)

# Endurance projection (NVMe)
predict_linear(disk_percentage_used[30d], 86400 * 365)
```

## Best Practices

### Monitoring Setup

1. **Set appropriate scrape intervals**: 30-60 seconds for disk metrics
2. **Use meaningful labels**: Include hostname, datacenter, etc.
3. **Set up proper alerting**: Don't ignore warning-level alerts
4. **Monitor tool availability**: Ensure monitoring tools are working

### Alert Thresholds

1. **Temperature**: 55°C warning, 65°C critical
2. **SMART errors**: Any increase is concerning
3. **SSD wear**: 80% warning, 90% critical
4. **RAID status**: Any non-optimal state is critical

### Performance Considerations

1. **Avoid too frequent scraping**: Disk operations can be expensive
2. **Use recording rules**: For complex queries used in dashboards
3. **Set proper timeouts**: Some disk operations may take time
4. **Monitor exporter performance**: Check for timeouts or errors

## Troubleshooting

### Common Issues

#### Missing Metrics

Check tool availability:

```bash
curl -s http://localhost:9100/metrics | grep tool_available
```

#### Incorrect Values

Verify tool output manually:

```bash
smartctl -a /dev/sda
megacli -PDList -aALL
```

#### Performance Issues

Monitor exporter logs and Prometheus scrape durations.
