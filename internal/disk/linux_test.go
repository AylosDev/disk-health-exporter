package disk

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"disk-health-exporter/pkg/types"
)

func TestParseMegaCLIOutput(t *testing.T) {
	// Sample MegaCLI output
	sampleOutput := `Adapter 0 -- Virtual Drive Information:
Virtual Drive: 0 (Target Id: 0)
Name                :
RAID Level          : Primary-5, Secondary-0, RAID Level Qualifier-3
Size                : 113.795 TB
Sector Size         : 4096
Is VD emulated      : No
Parity Size         : 12.643 TB
State               : Optimal
Strip Size          : 2.0 MB
Number Of Drives    : 10
Span Depth          : 1
Default Cache Policy: WriteBack, ReadAhead, Direct, No Write Cache if Bad BBU
Current Cache Policy: WriteBack, ReadAhead, Direct, No Write Cache if Bad BBU
Default Access Policy: Read/Write
Current Access Policy: Read/Write
Disk Cache Policy   : Disk's Default
Encryption Type     : None
Bad Blocks Exist: No
PI type: No PI

Is VD Cached: No

Virtual Drive: 1 (Target Id: 1)
Name                :
RAID Level          : Primary-1, Secondary-0, RAID Level Qualifier-0
Size                : 50.0 GB
State               : Optimal
Number Of Drives    : 2`

	// Parse the sample output using the same logic as checkMegaCLI
	raidArrays := parseMegaCLIOutput(sampleOutput)

	// Test expectations
	expectedArrays := []struct {
		arrayID   string
		raidLevel string
		size      int64
		state     string
		numDrives int
	}{
		{
			arrayID:   "0",
			raidLevel: "RAID 5",
			size:      125118925682769, // 113.795 TB in bytes
			state:     "Optimal",
			numDrives: 10,
		},
		{
			arrayID:   "1",
			raidLevel: "RAID 1",
			size:      53687091200, // 50.0 GB in bytes
			state:     "Optimal",
			numDrives: 2,
		},
	}

	if len(raidArrays) != len(expectedArrays) {
		t.Fatalf("Expected %d RAID arrays, got %d", len(expectedArrays), len(raidArrays))
	}

	for i, expected := range expectedArrays {
		array := raidArrays[i]

		if array.ArrayID != expected.arrayID {
			t.Errorf("Array %d: expected ArrayID %s, got %s", i, expected.arrayID, array.ArrayID)
		}

		if array.RaidLevel != expected.raidLevel {
			t.Errorf("Array %d: expected RaidLevel %s, got %s", i, expected.raidLevel, array.RaidLevel)
		}

		if array.Size != expected.size {
			t.Errorf("Array %d: expected Size %d, got %d", i, expected.size, array.Size)
		}

		if array.State != expected.state {
			t.Errorf("Array %d: expected State %s, got %s", i, expected.state, array.State)
		}

		if array.NumDrives != expected.numDrives {
			t.Errorf("Array %d: expected NumDrives %d, got %d", i, expected.numDrives, array.NumDrives)
		}

		if array.Type != "hardware" {
			t.Errorf("Array %d: expected Type 'hardware', got %s", i, array.Type)
		}

		if array.Controller != "MegaCLI" {
			t.Errorf("Array %d: expected Controller 'MegaCLI', got %s", i, array.Controller)
		}
	}
}

func TestParseSizeToBytes(t *testing.T) {
	testCases := []struct {
		input    string
		expected int64
	}{
		{"113.795 TB", 125118925682769}, // Actual calculation: 113.795 * 1024^4
		{"50.0 GB", 53687091200},
		{"1.5 MB", 1572864},
		{"512 KB", 524288},
		{"100 B", 100},
		{"2 TB", 2199023255552},
		{"", 0},
		{"invalid", 0},
		{"100", 0},    // Missing unit
		{"GB 100", 0}, // Wrong format
	}

	for _, tc := range testCases {
		result := parseSizeToBytes(tc.input)
		if result != tc.expected {
			t.Errorf("parseSizeToBytes(%q) = %d, expected %d", tc.input, result, tc.expected)
		}
	}
}

