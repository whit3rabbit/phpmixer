package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
)

// PhpRunner provides utilities for running PHP in integration tests
type PhpRunner struct {
	T *testing.T
}

// NewPhpRunner creates a new PHP runner for integration tests
func NewPhpRunner(t *testing.T) *PhpRunner {
	return &PhpRunner{T: t}
}

// SkipIfPhpNotAvailable skips the test if PHP is not installed
func (r *PhpRunner) SkipIfPhpNotAvailable() {
	if _, err := exec.LookPath("php"); err != nil {
		r.T.Skip("PHP not available, skipping integration test")
	}
}

// RunPHP executes a PHP file and returns its output
func (r *PhpRunner) RunPHP(file string) (string, error) {
	r.T.Helper()
	cmd := exec.Command("php", file)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ObfuscateFile obfuscates a PHP file with the given configuration and writes the output to a temporary file
func (r *PhpRunner) ObfuscateFile(inputFile string, cfg *config.Config) (string, string, error) {
	r.T.Helper()

	// Create the obfuscation context
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	if err != nil {
		return "", "", err
	}

	// Process the file
	r.T.Logf("Processing file: %s", inputFile)
	obfuscated, err := obfuscator.ProcessFile(inputFile, ctx)
	if err != nil {
		return "", "", err
	}

	// Create temporary directory for test outputs if needed
	tmpDir := r.T.TempDir()
	outputFileName := filepath.Base(inputFile)
	outputFileName = "obfuscated_" + outputFileName
	outputFile := filepath.Join(tmpDir, outputFileName)

	err = os.WriteFile(outputFile, []byte(obfuscated), 0644)
	if err != nil {
		return "", "", err
	}

	r.T.Logf("Successfully wrote obfuscated file to: %s", outputFile)

	return outputFile, obfuscated, nil
}

// TestPhpFile runs both the original and obfuscated PHP file and compares outputs
func (r *PhpRunner) TestPhpFile(originalFile, obfuscatedFile string) (string, string, error) {
	r.T.Helper()

	// Run original PHP file
	r.T.Log("=== Original PHP Output ===")
	originalOutput, err := r.RunPHP(originalFile)
	if err != nil {
		return "", "", err
	}
	r.T.Log(originalOutput)

	// Run obfuscated PHP file
	r.T.Log("=== Obfuscated PHP Output ===")
	obfuscatedOutput, err := r.RunPHP(obfuscatedFile)
	if err != nil {
		return "", "", err
	}
	r.T.Log(obfuscatedOutput)

	return originalOutput, obfuscatedOutput, nil
}

// IntegrationTest runs a complete integration test with the given config and input file
func (r *PhpRunner) IntegrationTest(inputFile string, cfg *config.Config) (string, string, string, error) {
	r.T.Helper()

	// Skip test if PHP is not available
	r.SkipIfPhpNotAvailable()

	// Get absolute path
	absPath, err := filepath.Abs(inputFile)
	require.NoError(r.T, err, "Error getting absolute path")

	// Obfuscate the file
	obfuscatedFile, obfuscatedCode, err := r.ObfuscateFile(absPath, cfg)
	if err != nil {
		return "", "", "", err
	}

	// Test the files
	originalOutput, obfuscatedOutput, err := r.TestPhpFile(absPath, obfuscatedFile)
	if err != nil {
		return "", "", "", err
	}

	return originalOutput, obfuscatedOutput, obfuscatedCode, nil
}

