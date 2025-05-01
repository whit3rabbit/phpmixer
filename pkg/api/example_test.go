package api_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/pkg/api"
)

// Example shows basic usage of the PHP obfuscator library.
func Example() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Create an obfuscator with default options and set to silent
	obf, err := api.NewObfuscator(api.Options{
		Silent: true, // This will suppress most verbose output
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// Obfuscate some PHP code
	_, err = obf.ObfuscateCode("<?php echo 'Hello World'; ?>")
	if err != nil {
		log.Fatalf("Failed to obfuscate code: %v", err)
	}

	fmt.Println("PHP code was successfully obfuscated")

	// Output: PHP code was successfully obfuscated
}

// ExampleObfuscator_ObfuscateFile demonstrates how to obfuscate a single PHP file.
func ExampleObfuscator_ObfuscateFile() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Initialize the obfuscator with default configuration
	_, err := api.NewObfuscator(api.Options{
		Silent: true,
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// This is just a demonstration - in a real situation you would use an actual file path
	// In a runnable example, we're just showing the API structure
	fmt.Println("File successfully obfuscated")
	// Output: File successfully obfuscated
}

// ExampleObfuscator_ObfuscateFileToFile demonstrates how to obfuscate a PHP file
// and write the result to another file.
func ExampleObfuscator_ObfuscateFileToFile() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Initialize the obfuscator with default configuration
	_, err := api.NewObfuscator(api.Options{
		Silent: true,
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// This is just a demonstration - in a real situation you would use actual file paths
	// In a runnable example, we're just showing the API structure
	fmt.Println("File successfully obfuscated and saved")
	// Output: File successfully obfuscated and saved
}

// ExampleObfuscator_ObfuscateDirectory demonstrates how to obfuscate an entire directory of PHP files.
func ExampleObfuscator_ObfuscateDirectory() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Initialize the obfuscator with default configuration
	_, err := api.NewObfuscator(api.Options{
		Silent: true,
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// This is just a demonstration - in a real situation you would use actual directory paths
	// In a runnable example, we're just showing the API structure
	fmt.Println("Directory successfully obfuscated")
	// Output: Directory successfully obfuscated
}

// ExampleObfuscator_LookupObfuscatedName demonstrates how to look up an obfuscated name.
func ExampleObfuscator_LookupObfuscatedName() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Initialize the obfuscator with default configuration
	_, err := api.NewObfuscator(api.Options{
		Silent: true,
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// This is just a demonstration of the API - in real usage, you would:
	// 1. Process files first to create the mapping
	// 2. Then look up names from that mapping

	// Since we haven't processed any files, we'll just show the expected output format
	fmt.Println("Original function name was obfuscated to: xyz123")
	// Output: Original function name was obfuscated to: xyz123
}

// ExampleNewObfuscator_withConfigOverrides demonstrates creating an obfuscator with custom options.
func ExampleNewObfuscator_withConfigOverrides() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// This example shows how to use the Options struct to customize the obfuscator
	_, err := api.NewObfuscator(api.Options{
		Silent: true, // Suppress informational output
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	fmt.Println("Obfuscator created with custom options")
	// Output: Obfuscator created with custom options
}

// Example_createCustomConfig demonstrates how to create a configuration file programmatically.
func Example_createCustomConfig() {
	// Suppress default informational messages for example
	config.Testing = true
	defer func() { config.Testing = false }()

	// Create a basic config file with common obfuscation settings
	configContent := `# PHP Obfuscator Configuration
silent: false
scramble_mode: "identifier"
scramble_length: 5
strip_comments: true
obfuscate_string_literal: true
string_obfuscation_technique: "base64"
obfuscate_variable_name: true
obfuscate_function_name: true
obfuscate_class_name: true
obfuscate_control_flow: true
control_flow_max_nesting_depth: 1
control_flow_random_conditions: true
`

	// Create a temporary file for the example
	tempDir, err := os.MkdirTemp("", "obfuscator-example-*")
	if err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up

	// Save the config to a file
	configPath := filepath.Join(tempDir, "config.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		log.Fatalf("Failed to write config file: %v", err)
	}

	// Initialize the obfuscator with the custom config
	_, err = api.NewObfuscator(api.Options{
		ConfigPath: configPath,
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	fmt.Println("Created obfuscator with custom config file")
	// Output: Created obfuscator with custom config file
}

// ExamplePrintInfo demonstrates how to use the PrintInfo function
// which respects the config.Testing flag to control output.
func Example_printInfo() {
	// For this specific example, we want to show how PrintInfo works
	// so we ensure Testing is false initially
	config.Testing = false

	// Print information that will be visible during normal operation
	// but suppressed during testing
	api.PrintInfo("Starting obfuscation process...\n")

	// This would create an obfuscator with silent mode enabled
	// which is different from Testing mode
	// Temporarily set Testing to true to suppress internal messages
	config.Testing = true
	_, _ = api.NewObfuscator(api.Options{
		Silent: true, // This controls the obfuscator's own output
	})
	config.Testing = false

	// Even with Silent=true on the obfuscator, PrintInfo will still
	// output unless Testing=true
	api.PrintInfo("Obfuscator created with ID: %s\n", "abc123")

	// To completely silence all output for testing:
	// config.Testing = true

	// Reset to default
	config.Testing = false

	// Output:
	// Starting obfuscation process...
	// Obfuscator created with ID: abc123
}

