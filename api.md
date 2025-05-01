# PHP Mixer Go API Documentation

## Overview

The PHP Mixer Go API provides programmatic access to the PHP obfuscator's functionality. It allows you to obfuscate PHP code, files, and directories using various techniques to protect your PHP code from reverse engineering.

## Installation

To use the API in your Go project:

```bash
go get github.com/whit3rabbit/phpmixer
```

Then import it in your code:

```go
import "github.com/whit3rabbit/phpmixer/pkg/api"
```

## Core API Components

### Obfuscator

The main entry point for the API is the `Obfuscator` type, which encapsulates the configuration and context for obfuscation operations.

```go
type Obfuscator struct {
    Context *obfuscator.ObfuscationContext
    Config  *config.Config // Parsed configuration
}
```

### Options

The `Options` struct allows you to configure the obfuscator when creating it:

```go
type Options struct {
    // Path to a YAML configuration file.
    // If empty or file not found, default configuration values are used.
    ConfigPath string

    // Suppresses informational messages printed by the API during operation.
    Silent bool

    // Allows programmatically overriding configuration values after loading from file or defaults.
    // Keys should match the nested structure of the config, e.g., "obfuscation.strings.enabled".
    // Values should be of the correct type (bool, int, string, []string, etc.).
    ConfigOverrides map[string]interface{}
}
```

Example using `ConfigOverrides`:
```go
options := api.Options{
    Silent: true,
    ConfigOverrides: map[string]interface{}{
        "obfuscation.strings.enabled":         false, // Disable string obfuscation
        "obfuscation.dead_code.enabled":       true,  // Enable dead code
        "obfuscation.dead_code.injection_rate": 50,   // Set dead code rate
        "obfuscation.ignore.functions":        []string{"framework_init", "render_page"}, // Add functions to ignore list
    },
}
obf, err := api.NewObfuscator(options)
// ... handle error
```

## Core Functions

### Creating an Obfuscator

```go
func NewObfuscator(options Options) (*Obfuscator, error)
```

Creates a new obfuscator instance with the specified options. Loads configuration from `options.ConfigPath` (if provided), applies defaults, and then applies `options.ConfigOverrides`.

Example:
```go
obf, err := api.NewObfuscator(api.Options{
    ConfigPath: "custom-config.yaml", // Optional path to config file
    Silent:     true,                 // Suppress informational messages
})
if err != nil {
    log.Fatalf("Failed to create obfuscator: %v", err)
}
```

### Obfuscating PHP Code

```go
func (o *Obfuscator) ObfuscateCode(code string) (string, error)
```

Obfuscates a PHP code string and returns the obfuscated code. Uses the configuration associated with the `Obfuscator` instance.

Example:
```go
result, err := obf.ObfuscateCode("<?php echo 'Hello World'; ?>")
if err != nil {
    log.Fatalf("Failed to obfuscate code: %v", err)
}
fmt.Println(result)
```

### Obfuscating a File

```go
func (o *Obfuscator) ObfuscateFile(filePath string) (string, error)
```

Obfuscates a single PHP file located at `filePath` and returns the obfuscated code as a string.

Example:
```go
result, err := obf.ObfuscateFile("input.php")
if err != nil {
    log.Fatalf("Failed to obfuscate file: %v", err)
}
fmt.Println(result)
```

### Obfuscating a File to Another File

```go
func (o *Obfuscator) ObfuscateFileToFile(inputPath, outputPath string) error
```

Obfuscates a PHP file at `inputPath` and writes the result directly to `outputPath`.

Example:
```go
err := obf.ObfuscateFileToFile("input.php", "obfuscated/output.php")
if err != nil {
    log.Fatalf("Failed to obfuscate file to file: %v", err)
}
```

### Obfuscating a Directory

```go
func (o *Obfuscator) ObfuscateDirectory(inputDir, outputDir string) error
```

Recursively obfuscates PHP files (matching configured extensions) from `inputDir` to `outputDir`. It processes files according to the `skip`, `keep`, and `follow_symlinks` configuration settings. It also automatically saves the name scrambling context (`*.scramble` files) within the `outputDir`.

Example:
```go
err := obf.ObfuscateDirectory("src", "dist/obfuscated")
if err != nil {
    log.Fatalf("Failed to obfuscate directory: %v", err)
}
```

### Context Management

```go
func (o *Obfuscator) LoadContext(baseDir string) error
func (o *Obfuscator) SaveContext(baseDir string) error
```

Manually load or save the obfuscation context (primarily name scrambling maps) from/to a specified directory. `ObfuscateDirectory` handles this automatically, but these functions are available for more complex workflows.

