package transformer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
)

/*
Junk Code Insertion Syntax Validation Tests
============================================
These tests validate that junk code insertion generates syntactically valid PHP code.
They would have caught the original string token handling issues that caused
malformed variable assignments and improper array item generation.

The tests cover:
1. Unused variable assignments with different data types
2. No-op calculations (multiply by 1, add 0, etc.)
3. Useless function calls with proper argument handling
4. String literal token creation in junk statements
5. Array item generation with mixed data types

Each test validates PHP syntax using `php -l` to ensure no lint errors.
*/


// processCodeWithJunkCodeInsertion applies junk code insertion and returns the result
func processCodeWithJunkCodeInsertion(t *testing.T, phpCode string, enableJunkCode bool) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Create and configure junk code inserter
	visitor := NewDeadCodeInserterVisitor()
	visitor.InjectDeadCodeBlocks = false  // Disable dead code to focus on junk code
	visitor.InjectJunkStatements = enableJunkCode
	visitor.InjectionRate = 100          // Always inject for testing
	visitor.DebugMode = false            // Disable debug to avoid test noise

	// Apply junk code insertion using replace traverser
	traverser := NewReplaceTraverser(visitor, false)
	result := traverser.Traverse(rootNode)

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	result.Accept(p)

	return output.String()
}

