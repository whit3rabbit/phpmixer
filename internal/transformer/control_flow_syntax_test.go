package transformer

import (
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
)

/*
Control Flow Obfuscation Syntax Validation Tests
===============================================
These tests validate that control flow obfuscation generates syntactically valid PHP code.
They would have caught the original boolean literal creation issues that caused
malformed condition expressions and improper ScalarString usage.

The tests cover:
1. Always-true condition generation
2. Always-false condition generation  
3. Boolean literal handling
4. Logical operator combinations
5. Nested control structures

Each test validates PHP syntax using `php -l` to ensure no lint errors.
*/

// processCodeWithControlFlowObfuscation applies control flow obfuscation and returns the result
func processCodeWithControlFlowObfuscation(t *testing.T, phpCode string, useRandomConditions bool) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Create and configure control flow obfuscator
	visitor := NewControlFlowObfuscatorVisitor()
	visitor.UseRandomConditions = useRandomConditions
	visitor.MaxNestingDepth = 1 // Keep simple for testing
	visitor.AddDeadBranches = false // Disable to focus on main logic
	visitor.DebugMode = false // Disable debug to avoid test noise

	// Apply control flow obfuscation
	traverser := traverser.NewTraverser(visitor)
	rootNode.Accept(traverser)

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	rootNode.Accept(p)

	return output.String()
}

func TestControlFlowObfuscationSyntaxValidation_TrueConditions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "simple_function",
			input: `<?php
function test() {
    return 42;
}`,
			description: "Simple functions should be wrapped with valid true conditions",
		},
		{
			name: "function_with_logic",
			input: `<?php
function calculate($a, $b) {
    $sum = $a + $b;
    $product = $a * $b;
    return [$sum, $product];
}`,
			description: "Functions with logic should maintain valid syntax",
		},
		{
			name: "class_method",
			input: `<?php
class Calculator {
    public function add($x, $y) {
        $result = $x + $y;
        return $result;
    }
}`,
			description: "Class methods should be wrapped with valid conditions",
		},
		{
			name: "static_method",
			input: `<?php
class MathUtils {
    public static function multiply($a, $b) {
        return $a * $b;
    }
}`,
			description: "Static methods should handle true conditions correctly",
		},
		{
			name: "constructor_method",
			input: `<?php
class Database {
    private $connection;
    
    public function __construct($host) {
        $this->connection = "mysql:host=$host";
    }
}`,
			description: "Constructor methods should handle obfuscation correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with deterministic conditions (non-random)
			result := processCodeWithControlFlowObfuscation(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure control flow obfuscation was applied (should contain if statements)
			if !strings.Contains(result, "if") {
				t.Errorf("Control flow obfuscation was not applied to: %s", tt.name)
			}
			
			// Ensure no malformed boolean literals (old bug would create if("true"))
			if strings.Contains(result, `if("true")`) || strings.Contains(result, `if('true')`) {
				t.Errorf("Found malformed boolean literal in: %s\nResult: %s", tt.name, result)
			}
			
			// Ensure no malformed false literals (old bug would create if("false"))
			if strings.Contains(result, `if("false")`) || strings.Contains(result, `if('false')`) {
				t.Errorf("Found malformed false literal in: %s\nResult: %s", tt.name, result)
			}
		})
	}
}

func TestControlFlowObfuscationSyntaxValidation_ConditionVariety(t *testing.T) {
	baseCode := `<?php
function test() {
    return "test";
}`

	t.Run("deterministic_conditions", func(t *testing.T) {
		result := processCodeWithControlFlowObfuscation(t, baseCode, false)
		validatePHPSyntax(t, result)
		
		// Should contain numeric true conditions
		hasValidCondition := strings.Contains(result, "if(1)") || 
						   strings.Contains(result, "if(2)") ||
						   strings.Contains(result, "if(3)")
		
		if !hasValidCondition {
			t.Logf("Generated deterministic result: %s", result)
		}
	})

	t.Run("random_conditions", func(t *testing.T) {
		result := processCodeWithControlFlowObfuscation(t, baseCode, true)
		validatePHPSyntax(t, result)
		
		// Should not contain string-based boolean literals
		if strings.Contains(result, `"true"`) || strings.Contains(result, `"false"`) {
			if strings.Contains(result, `if("true")`) || strings.Contains(result, `if("false")`) {
				t.Errorf("Found string-based boolean literals in conditions: %s", result)
			}
		}
	})
}

