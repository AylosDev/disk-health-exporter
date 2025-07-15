// Package utils provides utility functions for command execution and tool detection.
package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommandExists checks if a command is available in the system PATH.
// It returns true if the command can be found, false otherwise.
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// GetToolVersion gets the version string of a tool by executing it with
// the specified version flag. Returns the first line of output trimmed
// of whitespace, or an error if the command fails.
func GetToolVersion(tool string, versionFlag string) (string, error) {
	if tool == "" {
		return "", fmt.Errorf("tool name cannot be empty")
	}
	if versionFlag == "" {
		return "", fmt.Errorf("version flag cannot be empty")
	}

	output, err := exec.Command(tool, versionFlag).Output()
	if err != nil {
		return "", fmt.Errorf("failed to get version for %s: %w", tool, err)
	}

	// Extract version from output (simplified)
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return "", fmt.Errorf("no output from %s %s", tool, versionFlag)
}
