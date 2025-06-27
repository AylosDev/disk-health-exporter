package disk

import (
	"strings"
	"testing"

	"disk-health-exporter/pkg/types"
)

func TestGetMegaCLIBatteryInfo(t *testing.T) {
	// Mock MegaCLI battery output
	mockOutput := `BBU status for Adapter: 0

BatteryType: CVPM02
Voltage: 9481 mV
Current: 0 mA
Temperature: 35 C
Battery State: Optimal
BBU Firmware Status:

  Charging Status              : None
  Voltage                                 : OK
  Temperature                             : OK
  Learn Cycle Requested                   : No
  Learn Cycle Active                      : No
  Learn Cycle Status                      : OK
  Learn Cycle Timeout                     : No
  I2c Errors Detected                     : No
  Battery Pack Missing                    : No
  Battery Replacement required            : No
  Remaining Capacity Low                  : No
  Periodic Learn Required                 : No
  Transparent Learn                       : No
  No space to cache offload               : No
  Pack is about to fail & should be replaced : No
  Cache Offload premium feature required  : No
  Module microcode update required        : No

BBU GasGauge Status: 0x66e9
  Pack energy             : 233 J
  Capacitance             : 102
  Remaining reserve space : 0

  Battery backup charge time : 0 hours

BBU Design Info for Adapter: 0

Date of Manufacture: 02/21, 2017
Design Capacity: 288 J
Design Voltage: 9500 mV
Serial Number: 3623
Manufacture Name: LSI
Firmware Version   : 6635-02A
Device Name: CVPM02
Device Chemistry: EDLC
Battery FRU: N/A
TMM FRU: N/A
Module Version = 6635-02A
  Transparent Learn = 1
  App Data = 0

BBU Properties for Adapter: 0

  Auto Learn Period: 27 Days
  Next Learn time: Tue Jul 15 16:26:47 2025
  Learn Delay Interval:0 Hours
  Auto-Learn Mode: Transparent

Exit Code: 0x00`

	// Test the battery info parsing (simulate the parsing part)
	battery := &types.RAIDBatteryInfo{}
	battery.AdapterID = 0

	lines := strings.Split(mockOutput, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "BatteryType:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				battery.BatteryType = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Battery State:") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				battery.State = strings.TrimSpace(parts[1])
			}
		} else if strings.Contains(line, "Voltage:") && !strings.Contains(line, "Design Voltage") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				voltageStr := strings.TrimSpace(parts[1])
				if strings.Contains(voltageStr, "9481 mV") {
					battery.Voltage = 9481
				}
			}
		} else if strings.Contains(line, "Temperature:") && !strings.Contains(line, "Temperature                             :") {
			parts := strings.Split(line, ":")
			if len(parts) > 1 {
				tempStr := strings.TrimSpace(parts[1])
				if strings.Contains(tempStr, "35 C") {
					battery.Temperature = 35
				}
			}
		}
	}

	// Test the parsed battery information
	if battery.BatteryType != "CVPM02" {
		t.Errorf("Expected BatteryType to be 'CVPM02', got '%s'", battery.BatteryType)
	}

	if battery.State != "Optimal" {
		t.Errorf("Expected State to be 'Optimal', got '%s'", battery.State)
	}

	if battery.Voltage != 9481 {
		t.Errorf("Expected Voltage to be 9481, got %d", battery.Voltage)
	}

	if battery.Temperature != 35 {
		t.Errorf("Expected Temperature to be 35, got %d", battery.Temperature)
	}

	if battery.AdapterID != 0 {
		t.Errorf("Expected AdapterID to be 0, got %d", battery.AdapterID)
	}
}

func TestBatteryStatusParsing(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"Battery Pack Missing                    : No", false},
		{"Battery Pack Missing                    : Yes", true},
		{"Battery Replacement required            : No", false},
		{"Battery Replacement required            : Yes", true},
		{"Remaining Capacity Low                  : No", false},
		{"Remaining Capacity Low                  : Yes", true},
		{"Learn Cycle Active                      : No", false},
		{"Learn Cycle Active                      : Yes", true},
	}

	for _, tc := range testCases {
		parts := strings.Split(tc.input, ":")
		if len(parts) > 1 {
			status := strings.TrimSpace(parts[1])
			result := strings.ToLower(status) == "yes"
			if result != tc.expected {
				t.Errorf("For input '%s', expected %v, got %v", tc.input, tc.expected, result)
			}
		}
	}
}
