package types

import "testing"

func TestHealthStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		status   HealthStatus
		expected int
	}{
		{"Unknown", HealthStatusUnknown, 0},
		{"OK", HealthStatusOK, 1},
		{"Warning", HealthStatusWarning, 2},
		{"Critical", HealthStatusCritical, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.status) != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, int(tt.status))
			}
		})
	}
}

func TestDiskInfoStruct(t *testing.T) {
	disk := DiskInfo{
		Device:      "/dev/sda",
		Serial:      "123456",
		Model:       "Test Disk",
		Health:      "OK",
		Temperature: 45.5,
		Type:        "regular",
		Location:    "internal",
	}

	if disk.Device != "/dev/sda" {
		t.Errorf("Expected device /dev/sda, got %s", disk.Device)
	}

	if disk.Temperature != 45.5 {
		t.Errorf("Expected temperature 45.5, got %f", disk.Temperature)
	}
}

func TestRAIDInfoStruct(t *testing.T) {
	raid := RAIDInfo{
		ArrayID:   "0",
		RaidLevel: "RAID1",
		State:     "OPTIMAL",
		Status:    1,
	}

	if raid.ArrayID != "0" {
		t.Errorf("Expected array ID 0, got %s", raid.ArrayID)
	}

	if raid.Status != 1 {
		t.Errorf("Expected status 1, got %d", raid.Status)
	}
}
