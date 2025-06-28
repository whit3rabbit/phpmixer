package transformer

import (
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
)

/*
Dead Code Injection Syntax Validation Tests
===========================================
These tests validate that dead code injection generates syntactically valid PHP code.
They would have caught the original AST node construction issues that caused
malformed boolean conditions and improper string literal handling.

The tests cover:
1. Boolean false condition generation
2. String literal token creation
3. Complex condition expressions
4. Array item generation with strings
5. Various dead code patterns

Each test validates PHP syntax using `php -l` to ensure no lint errors.
*/


// processCodeWithDeadCodeInjection applies dead code injection and returns the result
func processCodeWithDeadCodeInjection(t *testing.T, phpCode string, enableDeadCode, enableJunkCode bool) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Create and configure dead code inserter
	visitor := NewDeadCodeInserterVisitor()
	visitor.InjectDeadCodeBlocks = enableDeadCode
	visitor.InjectJunkStatements = enableJunkCode
	visitor.InjectionRate = 100 // Always inject for testing
	visitor.DebugMode = false   // Disable debug to avoid test noise

	// Apply dead code injection using replace traverser
	traverser := NewReplaceTraverser(visitor, false)
	result := traverser.Traverse(rootNode)

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	result.Accept(p)

	return output.String()
}

func TestDeadCodeInjectionSyntaxValidation_FalseConditions(t *testing.T) {
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
			description: "False conditions in simple functions should be valid",
		},
		{
			name: "function_with_parameters",
			input: `<?php
function calculate($a, $b) {
    $result = $a + $b;
    return $result;
}`,
			description: "False conditions with parameters should be valid",
		},
		{
			name: "class_method",
			input: `<?php
class Calculator {
    public function add($x, $y) {
        return $x + $y;
    }
}`,
			description: "False conditions in class methods should be valid",
		},
		{
			name: "nested_conditions",
			input: `<?php
function complexLogic($value) {
    if ($value > 0) {
        return "positive";
    } else {
        return "negative";
    }
}`,
			description: "Nested conditions with false blocks should be valid",
		},
		{
			name: "loop_structures",
			input: `<?php
function processArray($arr) {
    for ($i = 0; $i < count($arr); $i++) {
        echo $arr[$i];
    }
}`,
			description: "False conditions in loops should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test dead code injection only
			result := processCodeWithDeadCodeInjection(t, tt.input, true, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure dead code was actually injected (should contain if statements)
			if !strings.Contains(result, "if") {
				t.Errorf("Dead code was not injected in: %s", tt.name)
			}
		})
	}
}

