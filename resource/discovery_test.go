package resource

import (
	"context"
	"testing"

	"github.com/goss-org/goss/system"
)

func TestDiscoveryPackage(t *testing.T) {
	discovery := &Discovery{
		id:       "test_discovery",
		Type:     "package",
		Register: "test_package",
		Resource: map[string]interface{}{
			"name": "bash",
		},
	}

	sys := system.New("")
	results := discovery.Validate(sys)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if !result.Successful {
		t.Errorf("Expected successful result, got: %v", result.Err)
	}

	if result.Result != SUCCESS {
		t.Fatalf("Expected SUCCESS result, got: %d", result.Result)
	}

	discovered := discovery.GetDiscoveredValue()
	if discovered == nil {
		t.Fatal("Expected discovered value to be stored")
	}

	if discovered.Raw == nil {
		t.Error("Expected Raw map to be populated")
	}
}

func TestDiscoveryFile(t *testing.T) {
	discovery := &Discovery{
		id:       "test_file_discovery",
		Type:     "file",
		Register: "test_file",
		Resource: map[string]interface{}{
			"path": "/etc/passwd",
		},
	}

	sys := system.New("")
	results := discovery.Validate(sys)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if !result.Successful {
		t.Errorf("Expected successful result, got: %v", result.Err)
	}

	discovered := discovery.GetDiscoveredValue()
	if discovered == nil {
		t.Fatal("Expected discovered value to be stored")
	}

	if !discovered.Exists {
		t.Error("/etc/passwd should exist")
	}

	if discovered.Raw["path"] != "/etc/passwd" {
		t.Errorf("Expected path to be /etc/passwd, got: %v", discovered.Raw["path"])
	}
}

func TestDiscoveryCommand(t *testing.T) {
	discovery := &Discovery{
		id:       "test_command_discovery",
		Type:     "command",
		Register: "test_command",
		Resource: map[string]interface{}{
			"command": "echo hello",
		},
	}

	sys := system.New("")
	results := discovery.Validate(sys)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if !result.Successful {
		t.Errorf("Expected successful result, got: %v", result.Err)
	}

	discovered := discovery.GetDiscoveredValue()
	if discovered == nil {
		t.Fatal("Expected discovered value to be stored")
	}

	if discovered.Raw["command"] != "echo hello" {
		t.Errorf("Expected command to be 'echo hello', got: %v", discovered.Raw["command"])
	}

	if discovered.Raw["exit-status"] != 0 {
		t.Errorf("Expected exit status 0, got: %v", discovered.Raw["exit-status"])
	}
}

func TestDiscoverySkip(t *testing.T) {
	discovery := &Discovery{
		id:       "test_skip",
		Type:     "package",
		Register: "test_package",
		Resource: map[string]interface{}{
			"name": "bash",
		},
		Skip: true,
	}

	sys := system.New("")
	results := discovery.Validate(sys)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if !result.Skipped {
		t.Error("Expected result to be skipped")
	}
}

func TestDiscoveryUnknownType(t *testing.T) {
	discovery := &Discovery{
		id:       "test_unknown",
		Type:     "unknown_type",
		Register: "test",
		Resource: map[string]interface{}{},
	}

	sys := system.New("")
	results := discovery.Validate(sys)

	if len(results) == 0 {
		t.Fatal("Expected at least one result")
	}

	result := results[0]
	if result.Successful {
		t.Error("Expected unsuccessful result for unknown type")
	}

	if result.Err == nil {
		t.Error("Expected error for unknown type")
	}
}

func TestDiscoveryGetRegister(t *testing.T) {
	discovery := &Discovery{
		Register: "my_register",
	}

	if discovery.GetRegister() != "my_register" {
		t.Errorf("Expected register name 'my_register', got: %s", discovery.GetRegister())
	}
}

func TestDiscoveryInterfaceMethods(t *testing.T) {
	discovery := &Discovery{
		id:    "test_id",
		Title: "Test Title",
		Meta: meta{
			"key": "value",
		},
	}

	if discovery.ID() != "test_id" {
		t.Errorf("Expected ID 'test_id', got: %s", discovery.ID())
	}

	if discovery.TypeKey() != DiscoveryResourceKey {
		t.Errorf("Expected TypeKey '%s', got: %s", DiscoveryResourceKey, discovery.TypeKey())
	}

	if discovery.TypeName() != DiscoveryResourceName {
		t.Errorf("Expected TypeName '%s', got: %s", DiscoveryResourceName, discovery.TypeName())
	}

	if discovery.GetTitle() != "Test Title" {
		t.Errorf("Expected Title 'Test Title', got: %s", discovery.GetTitle())
	}

	meta := discovery.GetMeta()
	if meta["key"] != "value" {
		t.Errorf("Expected meta key 'value', got: %v", meta["key"])
	}

	discovery.SetID("new_id")
	if discovery.ID() != "new_id" {
		t.Errorf("Expected ID 'new_id' after SetID, got: %s", discovery.ID())
	}

	discovery.SetSkip()
	if !discovery.Skip {
		t.Error("Expected Skip to be true after SetSkip")
	}
}

func TestDiscoverPackageDetails(t *testing.T) {
	discovery := &Discovery{
		id:       "package_discovery",
		Type:     "package",
		Register: "pkg_info",
		Resource: map[string]interface{}{
			"name": "bash",
		},
	}

	sys := system.New("")
	ctx := context.Background()
	discovered := &DiscoveredValue{
		Raw: make(map[string]interface{}),
	}

	discovery.discoverPackage(ctx, sys, discovered)

	if discovered.Raw["name"] != "bash" {
		t.Errorf("Expected name 'bash', got: %v", discovered.Raw["name"])
	}

	// Check that installed field exists
	if _, ok := discovered.Raw["installed"]; !ok {
		t.Error("Expected 'installed' field in Raw map")
	}
}

// Add a new test to verify the store mechanism
func TestDiscoveryValueStore(t *testing.T) {
	// Clear the store first
	discoveredValueStore = make(map[string]*DiscoveredValue)

	discovery := &Discovery{
		id:       "store_test",
		Type:     "file",
		Register: "test_store",
		Resource: map[string]interface{}{
			"path": "/etc/hosts",
		},
	}

	sys := system.New("")
	discovery.Validate(sys)

	// Check if value was stored
	stored := discovery.GetDiscoveredValue()
	if stored == nil {
		t.Error("Expected value to be stored in discoveredValueStore")
	}

	if stored.Raw["path"] != "/etc/hosts" {
		t.Errorf("Expected path /etc/hosts, got: %v", stored.Raw["path"])
	}
}