Example:
```go
// Load context before obfuscating individual files to maintain consistency
err := obf.LoadContext("dist/obfuscated")
if err != nil {
    log.Printf("Warning: Failed to load context (maybe first run?): %v", err)
}

// ... perform obfuscation ...

// Manually save context if needed
err = obf.SaveContext("dist/obfuscated")
if err != nil {
    log.Fatalf("Failed to save context: %v", err)
}
```

### Looking Up Obfuscated Names

```go
func (o *Obfuscator) LookupObfuscatedName(name string, typeStr string) (string, error)
```

Looks up the *obfuscated* version of an *original* name (`name`) of a specific type (`typeStr`, e.g., "function", "variable", "class") from the currently loaded context. Returns the obfuscated name or an error if not found.

Example:
```go
obfuscatedName, err := obf.LookupObfuscatedName("myOriginalFunction", "function")
if err != nil {
    log.Printf("Original name 'myOriginalFunction' not found in context: %v", err)
} else {
    fmt.Printf("Original: myOriginalFunction, Obfuscated: %s\n", obfuscatedName)
}
```
*Note: To find the original name from a scrambled name, use the CLI `whatis` command or manually inspect the `.scramble` files.*

### Controlling Output

```go
func PrintInfo(format string, args ...interface{})
```

A helper function (in the `api` package) to print informational messages to stdout. It respects the `Testing` flag (used internally for Go tests) and can be used for logging consistent with the library's own output when `Silent` is false.

Example:
```go
api.PrintInfo("Starting obfuscation of %d files...\n", fileCount)
```

## Configuration

Configuration is loaded from a YAML file (specified via `Options.ConfigPath` or defaulting to `./config.yaml`) and/or overridden programmatically using `Options.ConfigOverrides`.

Here's an example `config.yaml` reflecting the current structure:

```yaml
# PHP Obfuscator Configuration

# General behavior settings
silent: false                    # Suppress informational messages
abort_on_error: true             # Stop processing on the first error
debug_mode: false                # Enable verbose debug logging for development
parser_mode: "PREFER_PHP7"       # PHP Parser version preference (e.g., PREFER_PHP7, PREFER_PHP8)

# --- Obfuscation Techniques ---
obfuscation:
  # String obfuscation settings
  strings:
    enabled: true                # Enable string literal obfuscation
    technique: "base64"          # Options: "base64", "rot13", "xor"
    # Optional: Define XOR key if technique is XOR
    # xor_key: "YourSecretKey"

  # Name scrambling settings
  scrambling:
    mode: "identifier"           # Name generation style: "identifier", "hexa", "numeric"
    length: 5                    # Target length for generated names

  # Comment handling
  comments:
    strip: false                 # Remove comments from the PHP code

  # --- Granular Name Scrambling Toggles ---
  # Set 'scramble: true' to enable obfuscation for that specific type
  variables: { scramble: false } # Obfuscate variable names ($var)
  functions: { scramble: false } # Obfuscate function names (myFunc())
  classes:   { scramble: false } # Obfuscate class/interface/trait names (MyClass)
  # Add more specific types if needed (e.g., properties, methods, constants - check future versions)

  # Control flow obfuscation
  control_flow:
    enabled: true                # Wrap blocks in if(true){} statements
    max_nesting_depth: 2         # Max depth for nested if(true){} wrappers
    random_conditions: true      # Use random conditions (e.g., 1==1, 5>2) instead of just 'true'
    add_dead_branches: false     # Add empty or junk 'else {}' blocks to the if(true){} wrappers

  # Advanced loop obfuscation (potentially more complex/fragile)
  advanced_loops:
    enabled: false               # Enable techniques like sentinel variables in loops

  # Array access obfuscation (may affect performance or compatibility)
  array_access:
    enabled: true                # Transform $a['b'] to _phpmix_array_get($a, 'b')

  # Arithmetic expression obfuscation (experimental)
  # arithmetic_expressions:
  #   enabled: false
  #   complexity_level: 1        # Controls complexity (e.g., 1-3)
  #   transformation_rate: 50    # Percentage of eligible expressions to transform (0-100)

  # Dead code injection (code that never runs)
  dead_code:
    enabled: false               # Insert dead code blocks like if(false){...}
    injection_rate: 30           # Percentage chance (0-100) to inject dead code at each possible location

  # Junk code insertion (code that runs but has no effect)
  junk_code:
    enabled: false               # Insert useless code like unused vars, pointless math
    injection_rate: 30           # Percentage chance (0-100) to inject junk code at each possible location
    max_injection_depth: 3       # Limit how deep junk code can be injected inside nested structures

  # Statement shuffling (reorders independent lines of code)
  statement_shuffling:
    enabled: true                # Shuffle independent statements within blocks
    min_chunk_size: 2            # Minimum number of independent statements in a sequence to consider shuffling
    chunk_mode: "fixed"          # How to group statements: "fixed" (size based) or "ratio" (percentage based)
    chunk_ratio: 20              # Percentage of statements per chunk if chunk_mode is "ratio"

  # --- Exclusions/Ignores for Name Scrambling ---
  # Prevents specific names from being changed, useful for frameworks or APIs
  ignore:
    functions: []                # List of function names to ignore (e.g., ["wp_head", "drupal_add_js"])
    variables: []                # List of variable names (WITHOUT $) to ignore (e.g., ["post", "wpdb"])
    classes: []                  # List of class/interface/trait names to ignore (e.g., ["WP_Query", "MyFrameworkBase"])
    # Add more ignore lists if needed (e.g., properties, methods, constants - check future versions)

# --- File Processing Options (Top-Level) ---
skip: ["vendor/*", "*.git*", "*.svn*", "*.bak", "node_modules/*"] # Glob patterns for files/dirs to skip entirely
keep: []                         # Glob patterns for files/dirs to copy to target without obfuscating (e.g., ["assets/*", "*.css"])
obfuscate_php_extensions: ["php", "php5", "phtml"] # File extensions considered PHP code
follow_symlinks: false           # Follow symbolic links (process target) instead of copying the link itself
# target_directory: ""           # Output directory (set internally by API call or CLI flags, not read from config file)
```