func TestDeadCodeInjectionSyntaxValidation_StringLiterals(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "string_assignment",
			input: `<?php
function getMessage() {
    $msg = "Hello World";
    return $msg;
}`,
			description: "String literals in dead code should be properly quoted",
		},
		{
			name: "string_concatenation",
			input: `<?php
function buildPath($dir, $file) {
    return $dir . "/" . $file;
}`,
			description: "String concatenation with dead code should be valid",
		},
		{
			name: "array_with_strings",
			input: `<?php
function getColors() {
    return ["red", "green", "blue"];
}`,
			description: "Arrays containing strings should have proper token handling",
		},
		{
			name: "mixed_data_types",
			input: `<?php
function processData() {
    $data = [
        "name" => "John",
        "age" => 30,
        "active" => true
    ];
    return $data;
}`,
			description: "Mixed data types in arrays should be handled correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test both dead code and junk code injection
			result := processCodeWithDeadCodeInjection(t, tt.input, true, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestDeadCodeInjectionSyntaxValidation_ConditionTypes(t *testing.T) {
	// Test various condition types that were problematic in the original implementation
	baseCode := `<?php
function testConditions() {
    $x = 1;
    return $x;
}`

	t.Run("numeric_false_conditions", func(t *testing.T) {
		result := processCodeWithDeadCodeInjection(t, baseCode, true, false)
		validatePHPSyntax(t, result)
		
		// Check for various numeric false patterns
		hasValidCondition := strings.Contains(result, "if(0)") || 
						   strings.Contains(result, "if(1==2)") ||
						   strings.Contains(result, "if(1===2)")
		
		if !hasValidCondition {
			t.Logf("Generated code: %s", result)
		}
	})

	t.Run("comparison_conditions", func(t *testing.T) {
		result := processCodeWithDeadCodeInjection(t, baseCode, true, false)
		validatePHPSyntax(t, result)
		
		// Should not contain malformed string-based boolean conditions
		if strings.Contains(result, `if("false")`) || strings.Contains(result, `if('false')`) {
			t.Errorf("Found malformed string-based boolean condition: %s", result)
		}
	})

	t.Run("string_literal_conditions", func(t *testing.T) {
		result := processCodeWithDeadCodeInjection(t, baseCode, true, false)
		validatePHPSyntax(t, result)
		
		// Check for properly quoted string literals in conditions
		if strings.Contains(result, `'a'`) && strings.Contains(result, `'b'`) {
			// Ensure proper quoting
			if strings.Contains(result, `if('a'==='b')`) || strings.Contains(result, `if("a"==="b")`) {
				// This is valid
			} else if strings.Contains(result, `if(a===b)`) {
				t.Errorf("Found unquoted string literals in condition: %s", result)
			}
		}
	})
}

func TestDeadCodeInjectionSyntaxValidation_JunkCode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "variable_assignments",
			input: `<?php
function process() {
    $data = [1, 2, 3];
    return array_sum($data);
}`,
			description: "Junk variable assignments should be syntactically valid",
		},
		{
			name: "function_calls",
			input: `<?php
function calculateTime() {
    $start = microtime(true);
    // Some processing
    return microtime(true) - $start;
}`,
			description: "Junk function calls should be syntactically valid",
		},
		{
			name: "arithmetic_operations",
			input: `<?php
function mathOperations($a, $b) {
    $sum = $a + $b;
    $product = $a * $b;
    return [$sum, $product];
}`,
			description: "Junk arithmetic operations should be syntactically valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test junk code injection only
			result := processCodeWithDeadCodeInjection(t, tt.input, false, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestDeadCodeInjectionSyntaxValidation_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "nested_classes_and_functions",
			input: `<?php
class DatabaseManager {
    private $connection;
    
    public function __construct($host, $user, $pass) {
        $this->connection = new PDO("mysql:host=$host", $user, $pass);
    }
    
    public function query($sql, $params = []) {
        $stmt = $this->connection->prepare($sql);
        $stmt->execute($params);
        return $stmt->fetchAll();
    }
}

function createManager() {
    return new DatabaseManager("localhost", "user", "pass");
}`,
			description: "Complex nested structures should remain valid with dead code",
		},
		{
			name: "control_flow_structures",
			input: `<?php
function processRequest($data) {
    switch ($data['type']) {
        case 'user':
            return handleUser($data);
        case 'admin':
            return handleAdmin($data);
        default:
            throw new InvalidArgumentException("Unknown type");
    }
}

function handleUser($data) {
    try {
        return validateUser($data);
    } catch (Exception $e) {
        return false;
    }
}`,
			description: "Control flow structures should remain valid with dead code",
		},
		{
			name: "string_heavy_code",
			input: `<?php
function generateReport() {
    $template = "
        <html>
            <head><title>Report</title></head>
            <body>
                <h1>Monthly Report</h1>
                <p>Generated on: %s</p>
            </body>
        </html>
    ";
    
    return sprintf($template, date('Y-m-d H:i:s'));
}`,
			description: "String-heavy code should not be affected by dead code string token issues",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test both dead code and junk code injection
			result := processCodeWithDeadCodeInjection(t, tt.input, true, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestDeadCodeInjectionSyntaxValidation_EdgeCases(t *testing.T) {
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
			description: "Empty functions should handle dead code injection gracefully",
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
			description: "Minimal code should handle dead code injection",
		},
		{
			name: "special_characters_in_strings",
			input: `<?php
function specialChars() {
    $symbols = "!@#$%^&*()_+-=[]{}|;':\",./<>?";
    return $symbols;
}`,
			description: "Special characters in strings should not break token handling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with high injection rate to stress test
			result := processCodeWithDeadCodeInjection(t, tt.input, true, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

// Benchmark test to ensure performance doesn't degrade with syntax validation
func BenchmarkDeadCodeInjectionSyntaxValidation(b *testing.B) {
	phpCode := `<?php
function complexFunction($data) {
    $result = [];
    foreach ($data as $key => $value) {
        if (is_string($value)) {
            $result[$key] = strtoupper($value);
        } else {
            $result[$key] = $value;
        }
    }
    return $result;
}`

	for i := 0; i < b.N; i++ {
		result := processCodeWithDeadCodeInjection(nil, phpCode, true, true)
		_ = result // Prevent optimization
	}
}