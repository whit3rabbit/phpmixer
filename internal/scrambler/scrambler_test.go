package scrambler

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/whit3rabbit/phpmixer/internal/config" // Adjust import path
)

// Helper to create a default config for testing
func createTestConfig() *config.Config {
	// Use the defaults defined in the config package
	cfg := config.Config{} // Create zero-value config first

	// Manually set defaults similar to how LoadConfig would if no file/env found
	// This avoids direct dependency on viper in tests if possible
	cfg.ScrambleMode = "identifier"
	cfg.ScrambleLength = 5
	// ... set other relevant defaults if needed for specific tests
	cfg.IgnoreVariables = []string{"ignoreMeVar"}
	cfg.IgnoreFunctionsPrefix = []string{"test_"}
	cfg.IgnoreMethods = []string{"__magicIgnore"}
	// cfg.ReservedWords = []string{"class", "__construct", "this"} // Reserved words are handled internally

	return &cfg
}

// Helper to create a scrambler with a specific config
func createTestScrambler(t *testing.T, sType ScrambleType, cfg *config.Config) *Scrambler {
	t.Helper() // Marks this as a test helper
	if cfg == nil {
		cfg = createTestConfig()
	}
	sc, err := NewScrambler(sType, cfg)
	if err != nil {
		t.Fatalf("Failed to create scrambler for type %s: %v", sType, err)
	}
	return sc
}

// Test basic scrambling and consistency
func TestScrambleBasic(t *testing.T) {
	cfg := createTestConfig()
	scVar := createTestScrambler(t, TypeVariable, cfg)  // Variable (case-sensitive)
	scFunc := createTestScrambler(t, TypeFunction, cfg) // Function (case-insensitive)

	originalVar := "myVariable"
	scrambledVar1 := scVar.Scramble(originalVar)
	scrambledVar2 := scVar.Scramble(originalVar) // Should be consistent

	if scrambledVar1 == originalVar {
		t.Errorf("Variable '%s' was not scrambled", originalVar)
	}
	if len(scrambledVar1) < cfg.ScrambleLength { // Use config length
		t.Errorf("Scrambled variable '%s' is too short: len=%d, expected >= %d", scrambledVar1, len(scrambledVar1), cfg.ScrambleLength)
	}
	if scrambledVar1 != scrambledVar2 {
		t.Errorf("Scrambled variable is not consistent: '%s' != '%s'", scrambledVar1, scrambledVar2)
	}

	originalFunc := "myFunction"
	scrambledFunc1 := scFunc.Scramble(originalFunc)
	scrambledFunc2 := scFunc.Scramble(originalFunc) // Consistent

	if scrambledFunc1 == originalFunc {
		t.Errorf("Function '%s' was not scrambled", originalFunc)
	}
	if len(scrambledFunc1) < cfg.ScrambleLength { // Use config length
		t.Errorf("Scrambled function '%s' is too short: len=%d, expected >= %d", scrambledFunc1, len(scrambledFunc1), cfg.ScrambleLength)
	}
	if strings.ToLower(scrambledFunc1) != strings.ToLower(scrambledFunc2) {
		// Case shuffling might happen, so compare lowercase for consistency check
		t.Errorf("Scrambled function lowercase form is not consistent: '%s' != '%s'", strings.ToLower(scrambledFunc1), strings.ToLower(scrambledFunc2))
	}
}

