package tools

import (
	"os/exec"
	"strings"
)

// commandExists checks if a command is available in the system PATH
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// getToolVersion gets the version of a tool
func getToolVersion(tool string, versionFlag string) (string, error) {
	output, err := exec.Command(tool, versionFlag).Output()
	if err != nil {
		return "", err
	}

	// Extract version from output (simplified)
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "", nil
}

// parseSizeToBytes converts human-readable size strings to bytes
func parseSizeToBytes(sizeStr string) int64 {
	// Simplified implementation
	if sizeStr == "" {
		return 0
	}
	// TODO: Implement proper size parsing
	return 0
}

// getHealthStatusValue converts health status string to numeric value
func getHealthStatusValue(status string) int {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "PASSED", "OK", "HEALTHY":
		return 1
	case "WARNING", "CAUTION":
		return 2
	case "FAILED", "CRITICAL", "FAILING":
		return 3
	default:
		return 0
	}
}

// getRaidStatusValue converts RAID status string to numeric value
func getRaidStatusValue(status string) int {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "OPTIMAL", "OK", "HEALTHY":
		return 1
	case "DEGRADED", "WARNING":
		return 2
	case "FAILED", "CRITICAL", "OFFLINE":
		return 3
	default:
		return 0
	}
}
