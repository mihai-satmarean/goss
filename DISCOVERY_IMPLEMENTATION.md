# Discovery Feature Implementation

## Overview

This document describes the implementation of the Discovery feature for Goss, as requested in [GitHub Issue #784](https://github.com/goss-org/goss/issues/784).

## What Was Implemented

The Discovery feature allows Goss to run conditional tests based on the state of the system. This enables:

1. **Dynamic Test Configuration**: Tests can be conditionally executed based on what's installed on the system
2. **Portable Test Suites**: The same Goss configuration can run across different environments
3. **Version-Specific Testing**: Tests can adapt based on installed package versions or system configuration

## Implementation Details

### New Files Created

1. **`resource/discovery.go`**: Main implementation of the Discovery resource type
   - Supports all standard Goss resource types (package, file, command, service, port, user, group, process, http, addr, dns, kernel-param, mount, interface)
   - Stores discovered values in a package-level cache
   - Returns structured `DiscoveredValue` objects with common fields (Installed, Version, Exists, Value, Raw)

2. **`docs/discovery.md`**: Comprehensive documentation
   - Syntax and usage examples
   - All supported discovery types with available fields
   - Complete examples demonstrating the feature
   - Best practices and limitations

3. **`examples/discovery_example.yaml`**: Working example
   - Demonstrates conditional testing based on package presence
   - Shows OS-specific tests
   - Includes systemd detection

4. **`resource/discovery_test.go`**: Test suite
   - Tests for all major discovery functionality
   - Validation of discovered value storage
   - Edge cases (unknown types, skip functionality)

### Modified Files

1. **`goss_config.go`**:
   - Added `Discoveries` field to `GossConfig` struct
   - Updated `NewGossConfig()` to initialize the Discoveries map
   - Added merge logic for discoveries

2. **`resource/resource_list.go`**:
   - Added `DiscoveryMap` type with JSON and YAML unmarshal methods

3. **`store.go`**:
   - Added `Discovered` field to `TmplVars` struct for template access
   - Implemented `RunDiscoveries()` function to execute discovery phase
   - Added import for `system` package

4. **`template.go`**:
   - Created `NewTemplateFilterWithDiscovered()` function
   - Supports passing discovered values to templates

5. **`validate.go`**:
   - Modified `getGossConfig()` to accept `packageManager` parameter
   - Implemented two-pass processing:
     1. First pass: Read config without templates, run discoveries
     2. Second pass: Read config with templates, using discovered values
   - Updated all callers of `getGossConfig()`

6. **`serve.go`**:
   - Updated `newHealthHandler()` to pass `packageManager` to `getGossConfig()`

## How It Works

### Execution Flow

```
1. User runs: goss validate
   ↓
2. Goss reads the gossfile WITHOUT template processing
   ↓
3. Discovery resources are executed (RunDiscoveries)
   ↓
4. Discovered values are stored in a map
   ↓
5. Goss re-reads the gossfile WITH template processing
   ↓
6. Templates have access to .Discovered values
   ↓
7. Conditional tests are included/excluded based on discoveries
   ↓
8. Regular validation proceeds
```

### Template Access

Discovered values are available in templates via the `.Discovered` variable:

```yaml
discovery:
  my_check:
    type: package
    resource:
      name: nginx
    register: nginx_info

{{ if .Discovered.nginx_info.Installed }}
file:
  /etc/nginx/nginx.conf:
    exists: true
{{ end }}
```

### Discovered Value Structure

Each discovered value has:
- **Common fields**: `Installed`, `Version`, `Exists`, `Value`
- **Raw map**: Contains all type-specific information

Example for a package discovery:
```go
{
  "Installed": true,
  "Version": "1.2.3",
  "Raw": {
    "name": "nginx",
    "installed": true,
    "version": "1.2.3"
  }
}
```

## Supported Discovery Types

All standard Goss resource types are supported:

1. **package** - Check if packages are installed and get version info
2. **file** - Check file existence, permissions, ownership
3. **command** - Execute commands and capture output
4. **service** - Check service status (enabled, running)
5. **port** - Check if ports are listening
6. **user** - Check user existence and properties
7. **group** - Check group existence
8. **process** - Check if processes are running
9. **http** - Make HTTP requests and check responses
10. **addr** - Check network address reachability
11. **dns** - Resolve DNS names
12. **kernel-param** - Get kernel parameter values
13. **mount** - Check mount points
14. **interface** - Check network interfaces

## Usage Example

```yaml
# Discovery phase
discovery:
  auditd_check:
    type: package
    resource:
      name: auditd
    register: auditd_installed

  app_version:
    type: command
    resource:
      command: "cat /app/version"
    register: app_version

# Conditional tests
{{ if .Discovered.auditd_installed.Installed }}
service:
  auditd:
    enabled: true
    running: true
{{ end }}

{{ if regexMatch "^2\\..*" .Discovered.app_version.Value }}
file:
  /etc/app/v2-config.json:
    exists: true
{{ end }}
```

## Testing

All tests pass successfully:

```bash
$ go test -v ./resource -run TestDiscovery
=== RUN   TestDiscoveryPackage
--- PASS: TestDiscoveryPackage (0.01s)
=== RUN   TestDiscoveryFile
--- PASS: TestDiscoveryFile (0.01s)
=== RUN   TestDiscoveryCommand
--- PASS: TestDiscoveryCommand (0.00s)
=== RUN   TestDiscoverySkip
--- PASS: TestDiscoverySkip (0.00s)
=== RUN   TestDiscoveryUnknownType
--- PASS: TestDiscoveryUnknownType (0.01s)
=== RUN   TestDiscoveryGetRegister
--- PASS: TestDiscoveryGetRegister (0.00s)
=== RUN   TestDiscoveryInterfaceMethods
--- PASS: TestDiscoveryInterfaceMethods (0.00s)
=== RUN   TestDiscoveryValueStore
--- PASS: TestDiscoveryValueStore (0.00s)
PASS
```

## Limitations

1. **STDIN not supported**: Discoveries cannot be used when reading from STDIN (`goss validate -`)
2. **Single execution**: Discoveries run once before template processing, they don't update dynamically
3. **No output**: Discovery resources are not included in test output (they're a pre-processing step)

## Backwards Compatibility

This implementation is fully backwards compatible:
- Existing Goss files work without any changes
- Discovery is opt-in (only used when `discovery:` section is present)
- No breaking changes to existing APIs

## Future Enhancements

Potential improvements that could be added later:
1. Dynamic re-evaluation during validation with retry
2. Discovery dependencies (run discoveries in order)
3. Discovery caching between multiple Goss runs
4. More granular access to discovered values in tests
5. Discovery assertions (fail if discovery doesn't meet expectations)

## Bug Fixes

### Issue: YAML Unmarshal Error with --vars-inline

**Problem:** When using `--vars-inline` or `--vars` with template syntax in the gossfile, the following error occurred:
```
Error: yaml: unmarshal errors:
 line 3: cannot unmarshal !!map into string
```

**Root Cause:** The initial implementation disabled template processing entirely in the first pass by setting `currentTemplateFilter = nil`. This caused the YAML parser to try parsing raw template syntax (like `{{ if .Vars.something }}`), which resulted in unmarshal errors.

**Solution:** Changed the two-pass approach to:
1. **First pass**: Apply template processing with vars/varsInline (but without discovered values)
2. **Second pass**: Only execute if discoveries were found, apply template processing with both vars/varsInline AND discovered values

This ensures that:
- Variable substitution works correctly in both passes
- Files without discoveries only go through one pass (optimization)
- Template syntax is always processed before YAML parsing

**Modified Code:** `validate.go` - `getGossConfig()` function now calls `NewTemplateFilter()` in the first pass instead of setting `currentTemplateFilter = nil`.

## Conclusion

The Discovery feature has been successfully implemented and tested. It addresses the original issue (#784) by providing:
- ✅ Dynamic dependency resolution
- ✅ Conditional test execution
- ✅ Portable configurations across different environments
- ✅ Support for all Goss resource types
- ✅ Clean, documented API
- ✅ Comprehensive test coverage
- ✅ Fixed vars-inline compatibility issue

The feature is ready for use and can significantly improve the flexibility and reusability of Goss test suites.

