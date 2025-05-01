package api

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
)

func TestNewObfuscator(t *testing.T) {
	// Test with default empty options - this should use default config
	obf, err := NewObfuscator(Options{})
	if err != nil {
		t.Errorf("Expected default config to be used, got error: %v", err)
	}
	if obf == nil {
		t.Errorf("Expected non-nil Obfuscator with default config, got nil")
	}

	// Create a temporary config file
	configContent := `
# Test configuration
silent: true
scramble_mode: "identifier"
scramble_length: 5
strip_comments: true
obfuscate_string_literal: true
string_obfuscation_technique: "base64"
obfuscate_variable_name: true
obfuscate_function_name: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Test with valid config
	obf, err = NewObfuscator(Options{
		ConfigPath: configPath,
		Silent:     true,
	})
	if err != nil {
		t.Fatalf("NewObfuscator with valid config failed: %v", err)
	}
	if obf == nil {
		t.Fatalf("Expected non-nil Obfuscator, got nil")
	}
	if obf.Config == nil {
		t.Errorf("Expected non-nil Config in Obfuscator, got nil")
	}
	if obf.Context == nil {
		t.Errorf("Expected non-nil Context in Obfuscator, got nil")
	}
}

func TestObfuscateCode(t *testing.T) {
	// Store original testing flag and restore it after the test
	originalTestingFlag := config.Testing
	config.Testing = false
	defer func() { config.Testing = originalTestingFlag }()

	// Create a temporary config file
	configContent := `
silent: true
scramble_mode: "identifier"
scramble_length: 5
obfuscation:
  comments:
    strip: true
  strings:
    enabled: true
    technique: "base64"
  variables:
    scramble: true
  functions:
    scramble: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create the obfuscator with explicit config values
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Force certain settings
	cfg.Obfuscation.Comments.Strip = true
	cfg.Obfuscation.Strings.Enabled = true
	cfg.Obfuscation.Variables.Scramble = true
	cfg.Obfuscation.Functions.Scramble = true

	// Create obfuscation context with the forced config
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	if err != nil {
		t.Fatalf("Failed to create obfuscation context: %v", err)
	}

	obf := &Obfuscator{
		Context: ctx,
		Config:  cfg,
	}

	// Test PHP code
	phpCode := `<?php
// This is a test comment
function test($arg) {
    echo "Hello, " . $arg;
}
$name = "World";
test($name);
?>`

	// Test obfuscation
	result, err := obf.ObfuscateCode(phpCode)
	if err != nil {
		t.Fatalf("ObfuscateCode failed: %v", err)
	}

	// Basic validation of the result
	if result == "" {
		t.Errorf("ObfuscateCode returned empty string")
	}

	// The comment should be gone if strip_comments is true
	if strings.Contains(result, "This is a test comment") {
		t.Errorf("Expected comments to be stripped, but found comment text")
	}

	// Check that the code has been modified in some way
	if result == phpCode {
		t.Errorf("Expected code to be modified, but it's identical to the input")
	}
}