## Complete Example

Here's a complete example showing how to create a config file programmatically (if needed) and obfuscate a directory:

```go
package main

import (
	"log"
	"os"
	// "path/filepath" // Not needed for this specific example

	"github.com/whit3rabbit/phpmixer/pkg/api"
)

func main() {
	// --- Option 1: Define config content as a string ---
	// Note: Using the correct NESTED YAML structure
	configContent := `
# General settings
silent: false
abort_on_error: true
debug_mode: false

# File processing
skip: ["vendor/*", "*.git*"]
keep: ["assets/*"]
obfuscate_php_extensions: ["php"]
follow_symlinks: false

# Obfuscation details
obfuscation:
  strings:
    enabled: true
    technique: "base64"
  scrambling:
    mode: "identifier"
    length: 6
  comments:
    strip: true
  variables: { scramble: true } # Enable variable scrambling
  functions: { scramble: true } # Enable function scrambling
  classes:   { scramble: true } # Enable class scrambling
  control_flow:
    enabled: true
    max_nesting_depth: 1
  array_access:
    enabled: true
  statement_shuffling:
    enabled: true
  dead_code:
    enabled: false # Keep dead code off for this example
  junk_code:
    enabled: false # Keep junk code off for this example
  ignore:
    functions: ["framework_entry_point"]
    variables: ["config"]
`
	configPath := "temp-obfuscator-config.yaml"
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		log.Fatalf("Failed to write temporary config file: %v", err)
	}
	defer os.Remove(configPath) // Clean up the temp file

	// --- Option 2: Use ConfigOverrides (Alternative to writing a file) ---
	/*
	   options := api.Options{
	       Silent: false,
	       ConfigOverrides: map[string]interface{}{
	           "abort_on_error": true,
	           "skip": []string{"vendor/*", "*.git*"},
	           "keep": []string{"assets/*"},
	           "obfuscation.strings.enabled": true,
	           "obfuscation.comments.strip": true,
	           "obfuscation.variables.scramble": true,
	           // ... add other overrides ...
	       },
	   }
	*/

	// Create obfuscator using the temporary config file
	obf, err := api.NewObfuscator(api.Options{
		ConfigPath: configPath, // Use the file we just wrote
		Silent:     false,      // Can also be set in the config file
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// Define source and target directories (ensure they exist or handle creation)
	inputDir := "path/to/your/php/src"    // CHANGE THIS
	outputDir := "path/to/your/output/dist" // CHANGE THIS

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", outputDir, err)
	}

	api.PrintInfo("Starting obfuscation: %s -> %s\n", inputDir, outputDir)

	// Obfuscate the directory
	err = obf.ObfuscateDirectory(inputDir, outputDir)
	if err != nil {
		log.Fatalf("Failed to obfuscate directory: %v", err)
	}

	api.PrintInfo("Obfuscation completed successfully to %s\n", outputDir)
}

```

