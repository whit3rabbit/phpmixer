package transformer

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/stretchr/testify/assert"
)

// TestDeadCodeInsertion tests that the dead code insertion visitor correctly
// adds dead code blocks and junk statements to PHP code
func TestDeadCodeInsertion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		cfg      func(*DeadCodeInserterVisitor)
		verifyFn func(*testing.T, string)
	}{
		{
			name:  "Function with dead code injection",
			input: "<?php function test() { echo 'Hello'; return 'World'; }",
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = false
			},
			verifyFn: func(t *testing.T, output string) {
				// Should inject an if(false) block with some code
				assert.Contains(t, output, "if")
				// Original code should still be present
				assert.Contains(t, output, "echo 'Hello'")
				assert.Contains(t, output, "return 'World'")
			},
		},
		{
			name:  "Statement list with junk statements",
			input: "<?php $a = 1; $b = 2; $c = $a + $b;",
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = false
				v.InjectJunkStatements = true
			},
			verifyFn: func(t *testing.T, output string) {
				// Should inject unused variables or calculations
				assert.Contains(t, output, "_junk")
				// Original code should still be present
				assert.Contains(t, output, "$a = 1")
				assert.Contains(t, output, "$b = 2")
				assert.Contains(t, output, "$c = $a + $b")
			},
		},
		{
			name:  "If statement with both types of injection",
			input: "<?php if ($condition) { doSomething(); } else { doSomethingElse(); }",
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
			},
			verifyFn: func(t *testing.T, output string) {
				// Should inject both types of code
				// Original code should still be present
				assert.Contains(t, output, "if ($condition)")
				assert.Contains(t, output, "doSomething()")
				assert.Contains(t, output, "doSomethingElse()")

				// Check that something was added (either a junk variable or dead code)
				// The exact output is hard to predict, so we're just checking that code got bigger
				assert.Greater(t, len(output), len("<?php if ($condition) { doSomething(); } else { doSomethingElse(); }"))
			},
		},
		{
			name:  "Loop body injection",
			input: "<?php for ($i = 0; $i < 10; $i++) { echo $i; }",
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
			},
			verifyFn: func(t *testing.T, output string) {
				// Original code should still be present
				assert.Contains(t, output, "for ($i = 0; $i < 10; $i++)")
				assert.Contains(t, output, "echo $i")

				// Should be larger than original due to injected code
				assert.Greater(t, len(output), len("<?php for ($i = 0; $i < 10; $i++) { echo $i; }"))
			},
		},
		{
			name:  "Zero injection rate",
			input: "<?php function simple() { return true; }",
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 0                     // Never inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
			},
			verifyFn: func(t *testing.T, output string) {
				// Output should be identical to input (after formatting)
				normalizedOutput := strings.ReplaceAll(strings.TrimSpace(output), " ", "")
				normalizedInput := strings.ReplaceAll(strings.TrimSpace("<?php function simple() { return true; }"), " ", "")
				assert.Equal(t, normalizedInput, normalizedOutput)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse PHP code to AST
			rootNode, err := parser.Parse([]byte(tc.input), conf.Config{})
			if err != nil {
				t.Fatalf("Failed to parse PHP code: %v", err)
			}

			// Create the dead code inserter visitor
			visitor := NewDeadCodeInserterVisitor()
			visitor.DebugMode = true

			// Apply custom configuration for this test case
			if tc.cfg != nil {
				tc.cfg(visitor)
			}

			// Create a traverser with the visitor
			traverser := NewReplaceTraverser(visitor, true)

			// Apply the transformation
			rootNode = traverser.Traverse(rootNode)

			// Generate PHP code from the transformed AST
			var buf bytes.Buffer
			p := printer.NewPrinter(&buf)
			rootNode.Accept(p)
			output := buf.String()

			t.Logf("Input: %s", tc.input)
			t.Logf("Output: %s", output)

			// Verify the transformation
			tc.verifyFn(t, output)
		})
	}
}

// TestCountInjectedCode tests that the dead code insertion works with different injection rates
func TestCountInjectedCode(t *testing.T) {
	// Create a complex input with multiple statement types to test injection
	input := `<?php
function test($arg1, $arg2) {
    $result = $arg1 + $arg2;
    
    if ($result > 10) {
        echo "Result is greater than 10";
    } else {
        echo "Result is 10 or less";
    }
    
    for ($i = 0; $i < $result; $i++) {
        echo $i;
    }
    
    return $result;
}
`

	// Test with different injection rates
	injectionRates := []int{0, 25, 50, 75, 100}

	for _, rate := range injectionRates {
		t.Run(fmt.Sprintf("InjectionRate_%d", rate), func(t *testing.T) {
			// Parse PHP code to AST
			rootNode, err := parser.Parse([]byte(input), conf.Config{})
			if err != nil {
				t.Fatalf("Failed to parse PHP code: %v", err)
			}

			// Create visitor with set injection rate
			visitor := NewDeadCodeInserterVisitor()
			visitor.DebugMode = true                      // Enable debug mode to capture processing info
			visitor.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
			visitor.InjectionRate = rate
			visitor.InjectDeadCodeBlocks = true
			visitor.InjectJunkStatements = true

			// Count processing events to verify injection is working
			debugOutput := &bytes.Buffer{}
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stderr = w

			// Apply the transformation
			traverser := NewReplaceTraverser(visitor, true)
			transformedRoot := traverser.Traverse(rootNode)

			// Capture debug output
			w.Close()
			io.Copy(debugOutput, r)
			os.Stderr = oldStderr

			// Generate code for original and transformed AST
			var origBuf, transformedBuf bytes.Buffer

			p1 := printer.NewPrinter(&origBuf)
			rootNode.Accept(p1)

			p2 := printer.NewPrinter(&transformedBuf)
			transformedRoot.Accept(p2)

			originalCode := origBuf.String()
			transformedCode := transformedBuf.String()

			t.Logf("Injection rate: %d%%", rate)
			t.Logf("Original size: %d bytes", len(originalCode))
			t.Logf("Transformed size: %d bytes", len(transformedCode))

			// For 0% rate, verify no injections occurred
			if rate == 0 {
				debugString := debugOutput.String()
				assert.NotContains(t, debugString, "Injected code",
					"With 0%% injection rate, no code should be injected")
			} else if rate >= 75 {
				// For high rates, verify some injections occurred
				debugString := debugOutput.String()
				containsInjections := strings.Contains(debugString, "Injected code into") ||
					strings.Contains(debugString, "Added replacement")
				assert.True(t, containsInjections,
					"With high injection rate (%d%%), some code should be injected", rate)
			}
		})
	}
}