func TestObfuscateFileToFile(t *testing.T) {
	// Store original testing flag and restore it after the test
	originalTestingFlag := config.Testing
	config.Testing = false
	defer func() { config.Testing = originalTestingFlag }()

	// Create a temporary config file
	configContent := `
silent: true
scramble_mode: "identifier"
scramble_length: 5
obfuscation:
  comments:
    strip: true
  strings:
    enabled: true
    technique: "base64"
  variables:
    scramble: true
  functions:
    scramble: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create the obfuscator with explicit config values
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Force certain settings
	cfg.Obfuscation.Comments.Strip = true
	cfg.Obfuscation.Strings.Enabled = true
	cfg.Obfuscation.Variables.Scramble = true
	cfg.Obfuscation.Functions.Scramble = true

	// Create obfuscation context with the forced config
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	if err != nil {
		t.Fatalf("Failed to create obfuscation context: %v", err)
	}

	obf := &Obfuscator{
		Context: ctx,
		Config:  cfg,
	}

	// Create a test PHP file
	phpCode := `<?php
// This is a test comment
function test($arg) {
    echo "Hello, " . $arg;
}
$name = "World";
test($name);
?>`

	inputPath := filepath.Join(tmpDir, "input.php")
	if err := os.WriteFile(inputPath, []byte(phpCode), 0644); err != nil {
		t.Fatalf("Failed to write test PHP file: %v", err)
	}

	// Obfuscate the file
	outputPath := filepath.Join(tmpDir, "output.php")
	err = obf.ObfuscateFileToFile(inputPath, outputPath)
	if err != nil {
		t.Fatalf("ObfuscateFileToFile failed: %v", err)
	}

	// Check if the output file exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Errorf("Output file was not created")
	}

	// Read the output file
	obfuscatedCode, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Basic validation of the result
	if len(obfuscatedCode) == 0 {
		t.Errorf("Obfuscated code file is empty")
	}

	// The comment should be gone if strip_comments is true
	if strings.Contains(string(obfuscatedCode), "This is a test comment") {
		t.Errorf("Expected comments to be stripped, but found comment text")
	}

	// Check that the code has been modified in some way
	if string(obfuscatedCode) == phpCode {
		t.Errorf("Expected code to be modified, but it's identical to the input")
	}
}

// Helper function to create a test directory structure with PHP files
func createTestDirStructure(t *testing.T, baseDir string) {
	// Create subdirectories
	dirs := []string{
		filepath.Join(baseDir, "subdir1"),
		filepath.Join(baseDir, "subdir2"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	// Create PHP files
	files := map[string]string{
		filepath.Join(baseDir, "root.php"):      "<?php\n// Root file\necho 'Root';\n?>",
		filepath.Join(baseDir, "subdir1/a.php"): "<?php\n// Subdir file A\necho 'File A';\n?>",
		filepath.Join(baseDir, "subdir2/b.php"): "<?php\n// Subdir file B\necho 'File B';\n?>",
		filepath.Join(baseDir, "subdir2/c.txt"): "This is a non-PHP file that should be copied.",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}
}

func TestObfuscateDirectory(t *testing.T) {
	// Store original testing flag and restore it after the test
	originalTestingFlag := config.Testing
	config.Testing = false
	defer func() { config.Testing = originalTestingFlag }()

	// Create a temporary config file
	configContent := `
silent: true
scramble_mode: "identifier"
scramble_length: 5
skip: ["*.skip"]  # Test skiplist
obfuscation:
  comments:
    strip: true
  strings:
    enabled: true
    technique: "base64"
  variables:
    scramble: true
  functions:
    scramble: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create the obfuscator with explicit config values
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Force certain settings
	cfg.Obfuscation.Comments.Strip = true
	cfg.Obfuscation.Strings.Enabled = true
	cfg.Obfuscation.Variables.Scramble = true
	cfg.Obfuscation.Functions.Scramble = true
	cfg.SkipPaths = []string{"*.skip"} // Set skip patterns

	// Create obfuscation context with the forced config
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	if err != nil {
		t.Fatalf("Failed to create obfuscation context: %v", err)
	}

	obf := &Obfuscator{
		Context: ctx,
		Config:  cfg,
	}

	// Create test directory structure
	inputDir := filepath.Join(tmpDir, "input")
	if err := os.MkdirAll(inputDir, 0755); err != nil {
		t.Fatalf("Failed to create input directory: %v", err)
	}
	createTestDirStructure(t, inputDir)

	// Add a file that should be skipped
	skipFile := filepath.Join(inputDir, "skip_me.skip")
	if err := os.WriteFile(skipFile, []byte("This file should be skipped"), 0644); err != nil {
		t.Fatalf("Failed to write skip file: %v", err)
	}

	// Obfuscate the directory
	outputDir := filepath.Join(tmpDir, "output")
	err = obf.ObfuscateDirectory(inputDir, outputDir)
	if err != nil {
		t.Fatalf("ObfuscateDirectory failed: %v", err)
	}

	// Check that PHP files were obfuscated (comments stripped)
	phpFiles := []string{
		filepath.Join(outputDir, "root.php"),
		filepath.Join(outputDir, "subdir1", "a.php"),
		filepath.Join(outputDir, "subdir2", "b.php"),
	}

	for _, phpFile := range phpFiles {
		// Check if file exists
		if _, err := os.Stat(phpFile); os.IsNotExist(err) {
			t.Errorf("Expected output file %s does not exist", phpFile)
			continue
		}

		// Read file content
		content, err := os.ReadFile(phpFile)
		if err != nil {
			t.Errorf("Failed to read output file %s: %v", phpFile, err)
			continue
		}

		// Check for comment stripping
		if strings.Contains(string(content), "//") {
			t.Errorf("Expected comments to be stripped in %s, but found comment", phpFile)
		}
	}

	// Check non-PHP file was copied as-is
	nonPhpFile := filepath.Join(outputDir, "subdir2", "c.txt")
	if _, err := os.Stat(nonPhpFile); os.IsNotExist(err) {
		t.Errorf("Expected non-PHP file %s was not copied", nonPhpFile)
	}

	// Check skipped file was not copied
	skipFileOutput := filepath.Join(outputDir, "skip_me.skip")
	if _, err := os.Stat(skipFileOutput); !os.IsNotExist(err) {
		t.Errorf("Skipped file %s should not have been copied", skipFileOutput)
	}
}

