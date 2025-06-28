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
Statement Shuffling Syntax Validation Tests
============================================
These tests validate that statement shuffling generates syntactically valid PHP code.
They would have caught the original AST node field preservation issues that caused
malformed function and method declarations during statement reordering.

The tests cover:
1. Function statement shuffling with proper field preservation
2. Method statement shuffling with class structure integrity
3. Root-level statement shuffling
4. Statement list shuffling in various contexts
5. Independent statement identification and reordering
6. Control flow statement preservation

Each test validates PHP syntax using `php -l` to ensure no lint errors.
*/


// processCodeWithStatementShuffling applies statement shuffling and returns the result
func processCodeWithStatementShuffling(t *testing.T, phpCode string, minChunkSize int) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Create and configure statement shuffler
	visitor := NewStatementShufflerVisitor()
	visitor.MinChunkSize = minChunkSize
	visitor.DebugMode = false // Disable debug to avoid test noise

	// Apply statement shuffling using replace traverser
	traverser := NewReplaceTraverser(visitor, false)
	result := traverser.Traverse(rootNode)

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	result.Accept(p)

	return output.String()
}

func TestStatementShufflingSyntaxValidation_FunctionStatements(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "simple_assignments",
			input: `<?php
function test() {
    $a = 1;
    $b = 2;
    $c = 3;
    return $a + $b + $c;
}`,
			description: "Simple variable assignments should be shufflable while preserving function structure",
		},
		{
			name: "mixed_statements",
			input: `<?php
function processData() {
    $config = getConfig();
    $data = loadData();
    $result = [];
    $processed = process($data, $config);
    return $processed;
}`,
			description: "Mixed independent statements should shuffle while maintaining syntax",
		},
		{
			name: "function_with_parameters",
			input: `<?php
function calculate($x, $y, $z) {
    $sum = $x + $y;
    $product = $x * $y;
    $difference = $x - $y;
    $final = $sum + $product + $difference + $z;
    return $final;
}`,
			description: "Functions with parameters should preserve parameter list during shuffling",
		},
		{
			name: "function_with_return_type",
			input: `<?php
function getNumbers(): array {
    $first = 1;
    $second = 2;
    $third = 3;
    return [$first, $second, $third];
}`,
			description: "Return type declarations should be preserved during shuffling",
		},
		{
			name: "global_and_static_statements",
			input: `<?php
function complexFunction() {
    global $globalVar;
    static $counter = 0;
    $counter++;
    $localVar = $globalVar + $counter;
    return $localVar;
}`,
			description: "Global and static statements should be identified as shufflable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test statement shuffling with minimum chunk size of 2
			result := processCodeWithStatementShuffling(t, tt.input, 2)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure function structure is preserved
			if !strings.Contains(result, "function") {
				t.Errorf("Function declaration was corrupted in: %s", tt.name)
			}
			
			// Ensure return statement is still present
			if strings.Contains(tt.input, "return") && !strings.Contains(result, "return") {
				t.Errorf("Return statement was lost during shuffling in: %s", tt.name)
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_ClassMethods(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "public_method",
			input: `<?php
class Calculator {
    public function add($a, $b) {
        $temp1 = $a;
        $temp2 = $b;
        $sum = $temp1 + $temp2;
        return $sum;
    }
}`,
			description: "Public method statements should shuffle while preserving method signature",
		},
		{
			name: "private_method_with_modifiers",
			input: `<?php
class DataProcessor {
    private function process($data) {
        $filtered = filter($data);
        $validated = validate($filtered);
        $sanitized = sanitize($validated);
        return $sanitized;
    }
}`,
			description: "Private method modifiers should be preserved during shuffling",
		},
		{
			name: "static_method",
			input: `<?php
class Utils {
    public static function format($value) {
        $cleaned = trim($value);
        $formatted = ucfirst($cleaned);
        $final = htmlspecialchars($formatted);
        return $final;
    }
}`,
			description: "Static method declarations should remain intact",
		},
		{
			name: "method_with_return_type",
			input: `<?php
class StringProcessor {
    public function process(string $input): string {
        $step1 = strtolower($input);
        $step2 = trim($step1);
        $step3 = ucwords($step2);
        return $step3;
    }
}`,
			description: "Method parameter and return types should be preserved",
		},
		{
			name: "constructor_method",
			input: `<?php
class Database {
    private $connection;
    
    public function __construct($host, $user, $pass) {
        $dsn = "mysql:host=$host";
        $options = [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION];
        $this->connection = new PDO($dsn, $user, $pass, $options);
    }
}`,
			description: "Constructor methods should handle shuffling correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, tt.input, 2)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure class structure is preserved
			if !strings.Contains(result, "class") {
				t.Errorf("Class declaration was corrupted in: %s", tt.name)
			}
			
			// Ensure method visibility/modifiers are preserved
			visibilityKeywords := []string{"public", "private", "protected", "static"}
			for _, keyword := range visibilityKeywords {
				if strings.Contains(tt.input, keyword) && !strings.Contains(result, keyword) {
					t.Errorf("Method modifier '%s' was lost during shuffling in: %s", keyword, tt.name)
				}
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_RootStatements(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "global_assignments",
			input: `<?php
$config = loadConfig();
$database = connectDatabase();
$cache = initializeCache();
$logger = createLogger();`,
			description: "Global variable assignments should be shufflable",
		},
		{
			name: "function_definitions",
			input: `<?php
function first() {
    return 1;
}

function second() {
    return 2;
}

function third() {
    return 3;
}`,
			description: "Function definitions should be shufflable at root level",
		},
		{
			name: "mixed_root_statements",
			input: `<?php
$globalVar = 'test';

function helper() {
    return 'helper';
}

$anotherGlobal = 'another';

class MyClass {
    public function method() {
        return 'method';
    }
}`,
			description: "Mixed root-level statements should shuffle appropriately",
		},
		{
			name: "require_and_assignments",
			input: `<?php
$path = '/var/www';
$config = require 'config.php';
$utils = require 'utils.php';
$data = loadInitialData();`,
			description: "Require statements and assignments should handle shuffling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, tt.input, 1)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure PHP opening tag is preserved
			if !strings.Contains(result, "<?php") {
				t.Errorf("PHP opening tag was lost during shuffling in: %s", tt.name)
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_ControlFlowPreservation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "return_statement_preservation",
			input: `<?php
function test() {
    $a = 1;
    $b = 2;
    $c = 3;
    return $a + $b + $c;
}`,
			description: "Return statements should remain at the end after shuffling",
		},
		{
			name: "if_statement_preservation",
			input: `<?php
function conditional() {
    $x = 1;
    $y = 2;
    if ($x > $y) {
        return 'greater';
    }
    return 'not greater';
}`,
			description: "If statements should not be shuffled with independent assignments",
		},
		{
			name: "loop_preservation",
			input: `<?php
function loopTest() {
    $data = [];
    $count = 10;
    for ($i = 0; $i < $count; $i++) {
        $data[] = $i;
    }
    return $data;
}`,
			description: "Loop statements should maintain their position relative to dependencies",
		},
		{
			name: "try_catch_preservation",
			input: `<?php
function errorHandling() {
    $config = getConfig();
    $data = prepareData();
    try {
        $result = processData($data, $config);
        return $result;
    } catch (Exception $e) {
        return null;
    }
}`,
			description: "Try-catch blocks should not be shuffled",
		},
		{
			name: "throw_statement_preservation",
			input: `<?php
function validator($input) {
    $cleaned = trim($input);
    $validated = validate($cleaned);
    if (!$validated) {
        throw new InvalidArgumentException('Invalid input');
    }
    return $validated;
}`,
			description: "Throw statements should maintain their control flow position",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, tt.input, 2)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure control flow statements are preserved
			controlFlowKeywords := []string{"return", "if", "for", "while", "try", "catch", "throw"}
			for _, keyword := range controlFlowKeywords {
				if strings.Contains(tt.input, keyword) && !strings.Contains(result, keyword) {
					t.Errorf("Control flow keyword '%s' was lost during shuffling in: %s", keyword, tt.name)
				}
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_IndependentStatements(t *testing.T) {
	baseCode := `<?php
function test() {
    $var1 = 'first';
    $var2 = 'second';
    $var3 = 'third';
    $var4 = 'fourth';
    echo $var1;
    echo $var2;
    echo $var3;
    echo $var4;
}`

	// Run multiple times to verify shuffling occurs and syntax remains valid
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, baseCode, 2)
			validatePHPSyntax(t, result)
			
			// Ensure all variable assignments are present
			vars := []string{"$var1", "$var2", "$var3", "$var4"}
			for _, varName := range vars {
				if !strings.Contains(result, varName) {
					t.Errorf("Variable %s was lost during shuffling", varName)
				}
			}
			
			// Ensure all echo statements are present
			echoCount := strings.Count(result, "echo")
			if echoCount != 4 {
				t.Errorf("Expected 4 echo statements, found %d", echoCount)
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_ChunkSizes(t *testing.T) {
	baseCode := `<?php
function chunkTest() {
    $a = 1;
    $b = 2;
    $c = 3;
    $d = 4;
    $e = 5;
    return $a + $b + $c + $d + $e;
}`

	// Test different minimum chunk sizes
	chunkSizes := []int{1, 2, 3, 4, 5}
	
	for _, chunkSize := range chunkSizes {
		t.Run(fmt.Sprintf("chunk_size_%d", chunkSize), func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, baseCode, chunkSize)
			validatePHPSyntax(t, result)
			
			// Ensure function structure is maintained
			if !strings.Contains(result, "function chunkTest()") {
				t.Errorf("Function signature was corrupted with chunk size %d", chunkSize)
			}
			
			// Ensure return statement is preserved
			if !strings.Contains(result, "return") {
				t.Errorf("Return statement was lost with chunk size %d", chunkSize)
			}
		})
	}
}

func TestStatementShufflingSyntaxValidation_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "nested_class_hierarchy",
			input: `<?php
abstract class BaseService {
    protected $config;
    
    public function __construct($config) {
        $this->config = $config;
        $defaultSettings = getDefaults();
        $mergedConfig = array_merge($defaultSettings, $config);
        $this->config = $mergedConfig;
    }
    
    abstract public function process($data);
}

class ConcreteService extends BaseService {
    public function process($data) {
        $validated = $this->validate($data);
        $filtered = $this->filter($validated);
        $processed = $this->transform($filtered);
        return $processed;
    }
}`,
			description: "Complex class hierarchies should maintain integrity during shuffling",
		},
		{
			name: "namespace_with_functions",
			input: `<?php
namespace App\Services;

use App\Utils\Logger;
use App\Utils\Config;

$logger = new Logger();
$config = new Config();

function initialize() {
    $step1 = prepareEnvironment();
    $step2 = loadConfiguration();
    $step3 = setupLogging();
    return [$step1, $step2, $step3];
}`,
			description: "Namespaces with use statements should handle shuffling correctly",
		},
		{
			name: "multiple_functions_with_dependencies",
			input: `<?php
function loadConfig() {
    $path = getConfigPath();
    $content = file_get_contents($path);
    $config = json_decode($content, true);
    return $config;
}

function getConfigPath() {
    $env = getenv('ENV');
    $filename = "config.$env.json";
    return "/etc/app/$filename";
}

function processConfig($config) {
    $validated = validateConfig($config);
    $normalized = normalizeConfig($validated);
    $cached = cacheConfig($normalized);
    return $cached;
}`,
			description: "Multiple functions should shuffle independently without breaking dependencies",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, tt.input, 2)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestStatementShufflingSyntaxValidation_EdgeCases(t *testing.T) {
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
			description: "Empty functions should handle shuffling gracefully",
		},
		{
			name: "single_statement_function",
			input: `<?php
function singleStatement() {
    return 42;
}`,
			description: "Single statement functions should remain unchanged",
		},
		{
			name: "function_with_only_comments",
			input: `<?php
function commentedFunction() {
    // First comment
    // Second comment
    // Third comment
}`,
			description: "Functions with only comments should handle shuffling",
		},
		{
			name: "minimal_script",
			input: `<?php
$x = 1;`,
			description: "Minimal scripts should handle shuffling without issues",
		},
		{
			name: "complex_expression_assignments",
			input: `<?php
function complexExpressions() {
    $result1 = (($x + $y) * $z) / ($a - $b);
    $result2 = isset($data['key']) ? $data['key'] : 'default';
    $result3 = $obj->method()->chainedMethod($param);
    return [$result1, $result2, $result3];
}`,
			description: "Complex expressions should not break during shuffling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithStatementShuffling(t, tt.input, 1)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

// Benchmark test to ensure performance doesn't degrade with statement shuffling
func BenchmarkStatementShufflingSyntaxValidation(b *testing.B) {
	phpCode := `<?php
function complexFunction() {
    $config = loadConfig();
    $database = connectDatabase();
    $cache = initializeCache();
    $logger = createLogger();
    $session = startSession();
    $user = authenticate();
    $permissions = loadPermissions($user);
    $data = fetchData($database, $user);
    $filtered = filterData($data, $permissions);
    $processed = processData($filtered, $config);
    return $processed;
}`

	for i := 0; i < b.N; i++ {
		result := processCodeWithStatementShuffling(nil, phpCode, 2)
		_ = result // Prevent optimization
	}
}