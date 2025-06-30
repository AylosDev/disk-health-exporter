package tools

import (
	"testing"
)

func TestArcconfTool_NewArcconfTool(t *testing.T) {
	tool := NewArcconfTool()
	if tool == nil {
		t.Fatal("NewArcconfTool returned nil")
	}
}

func TestArcconfTool_GetName(t *testing.T) {
	tool := NewArcconfTool()
	expected := "arcconf"
	if tool.GetName() != expected {
		t.Errorf("Expected name %s, got %s", expected, tool.GetName())
	}
}

func TestArcconfTool_IsAvailable(t *testing.T) {
	tool := NewArcconfTool()
	// This test will pass/fail depending on whether arcconf is installed
	// We're just testing that the method doesn't panic
	available := tool.IsAvailable()
	t.Logf("arcconf available: %v", available)
}

func TestArcconfTool_Interfaces(t *testing.T) {
	tool := NewArcconfTool()

	// Verify that ArcconfTool implements RAIDToolInterface
	var _ RAIDToolInterface = tool

	// Test that methods don't panic
	arrays := tool.GetRAIDArrays()
	disks := tool.GetRAIDDisks()

	t.Logf("arcconf found %d RAID arrays and %d RAID disks", len(arrays), len(disks))
}
