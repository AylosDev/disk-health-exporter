package utils

import (
	"os/exec"
	"strings"
)

// CommandExists checks if a command is available in the system PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetToolVersion gets the version of a tool
func GetToolVersion(tool string, versionFlag string) (string, error) {
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
