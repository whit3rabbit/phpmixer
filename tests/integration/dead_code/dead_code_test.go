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

func TestDeadJunkCodeInjection(t *testing.T) {
	// Skip test if PHP is not available
	if _, err := exec.LookPath("php"); err != nil {
		t.Skip("PHP not available, skipping integration test")
	}

	// Test different configurations
	testCases := []struct {
		name              string
		injectDeadCode    bool
		injectJunkCode    bool
		injectionRate     int
		maxInjectionDepth int
	}{
		{
			name:              "DeadCodeOnly",
			injectDeadCode:    true,
			injectJunkCode:    false,
			injectionRate:     75,
			maxInjectionDepth: 3,
		},
		{
			name:              "JunkCodeOnly",
			injectDeadCode:    false,
			injectJunkCode:    true,
			injectionRate:     75,
			maxInjectionDepth: 3,
		},
		{
			name:              "BothTypesHighRate",
			injectDeadCode:    true,
			injectJunkCode:    true,
			injectionRate:     90,
			maxInjectionDepth: 3,
		},
		{
			name:              "BothTypesLowRate",
			injectDeadCode:    true,
			injectJunkCode:    true,
			injectionRate:     30,
			maxInjectionDepth: 2,
		},
	}

	// Get the test file path
	inputFile := filepath.Join("..", "..", "testdata", "integration", "dead_code", "input.php")
	absPath, err := filepath.Abs(inputFile)
	require.NoError(t, err, "Error getting absolute path")

	// Run original file to verify it works and get baseline output
	t.Log("=== Original PHP Output ===")
	originalOutput, err := runPHP(t, absPath)
	require.NoError(t, err, "Error running original PHP file")
	t.Log(originalOutput)

	// Create temporary directory for test outputs
	tmpDir := t.TempDir()

	// Run tests with different configurations
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create configuration
			cfg := &config.Config{
				DebugMode:             true,
				InjectDeadCode:        tc.injectDeadCode,
				InjectJunkCode:        tc.injectJunkCode,
				DeadJunkInjectionRate: tc.injectionRate,
				MaxInjectionDepth:     tc.maxInjectionDepth,
				// Disable other obfuscations for clean test
				ObfuscateArrayAccess:   false,
				ObfuscateControlFlow:   false,
				ObfuscateStringLiteral: false,
			}

			// Create obfuscation context
			ctx, err := obfuscator.NewObfuscationContext(cfg)
			require.NoError(t, err, "Error creating obfuscation context")

			// Process the file
			t.Logf("Processing file with: Dead Code=%v, Junk Code=%v, Rate=%d%%, Depth=%d",
				tc.injectDeadCode, tc.injectJunkCode, tc.injectionRate, tc.maxInjectionDepth)
			obfuscated, err := obfuscator.ProcessFile(absPath, ctx)
			require.NoError(t, err, "Error obfuscating file")

			// Write output to file
			outputFile := filepath.Join(tmpDir, "dead_junk_test_"+tc.name+".php")
			err = os.WriteFile(outputFile, []byte(obfuscated), 0644)
			require.NoError(t, err, "Error writing output file")

			t.Logf("Successfully wrote obfuscated file to: %s", outputFile)

			// Compare file sizes
			originalSize := int64(len(originalOutput))
			obfuscatedSize := int64(len(obfuscated))
			sizeDiff := obfuscatedSize - originalSize
			percentIncrease := float64(sizeDiff) / float64(originalSize) * 100

			t.Logf("Original size: %d bytes", originalSize)
			t.Logf("Obfuscated size: %d bytes", obfuscatedSize)
			t.Logf("Size increase: %d bytes (%.2f%%)", sizeDiff, percentIncrease)

			// Run the obfuscated file
			t.Logf("=== Obfuscated PHP Output (%s) ===", tc.name)
			obfuscatedOutput, err := runPHP(t, outputFile)
			require.NoError(t, err, "Error running obfuscated PHP file")
			t.Log(obfuscatedOutput)

			// Verify that the outputs match (ignoring whitespace differences)
			assert.Equal(t,
				strings.TrimSpace(originalOutput),
				strings.TrimSpace(obfuscatedOutput),
				"Original and obfuscated outputs should match")

			// Verify file size increased (except for the "None" test case)
			if tc.injectDeadCode || tc.injectJunkCode {
				assert.Greater(t, obfuscatedSize, originalSize,
					"Obfuscated file should be larger than original when injection is enabled")
			}
		})
	}
}

// runPHP executes a PHP file and returns its output
func runPHP(t *testing.T, file string) (string, error) {
	t.Helper()
	cmd := exec.Command("php", file)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

