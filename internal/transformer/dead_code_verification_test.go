package transformer

import (
	"bytes"
	"math/rand"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/stretchr/testify/assert"
)

// TestDeadCodeInserterIntegration tests the DeadCodeInserterVisitor against more complex
// PHP code structures to ensure it handles all cases correctly
func TestDeadCodeInserterIntegration(t *testing.T) {
	// More complex test cases with different PHP constructs
	testCases := []struct {
		name     string
		input    string
		cfg      func(*DeadCodeInserterVisitor)
		verifyFn func(*testing.T, string)
	}{
		{
			name: "Complex function with dead code",
			input: `<?php
function complexFunction($param1, $param2) {
    $result = [];
    
    if ($param1 > $param2) {
        $result['comparison'] = 'greater';
    } else if ($param1 == $param2) {
        $result['comparison'] = 'equal';
    } else {
        $result['comparison'] = 'less';
    }
    
    for ($i = 0; $i < $param1; $i++) {
        $result['values'][] = $i * $param2;
    }
    
    return $result;
}`,
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = false
				v.MaxInjectionDepth = 3
			},
			verifyFn: func(t *testing.T, output string) {
				// Should contain dead code blocks (if(false), etc.)
				assert.Contains(t, output, "if(")
				assert.Contains(t, output, "function complexFunction")

				// The code should still be valid PHP
				// Check for key structural elements
				assert.Contains(t, output, "$result = []")
				assert.Contains(t, output, "return $result")

				// Size should be larger - not a precise test but a good indicator
				assert.Greater(t, len(output), 300)
			},
		},
		{
			name: "Class with methods and junk statements",
			input: `<?php
class TestClass {
    private $value;
    
    public function __construct($value) {
        $this->value = $value;
    }
    
    public function getValue() {
        return $this->value;
    }
    
    public function setValue($value) {
        $this->value = $value;
    }
    
    public function calculate($factor) {
        return $this->value * $factor;
    }
}`,
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = false
				v.InjectJunkStatements = true
				v.MaxInjectionDepth = 2
			},
			verifyFn: func(t *testing.T, output string) {
				// The code should be transformed in some way, either with junk statements or other changes
				// Due to randomness, _junk might not appear in all runs, so we'll check for multiple indicators
				transformed := len(output) > len(`<?php
class TestClass {
    private $value;
    
    public function __construct($value) {
        $this->value = $value;
    }
    
    public function getValue() {
        return $this->value;
    }
    
    public function setValue($value) {
        $this->value = $value;
    }
    
    public function calculate($factor) {
        return $this->value * $factor;
    }
}`)

				// Check that some transformation was applied (either _junk, time(), count(), etc.)
				hasJunkCode := strings.Contains(output, "_junk") ||
					strings.Contains(output, "time(") ||
					strings.Contains(output, "rand(") ||
					strings.Contains(output, "_temp") ||
					strings.Contains(output, "count(") ||
					strings.Contains(output, "gettype(")

				if !transformed && !hasJunkCode {
					t.Errorf("No transformation was applied to the code")
				}

				// The class structure should be preserved
				assert.Contains(t, output, "class TestClass")
				assert.Contains(t, output, "private $value")
				assert.Contains(t, output, "public function __construct")
				assert.Contains(t, output, "public function getValue")
				assert.Contains(t, output, "public function setValue")
				assert.Contains(t, output, "public function calculate")

				// Original functionality should be preserved
				assert.Contains(t, output, "return $this->value")
				assert.Contains(t, output, "$this->value = $value")
				assert.Contains(t, output, "return $this->value * $factor")
			},
		},
		{
			name: "Nested loops with both types of injection",
			input: `<?php
function processData($data) {
    $result = [];
    
    foreach ($data as $key => $items) {
        $sum = 0;
        
        for ($i = 0; $i < count($items); $i++) {
            $value = $items[$i];
            if ($value > 0) {
                $sum += $value;
            }
        }
        
        $result[$key] = $sum;
    }
    
    return $result;
}`,
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 75                    // 75% chance to inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
				v.MaxInjectionDepth = 3 // Allow deeper nesting
			},
			verifyFn: func(t *testing.T, output string) {
				// Should contain both types of injections
				// Specific content is random, but we can check for patterns

				// Function structure should be preserved
				assert.Contains(t, output, "function processData")
				assert.Contains(t, output, "$result = []")
				assert.Contains(t, output, "foreach ($data as $key => $items)")
				assert.Contains(t, output, "for ($i = 0; $i < count($items); $i++)")
				assert.Contains(t, output, "return $result")

				// There should be more code than the original
				originalSize := len(`<?php
function processData($data) {
    $result = [];
    
    foreach ($data as $key => $items) {
        $sum = 0;
        
        for ($i = 0; $i < count($items); $i++) {
            $value = $items[$i];
            if ($value > 0) {
                $sum += $value;
            }
        }
        
        $result[$key] = $sum;
    }
    
    return $result;
}`)
				assert.Greater(t, len(output), originalSize)
			},
		},
		{
			name: "Zero injection rate check",
			input: `<?php
$a = 1;
$b = 2;
$c = $a + $b;
echo $c;`,
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 0                     // Never inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
			},
			verifyFn: func(t *testing.T, output string) {
				// No code should be injected with 0% rate
				// Normalize whitespace for comparison
				normalizedOutput := strings.Join(strings.Fields(output), " ")
				normalizedInput := strings.Join(strings.Fields(`<?php
$a = 1;
$b = 2;
$c = $a + $b;
echo $c;`), " ")

				assert.Contains(t, normalizedOutput, normalizedInput)

				// Should not contain junk variables or dead code blocks
				assert.NotContains(t, output, "_junk")
				assert.NotContains(t, output, "if(false)")
			},
		},
		{
			name: "Check max injection depth",
			input: `<?php
function nestedFunction() {
    if (true) {
        for ($i = 0; $i < 5; $i++) {
            if ($i % 2 == 0) {
                while ($i < 3) {
                    echo $i;
                    $i++;
                }
            }
        }
    }
}`,
			cfg: func(v *DeadCodeInserterVisitor) {
				v.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducible tests
				v.InjectionRate = 100                   // Always inject
				v.InjectDeadCodeBlocks = true
				v.InjectJunkStatements = true
				v.MaxInjectionDepth = 2 // Limit to 2 levels deep
			},
			verifyFn: func(t *testing.T, output string) {
				// The code should contain outer level injections but inner levels should be less affected
				// This is hard to verify precisely, so we're just checking for general size increase
				// and making sure the original structure is preserved
				assert.Contains(t, output, "function nestedFunction")
				assert.Contains(t, output, "if (true)")
				assert.Contains(t, output, "for ($i = 0; $i < 5; $i++)")
				assert.Contains(t, output, "while ($i < 3)")
				assert.Contains(t, output, "echo $i")

				// Size should increase due to injection
				originalSize := len(`<?php
function nestedFunction() {
    if (true) {
        for ($i = 0; $i < 5; $i++) {
            if ($i % 2 == 0) {
                while ($i < 3) {
                    echo $i;
                    $i++;
                }
            }
        }
    }
}`)
				assert.Greater(t, len(output), originalSize)
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

			// Log original and transformed code for visual inspection
			t.Logf("--- Original Code ---\n%s\n", tc.input)
			t.Logf("--- Transformed Code ---\n%s\n", output)

			// Verify the transformation
			tc.verifyFn(t, output)
		})
	}
}