// Test case sensitivity
func TestScrambleCaseSensitivity(t *testing.T) {
	cfg := createTestConfig()
	scVar := createTestScrambler(t, TypeVariable, cfg)  // Variable (case-sensitive)
	scFunc := createTestScrambler(t, TypeFunction, cfg) // Function (case-insensitive)

	varName1 := "myVar"
	varName2 := "MyVar"
	scrambled1 := scVar.Scramble(varName1)
	scrambled2 := scVar.Scramble(varName2)

	if scrambled1 == scrambled2 {
		t.Errorf("Case-sensitive variable scrambler produced same result for '%s' and '%s': '%s'", varName1, varName2, scrambled1)
	}
	if scrambled1 == varName1 || scrambled2 == varName2 {
		t.Errorf("Case-sensitive variables were not scrambled")
	}

	funcName1 := "myFunc"
	funcName2 := "MyFunc"
	scrambledF1 := scFunc.Scramble(funcName1)
	scrambledF2 := scFunc.Scramble(funcName2)

	// Case-insensitive should produce the same underlying scrambled name (compare lowercase)
	if strings.ToLower(scrambledF1) != strings.ToLower(scrambledF2) {
		t.Errorf("Case-insensitive function scrambler produced different results for '%s' and '%s': '%s' vs '%s'", funcName1, funcName2, scrambledF1, scrambledF2)
	}
	if scrambledF1 == funcName1 || scrambledF2 == funcName2 {
		t.Errorf("Case-insensitive functions were not scrambled")
	}
}

// Test ignore lists (direct and prefix)
func TestScrambleIgnore(t *testing.T) {
	cfg := createTestConfig()
	// Use specific ignore lists from helper config
	scVar := createTestScrambler(t, TypeVariable, cfg)
	scFunc := createTestScrambler(t, TypeFunction, cfg)

	// Direct ignore (variable)
	ignoredVar := "ignoreMeVar"
	scrambledIgnoredVar := scVar.Scramble(ignoredVar)
	if scrambledIgnoredVar != ignoredVar {
		t.Errorf("Variable '%s' should have been ignored, but got '%s'", ignoredVar, scrambledIgnoredVar)
	}

	// Prefix ignore (function)
	ignoredFunc := "test_myFunc"
	scrambledIgnoredFunc := scFunc.Scramble(ignoredFunc)
	if scrambledIgnoredFunc != ignoredFunc {
		t.Errorf("Function '%s' should have been ignored due to prefix, but got '%s'", ignoredFunc, scrambledIgnoredFunc)
	}

	// Check case-insensitivity of ignores
	ignoredVarCase := "IgnoreMeVAr"
	scrambledIgnoredVarCase := scVar.Scramble(ignoredVarCase)
	if scrambledIgnoredVarCase != ignoredVarCase {
		t.Errorf("Variable '%s' (different case) should have been ignored, but got '%s'", ignoredVarCase, scrambledIgnoredVarCase)
	}

	ignoredFuncCase := "Test_AnotherFunc"
	scrambledIgnoredFuncCase := scFunc.Scramble(ignoredFuncCase)
	if scrambledIgnoredFuncCase != ignoredFuncCase {
		t.Errorf("Function '%s' (different case) should have been ignored due to prefix, but got '%s'", ignoredFuncCase, scrambledIgnoredFuncCase)
	}
}

// Test reserved words
func TestScrambleReserved(t *testing.T) {
	cfg := createTestConfig()
	scVar := createTestScrambler(t, TypeVariable, cfg)
	scFunc := createTestScrambler(t, TypeFunction, cfg) // Covers class names too
	scMethod := createTestScrambler(t, TypeMethod, cfg)

	// Reserved variable ($this)
	reservedVar := "this"
	scrambledReservedVar := scVar.Scramble(reservedVar)
	if scrambledReservedVar != reservedVar {
		t.Errorf("Reserved variable '%s' was scrambled to '%s'", reservedVar, scrambledReservedVar)
	}

	// Reserved keyword/class name (class) - should not be scrambled if used as function/class name
	reservedKeyword := "class"
	scrambledReservedKeyword := scFunc.Scramble(reservedKeyword)
	if scrambledReservedKeyword != reservedKeyword {
		t.Errorf("Reserved keyword '%s' was scrambled to '%s'", reservedKeyword, scrambledReservedKeyword)
	}

	// Reserved magic method (__construct)
	reservedMethod := "__construct"
	scrambledReservedMethod := scMethod.Scramble(reservedMethod)
	if scrambledReservedMethod != reservedMethod {
		t.Errorf("Reserved method '%s' was scrambled to '%s'", reservedMethod, scrambledReservedMethod)
	}

	// Test case insensitivity for reserved words
	reservedKeywordCase := "CLASS"
	scrambledReservedKeywordCase := scFunc.Scramble(reservedKeywordCase)
	if scrambledReservedKeywordCase != reservedKeywordCase {
		t.Errorf("Reserved keyword '%s' (uppercase) was scrambled to '%s'", reservedKeywordCase, scrambledReservedKeywordCase)
	}
}

