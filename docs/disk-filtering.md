# Disk Filtering

The Disk Health Exporter supports filtering disks to monitor specific devices and automatically ignores certain device types that are typically not relevant for monitoring.

## User-Configurable Filtering

### Target Specific Disks

You can specify exactly which disks to monitor using the `-target-disks` flag:

```bash
# Monitor only specific disks
./disk-health-exporter -target-disks "/dev/sda,/dev/nvme0n1"

# Monitor multiple NVMe drives
./disk-health-exporter -target-disks "/dev/nvme0n1,/dev/nvme1n1,/dev/nvme2n1"

# Monitor a single disk
./disk-health-exporter -target-disks "/dev/sda"
```

### Environment Variable Support

You can also use the `TARGET_DISKS` environment variable:

```bash
# Set via environment variable
export TARGET_DISKS="/dev/sda,/dev/nvme0n1"
./disk-health-exporter

# Or inline
TARGET_DISKS="/dev/sdb" ./disk-health-exporter
```

### Command Line Priority

Command-line flags take priority over environment variables:

```bash
# This will monitor /dev/sda (flag overrides env var)
TARGET_DISKS="/dev/sdb" ./disk-health-exporter -target-disks "/dev/sda"
```

## Automatic Filtering (Internal)

The exporter automatically ignores certain device types that are typically not useful for monitoring:

### Ignored Device Patterns

- **`/dev/loop*`** - Loop devices (mounted images, snaps, etc.)
- **`/dev/ram*`** - RAM disks and tmpfs mounts
- **`/dev/dm-*`** - Device mapper devices (LVM, LUKS - monitored via underlying devices)

### Examples of Ignored Devices

```text
/dev/loop0    # Snap packages
/dev/loop1    # Mounted ISO files
/dev/ram0     # RAM disk
/dev/dm-0     # LVM logical volume (underlying /dev/sda monitored instead)
```

## Filtering Behavior

### Default Behavior (No Target Disks)

When no target disks are specified:

1. ✅ **Include**: All detected physical disks (SATA, NVMe, SAS, etc.)
2. ❌ **Exclude**: Devices matching ignore patterns
3. ❌ **Exclude**: Devices that are part of software RAID arrays (individual components)

### With Target Disks Specified

When target disks are specified:

1. ✅ **Include**: Only disks in the target list
2. ❌ **Exclude**: All other disks, even if they would normally be detected
3. ❌ **Exclude**: Target disks that match ignore patterns (safety override)

## Logging and Debugging

The exporter provides detailed logging about filtering decisions:

```text
2025/06/26 12:55:11 Target disks specified: [/dev/sda /dev/nvme0n1]
2025/06/26 12:55:11 Ignore patterns: [/dev/loop /dev/ram /dev/dm-]
2025/06/26 12:55:11 Including target disk: /dev/sda
2025/06/26 12:55:11 Skipping disk /dev/sdb (not in target list)
2025/06/26 12:55:11 Ignoring disk /dev/loop0 (matches ignore pattern: /dev/loop)
2025/06/26 12:55:11 Found 3 disks detected, 1 disks after filtering
```

Use `-log-level debug` for even more detailed output.

## Use Cases

### Monitor Only NVMe Drives

```bash
./disk-health-exporter -target-disks "/dev/nvme0n1,/dev/nvme1n1"
```

### Monitor Only System Drive

```bash
./disk-health-exporter -target-disks "/dev/sda"
```

### Monitor Specific Drives in a Server

```bash
./disk-health-exporter -target-disks "/dev/sda,/dev/sdb,/dev/sdc,/dev/sdd"
```

### Exclude Problematic Drives

If a particular drive causes issues, simply don't include it in the target list:

```bash
# Monitor all except /dev/sdc
./disk-health-exporter -target-disks "/dev/sda,/dev/sdb,/dev/sdd"
```

## Best Practices

1. **Use target disks for servers**: In production environments, explicitly specify which disks to monitor
2. **Test first**: Run with `-log-level debug` to see what disks would be monitored
3. **Use absolute paths**: Always use full device paths like `/dev/sda`, not relative paths
4. **Monitor underlying devices**: For RAID/LVM setups, monitor the underlying physical devices
5. **Be specific**: Don't rely on device naming patterns, use explicit device names

## Troubleshooting

### No Disks Found

```text
Found 0 disks after filtering
```

**Possible causes:**

- Target disks don't exist: Check `/dev/` for actual device names
- Target disks match ignore patterns: Remove ignore pattern override if needed
- Permissions: Ensure the exporter can access device files

### Wrong Disks Monitored

```text
Found 5 disks detected, 3 disks after filtering
```

**Solutions:**

- Use `-target-disks` to be explicit about which disks to monitor
- Check the log output to see which disks are being included/excluded
- Use `lsblk` or `fdisk -l` to see available disks