func TestControlFlowObfuscationSyntaxValidation_LoopStructures(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "for_loop",
			input: `<?php
function processArray($arr) {
    for ($i = 0; $i < count($arr); $i++) {
        echo $arr[$i];
    }
}`,
			description: "For loops should handle control flow obfuscation correctly",
		},
		{
			name: "while_loop",
			input: `<?php
function countdown($n) {
    while ($n > 0) {
        echo $n;
        $n--;
    }
}`,
			description: "While loops should handle control flow obfuscation correctly",
		},
		{
			name: "foreach_loop",
			input: `<?php
function printValues($data) {
    foreach ($data as $key => $value) {
        echo "$key: $value\n";
    }
}`,
			description: "Foreach loops should handle control flow obfuscation correctly",
		},
		{
			name: "do_while_loop",
			input: `<?php
function processUntil($condition) {
    do {
        echo "Processing...\n";
    } while (!$condition());
}`,
			description: "Do-while loops should handle control flow obfuscation correctly",
		},
		{
			name: "nested_loops",
			input: `<?php
function createMatrix($rows, $cols) {
    $matrix = [];
    for ($i = 0; $i < $rows; $i++) {
        for ($j = 0; $j < $cols; $j++) {
            $matrix[$i][$j] = $i * $cols + $j;
        }
    }
    return $matrix;
}`,
			description: "Nested loops should handle control flow obfuscation correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithControlFlowObfuscation(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure loop structures were obfuscated
			if !strings.Contains(result, "if") {
				t.Errorf("Loop structures were not obfuscated in: %s", tt.name)
			}
		})
	}
}

func TestControlFlowObfuscationSyntaxValidation_ConditionalStructures(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "if_statement",
			input: `<?php
function checkValue($value) {
    if ($value > 0) {
        return "positive";
    }
    return "non-positive";
}`,
			description: "If statements should handle nested obfuscation correctly",
		},
		{
			name: "if_else_statement",
			input: `<?php
function getSign($number) {
    if ($number > 0) {
        return "positive";
    } else {
        return "negative or zero";
    }
}`,
			description: "If-else statements should handle obfuscation correctly",
		},
		{
			name: "if_elseif_else",
			input: `<?php
function categorize($score) {
    if ($score >= 90) {
        return "A";
    } elseif ($score >= 80) {
        return "B";
    } elseif ($score >= 70) {
        return "C";
    } else {
        return "F";
    }
}`,
			description: "Complex if-elseif-else should handle obfuscation correctly",
		},
		{
			name: "switch_statement",
			input: `<?php
function handleRequest($type) {
    switch ($type) {
        case 'GET':
            return handleGet();
        case 'POST':
            return handlePost();
        default:
            return handleDefault();
    }
}`,
			description: "Switch statements should handle obfuscation correctly",
		},
		{
			name: "try_catch",
			input: `<?php
function safeOperation() {
    try {
        return riskyOperation();
    } catch (Exception $e) {
        return null;
    }
}`,
			description: "Try-catch blocks should handle obfuscation correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithControlFlowObfuscation(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestControlFlowObfuscationSyntaxValidation_LogicalOperators(t *testing.T) {
	baseCode := `<?php
function test() {
    return true;
}`

	t.Run("logical_or_conditions", func(t *testing.T) {
		result := processCodeWithControlFlowObfuscation(t, baseCode, true)
		validatePHPSyntax(t, result)
		
		// Check for proper logical OR usage (should be numeric, not string-based)
		if strings.Contains(result, "||") || strings.Contains(result, "or") {
			// Ensure no string-based operands
			if strings.Contains(result, `"true" ||`) || strings.Contains(result, `"true" or`) {
				t.Errorf("Found string-based logical OR operands: %s", result)
			}
		}
	})

	t.Run("comparison_operators", func(t *testing.T) {
		result := processCodeWithControlFlowObfuscation(t, baseCode, true)
		validatePHPSyntax(t, result)
		
		// Should not have malformed comparison operators
		malformedPatterns := []string{
			`"1"=="1"`, // Should be 1==1
			`"true"=="true"`, // Should be numeric
			`"false"=="false"`, // Should be numeric
		}
		
		for _, pattern := range malformedPatterns {
			if strings.Contains(result, pattern) {
				t.Errorf("Found malformed comparison pattern '%s' in: %s", pattern, result)
			}
		}
	})

	t.Run("boolean_not_operations", func(t *testing.T) {
		result := processCodeWithControlFlowObfuscation(t, baseCode, true)
		validatePHPSyntax(t, result)
		
		// Check for proper NOT operations (!0 should be valid, !"false" should not exist)
		if strings.Contains(result, "!") {
			if strings.Contains(result, `!"false"`) || strings.Contains(result, `!"true"`) {
				t.Errorf("Found string-based NOT operations: %s", result)
			}
		}
	})
}