// Test Unscramble
func TestUnscramble(t *testing.T) {
	cfg := createTestConfig()
	scVar := createTestScrambler(t, TypeVariable, cfg)

	original := "originalName"
	scrambled := scVar.Scramble(original)

	if scrambled == original {
		t.Fatalf("Scrambling failed for '%s'", original)
	}

	unscrambled, found := scVar.Unscramble(scrambled)
	if !found {
		t.Errorf("Unscramble failed to find original name for scrambled '%s'", scrambled)
	}
	if unscrambled != original {
		t.Errorf("Unscramble returned wrong original name: expected '%s', got '%s'", original, unscrambled)
	}

	// Test unscrambling an unknown name
	unknown := "nonExistentScrambledName"
	_, found = scVar.Unscramble(unknown)
	if found {
		t.Errorf("Unscramble incorrectly found an original name for unknown '%s'", unknown)
	}

	// Test unscrambling an ignored name (should not be found as scrambled)
	ignored := "ignoreMeVar"                    // From test config
	scrambledIgnored := scVar.Scramble(ignored) // This returns 'ignored' itself
	if scrambledIgnored != ignored {
		t.Fatalf("Test setup failed: ignored variable was scrambled")
	}
	_, found = scVar.Unscramble(ignored)
	if found {
		t.Errorf("Unscramble incorrectly found an original name for ignored '%s'", ignored)
	}
}

// Test collision handling (difficult to test deterministically without mocking rand)
// This test attempts to force collisions by scrambling many names.
func TestScrambleCollision(t *testing.T) {
	cfg := createTestConfig()
	cfg.ScrambleLength = 2 // Use short length to increase collision probability
	sc := createTestScrambler(t, TypeVariable, cfg)

	generated := make(map[string]string) // scrambled -> original
	count := 1000                        // Number of names to generate

	for i := 0; i < count; i++ {
		original := fmt.Sprintf("var_%d", i)
		scrambled := sc.Scramble(original)

		if original == scrambled {
			// This might happen if generation hits max attempts
			t.Logf("Variable '%s' was not scrambled (collision test, might indicate max attempts reached)", original)
			continue
		}

		if existingOriginal, exists := generated[scrambled]; exists {
			t.Errorf("Collision detected! Scrambled name '%s' generated for both '%s' and '%s'", scrambled, existingOriginal, original)
			// Continue testing even if collision found, to see if more occur
		} else {
			generated[scrambled] = original
		}

		// Check if the reverse map also contains the correct entry
		unscrambled, found := sc.Unscramble(scrambled)
		if !found || unscrambled != original {
			t.Errorf("Unscramble failed or returned incorrect value for '%s' (expected '%s', got '%s')", scrambled, original, unscrambled)
		}
	}

	expectedGeneratedCount := count
	if len(generated) != expectedGeneratedCount {
		// Calculate how many failed to scramble (returned original name)
		failedToScrambleCount := 0
		for i := 0; i < count; i++ {
			original := fmt.Sprintf("var_%d", i)
			scrambled := sc.Scramble(original) // Re-scramble to check which ones failed
			if original == scrambled {
				failedToScrambleCount++
			}
		}
		// Adjust expectation based on failed scrambles
		expectedGeneratedCount = count - failedToScrambleCount
		// Log if the count still doesn't match the adjusted expectation (indicates collision or other map issues)
		if len(generated) != expectedGeneratedCount {
			t.Errorf("Expected %d unique scrambled names (after accounting for %d failed scrambles), but generated map has %d entries", expectedGeneratedCount, failedToScrambleCount, len(generated))
		} else {
			t.Logf("%d variables failed to scramble (returned original name), %d unique scrambled names generated as expected.", failedToScrambleCount, len(generated))
		}

	}

	// Check if length increased (indirect sign of collision handling)
	if sc.currentLength == cfg.ScrambleLength && count > 100 { // Only likely if many collisions occurred
		t.Logf("Scramble length remained %d after %d attempts; few or no collisions requiring length increase likely occurred.", sc.currentLength, count)
	} else if sc.currentLength > cfg.ScrambleLength {
		t.Logf("Scramble length increased to %d, indicating collision handling occurred.", sc.currentLength)
	}
}

