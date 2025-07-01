# Development Guide

This guide covers development setup, architecture, and contribution guidelines for the Disk Health Exporter.

## Project Structure

```
disk-health-exporter/
├── cmd/
│   └── disk-health-exporter/    # Main application entry point
│       ├── main.go              # Application main function
│       └── main_test.go         # Main function tests
├── internal/                    # Private application code
│   ├── collector/               # Metrics collection logic
│   │   └── collector.go         # Main collector implementation
│   ├── config/                  # Configuration management
│   │   ├── config.go            # Configuration struct and loading
│   │   └── config_test.go       # Configuration tests
│   ├── disk/                    # Disk detection and monitoring
│   │   ├── common.go            # Common disk detection functions
│   │   ├── linux.go             # Linux-specific disk detection
│   │   └── macos.go             # macOS-specific disk detection
│   └── metrics/                 # Prometheus metrics definitions
│       └── metrics.go           # Metrics registration and management
├── pkg/
│   └── types/                   # Shared types and structs
│       ├── types.go             # Type definitions
│       └── types_test.go        # Type tests
├── scripts/                     # Installation and utility scripts
│   ├── install.sh               # Universal installation script
│   ├── test.sh                  # Universal testing script
│   └── demo.sh                  # System capabilities demo
├── deployments/                 # Deployment configurations
│   ├── Dockerfile               # Docker container definition
│   └── disk-health-exporter.service # systemd service file
├── docs/                        # Documentation
│   ├── metrics.md               # Metrics reference
│   ├── installation.md          # Installation guide
│   ├── usage.md                 # Usage guide
│   └── development.md           # This file
├── go.mod                       # Go module definition
├── go.sum                       # Go module checksums
├── Makefile                     # Build automation
└── README.md                    # Basic project information
```

## Development Setup

### Prerequisites

- Go 1.19 or later
- Make (optional, for using Makefile)
- Docker (optional, for container development)

### Local Development

```bash
# Clone the repository
git clone <repository-url>
cd disk-health-exporter

# Install dependencies
go mod download

# Build the project
make build
# or
go build -o disk-health-exporter ./cmd/disk-health-exporter

# Run tests
make test
# or
go test ./...

# Run the application
./disk-health-exporter
```

### Development Tools

#### Useful Make Targets

```bash
make build          # Build the binary
make test           # Run tests
make clean          # Clean build artifacts
make install        # Install as system service
make docker-build   # Build Docker image
```

#### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific test
go test ./internal/disk -v
```

## Architecture Overview

### Core Components

#### 1. Collector (`internal/collector/`)

The collector is responsible for:

- Orchestrating metric collection
- Managing collection intervals
- Handling OS-specific collection strategies
- Updating Prometheus metrics

Key functions:

- `New()`: Creates a new collector instance
- `Start()`: Begins the metric collection loop
- `updateMetrics()`: Collects and updates all metrics

#### 2. Disk Manager (`internal/disk/`)

The disk manager handles:

- Tool detection and availability
- Multi-tool disk discovery
- SMART data extraction
- RAID array detection

Key files:

- `common.go`: Shared functionality and tool detection
- `linux.go`: Linux-specific implementations
- `macos.go`: macOS-specific implementations

#### 3. Metrics (`internal/metrics/`)

The metrics package manages:

- Prometheus metric registration
- Metric definitions and labels
- Metric reset functionality

#### 4. Types (`pkg/types/`)

Shared type definitions:

- `DiskInfo`: Comprehensive disk information
- `RAIDInfo`: RAID array information
- `SoftwareRAIDInfo`: Software RAID specifics
- `ToolInfo`: Available tool information

### Data Flow

1. **Initialization**
   - Detect available tools (smartctl, megacli, etc.)
   - Register Prometheus metrics
   - Start HTTP server

2. **Collection Loop**
   - Determine OS type
   - Execute OS-specific collection
   - Update Prometheus metrics
   - Wait for next interval

3. **Tool Integration**
   - Try multiple tools for disk detection
   - Combine information from different sources
   - Fallback gracefully when tools are unavailable

## Adding New Features

### Adding a New Monitoring Tool

1. **Add tool to ToolInfo** (`pkg/types/types.go`):

   ```go
   type ToolInfo struct {
       // ...existing tools...
       NewTool bool `json:"new_tool"`
       NewToolVersion string `json:"new_tool_version"`
   }
   ```

2. **Add detection logic** (`internal/disk/common.go`):

   ```go
   func (m *Manager) detectTools() {
       // ...existing detection...
       m.tools.NewTool = commandExists("newtool")
       
       if m.tools.NewTool {
           if version, err := getToolVersion("newtool", "--version"); err == nil {
               m.tools.NewToolVersion = version
           }
       }
   }
   ```

3. **Implement tool-specific logic** (in appropriate OS file):

   ```go
   func (m *Manager) getDisksFromNewTool() []types.DiskInfo {
       var disks []types.DiskInfo
       
       if !m.tools.NewTool {
           return disks
       }
       
       // Tool-specific implementation
       return disks
   }
   ```

4. **Integrate into collection** (`internal/disk/linux.go` or `macos.go`):

   ```go
   // Add to getRegularDisksMultiTool or similar function
   if m.tools.NewTool {
       newToolDisks := m.getDisksFromNewTool()
       for _, disk := range newToolDisks {
           if !seenDevices[disk.Device] {
               allDisks = append(allDisks, disk)
               seenDevices[disk.Device] = true
           }
       }
   }
   ```

### Adding New Metrics

1. **Add metric to Metrics struct** (`internal/metrics/metrics.go`):

   ```go
   type Metrics struct {
       // ...existing metrics...
       NewMetric *prometheus.GaugeVec
   }
   ```

2. **Initialize metric in New()** function:

   ```go
   NewMetric: prometheus.NewGaugeVec(
       prometheus.GaugeOpts{
           Name: "disk_new_metric",
           Help: "Description of the new metric",
       },
       []string{"device", "serial", "model"},
   ),
   ```

3. **Register metric**:

   ```go
   prometheus.MustRegister(
       // ...existing metrics...
       m.NewMetric,
   )
   ```

4. **Add to Reset() function**:

   ```go
   func (m *Metrics) Reset() {
       // ...existing resets...
       m.NewMetric.Reset()
   }
   ```

5. **Update metric in collector** (`internal/collector/collector.go`):

   ```go
   func (c *Collector) updateComprehensiveDiskMetrics(disks []types.DiskInfo) {
       for _, disk := range disks {
           // ...existing metric updates...
           
           if disk.NewValue > 0 {
               c.metrics.NewMetric.WithLabelValues(
                   disk.Device,
                   disk.Serial,
                   disk.Model,
               ).Set(float64(disk.NewValue))
           }
       }
   }
   ```

### Adding OS Support

1. **Create new OS file** (`internal/disk/newos.go`):

   ```go
   //go:build newos
   
   package disk
   
   func (m *Manager) GetNewOSDisks() []types.DiskInfo {
       // OS-specific implementation
   }
   ```

2. **Add to collector** (`internal/collector/collector.go`):

   ```go
   func (c *Collector) updateMetrics() {
       osType := runtime.GOOS
       
       switch osType {
       case "linux":
           c.collectLinuxMetrics()
       case "darwin":
           c.collectMacOSMetrics()
       case "newos":
           c.collectNewOSMetrics()
       default:
           c.collectFallbackMetrics()
       }
   }
   ```

## Testing

### Unit Tests

Write tests for all new functionality:

```go
func TestNewFunction(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"
    
    // Act
    result := NewFunction(input)
    
    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### Integration Tests

Test tool integration:

```go
func TestToolIntegration(t *testing.T) {
    if !commandExists("smartctl") {
        t.Skip("smartctl not available")
    }
    
    // Test actual tool integration
    disks := getSmartCtlInfo("/dev/sda")
    
    // Validate results
    if len(disks) == 0 {
        t.Error("No disks detected")
    }
}
```

### Mock Testing

For testing without requiring actual tools:

```go
type MockDiskManager struct{}

func (m *MockDiskManager) GetLinuxDisks() ([]types.DiskInfo, []types.RAIDInfo) {
    return []types.DiskInfo{
        {Device: "/dev/sda", Health: "OK"},
    }, []types.RAIDInfo{}
}
```

## Code Style and Standards

### Go Code Style

- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Handle errors appropriately
- Use context for cancellation where appropriate

### Error Handling

```go
// Good: Specific error handling
output, err := exec.Command("smartctl", "--scan").Output()
if err != nil {
    log.Printf("Error scanning for devices: %v", err)
    return disks
}

// Bad: Ignoring errors
output, _ := exec.Command("smartctl", "--scan").Output()
```

### Logging

```go
// Use structured logging
log.Printf("Tool detection complete: smartctl=%v, megacli=%v", 
    m.tools.SmartCtl, m.tools.MegaCLI)

// Include context in error messages
log.Printf("Error getting smartctl info for %s: %v", device, err)
```

## Contributing

### Development Workflow

1. Fork the repository
2. Create a feature branch
3. Make changes and add tests
4. Ensure all tests pass
5. Submit a pull request

### Pull Request Guidelines

- Include a clear description of changes
- Add tests for new functionality
- Update documentation as needed
- Ensure CI passes
- Follow code style guidelines

### Commit Messages

Use clear, descriptive commit messages:

```
feat: add support for Adaptec RAID controllers

- Add arcconf tool detection
- Implement Adaptec RAID array discovery
- Add comprehensive error handling
- Update documentation

Closes #123
```

## Debugging

### Debug Logging

Enable verbose logging during development:

```go
log.SetLevel(log.DebugLevel)
```

### Tool Testing

Test individual tools manually:

```bash
# Test smartctl
smartctl --scan
smartctl -a /dev/sda

# Test MegaCLI
megacli -PDList -aALL
megacli -LDInfo -Lall -aALL
```

### Metric Validation

Verify metrics are correctly formatted:

```bash
# Check metric format
curl -s http://localhost:9100/metrics | promtool check metrics

# Validate specific metrics
curl -s http://localhost:9100/metrics | grep disk_health_status
```

## Performance Considerations

### Optimization Guidelines

- Cache tool detection results
- Avoid excessive system calls
- Use efficient data structures
- Implement proper timeouts
- Handle large numbers of disks gracefully

### Monitoring Performance

- Track collection duration
- Monitor memory usage
- Watch for goroutine leaks
- Validate metric cardinality

### Scalability

- Test with many disks (100+)
- Validate memory usage patterns
- Ensure reasonable collection times
- Test concurrent access patterns

## References

- [megacli](https://docs.broadcom.com/docs/12352815)
-^[storcli](https://docs.broadcom.com/doc/12352476)
- [smartctl](https://www.smartmontools.org/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang/tree/main/examples)
- [Megacli usful commands](https://gist.github.com/dubcl/ee3c85d561cc39cc4096276b728b1502)