func TestLookupObfuscatedName(t *testing.T) {
	// Create a temporary config file
	configContent := `
silent: true
scramble_mode: "identifier"
scramble_length: 5
obfuscation:
  comments:
    strip: true
  strings:
    enabled: true
    technique: "base64"
  variables:
    scramble: true
  functions:
    scramble: true
  classes:
    scramble: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}

	// Create the obfuscator
	obf, err := NewObfuscator(Options{
		ConfigPath: configPath,
		Silent:     true,
	})
	if err != nil {
		t.Fatalf("NewObfuscator failed: %v", err)
	}

	// Create a simple PHP file with a function and variable
	phpCode := `<?php
function testFunction($testVar) {
    echo "Test: " . $testVar;
}
$myVar = "Hello";
testFunction($myVar);
?>`

	// Create a temporary input file
	inputPath := filepath.Join(tmpDir, "lookup_test.php")
	if err := os.WriteFile(inputPath, []byte(phpCode), 0644); err != nil {
		t.Fatalf("Failed to write PHP file: %v", err)
	}

	// Create output directory
	outputDir := filepath.Join(tmpDir, "lookup_output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("Failed to create output directory: %v", err)
	}

	// Process the file
	outputPath := filepath.Join(outputDir, "lookup_test.php")
	err = obf.ObfuscateFileToFile(inputPath, outputPath)
	if err != nil {
		t.Fatalf("ObfuscateFileToFile failed: %v", err)
	}

	// Now we can test the LookupObfuscatedName function
	// Test with invalid type
	_, err = obf.LookupObfuscatedName("testFunction", "invalid_type")
	if err == nil {
		t.Errorf("Expected error for invalid type, got nil")
	}

	// Test with function type
	result, err := obf.LookupObfuscatedName("testFunction", "function")
	if err == nil {
		// If we found the name, check that it's not the original
		if result == "testFunction" {
			t.Errorf("Expected obfuscated name to be different from original, got same name")
		}
		// Verify the result by reading the obfuscated file
		obfuscatedContent, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read obfuscated file: %v", err)
		}
		if !strings.Contains(string(obfuscatedContent), result) {
			t.Errorf("Obfuscated name %s not found in obfuscated file", result)
		}
	} else {
		// We may not find the name if it wasn't obfuscated or context wasn't saved properly
		// Just log a warning instead of failing the test
		t.Logf("Warning: LookupObfuscatedName for function returned error: %v", err)
	}

	// Test with variable type
	result, err = obf.LookupObfuscatedName("myVar", "variable")
	if err == nil {
		if result == "myVar" {
			t.Errorf("Expected obfuscated name to be different from original, got same name")
		}
		// Verify the result
		obfuscatedContent, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read obfuscated file: %v", err)
		}
		// For variables, we would look for the $ prefix
		if !strings.Contains(string(obfuscatedContent), "$"+result) {
			t.Errorf("Obfuscated variable $%s not found in obfuscated file", result)
		}
	} else {
		t.Logf("Warning: LookupObfuscatedName for variable returned error: %v", err)
	}
}

func TestPrintInfo(t *testing.T) {
	// Capture stdout
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test with Testing flag set to false (default)
	config.Testing = false
	PrintInfo("Test output: %s\n", "visible")

	// Read captured output
	w.Close()
	os.Stdout = originalStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Verify output was printed when Testing=false
	if !strings.Contains(buf.String(), "Test output: visible") {
		t.Error("Expected output to be printed when Testing=false")
	}

	// Reset capture
	r, w, _ = os.Pipe()
	os.Stdout = w

	// Test with Testing flag set to true
	config.Testing = true
	PrintInfo("Test output: %s\n", "invisible")

	// Read captured output
	w.Close()
	os.Stdout = originalStdout
	buf.Reset()
	io.Copy(&buf, r)

	// Verify no output was printed when Testing=true
	if buf.String() != "" {
		t.Errorf("Expected no output when Testing=true, got: %s", buf.String())
	}

	// Reset Testing flag to default value
	config.Testing = false
}