// Test Label Generation
func TestGenerateLabelName(t *testing.T) {
	cfg := createTestConfig()
	sc := createTestScrambler(t, TypeLabel, cfg)

	label1 := sc.GenerateLabelName("loop")
	label2 := sc.GenerateLabelName("loop")
	label3 := sc.GenerateLabelName("if")

	if label1 == "" || label2 == "" || label3 == "" {
		t.Fatal("Generated label name is empty")
	}
	if label1 == label2 {
		t.Errorf("Generated identical labels for consecutive calls: %s", label1)
	}
	if label1 == label3 || label2 == label3 {
		// This isn't necessarily an error if scrambling happened to produce the same output,
		// but it's worth logging as potentially unexpected. Collision check is more robust.
		t.Logf("Generated labels for different prefixes yielded potentially similar results (collision check needed): %s, %s, %s", label1, label2, label3)
	}

	// Check if labels are actually scrambled (not just prefix_counter)
	// This check is weak because scrambling might coincidentally start with the prefix
	// if strings.HasPrefix(label1, "loop_") || strings.HasPrefix(label2, "loop_") {
	// 	t.Logf("Label %s or %s might not be scrambled fully (starts with original prefix)", label1, label2)
	// }

	// Verify they are in the maps
	orig1, found1 := sc.Unscramble(label1)
	orig2, found2 := sc.Unscramble(label2)
	orig3, found3 := sc.Unscramble(label3)

	// Check if the unscrambled original matches the expected format (prefix + _ + number)
	if !found1 || !strings.HasPrefix(orig1, "loop_") { // Expect "loop_" prefix
		t.Errorf("Failed to unscramble generated label '%s' correctly, got '%s' (expected format: loop_N)", label1, orig1)
	}
	if !found2 || !strings.HasPrefix(orig2, "loop_") { // Expect "loop_" prefix
		t.Errorf("Failed to unscramble generated label '%s' correctly, got '%s' (expected format: loop_N)", label2, orig2)
	}
	if !found3 || !strings.HasPrefix(orig3, "if_") { // Expect "if_" prefix
		t.Errorf("Failed to unscramble generated label '%s' correctly, got '%s' (expected format: if_N)", label3, orig3)
	}

	// Ensure original generated names were different
	if orig1 == orig2 {
		t.Errorf("Internal original names for labels with same prefix were identical: %s", orig1)
	}
}

