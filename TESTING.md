# Discovery Feature - Testing Guide

## Prerequisites

**Go Version Required:** Go 1.21+ (tested with Go 1.21.0)
```bash
# Install Go if needed
curl -L https://go.dev/dl/go1.21.0.linux-amd64.tar.gz | tar -C ~/go -xzf - --strip-components=1
export PATH=~/go/bin:$PATH
```

## Building

```bash
# Build from source
go build -o goss-discovery cmd/goss/goss.go
```

## Testing Discovery Features

### Basic Functionality Test
```bash
# Test discovery bug fix
./goss-discovery --gossfile discovery_option2/debug_test.goss.yml validate

# Expected: Shows discovery values correctly populated
```

### Mark's Sequential Architecture Test
```bash
# Test complete sequential implementation  
./goss-discovery --gossfile discovery_option2/mark_sequential_fixed.goss.yml validate --format documentation

# Expected: 15/15 tests pass with conditional loading based on discoveries
```

### Stability Test
```bash
# Run multiple times to verify no threading issues
for i in {1..10}; do 
  echo "Run $i:"
  ./goss-discovery --gossfile discovery_option2/mark_sequential_fixed.goss.yml validate --format silent
  echo "Exit code: $?"
done

# Expected: All runs should exit with code 0
```

## Key Fixes Applied

1. **Type Conversion Bug:** Fixed `normalizeDiscoveredValue()` in `store.go` to properly handle `*resource.DiscoveredValue`
2. **Sequential Execution:** Implemented Mark's architecture to eliminate threading race conditions
3. **Conditional Logic:** Discovery-based template conditions now work reliably

## Architecture Pattern

```yaml
discovery:
  component_check:
    type: package|file|command
    register: variable_name

{{ if .Discovered.variable_name.Installed/Exists/Value }}
resource_type:
  conditional_tests: {}
{{ end }}
```

## Validation

Working discovery conditions:
- `.Discovered.file.Exists` - File existence checks
- `.Discovered.package.Installed` - Package installation status  
- `.Discovered.command.Value` - Command output values
- All discoveries populate before test execution (no race conditions)