func TestGetRaidStatusValue(t *testing.T) {
	testCases := []struct {
		state    string
		expected int
	}{
		{"Optimal", 1},
		{"optimal", 1},
		{"OPTIMAL", 1},
		{"Degraded", 2},
		{"degraded", 2},
		{"Rebuilding", 2},
		{"Failed", 3},
		{"failed", 3},
		{"Offline", 3},
		{"Unknown", 0},
		{"", 0},
	}

	for _, tc := range testCases {
		result := getRaidStatusValue(tc.state)
		if result != tc.expected {
			t.Errorf("getRaidStatusValue(%q) = %d, expected %d", tc.state, result, tc.expected)
		}
	}
}

func TestParseMegaCLIRaidLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"Primary-5, Secondary-0, RAID Level Qualifier-3", "RAID 5"},
		{"Primary-1, Secondary-0, RAID Level Qualifier-0", "RAID 1"},
		{"Primary-0, Secondary-0, RAID Level Qualifier-0", "RAID 0"},
		{"Primary-6, Secondary-0, RAID Level Qualifier-3", "RAID 6"},
		{"Primary-10, Secondary-0, RAID Level Qualifier-3", "RAID 10"},
		{"RAID-5", "RAID-5"},           // Fallback case
		{"Simple RAID", "Simple RAID"}, // No Primary- pattern
		{"", ""},
	}

	for _, tc := range testCases {
		result := parseMegaCLIRaidLevel(tc.input)
		if result != tc.expected {
			t.Errorf("parseMegaCLIRaidLevel(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

// Helper function to parse MegaCLI output (extracted from checkMegaCLI logic)
func parseMegaCLIOutput(output string) []types.RAIDInfo {
	var raidArrays []types.RAIDInfo
	lines := strings.Split(output, "\n")
	var currentArray types.RAIDInfo

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "Virtual Drive:") {
			// If we have a previous array, save it
			if currentArray.ArrayID != "" {
				raidArrays = append(raidArrays, currentArray)
			}
			// Start new array
			currentArray = types.RAIDInfo{
				Type:       "hardware",
				Controller: "MegaCLI",
			}

			// Extract array ID
			re := regexp.MustCompile(`Virtual Drive: (\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				currentArray.ArrayID = matches[1]
			}
		} else if strings.Contains(line, "RAID Level") {
			// Extract RAID level
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				raidLevelStr := strings.TrimSpace(parts[1])
				currentArray.RaidLevel = parseMegaCLIRaidLevel(raidLevelStr)
			}
		} else if strings.Contains(line, "Size") && strings.Contains(line, ":") &&
			!strings.Contains(line, "Sector Size") &&
			!strings.Contains(line, "Parity Size") &&
			!strings.Contains(line, "Strip Size") {
			// Extract array size
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				sizeStr := strings.TrimSpace(parts[1])
				currentArray.Size = parseSizeToBytes(sizeStr)
			}
		} else if strings.Contains(line, "Number Of Drives") {
			// Extract number of drives
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				numDrivesStr := strings.TrimSpace(parts[1])
				if num, err := strconv.Atoi(numDrivesStr); err == nil && num > 0 {
					currentArray.NumDrives = num
				}
			}
		} else if strings.Contains(line, "State") && strings.Contains(line, ":") {
			// Extract state
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				state := strings.TrimSpace(parts[1])
				currentArray.State = state
				currentArray.Status = getRaidStatusValue(state)
			}
		}
	}

	// Don't forget the last array
	if currentArray.ArrayID != "" {
		raidArrays = append(raidArrays, currentArray)
	}

	return raidArrays
}

// Helper function to parse MegaCLI RAID level format
func parseMegaCLIRaidLevel(raidLevelStr string) string {
	if strings.Contains(raidLevelStr, "Primary-") {
		re := regexp.MustCompile(`Primary-(\d+)`)
		matches := re.FindStringSubmatch(raidLevelStr)
		if len(matches) > 1 {
			return "RAID " + matches[1]
		}
	}
	return raidLevelStr
}

func TestIsPartOfSoftwareRAID(t *testing.T) {
	// This test would require mocking /proc/mdstat or similar
	// For now, we'll test the basic functionality

	// Test with empty/non-existent device
	result := isPartOfSoftwareRAID("/dev/nonexistent")
	if result {
		t.Errorf("Expected false for non-existent device, got %v", result)
	}
}

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on most systems
	if !commandExists("ls") {
		t.Error("Expected 'ls' command to exist")
	}

	// Test with a command that shouldn't exist
	if commandExists("definitely_does_not_exist_command_12345") {
		t.Error("Expected non-existent command to return false")
	}
}
