package tools

import (
	"testing"
)

func TestNewStoreCLITool(t *testing.T) {
	// Test creating a new StoreCLI tool instance
	storeTool := NewStoreCLITool()

	// Verify the tool is correctly initialized
	if storeTool == nil {
		t.Error("NewStoreCLITool should not return nil")
	}

	if storeTool.GetName() != "StoreCLI" {
		t.Errorf("Expected tool name to be 'StoreCLI', got '%s'", storeTool.GetName())
	}

	// Test availability check (this will depend on system)
	// We just test that the function doesn't panic
	_ = storeTool.IsAvailable()

	// Test version check (this will depend on system)
	// We just test that the function doesn't panic
	_ = storeTool.GetVersion()
}

func TestStoreCLIToolInterfaces(t *testing.T) {
	// Test that StoreCLITool implements the required interfaces
	storeTool := NewStoreCLITool()

	// Test as ToolInterface
	var _ ToolInterface = storeTool

	// Test as DiskToolInterface
	var _ DiskToolInterface = storeTool

	// Test as RAIDToolInterface
	var _ RAIDToolInterface = storeTool

	// Test as CombinedToolInterface
	var _ CombinedToolInterface = storeTool
}

func TestStoreCLIToolMethods(t *testing.T) {
	// Test that all methods can be called without panicking
	storeTool := NewStoreCLITool()

	// These methods should not panic even if StoreCLI is not available
	_ = storeTool.GetRAIDArrays()
	_ = storeTool.GetRAIDDisks()
	_ = storeTool.GetDisks()
}
