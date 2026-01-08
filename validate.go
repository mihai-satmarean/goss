package goss

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/onsi/gomega/format"

	"github.com/goss-org/goss/outputs"
	"github.com/goss-org/goss/resource"
	"github.com/goss-org/goss/system"
	"github.com/goss-org/goss/util"
)

// extractRegisterKeys scans a file for 'register:' declarations and returns the keys
func extractRegisterKeys(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keys []string
	seen := make(map[string]bool)
	registerRegex := regexp.MustCompile(`^\s*register:\s*(\w+)\s*$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if matches := registerRegex.FindStringSubmatch(line); matches != nil {
			key := strings.TrimSpace(matches[1])
			if key != "" && !seen[key] {
				keys = append(keys, key)
				seen[key] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func getGossConfig(vars string, varsInline string, specFile string, packageManager string) (cfg *GossConfig, err error) {
	// handle stdin
	var fh *os.File
	var path, source string
	var gossConfig GossConfig

	// IMPORTANT: Discoveries must run FIRST, before any threading
	// Strategy: 
	// 1. Pre-scan file to find all 'register:' keys (without parsing)
	// 2. Pre-populate .Discovered with placeholder values for those keys
	// 3. Read config WITH template processing using placeholders (with lenient mode)
	// 4. Run discoveries to get real values
	// 5. Re-read config WITH template processing using real discovered values
	
	// Step 1: Pre-scan to extract register keys
	var registerKeys []string
	if specFile != "-" {
		source = specFile
		path = filepath.Dir(specFile)
		outStoreFormat, err = getStoreFormatFromFileName(specFile)
		if err != nil {
			return nil, err
		}

		// Read file as text to extract register keys
		registerKeys, err = extractRegisterKeys(specFile)
		if err != nil {
			return nil, fmt.Errorf("error extracting register keys: %v", err)
		}
	} else {
		source = "STDIN"
		fh = os.Stdin
		var data []byte
		data, err = io.ReadAll(fh)
		if err != nil {
			return nil, err
		}
		outStoreFormat, err = getStoreFormatFromData(data)
		if err != nil {
			return nil, err
		}
		// For STDIN, we can't re-read, so discoveries are not supported
		registerKeys = nil
	}

	// Step 2: Pre-populate .Discovered with placeholder values
	placeholderDiscovered := make(map[string]DiscoveredValue)
	for _, key := range registerKeys {
		placeholderDiscovered[key] = DiscoveredValue{}
	}

	// Step 3: Read config WITH template processing using placeholders (lenient mode)
	currentTemplateFilter, err = NewTemplateFilterLenient(vars, varsInline, placeholderDiscovered)
	if err != nil {
		return nil, err
	}

	gossConfig, err = ReadJSON(specFile)
	if err != nil {
		return nil, err
	}

	// Merge to get all discoveries from included files
	gossConfig, err = mergeJSONData(gossConfig, 0, path)
	if err != nil {
		return nil, err
	}

	// Step 4: Run ALL discoveries FIRST to get real values
		discovered, err := RunDiscoveries(gossConfig, packageManager)
	if err != nil {
		return nil, fmt.Errorf("error running discoveries: %v", err)
	}

	// Merge real discovered values with placeholders (real values overwrite placeholders)
	for k, v := range discovered {
		placeholderDiscovered[k] = v
	}

	// If no discoveries and no vars, we're done (no second pass needed)
	if len(gossConfig.Discoveries) == 0 && vars == "" && varsInline == "" {
		if len(gossConfig.Resources()) == 0 {
			return nil, fmt.Errorf("found 0 tests, source: %v", source)
		}
		return &gossConfig, nil
	}

	// Step 5: Re-read with template processing using real discovered values
	currentTemplateFilter, err = NewTemplateFilterWithDiscovered(vars, varsInline, placeholderDiscovered)
	if err != nil {
		return nil, err
	}

	if specFile == "-" {
		// For STDIN, we can't re-read
		if len(gossConfig.Discoveries) > 0 {
			return nil, fmt.Errorf("discoveries are not supported when reading from STDIN")
		}
		// For vars-only, we couldn't process templates in first pass
		// This is a limitation
		return &gossConfig, nil
	}

	// Re-read and process with full context (vars + discovered values)
	gossConfig, err = ReadJSON(specFile)
	if err != nil {
		return nil, err
	}

	gossConfig, err = mergeJSONData(gossConfig, 0, path)
	if err != nil {
		return nil, err
	}

	if len(gossConfig.Resources()) == 0 {
		return nil, fmt.Errorf("found 0 tests, source: %v", source)
	}

	return &gossConfig, nil
}

func getOutputer(c *bool, format string) (outputs.Outputer, error) {
	if c != nil && *c {
		color.NoColor = true
	}
	if c != nil && !*c {
		color.NoColor = false
	}

	return outputs.GetOutputer(format)
}

// ValidateResults performs validation and provides programmatic access to validation results
// no retries or outputs are supported
func ValidateResults(c *util.Config) (results <-chan []resource.TestResult, err error) {
	gossConfig, err := getGossConfig(c.Vars, c.VarsInline, c.Spec, c.PackageManager)
	if err != nil {
		return nil, err
	}

	sys := system.New(c.PackageManager)

	return validate(sys, *gossConfig, c.DisabledResourceTypes, c.MaxConcurrent), nil
}

// Validate performs validation, writes formatted output to stdout by default
// and supports retries and more, this is the full featured Validate used
// by the typical CLI invocation and will produce output to StdOut.  Use
// ValidateResults for programmatic access
func Validate(c *util.Config) (code int, err error) {
	err = setLogLevel(c)
	if err != nil {
		return 1, err
	}
	gossConfig, err := getGossConfig(c.Vars, c.VarsInline, c.Spec, c.PackageManager)
	if err != nil {
		return 78, err
	}
	return ValidateConfig(c, gossConfig)
}

func ValidateConfig(c *util.Config, gossConfig *GossConfig) (code int, err error) {
	// Needed for contains-elements
	// Maybe we don't use this and use custom
	// contain_element_matcher is needed because it's single entry to avoid
	// transform message
	format.UseStringerRepresentation = true
	outputConfig := util.OutputConfig{
		FormatOptions: c.FormatOptions,
	}

	sys := system.New(c.PackageManager)
	outputer, err := getOutputer(c.NoColor, c.OutputFormat)
	if err != nil {
		return 1, err
	}

	var ofh io.Writer
	ofh = os.Stdout
	if c.OutputWriter != nil {
		ofh = c.OutputWriter
	}

	sleep := c.Sleep
	retryTimeout := c.RetryTimeout
	i := 1
	startTime := time.Now()
	for {
		out := validate(sys, *gossConfig, c.DisabledResourceTypes, c.MaxConcurrent)
		exitCode := outputer.Output(ofh, out, outputConfig)
		if retryTimeout == 0 || exitCode == 0 {
			return exitCode, nil
		}
		elapsed := time.Since(startTime)
		if elapsed+sleep > retryTimeout {
			return 3, fmt.Errorf("timeout of %s reached before tests entered a passing state", retryTimeout)
		}
		color.Red("Retrying in %s (elapsed/timeout time: %.3fs/%s)\n\n\n", sleep, elapsed.Seconds(), retryTimeout)
		// Reset cache
		sys = system.New(c.PackageManager)
		time.Sleep(sleep)
		i++
		fmt.Printf("Attempt #%d:\n", i)
	}
}

func validate(sys *system.System, gossConfig GossConfig, skipList []string, maxConcurrent int) <-chan []resource.TestResult {
	out := make(chan []resource.TestResult)
	in := make(chan resource.Resource)

	go func() {
		for _, t := range gossConfig.Resources() {
			if util.IsValueInList(t.TypeName(), skipList) || util.IsValueInList(t.TypeKey(), skipList) {
				t.SetSkip()
			}

			in <- t
		}
		close(in)
	}()

	workerCount := runtime.NumCPU() * 5
	if workerCount > maxConcurrent {
		workerCount = maxConcurrent
	}
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range in {
				out <- f.Validate(sys)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
