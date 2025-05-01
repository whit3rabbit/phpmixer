package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
)

func TestArrayAccessObfuscation(t *testing.T) {
	// Skip test if PHP is not available
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("PHP not available, skipping integration test")
	}

	// Create a configuration with array access obfuscation enabled
	cfg := &config.Config{
		DebugMode:              true,
		ObfuscateArrayAccess:   true,
		StripComments:          false,
		ObfuscateControlFlow:   false,
		ObfuscateStringLiteral: false,
	}

	// Create the obfuscation context
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	require.NoError(t, err, "Error creating obfuscation context")

	// Get the test file path
	inputFile := filepath.Join("..", "..", "testdata", "integration", "array_access", "input.php")
	absPath, err := filepath.Abs(inputFile)
	require.NoError(t, err, "Error getting absolute path")

	// Process the file
	t.Logf("Processing file: %s", absPath)
	obfuscated, err := obfuscator.ProcessFile(absPath, ctx)
	require.NoError(t, err, "Error obfuscating file")

	// Create temporary directory for test outputs
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "array_access_obfuscated.php")
	err = os.WriteFile(outputFile, []byte(obfuscated), 0644)
	require.NoError(t, err, "Error writing output file")

	t.Logf("Successfully wrote obfuscated file to: %s", outputFile)

	// Run PHP to test if the obfuscated code works
	t.Log("=== Original PHP Output ===")
	originalOutput, err := runPHP(t, absPath)
	require.NoError(t, err, "Error running original PHP file")
	t.Log(originalOutput)

	t.Log("=== Obfuscated PHP Output ===")
	obfuscatedOutput, err := runPHP(t, outputFile)
	require.NoError(t, err, "Error running obfuscated PHP file")
	t.Log(obfuscatedOutput)

	// Verify that the outputs match (ignoring whitespace differences)
	assert.Equal(t,
		strings.TrimSpace(originalOutput),
		strings.TrimSpace(obfuscatedOutput),
		"Original and obfuscated outputs should match")
}

// runPHP executes a PHP file and returns its output
func runPHP(t *testing.T, file string) (string, error) {
	t.Helper()
	cmd := exec.Command("php", file)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

