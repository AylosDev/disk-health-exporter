package utils

import (
	"strings"
	"testing"
)

func TestCommandExists(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		{
			name:     "existing command",
			command:  "ls",
			expected: true,
		},
		{
			name:     "non-existing command",
			command:  "definitely_does_not_exist_command_12345",
			expected: false,
		},
		{
			name:     "empty command",
			command:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CommandExists(tt.command)
			if result != tt.expected {
				t.Errorf("CommandExists(%q) = %v, expected %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestGetToolVersion(t *testing.T) {
	t.Helper()

	tests := []struct {
		name        string
		tool        string
		versionFlag string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty tool name",
			tool:        "",
			versionFlag: "--version",
			expectError: true,
			errorMsg:    "tool name cannot be empty",
		},
		{
			name:        "empty version flag",
			tool:        "ls",
			versionFlag: "",
			expectError: true,
			errorMsg:    "version flag cannot be empty",
		},
		{
			name:        "non-existing tool",
			tool:        "nonexistentcommand12345",
			versionFlag: "--version",
			expectError: true,
			errorMsg:    "failed to get version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetToolVersion(tt.tool, tt.versionFlag)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}
