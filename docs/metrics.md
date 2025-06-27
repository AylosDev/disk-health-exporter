# Metrics Reference

This document provides a comprehensive reference for all metrics exported by the Disk Health Exporter.

## Disk Health Metrics

### Basic Health and Status

- **`disk_health_status`**: Disk health status with labels: device, type, serial, model, location, interface
  - Values: `0` (Unknown), `1` (OK/Healthy), `2` (Warning), `3` (Critical/Failed)

- **`disk_smart_enabled`**: Whether SMART is enabled
  - Values: `1` (enabled), `0` (disabled)
  - Labels: device, serial, model

- **`disk_smart_healthy`**: SMART overall health assessment
  - Values: `1` (healthy), `0` (unhealthy)
  - Labels: device, serial, model

### Capacity and Physical Properties

- **`disk_capacity_bytes`**: Disk capacity in bytes
  - Labels: device, serial, model, interface

### Temperature Metrics

- **`disk_temperature_celsius`**: Current disk temperature in Celsius
  - Labels: device, serial, model, interface

- **`disk_temperature_max_celsius`**: Maximum recorded disk temperature in Celsius
  - Labels: device, serial, model

- **`disk_temperature_min_celsius`**: Minimum recorded disk temperature in Celsius
  - Labels: device, serial, model

### Power and Lifecycle Metrics

- **`disk_power_on_hours_total`**: Total power-on hours for the disk
  - Labels: device, serial, model

- **`disk_power_cycles_total`**: Total number of power cycles
  - Labels: device, serial, model

## Disk Error Metrics

### Sector Errors

- **`disk_sector_errors_total`**: Total number of disk sector errors
  - Labels: device, serial, model, error_type
  - Error types: `reallocated_sectors`, `pending_sectors`, `uncorrectable_errors`

- **`disk_reallocated_sectors_total`**: Total number of reallocated sectors
  - Labels: device, serial, model

- **`disk_pending_sectors_total`**: Total number of pending sectors
  - Labels: device, serial, model

- **`disk_uncorrectable_errors_total`**: Total number of uncorrectable errors
  - Labels: device, serial, model

### Media and Log Errors

- **`disk_media_errors_total`**: Total number of media errors (NVMe specific)
  - Labels: device, serial, model

- **`disk_error_log_entries_total`**: Total number of error log entries
  - Labels: device, serial, model

## Disk I/O Metrics

- **`disk_data_units_written_total`**: Total data units written
  - Labels: device, serial, model

- **`disk_data_units_read_total`**: Total data units read
  - Labels: device, serial, model

## SSD/NVMe Specific Metrics

### Wear and Endurance

- **`disk_wear_leveling_percentage`**: SSD wear leveling percentage (0-100)
  - Labels: device, serial, model

- **`disk_percentage_used`**: NVMe percentage used (0-100)
  - Labels: device, serial, model

- **`disk_available_spare_percentage`**: NVMe available spare percentage
  - Labels: device, serial, model

### Health Warnings

- **`disk_critical_warning`**: NVMe critical warning flags
  - Labels: device, serial, model

## Hardware RAID Metrics

### Array Status

- **`raid_array_status`**: RAID array status
  - Values: `0` (unknown), `1` (ok), `2` (degraded), `3` (failed)
  - Labels: array_id, raid_level, state, type, controller

### Array Capacity

- **`raid_array_size_bytes`**: RAID array size in bytes
  - Labels: array_id, raid_level, type

- **`raid_array_used_size_bytes`**: RAID array used size in bytes
  - Labels: array_id, raid_level, type

### Drive Counts

- **`raid_array_drives_total`**: Total number of drives in RAID array
  - Labels: array_id, raid_level, type

- **`raid_array_active_drives`**: Number of active drives in RAID array
  - Labels: array_id, raid_level, type

- **`raid_array_spare_drives`**: Number of spare drives in RAID array
  - Labels: array_id, raid_level, type

- **`raid_array_failed_drives`**: Number of failed drives in RAID array
  - Labels: array_id, raid_level, type

### Maintenance Progress

- **`raid_array_rebuild_progress_percentage`**: RAID array rebuild progress (0-100)
  - Labels: array_id, raid_level, type

- **`raid_array_scrub_progress_percentage`**: RAID array scrub progress (0-100)
  - Labels: array_id, raid_level, type

## Software RAID Metrics