func TestControlFlowObfuscationSyntaxValidation_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "complex_class_hierarchy",
			input: `<?php
abstract class BaseProcessor {
    protected $data;
    
    public function __construct($data) {
        $this->data = $data;
    }
    
    abstract public function process();
    
    protected function validate() {
        return !empty($this->data);
    }
}

class StringProcessor extends BaseProcessor {
    public function process() {
        if ($this->validate()) {
            return strtoupper($this->data);
        }
        return null;
    }
}`,
			description: "Complex class hierarchies should handle obfuscation correctly",
		},
		{
			name: "closures_and_callbacks",
			input: `<?php
function processWithCallback($data, $callback) {
    $filtered = array_filter($data, function($item) {
        return $item > 0;
    });
    
    return array_map($callback, $filtered);
}`,
			description: "Closures and callbacks should handle obfuscation correctly",
		},
		{
			name: "generators",
			input: `<?php
function generateNumbers($max) {
    for ($i = 1; $i <= $max; $i++) {
        yield $i;
    }
}`,
			description: "Generator functions should handle obfuscation correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithControlFlowObfuscation(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestControlFlowObfuscationSyntaxValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "empty_function",
			input: `<?php
function emptyFunction() {
    // This function is empty
}`,
			description: "Empty functions should handle obfuscation gracefully",
		},
		{
			name: "single_expression",
			input: `<?php
function singleExpression() {
    return 42;
}`,
			description: "Single expression functions should remain valid",
		},
		{
			name: "function_with_comments",
			input: `<?php
function documentedFunction() {
    // This does something important
    $result = calculateSomething();
    // Return the result
    return $result;
}`,
			description: "Functions with comments should handle obfuscation correctly",
		},
		{
			name: "special_php_features",
			input: `<?php
function useSpecialFeatures() {
    $array = [1, 2, 3];
    $result = $array[0] ?? 'default';
    return $result;
}`,
			description: "Modern PHP features should handle obfuscation correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithControlFlowObfuscation(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

// Benchmark test to ensure performance doesn't degrade
func BenchmarkControlFlowObfuscationSyntaxValidation(b *testing.B) {
	phpCode := `<?php
class ComplexClass {
    private $data;
    
    public function __construct($data) {
        $this->data = $data;
    }
    
    public function processData() {
        $result = [];
        foreach ($this->data as $item) {
            if ($item > 0) {
                $result[] = $item * 2;
            }
        }
        return $result;
    }
}`

	for i := 0; i < b.N; i++ {
		result := processCodeWithControlFlowObfuscation(nil, phpCode, false)
		_ = result // Prevent optimization
	}
}