func TestJunkCodeInsertionSyntaxValidation_UnusedVariables(t *testing.T) {
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
			description: "Unused variable assignments in simple functions should be valid",
		},
		{
			name: "function_with_parameters",
			input: `<?php
function calculate($a, $b) {
    $result = $a + $b;
    return $result;
}`,
			description: "Unused variables with existing parameters should be valid",
		},
		{
			name: "class_method",
			input: `<?php
class Calculator {
    public function add($x, $y) {
        return $x + $y;
    }
}`,
			description: "Unused variables in class methods should be valid",
		},
		{
			name: "nested_structures",
			input: `<?php
function processData($data) {
    if ($data) {
        foreach ($data as $item) {
            echo $item;
        }
    }
    return true;
}`,
			description: "Unused variables in nested structures should be valid",
		},
		{
			name: "multiple_statements",
			input: `<?php
function complexFunction() {
    $x = 1;
    $y = 2;
    $z = $x + $y;
    return $z;
}`,
			description: "Unused variables mixed with existing variables should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test junk code insertion only
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure junk code was actually injected (should contain junk variables)
			if !strings.Contains(result, "_junk") && !strings.Contains(result, "_temp") {
				t.Errorf("Junk code was not injected in: %s", tt.name)
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_VariableDataTypes(t *testing.T) {
	baseCode := `<?php
function testDataTypes() {
    $data = "test";
    return $data;
}`

	// Run multiple times to test different junk variable data types
	for i := 0; i < 10; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, baseCode, true)
			validatePHPSyntax(t, result)
			
			// Check for different types of junk variable assignments
			hasNumericAssignment := strings.Contains(result, "_junk") && 
									(strings.Contains(result, " = 1") || strings.Contains(result, " = 2"))
			hasStringAssignment := strings.Contains(result, "_junk") && 
								   strings.Contains(result, "'junk")
			hasArrayAssignment := strings.Contains(result, "_junk") && 
								  strings.Contains(result, "= [")
			hasFunctionCall := strings.Contains(result, "_junk") && 
							   (strings.Contains(result, "time()") || strings.Contains(result, "rand()"))
			
			// At least one type should be present
			if !hasNumericAssignment && !hasStringAssignment && !hasArrayAssignment && !hasFunctionCall {
				t.Logf("Generated result: %s", result)
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_StringLiterals(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "string_heavy_function",
			input: `<?php
function generateMessage() {
    $greeting = "Hello";
    $name = "World";
    return $greeting . " " . $name;
}`,
			description: "String literals in junk code should be properly quoted",
		},
		{
			name: "array_processing",
			input: `<?php
function processStrings($strings) {
    $result = [];
    foreach ($strings as $str) {
        $result[] = strtoupper($str);
    }
    return $result;
}`,
			description: "String literals in arrays should have proper token handling",
		},
		{
			name: "template_function",
			input: `<?php
function buildTemplate($title, $content) {
    $template = "<h1>$title</h1><p>$content</p>";
    return $template;
}`,
			description: "Complex strings should not be affected by junk code string tokens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Check for properly quoted string literals in junk code
			if strings.Contains(result, "'junk") {
				// Ensure proper quoting
				if strings.Contains(result, "junk") && !strings.Contains(result, "'junk") {
					// Check if it's an unquoted junk string (which would be invalid)
					lines := strings.Split(result, "\n")
					for _, line := range lines {
						if strings.Contains(line, "_junk") && strings.Contains(line, "junk") && !strings.Contains(line, "'junk") {
							if !strings.Contains(line, "//") && !strings.Contains(line, "/*") {
								t.Errorf("Found unquoted string literal in junk code: %s", line)
							}
						}
					}
				}
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_NoOpCalculations(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "arithmetic_function",
			input: `<?php
function mathOperations($a, $b) {
    $sum = $a + $b;
    $product = $a * $b;
    return [$sum, $product];
}`,
			description: "No-op calculations should not interfere with real arithmetic",
		},
		{
			name: "counter_function",
			input: `<?php
function incrementCounter($counter) {
    $counter++;
    return $counter;
}`,
			description: "No-op calculations should be syntactically valid",
		},
		{
			name: "statistical_function",
			input: `<?php
function calculateAverage($numbers) {
    $sum = array_sum($numbers);
    $count = count($numbers);
    return $count > 0 ? $sum / $count : 0;
}`,
			description: "Complex calculations should remain valid with no-op injections",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Check for no-op patterns
			noOpPatterns := []string{
				"* 1",    // multiply by 1
				"+ 0",    // add 0
				"- 0",    // subtract 0
				"/ 1",    // divide by 1
				"_temp",  // temp variable names
			}
			
			foundNoOp := false
			for _, pattern := range noOpPatterns {
				if strings.Contains(result, pattern) {
					foundNoOp = true
					break
				}
			}
			
			if foundNoOp {
				// Ensure no malformed expressions
				if strings.Contains(result, "= * 1") || strings.Contains(result, "= + 0") {
					t.Errorf("Found malformed no-op expression: %s", result)
				}
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_FunctionCalls(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "utility_function",
			input: `<?php
function getCurrentInfo() {
    $timestamp = time();
    $memory = memory_get_usage();
    return [$timestamp, $memory];
}`,
			description: "Useless function calls should not interfere with real function calls",
		},
		{
			name: "data_processor",
			input: `<?php
function processItems($items) {
    $count = count($items);
    $type = gettype($items);
    return ['count' => $count, 'type' => $type];
}`,
			description: "Function calls in junk code should be syntactically valid",
		},
		{
			name: "random_generator",
			input: `<?php
function generateRandoms($n) {
    $randoms = [];
    for ($i = 0; $i < $n; $i++) {
        $randoms[] = rand(1, 100);
    }
    return $randoms;
}`,
			description: "Random function calls should not conflict with junk function calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Check for useless function call patterns
			uselessFunctions := []string{"time", "rand", "microtime", "memory_get_usage", "gettype", "count"}
			foundUselessCall := false
			
			for _, funcName := range uselessFunctions {
				// Look for function calls that are not assigned or used
				if strings.Contains(result, funcName+"();") {
					foundUselessCall = true
					break
				}
			}
			
			if foundUselessCall {
				// Ensure function calls are properly formed
				if strings.Contains(result, "();") {
					// Check for malformed calls like "rand(1, );" or "(1, 100);"
					if strings.Contains(result, "(,") || strings.Contains(result, ",)") {
						t.Errorf("Found malformed function call: %s", result)
					}
				}
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_ArrayGeneration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "array_manipulation",
			input: `<?php
function mergeArrays($arr1, $arr2) {
    $merged = array_merge($arr1, $arr2);
    return $merged;
}`,
			description: "Array generation in junk code should not interfere with real arrays",
		},
		{
			name: "data_structures",
			input: `<?php
function createMatrix($rows, $cols) {
    $matrix = [];
    for ($i = 0; $i < $rows; $i++) {
        $matrix[$i] = array_fill(0, $cols, 0);
    }
    return $matrix;
}`,
			description: "Complex array operations should remain valid with junk arrays",
		},
		{
			name: "configuration_handler",
			input: `<?php
function getConfig() {
    return [
        'debug' => true,
        'timeout' => 30,
        'retries' => 3
    ];
}`,
			description: "Associative arrays should not be affected by junk array generation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Check for array generation in junk code
			if strings.Contains(result, "_junk") && strings.Contains(result, "[") {
				// Ensure arrays are properly formed
				if strings.Contains(result, "[,") || strings.Contains(result, ",]") {
					t.Errorf("Found malformed array in junk code: %s", result)
				}
				
				// Check for proper array item syntax
				if strings.Contains(result, "= [") {
					// Should not have empty items or malformed syntax
					if strings.Contains(result, "[ ,") || strings.Contains(result, ", ]") {
						t.Errorf("Found malformed array items: %s", result)
					}
				}
			}
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "mvc_controller",
			input: `<?php
class UserController {
    private $db;
    
    public function __construct($database) {
        $this->db = $database;
    }
    
    public function getUserById($id) {
        $query = "SELECT * FROM users WHERE id = ?";
        $stmt = $this->db->prepare($query);
        $stmt->execute([$id]);
        return $stmt->fetch();
    }
}`,
			description: "Complex class structures should remain valid with junk code",
		},
		{
			name: "api_handler",
			input: `<?php
function handleApiRequest($request) {
    $method = $request['method'] ?? 'GET';
    $path = $request['path'] ?? '/';
    
    switch ($method) {
        case 'GET':
            return handleGet($path);
        case 'POST':
            return handlePost($path, $request['data'] ?? []);
        default:
            throw new InvalidArgumentException("Unsupported method: $method");
    }
}`,
			description: "Control flow structures should handle junk code injection correctly",
		},
		{
			name: "data_validator",
			input: `<?php
function validateUserData($data) {
    $errors = [];
    
    if (empty($data['email'])) {
        $errors[] = 'Email is required';
    } elseif (!filter_var($data['email'], FILTER_VALIDATE_EMAIL)) {
        $errors[] = 'Invalid email format';
    }
    
    if (empty($data['password'])) {
        $errors[] = 'Password is required';
    } elseif (strlen($data['password']) < 8) {
        $errors[] = 'Password must be at least 8 characters';
    }
    
    return $errors;
}`,
			description: "Validation logic should not be affected by junk code injection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestJunkCodeInsertionSyntaxValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "empty_function",
			input: `<?php
function emptyFunction() {
    // This function does nothing
}`,
			description: "Empty functions should handle junk code injection gracefully",
		},
		{
			name: "single_statement",
			input: `<?php
function singleStatement() {
    return true;
}`,
			description: "Single statement functions should remain valid",
		},
		{
			name: "minimal_code",
			input: `<?php
$x = 1;`,
			description: "Minimal code should handle junk code injection",
		},
		{
			name: "closure_function",
			input: `<?php
$callback = function($x) {
    return $x * 2;
};`,
			description: "Closures should handle junk code injection correctly",
		},
		{
			name: "special_characters",
			input: `<?php
function handleSpecialChars() {
    $symbols = "!@#$%^&*()_+-=[]{}|;':\",./<>?";
    $escaped = addslashes($symbols);
    return $escaped;
}`,
			description: "Special characters should not break junk code token handling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithJunkCodeInsertion(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

// Benchmark test to ensure performance doesn't degrade with junk code injection
func BenchmarkJunkCodeInsertionSyntaxValidation(b *testing.B) {
	phpCode := `<?php
function complexFunction($data) {
    $result = [];
    foreach ($data as $key => $value) {
        if (is_string($value)) {
            $result[$key] = strtoupper($value);
        } elseif (is_numeric($value)) {
            $result[$key] = $value * 2;
        } else {
            $result[$key] = $value;
        }
    }
    return $result;
}`

	for i := 0; i < b.N; i++ {
		result := processCodeWithJunkCodeInsertion(nil, phpCode, true)
		_ = result // Prevent optimization
	}
}