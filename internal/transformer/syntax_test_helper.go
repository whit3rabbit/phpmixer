package transformer

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// validatePHPSyntax checks if the given PHP code is syntactically valid
func validatePHPSyntax(t *testing.T, phpCode string) {
	t.Helper()
	
	// Create a temporary file with the PHP code
	tmpFile, err := os.CreateTemp("", "test_*.php")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(phpCode); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Run php -l to check syntax
	cmd := exec.Command("php", "-l", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		t.Errorf("PHP syntax validation failed:\nCode:\n%s\nError: %v\nOutput: %s", 
			phpCode, err, string(output))
	}
	
	if strings.Contains(string(output), "syntax error") {
		t.Errorf("PHP syntax error detected:\nCode:\n%s\nOutput: %s", 
			phpCode, string(output))
	}
}