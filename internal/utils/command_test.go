package utils

import (
	"testing"
)

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on most systems
	if !CommandExists("ls") {
		t.Error("Expected 'ls' command to exist")
	}

	// Test with a command that shouldn't exist
	if CommandExists("definitely_does_not_exist_command_12345") {
		t.Error("Expected non-existent command to return false")
	}
}
