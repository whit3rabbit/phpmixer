# PHP Mixer (phpmixer)

<p align="center">
  <img src="logo.png" alt="Logo" />
</p>

PHP Mixer is a PHP code obfuscator written in Go. It helps protect your PHP code by applying various obfuscation techniques to make it difficult to reverse engineer.

## Features

- **String Obfuscation**: Transforms string literals using techniques like Base64, ROT13, or XOR encryption. Handles `ScalarString` and `ScalarEncapsed` parts.
- **Name Scrambling**: Obfuscates variable, function, class, interface, and trait names with configurable modes (identifier, hexa, numeric) and length. Allows fine-grained control over which types are scrambled and provides ignore lists.
- **Comment Stripping**: Removes comments from the code.
- **Control Flow Obfuscation**: Wraps code blocks in redundant conditional statements (`if(true){...}`), potentially with randomized conditions, nested levels, and dead `else` branches.
- **Advanced Loop Obfuscation**: Modifies loops using techniques like sentinel variables and conditional breaks (optional).
- **Array Access Obfuscation**: Replaces array access (`$arr['key']`, `$arr[0]`) with helper function calls (experimental).
- **Arithmetic Expression Obfuscation**: Transforms simple arithmetic operations into more complex but equivalent expressions (experimental).
- **Dead Code Injection**: Inserts code blocks that are never executed (e.g., within `if(false)`).
- **Junk Code Insertion**: Inserts code that executes but has no side effects (e.g., unused variables, pointless calculations).
- **Statement Shuffling**: Reorders independent statements within code blocks based on dependency analysis to hinder readability while preserving logic.
- **Indentation Stripping**: Removes unnecessary whitespace and formatting during the printing process.
- **Context Persistence**: Saves and loads name scrambling maps for consistent obfuscation across multiple runs or related projects.

## Installation

### Using go install

```bash
go install github.com/whit3rabbit/phpmixer@latest
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/whit3rabbit/phpmixer.git
cd phpmixer

# Build the binary
go build -o phpmixer ./cmd/go-php-obfuscator

# Optional: Move to a directory in your PATH
mv phpmixer /usr/local/bin/  # Linux/macOS
# or
move phpmixer C:\Windows\System32\  # Windows
```

## Usage

### Command Line Interface

```bash
# Obfuscate a single file (output to stdout)
phpmixer file path/to/file.php

# Obfuscate a single file (output to another file)
phpmixer file path/to/file.php -o path/to/output.php

# Obfuscate a directory
phpmixer dir -s path/to/source -t path/to/target

# Obfuscate a directory, cleaning the target first
phpmixer dir -s path/to/source -t path/to/target --clean

# Look up the original name of a scrambled identifier
phpmixer whatis -t path/to/target scrambled_name

# Look up a specific type of scrambled name
phpmixer whatis -t path/to/target --type function scrambled_func_name

# Override config options via flags (see --help for all flags)
phpmixer file input.php --strip-comments=true --dead-code=true
```

### Library API

PHP Mix can also be used as a Go library in your own projects:

```go
import (
    "log"
    "github.com/whit3rabbit/phpmixer/pkg/api"
)

func main() {
    // Create a new obfuscator using options
    obf, err := api.NewObfuscator(api.Options{
        ConfigPath: "config.yaml", // Path to config file (optional, defaults used if empty)
        Silent:     false,         // Suppress informational messages
        // ConfigOverrides can be used to programmatically override config values
        // ConfigOverrides: map[string]interface{}{
        //     "obfuscation.strings.enabled": false,
        //     "obfuscation.dead_code.enabled": true,
        // },
    })
    if err != nil {
        log.Fatalf("Failed to create obfuscator: %v", err)
    }

    // Example: Obfuscate a PHP code string
    phpCode := "<?php function hello() { echo 'Hello World!'; } hello(); ?>"
    obfuscatedCode, err := obf.ObfuscateCode(phpCode)
    if err != nil {
        log.Fatalf("Failed to obfuscate code: %v", err)
    }
    log.Printf("Obfuscated Code:\n%s", obfuscatedCode)

    // Example: Obfuscate a file and return the result as string
    // obfuscatedFileContent, err := obf.ObfuscateFile("input.php")

    // Example: Obfuscate a file and write to another file
    // err = obf.ObfuscateFileToFile("input.php", "output.php")

    // Example: Obfuscate a directory
    // err = obf.ObfuscateDirectory("input_dir", "output_dir")

    // After directory obfuscation, context is saved automatically.
    // You can explicitly load/save context if needed:
    // err = obf.LoadContext("output_dir")
    // err = obf.SaveContext("output_dir")

    // Example: Look up an obfuscated name (requires context to be loaded/generated)
    // originalName, err := obf.LookupObfuscatedName("scrambled_func_abc", "function")
    // if err != nil {
    //     log.Printf("Lookup failed: %v", err)
    // } else {
    //     log.Printf("Original name: %s", originalName)
    // }
}

```