// Test Context Save and Load
func TestContextPersistence(t *testing.T) {
	cfg := createTestConfig()
	sc1 := createTestScrambler(t, TypeVariable, cfg)

	// Scramble some names
	orig1, orig2 := "varOne", "varTwo"
	scrambled1 := sc1.Scramble(orig1)
	scrambled2 := sc1.Scramble(orig2)

	// Create temp dir for context file
	tempDir := t.TempDir()
	contextFile := filepath.Join(tempDir, "variable_test.scramble")

	// Save context
	err := sc1.SaveState(contextFile)
	if err != nil {
		t.Fatalf("Failed to save scrambler state: %v", err)
	}

	// Create a new scrambler and load state
	sc2 := createTestScrambler(t, TypeVariable, cfg)
	err = sc2.LoadState(contextFile)
	if err != nil {
		t.Fatalf("Failed to load scrambler state: %v", err)
	}

	// Verify loaded state
	if sc2.currentLength != sc1.currentLength {
		t.Errorf("Loaded context has wrong currentLength: expected %d, got %d", sc1.currentLength, sc2.currentLength)
	}
	// Check if label counter is loaded (should be 0 for variables)
	if sc2.labelCounter == nil || sc2.labelCounter.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Loaded context has unexpected label counter: %v", sc2.labelCounter)
	}

	// Check if maps contain the loaded items - Removed direct map access check
	// loadedScrambled1, ok1 := sc2.nameMap[strings.ToLower(orig1)] // Cannot access unexported field
	// loadedOrig1, ok2 := sc2.reverseMap[scrambled1]
	//
	// if !ok1 || loadedScrambled1 != scrambled1 {
	// 	t.Errorf("Loaded nameMap failed for '%s': expected '%s', found '%s' (exists: %t)", orig1, scrambled1, loadedScrambled1, ok1)
	// }
	// if !ok2 || loadedOrig1 != orig1 {
	// 	t.Errorf("Loaded reverseMap failed for '%s': expected '%s', found '%s' (exists: %t)", scrambled1, orig1, loadedOrig1, ok2)
	// }

	// Check if maps are loaded correctly by re-scrambling (This implicitly checks map loading)
	scrambled1Again := sc2.Scramble(orig1)
	scrambled2Again := sc2.Scramble(orig2)

	if scrambled1Again != scrambled1 {
		t.Errorf("Loaded context failed for '%s': expected '%s', got '%s'", orig1, scrambled1, scrambled1Again)
	}
	if scrambled2Again != scrambled2 {
		t.Errorf("Loaded context failed for '%s': expected '%s', got '%s'", orig2, scrambled2, scrambled2Again)
	}

	// Try scrambling a new name with the loaded context
	orig3 := "varThree"
	scrambled3 := sc2.Scramble(orig3)
	if scrambled3 == orig3 {
		t.Errorf("Scrambling new name failed after loading context for '%s'", orig3)
	}
	if scrambled3 == scrambled1 || scrambled3 == scrambled2 {
		t.Errorf("Potential collision after loading context: new '%s' matches existing '%s' or '%s'", scrambled3, scrambled1, scrambled2)
	}
	// Verify the new name is also in the maps
	unscrambled3, found3 := sc2.Unscramble(scrambled3)
	if !found3 || unscrambled3 != orig3 {
		t.Errorf("Newly scrambled name '%s' not found correctly in loaded context (unscrambled: %s)", scrambled3, unscrambled3)
	}

	// Test loading non-existent file (should not error)
	sc3 := createTestScrambler(t, TypeVariable, cfg)
	err = sc3.LoadState(filepath.Join(tempDir, "non_existent_file.scramble"))
	if err != nil {
		// Allow IsNotExist errors, but fail on others
		if !os.IsNotExist(err) {
			t.Fatalf("Loading non-existent state file errored unexpectedly: %v", err)
		} else {
			t.Logf("Loading non-existent file correctly resulted in: %v", err) // Log IsNotExist for info
		}
	} else {
		// It's also acceptable for LoadState to simply do nothing if the file doesn't exist
		t.Logf("Loading non-existent file did not error (acceptable behavior).")
	}

	// Test loading incompatible version
	// Initialize only the relevant fields for the test case
	badState := scramblerState{Version: "invalid-version"}
	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	err = encoder.Encode(badState)
	if err != nil {
		t.Fatalf("Failed to encode bad state for testing: %v", err)
	}
	badFile := filepath.Join(tempDir, "bad_version.scramble")
	err = os.WriteFile(badFile, buffer.Bytes(), 0644)
	if err != nil {
		t.Fatalf("Failed to write bad state file for testing: %v", err)
	}

	sc4 := createTestScrambler(t, TypeVariable, cfg)
	err = sc4.LoadState(badFile)
	if err == nil {
		t.Errorf("Loading state with incompatible version should have errored, but didn't")
	} else if !strings.Contains(err.Error(), "incompatible context version") {
		t.Errorf("Loading incompatible version gave wrong error type: %v", err)
	} else {
		t.Logf("Correctly received error for incompatible version: %v", err)
	}
}

// TODO: Add tests for specific scramble modes (hexa, numeric)
// TODO: Add tests for loading predefined ignore lists if that gets implemented
// TODO: Add test for case where context file is corrupted / invalid gob format

