# Discovery Option2 Implementation Analysis and Solution

## Problem Identified
Mark's threading issue is now clear from testing:
1. Current discovery implementation has bugs - file discoveries don't properly set `Exists` field
2. Race conditions occur because incomplete/incorrect discovery data gets accessed by parallel test threads
3. Template conditions fail because `.Discovered` values are not properly populated

## Mark's Solution: Discovery Option2 Architecture

### Current Implementation Problems:
- Two-pass template processing with placeholder pre-population
- Parallel test execution accesses discovery data before it's fully ready
- File discovery bug: `.Discovered.file.Exists` returns false even when file exists
- Threading race conditions in accessing `.Discovered` values

### Mark's Suggested Architecture:
```
1. Main orchestration file (main.goss.yml)
2. Run ALL discoveries first (sequential, no threading)
3. Load individual test files AFTER discoveries complete
4. Each test file is purpose-built for specific discovered conditions
5. Sequential file loading eliminates threading issues
```

### Benefits of Mark's Approach:
- **No Threading Issues**: Files loaded sequentially after discoveries complete
- **Better Organization**: Separate files for different system configurations  
- **Expandability**: Easy to add new discovery conditions and corresponding test files
- **Clearer Logic**: Main file orchestrates, test files focus on specific scenarios
- **Debugging**: Easier to trace which tests run for which conditions

## Implementation Strategy

### 1. Fix Discovery Implementation
The current bug where file discoveries don't set `Exists` properly needs to be fixed first.

### 2. Create Sequential Discovery Engine
Modify the validation engine to:
- Run all discoveries in a single thread FIRST
- Wait for ALL discoveries to complete
- THEN start loading and processing test files
- Load test files sequentially based on discovery results

### 3. File-Based Architecture
```
main.goss.yml           # Orchestration file with discoveries and conditional includes
├── discoveries/        # Optional: separate discovery definitions
├── tests/
    ├── core.yml        # Always-run tests
    ├── bash.yml        # Bash-specific tests  
    ├── systemd.yml     # Systemd-specific tests
    ├── docker.yml      # Docker-specific tests
    └── ...
```

### 4. Sequential Processing Flow
```
1. Parse main.goss.yml
2. Extract all discovery definitions
3. Run ALL discoveries sequentially in single thread
4. Store results in global discovery map
5. Re-parse main.goss.yml with discovery results available
6. Load included files sequentially based on conditions
7. Run tests (can still be parallel since discoveries are complete)
```

This eliminates the threading issues because:
- All discoveries complete before any test file loading
- No race conditions accessing `.Discovered` values
- Template processing happens with complete discovery data
- File includes are resolved sequentially