- **`software_raid_array_status`**: Software RAID array status
  - Values: `0` (unknown), `1` (clean), `2` (degraded), `3` (failed)
  - Labels: device, level, state

- **`software_raid_sync_progress_percentage`**: Software RAID sync progress (0-100)
  - Labels: device, level, sync_action

- **`software_raid_array_size_bytes`**: Software RAID array size in bytes
  - Labels: device, level

## RAID Controller Battery Metrics

RAID controllers often have backup batteries (BBU - Backup Battery Unit) to ensure data integrity during power failures. These metrics provide comprehensive monitoring of battery health and status.

### Battery Status and Health

- **`raid_battery_status`**: RAID controller battery status
  - Values: `0` (unknown), `1` (optimal), `2` (warning), `3` (critical)
  - Labels: adapter_id, battery_type, state, controller

- **`raid_battery_missing`**: Battery pack missing indicator
  - Values: `0` (present), `1` (missing)
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_replacement_required`**: Battery replacement required indicator
  - Values: `0` (not required), `1` (required)
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_capacity_low`**: Battery remaining capacity low indicator
  - Values: `0` (capacity normal), `1` (capacity low)
  - Labels: adapter_id, battery_type, controller

### Battery Physical Measurements

- **`raid_battery_voltage_millivolts`**: Battery voltage in millivolts
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_current_milliamps`**: Battery current in milliamps
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_temperature_celsius`**: Battery temperature in Celsius
  - Labels: adapter_id, battery_type, controller

### Battery Energy and Capacity

- **`raid_battery_pack_energy_joules`**: Battery pack energy in joules
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_capacitance`**: Battery capacitance
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_backup_charge_time_hours`**: Battery backup charge time in hours
  - Labels: adapter_id, battery_type, controller

### Battery Design Specifications

- **`raid_battery_design_capacity_joules`**: Battery design capacity in joules
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_design_voltage_millivolts`**: Battery design voltage in millivolts
  - Labels: adapter_id, battery_type, controller

### Battery Maintenance

- **`raid_battery_learn_cycle_active`**: Battery learn cycle active indicator
  - Values: `0` (not active), `1` (active)
  - Labels: adapter_id, battery_type, controller

- **`raid_battery_auto_learn_period_days`**: Battery auto learn period in days
  - Labels: adapter_id, battery_type, controller

## Tool Availability Metrics

- **`disk_monitoring_tool_available`**: Whether a disk monitoring tool is available
  - Values: `1` (available), `0` (not available)
  - Labels: tool, version
  - Tools: `smartctl`, `megacli`, `storcli`, `arcconf`, `mdadm`, `nvme`, `hdparm`, `lsblk`, `diskutil`, `zpool`

## Exporter Metrics

- **`disk_health_exporter_up`**: Whether the disk health exporter is up and running
  - Values: `1` (up), `0` (down)

## Health Status Values Reference

### Disk Health Status

- `0`: Unknown status
- `1`: OK/Healthy
- `2`: Warning (e.g., rebuilding, high temperature)
- `3`: Critical/Failed

### RAID Array Status

- `0`: Unknown status
- `1`: Optimal/OK
- `2`: Degraded/Rebuilding
- `3`: Failed/Offline

### Software RAID Status

- `0`: Unknown status
- `1`: Clean/Active
- `2`: Degraded/Recovering
- `3`: Failed/Inactive

### RAID Battery Status

- `0`: Unknown status
- `1`: Optimal/Charging
- `2`: Warning/Discharging/Low
- `3`: Critical/Failed/Missing

## Label Descriptions

### Common Labels

- **device**: Device path (e.g., `/dev/sda`, `/dev/nvme0n1`)
- **serial**: Device serial number
- **model**: Device model name
- **interface**: Interface type (SATA, NVMe, SAS, etc.)
- **type**: Device type (regular, raid, nvme, macos-smart, etc.)

### RAID-Specific Labels

- **array_id**: RAID array identifier
- **raid_level**: RAID level (raid0, raid1, raid5, etc.)
- **state**: Current array state (Optimal, Degraded, Failed, etc.)
- **controller**: RAID controller type (MegaCLI, StorCLI, mdadm, etc.)
- **adapter_id**: RAID controller adapter identifier
- **battery_type**: Battery type (e.g., CVPM02, iBBU, etc.)

### Error-Specific Labels

- **error_type**: Type of error (reallocated_sectors, pending_sectors, uncorrectable_errors)
- **sync_action**: Software RAID sync action (resync, recover, check)