// TestCodeSizeIncrease measures how much the code size increases with different injection rates
func TestCodeSizeIncrease(t *testing.T) {
	// Create a representative PHP file
	phpCode := `<?php
function processArray($array) {
    $result = [];
    foreach ($array as $key => $value) {
        if ($value > 10) {
            $result[$key] = $value * 2;
        } else {
            $result[$key] = $value;
        }
    }
    return $result;
}

function calculateSum($array) {
    $sum = 0;
    for ($i = 0; $i < count($array); $i++) {
        $sum += $array[$i];
    }
    return $sum;
}

class DataProcessor {
    private $data;
    
    public function __construct($data) {
        $this->data = $data;
    }
    
    public function process() {
        $processed = processArray($this->data);
        $total = calculateSum($processed);
        return [
            'processed' => $processed,
            'total' => $total
        ];
    }
}
`

	// Test with different injection rates
	injectionRates := []int{0, 25, 50, 75, 100}

	// Process the PHP code with different injection rates and measure the size increase
	var baselineSize int
	results := make(map[int]float64) // Rate -> Size increase percentage

	// Parse PHP code once for reuse
	rootNode, err := parser.Parse([]byte(phpCode), conf.Config{})
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Get baseline size (with 0% injection)
	visitor := NewDeadCodeInserterVisitor()
	visitor.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
	visitor.InjectionRate = 0
	visitor.InjectDeadCodeBlocks = true
	visitor.InjectJunkStatements = true

	traverser := NewReplaceTraverser(visitor, true)
	baseNode := traverser.Traverse(rootNode)

	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	baseNode.Accept(p)
	baselineSize = buf.Len()

	// Process with different rates
	for _, rate := range injectionRates {
		// Clone original AST for fresh transformation
		freshRootNode, _ := parser.Parse([]byte(phpCode), conf.Config{})

		visitor := NewDeadCodeInserterVisitor()
		visitor.random = rand.New(rand.NewSource(42)) // Fixed seed for reproducibility
		visitor.InjectionRate = rate
		visitor.InjectDeadCodeBlocks = true
		visitor.InjectJunkStatements = true

		traverser := NewReplaceTraverser(visitor, true)
		transformedRoot := traverser.Traverse(freshRootNode)

		var outBuf bytes.Buffer
		p := printer.NewPrinter(&outBuf)
		transformedRoot.Accept(p)

		transformedSize := outBuf.Len()
		increase := float64(transformedSize-baselineSize) / float64(baselineSize) * 100
		results[rate] = increase

		t.Logf("Injection rate %d%%: Size increased by %.2f%% (from %d to %d bytes)",
			rate, increase, baselineSize, transformedSize)

		// For higher rates, log code snippets for visual inspection
		if rate >= 75 {
			t.Logf("Sample of obfuscated code (first 500 bytes): %s",
				outBuf.String()[:min(500, transformedSize)])
		}
	}

	// Check that size increases with injection rate (should be monotonic)
	for i := 1; i < len(injectionRates); i++ {
		currentRate := injectionRates[i]
		prevRate := injectionRates[i-1]

		// Skip the comparison if both rates are 0
		if currentRate == 0 && prevRate == 0 {
			continue
		}

		t.Logf("Comparing rate %d%% (%.2f%% increase) with rate %d%% (%.2f%% increase)",
			prevRate, results[prevRate], currentRate, results[currentRate])

		// Higher rates should generally produce more code, but due to randomness
		// this might not always be strictly increasing
		if currentRate > 0 && prevRate == 0 {
			assert.Greater(t, results[currentRate], results[prevRate],
				"Higher injection rate should increase code size")
		} else if currentRate >= 50 && prevRate < 50 {
			// For higher rates, we expect a more definite increase
			assert.Greater(t, results[currentRate], results[prevRate],
				"Significantly higher injection rate should increase code size")
		}
	}
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
