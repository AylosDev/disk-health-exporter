package tools

import (
	"testing"
)

func TestZpoolTool_NewZpoolTool(t *testing.T) {
	tool := NewZpoolTool()
	if tool == nil {
		t.Fatal("NewZpoolTool returned nil")
	}
}

func TestZpoolTool_GetName(t *testing.T) {
	tool := NewZpoolTool()
	expected := "zpool"
	if tool.GetName() != expected {
		t.Errorf("Expected name %s, got %s", expected, tool.GetName())
	}
}

func TestZpoolTool_IsAvailable(t *testing.T) {
	tool := NewZpoolTool()
	// This test will pass/fail depending on whether zpool is installed
	// We're just testing that the method doesn't panic
	available := tool.IsAvailable()
	t.Logf("zpool available: %v", available)
}

func TestZpoolTool_GetDisks_Interface(t *testing.T) {
	tool := NewZpoolTool()

	// Verify that ZpoolTool implements DiskToolInterface
	var _ DiskToolInterface = tool

	// Test that GetDisks doesn't panic
	disks := tool.GetDisks()
	pools := tool.GetZFSPools()

	t.Logf("zpool found %d disks and %d pools", len(disks), len(pools))
}