## Best Practices

1.  **Configuration Files**: Store your primary obfuscation configuration in a version-controlled YAML file (`config.yaml`). Use `Options.ConfigOverrides` for minor, programmatic adjustments if needed.
2.  **Handle Errors**: Always check for errors returned by API functions, especially `ObfuscateDirectory`, as partial failures can occur. Check the `abort_on_error` setting.
3.  **Testing**: Thoroughly test your obfuscated code to ensure it functions identically to the original. Some complex code patterns might be affected by aggressive obfuscation (like array access or statement shuffling).
4.  **Context Management**: For consistent name scrambling across related projects or deployment steps, ensure the context files (`*.scramble`) generated in the output directory are preserved and potentially re-used via `LoadContext` or by running `ObfuscateDirectory` repeatedly on the same target.
5.  **Skip/Keep Lists**: Carefully configure the `skip` and `keep` patterns to exclude third-party libraries, frameworks, assets, or any code that should not be modified.
6.  **Ignore Lists**: Use the `obfuscation.ignore` lists to prevent scrambling of specific identifiers required by frameworks, APIs, or reflection mechanisms.

## Advanced Usage

### Customizing Obfuscation Techniques

Selectively enable/disable features or adjust parameters using `ConfigOverrides` or a YAML file:

```go
// Example using ConfigOverrides to disable most features except string/variable obfuscation
options := api.Options{
    Silent: true,
    ConfigOverrides: map[string]interface{}{
        "obfuscation.strings.enabled":             true,
        "obfuscation.strings.technique":           "xor", // Use XOR
        "obfuscation.variables.scramble":          true,
        // Disable others explicitly (assuming defaults might be true)
        "obfuscation.functions.scramble":          false,
        "obfuscation.classes.scramble":            false,
        "obfuscation.comments.strip":              false,
        "obfuscation.control_flow.enabled":        false,
        "obfuscation.array_access.enabled":        false,
        "obfuscation.statement_shuffling.enabled": false,
        "obfuscation.dead_code.enabled":           false,
        "obfuscation.junk_code.enabled":           false,
    },
}
obf, _ := api.NewObfuscator(options)
```

### Statement Shuffling Configuration

Adjust how statement shuffling works:

```yaml
obfuscation:
  statement_shuffling:
    enabled: true
    min_chunk_size: 3      # Only shuffle sequences of 3 or more independent statements
    chunk_mode: "ratio"    # Group statements based on a percentage of the block size
    chunk_ratio: 30        # Try to group ~30% of statements together for shuffling
```

### Ignoring Specific Items

Prevent critical identifiers from being scrambled:

```yaml
obfuscation:
  ignore:
    functions: ["wp_head", "wp_footer", "get_header", "drupal_set_message"]
    variables: ["wpdb", "post", "user"] # Note: Variable names *without* the leading '$'
    classes: ["WP_Query", "Drupal", "MyPlugin_ApiHandler"]
```

## Directory/File Tree Reference

(This tree shows the general structure of the `phpmixer` repository itself)

```
├── LICENSE
├── README.md
├── api.md  # <- You are here
├── cmd
│   └── go-php-obfuscator # (The main binary source)
│       ├── cmd
│       │   ├── dir.go
│       │   ├── file.go
│       │   ├── obfuscate.go
│       │   ├── root.go
│       │   └── whatis.go
│       └── main.go
├── config.yaml # Default configuration
├── dist_test # (Example outputs used by some tests)
├── examples
│   ├── config-example.yaml
│   ├── library_usage # API usage example
│   └── ... (other examples)
├── go.mod
├── go.sum
├── internal # (Internal implementation details, includes *.go AND *_test.go files)
│   ├── astutil
│   ├── config
│   ├── obfuscator
│   ├── scrambler
│   ├── testutil # (Test helpers for unit tests)
│   └── transformer
├── pkg # (Exported packages intended for public use, includes *.go AND *_test.go files)
│   └── api # The public API package documented here
├── testdata # (General test data, primarily for older/simpler UNIT tests)
│   ├── classes
│   ├── complex
│   └── simple
├── tests # Top-Level Directory for Integration Tests
│   ├── integration # End-to-end tests
│   │   ├── array_access
│   │   │   └── array_access_test.go
│   │   └── dead_code
│   │       └── dead_code_test.go
│   ├── internal # (Go helpers specifically for integration tests)
│   │   └── phprunner.go
│   └── testdata # (Test data specifically for INTEGRATION tests)
│       └── integration
│           ├── array_access
│           │   └── input.php
│           └── dead_code
│               └── input.php
```