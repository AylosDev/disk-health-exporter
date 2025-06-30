package tools

import (
	"testing"
)

func TestHdparmTool_NewHdparmTool(t *testing.T) {
	tool := NewHdparmTool()
	if tool == nil {
		t.Fatal("NewHdparmTool returned nil")
	}
}

func TestHdparmTool_GetName(t *testing.T) {
	tool := NewHdparmTool()
	expected := "hdparm"
	if tool.GetName() != expected {
		t.Errorf("Expected name %s, got %s", expected, tool.GetName())
	}
}

func TestHdparmTool_IsAvailable(t *testing.T) {
	tool := NewHdparmTool()
	// This test will pass/fail depending on whether hdparm is installed
	// We're just testing that the method doesn't panic
	available := tool.IsAvailable()
	t.Logf("hdparm available: %v", available)
}

func TestHdparmTool_GetDisks_Interface(t *testing.T) {
	tool := NewHdparmTool()

	// Verify that HdparmTool implements DiskToolInterface
	var _ DiskToolInterface = tool

	// Test that GetDisks doesn't panic
	disks := tool.GetDisks()
	t.Logf("hdparm found %d disks", len(disks))
}