For a complete example, see [examples/library_usage/main.go](examples/library_usage/main.go).

#### API Documentation

For detailed API documentation, run:

```bash
godoc -http=:6060
```

Then visit http://localhost:6060/pkg/github.com/whit3rabbit/phpmixer/pkg/api/

The API includes working examples (`example_test.go`) for each function that demonstrate proper usage.

### Name Scrambling and the "whatis" Command

When PHP Mixer obfuscates your code using name scrambling, it maintains a mapping between original and scrambled names. This mapping is stored in a context file (`.phpmixer-context.json`) in the target directory during obfuscation. This feature is useful for:

1. **Consistent Obfuscation**: When obfuscating multiple files or running obfuscation multiple times, the same original names will always map to the same scrambled names.

2. **Debugging Obfuscated Code**: When working with obfuscated code, you may need to trace back a scrambled identifier to its original name.

The `whatis` command allows you to look up the original name of a scrambled identifier:

```bash
# Basic usage - look up any scrambled name
phpmixer whatis -t path/to/target_dir scrambled_name

# Look up a specific type of scrambled name
phpmixer whatis -t path/to/target_dir --type function scrambled_func_name
phpmixer whatis -t path/to/target_dir --type variable scrambled_var_name
phpmixer whatis -t path/to/target_dir --type class scrambled_class_name
```

Examples:

```bash
# If original function "getUserData" was scrambled to "fn_a5x2"
$ phpmixer whatis -t ./obfuscated_code fn_a5x2
Original name: getUserData (Type: function)

# If you know it's a variable
$ phpmixer whatis -t ./obfuscated_code --type variable v_b7r9
Original name: userAccount (Type: variable)
```

You can also perform lookups programmatically using the API:

```go
// Load context from an obfuscated directory
err := obf.LoadContext("obfuscated_dir")
if err != nil {
    log.Fatalf("Failed to load context: %v", err)
}

// Look up an obfuscated name
originalName, err := obf.LookupObfuscatedName("scrambled_name", "function")
if err != nil {
    log.Printf("Lookup failed: %v", err)
} else {
    log.Printf("Original name: %s", originalName)
}
```

### Configuration

Create a `config.yaml` file (or use `-c path/to/config.yaml`) to customize obfuscation settings. The tool looks for `config.yaml` in the current directory by default.

Here's an example configuration with explanations:

```yaml
# PHP Obfuscator Configuration

# General behavior settings
silent: false                    # Suppress informational messages (overridden by --silent flag)
abort_on_error: true             # Stop processing on the first error (overridden by --abort-on-error flag)
debug_mode: false                # Enable verbose debug logging for development
parser_mode: "PREFER_PHP7"       # PHP Parser version preference (e.g., PREFER_PHP7, PREFER_PHP8)

# --- Obfuscation Techniques ---
obfuscation:
  # String obfuscation settings
  strings:
    enabled: true                # Enable string literal obfuscation (overridden by --obfuscate-strings)
    technique: "base64"          # Options: "base64", "rot13", "xor"
    # Optional: Define XOR key if technique is XOR
    # xor_key: "YourSecretKey"

  # Name scrambling settings
  scrambling:
    mode: "identifier"           # Name generation style: "identifier", "hexa", "numeric"
    length: 5                    # Target length for generated names

  # Comment handling
  comments:
    strip: false                 # Remove comments from the PHP code (overridden by --strip-comments)

  # --- Granular Name Scrambling Toggles ---
  # Set 'scramble: true' to enable obfuscation for that specific type
  variables: { scramble: false } # Obfuscate variable names ($var)
  functions: { scramble: false } # Obfuscate function names (myFunc())
  classes:   { scramble: false } # Obfuscate class/interface/trait names (MyClass)
  # Add more specific types if needed (e.g., properties, methods, constants - check future versions)

  # Control flow obfuscation
  control_flow:
    enabled: true                # Wrap blocks in if(true){} statements (overridden by --control-flow)
    max_nesting_depth: 2         # Max depth for nested if(true){} wrappers
    random_conditions: true      # Use random conditions (e.g., 1==1, 5>2) instead of just 'true'
    add_dead_branches: false     # Add empty or junk 'else {}' blocks to the if(true){} wrappers

  # Advanced loop obfuscation (potentially more complex/fragile)
  advanced_loops:
    enabled: false               # Enable techniques like sentinel variables in loops

  # Array access obfuscation (may affect performance or compatibility)
  array_access:
    enabled: true                # Transform $a['b'] to _phpmix_array_get($a, 'b') (overridden by --array-access)

  # Arithmetic expression obfuscation (experimental)
  # arithmetic_expressions:
  #   enabled: false
  #   complexity_level: 1        # Controls complexity (e.g., 1-3)
  #   transformation_rate: 50    # Percentage of eligible expressions to transform (0-100)

  # Dead code injection (code that never runs)
  dead_code:
    enabled: false               # Insert dead code blocks like if(false){...} (overridden by --dead-code)
    injection_rate: 30           # Percentage chance (0-100) to inject dead code at each possible location

  # Junk code insertion (code that runs but has no effect)
  junk_code:
    enabled: false               # Insert useless code like unused vars, pointless math (overridden by --junk-code)
    injection_rate: 30           # Percentage chance (0-100) to inject junk code at each possible location
    max_injection_depth: 3       # Limit how deep junk code can be injected inside nested structures

  # Statement shuffling (reorders independent lines of code)
  statement_shuffling:
    enabled: true                # Shuffle independent statements within blocks (overridden by --shuffle-statements)
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
# target_directory: ""           # Output directory (set internally by CLI flag -o/--output or API call)

```
## Testing

The project uses Go's standard testing tools. Tests are categorized into unit and integration tests.

### Unit Tests

Unit tests verify the functionality of individual packages. They are located in `_test.go` files alongside the code they test within the `internal/` and `pkg/` directories, following Go conventions. This allows testing of both exported and unexported package members.

To run all unit tests:

```bash
go test ./internal/... ./pkg/...
# Or simply run from the root:
go test ./...
```

Input data for unit tests is located in package-specific `testdata` directories:
- `internal/obfuscator/testdata/` - Test data for obfuscator unit tests
- `internal/transformer/testdata/` - Test data for transformer unit tests
- `internal/scrambler/testdata/` - Test data for scrambler unit tests
- `pkg/api/testdata/` - Test data for API unit tests

### Integration Tests

Integration tests verify end-to-end features, such as ensuring specific obfuscation techniques produce valid and runnable PHP code. These tests are located in the `/tests/integration` directory. They often use the public `pkg/api` or invoke the compiled binary and may execute the resulting PHP code using an installed PHP interpreter.

**Prerequisites:** Integration tests require PHP to be installed and available in your PATH. Tests will be skipped if PHP is not available.

To run all integration tests:

```bash
go test ./tests/...
```

Input data specific to integration tests is found in `/tests/testdata/integration`.

### Running All Tests

To run both unit and integration tests:

```bash
go test ./... ./tests/...
```

### Testing Tips

- **Verbose output:** Add the `-v` flag to see detailed test output:
  ```bash
  go test -v ./... ./tests/...
  ```

- **Running specific tests:** You can run tests in a specific package or file:
  ```bash
  # Test a specific package
  go test -v ./internal/transformer/...

  # Test a specific component
  go test -v ./tests/integration/array_access/...
  ```

- **Test with coverage:** To see test coverage information:
  ```bash
  go test -cover ./...

  # For detailed coverage by function:
  go test -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out

  # For HTML coverage report:
  go tool cover -html=coverage.out
  ```

- **Run tests multiple times:** To verify tests are stable (especially integration tests):
  ```bash
  go test -count=5 ./tests/...
  ```

- **Race detection:** To detect race conditions in concurrent code:
  ```bash
  go test -race ./...
  ```

## License

This project is licensed under MIT License - see the [LICENSE](LICENSE) file for details.
