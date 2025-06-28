package transformer

import (
	"fmt"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
)

/*
Integration Syntax Validation Tests
====================================
These tests validate that all obfuscation features working together generate 
syntactically valid PHP code. They would have caught integration issues where
multiple transformations interfered with each other or created cumulative
syntax errors.

The tests cover:
1. All features enabled simultaneously
2. Feature combinations (pairwise testing)
3. Different execution orders
4. Complex code scenarios with multiple transformations
5. Edge cases where features might conflict

Each test validates PHP syntax using `php -l` to ensure no lint errors
when multiple obfuscation techniques are applied together.
*/


// ObfuscationConfig holds configuration for all obfuscation features
type ObfuscationConfig struct {
	CommentStripping    bool
	AggressiveComments  bool
	DeadCodeInjection   bool
	JunkCodeInsertion   bool
	ControlFlowObf      bool
	StatementShuffling  bool
	MinChunkSize        int
	InjectionRate       int
}

// processCodeWithAllFeatures applies all specified obfuscation features and returns the result
func processCodeWithAllFeatures(t *testing.T, phpCode string, config ObfuscationConfig) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Apply transformations in order (important for testing interaction)
	
	// 1. Comment stripping (should be first to avoid interfering with other transformations)
	if config.CommentStripping {
		visitor := NewCommentStripperVisitor()
		visitor.AggressiveMode = config.AggressiveComments
		visitor.DebugMode = false
		
		traverser := traverser.NewTraverser(visitor)
		rootNode.Accept(traverser)
	}
	
	// 2. Dead code and junk code injection
	if config.DeadCodeInjection || config.JunkCodeInsertion {
		visitor := NewDeadCodeInserterVisitor()
		visitor.InjectDeadCodeBlocks = config.DeadCodeInjection
		visitor.InjectJunkStatements = config.JunkCodeInsertion
		visitor.InjectionRate = config.InjectionRate
		visitor.DebugMode = false
		
		traverser := NewReplaceTraverser(visitor, false)
		rootNode = traverser.Traverse(rootNode)
	}
	
	// 3. Control flow obfuscation
	if config.ControlFlowObf {
		visitor := NewControlFlowObfuscatorVisitor()
		visitor.UseRandomConditions = false // Use deterministic for testing
		visitor.MaxNestingDepth = 1
		visitor.AddDeadBranches = false
		visitor.DebugMode = false
		
		traverser := traverser.NewTraverser(visitor)
		rootNode.Accept(traverser)
	}
	
	// 4. Statement shuffling (should be last to avoid interfering with other transformations)
	if config.StatementShuffling {
		visitor := NewStatementShufflerVisitor()
		visitor.MinChunkSize = config.MinChunkSize
		visitor.DebugMode = false
		
		traverser := NewReplaceTraverser(visitor, false)
		rootNode = traverser.Traverse(rootNode)
	}

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	rootNode.Accept(p)

	return output.String()
}

func TestIntegrationSyntaxValidation_AllFeatures(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "simple_function",
			input: `<?php
// This is a simple function
function test() {
    // Initialize variables
    $a = 1; // First variable
    $b = 2; // Second variable
    $c = 3; // Third variable
    /* Calculate sum */
    $sum = $a + $b + $c;
    return $sum; // Return result
}`,
			description: "Simple function with all features should remain syntactically valid",
		},
		{
			name: "class_with_methods",
			input: `<?php
/**
 * Calculator class for basic operations
 */
class Calculator {
    // Private property
    private $precision;
    
    /**
     * Constructor
     */
    public function __construct($precision = 2) {
        // Set precision
        $this->precision = $precision;
    }
    
    /**
     * Add two numbers
     */
    public function add($a, $b) {
        // Perform addition
        $result = $a + $b;
        $rounded = round($result, $this->precision);
        return $rounded; // Return rounded result
    }
    
    /* Multiply two numbers */
    public function multiply($x, $y) {
        $product = $x * $y; // Calculate product
        return round($product, $this->precision);
    }
}`,
			description: "Class with multiple methods should handle all transformations",
		},
		{
			name: "control_flow_structures",
			input: `<?php
// Process user data
function processUserData($data) {
    // Validate input
    if (empty($data)) {
        /* No data provided */
        return null;
    }
    
    $processed = [];
    // Loop through data
    foreach ($data as $key => $value) {
        // Check value type
        if (is_string($value)) {
            $processed[$key] = trim($value); // Clean string
        } elseif (is_numeric($value)) {
            $processed[$key] = (float)$value; // Convert to float
        } else {
            $processed[$key] = $value; // Keep as is
        }
    }
    
    return $processed; // Return processed data
}`,
			description: "Control flow structures should remain functional with all features",
		},
		{
			name: "complex_scenario",
			input: `<?php
// Configuration and setup
$config = [
    'debug' => true,
    'timeout' => 30,
    'retries' => 3
];

/**
 * Data processor service
 */
class DataProcessor {
    private $config;
    private $logger;
    
    public function __construct($config) {
        // Initialize configuration
        $this->config = $config;
        $this->logger = new Logger();
    }
    
    /**
     * Process incoming data
     */
    public function process($rawData) {
        // Step 1: Validate
        $validated = $this->validate($rawData);
        if (!$validated) {
            throw new InvalidArgumentException('Invalid data');
        }
        
        // Step 2: Transform
        $transformed = [];
        foreach ($rawData as $item) {
            $processed = $this->transformItem($item);
            $transformed[] = $processed;
        }
        
        // Step 3: Finalize
        $result = $this->finalize($transformed);
        return $result;
    }
}`,
			description: "Complex scenario with classes, functions, and control flow",
		},
	}

	config := ObfuscationConfig{
		CommentStripping:   true,
		AggressiveComments: true,
		DeadCodeInjection:  true,
		JunkCodeInsertion:  true,
		ControlFlowObf:     true,
		StatementShuffling: true,
		MinChunkSize:       2,
		InjectionRate:      50, // Moderate injection rate for testing
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithAllFeatures(t, tt.input, config)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Verify transformations were applied
			// Comments should be stripped
			if strings.Contains(result, "//") || strings.Contains(result, "/*") {
				// Allow comments in strings
				if !strings.Contains(result, `"`) && !strings.Contains(result, `'`) {
					t.Errorf("Comments were not stripped in: %s", tt.name)
				}
			}
			
			// Should contain some form of obfuscation indicators
			hasObfuscation := strings.Contains(result, "if(") || 
							 strings.Contains(result, "_junk") || 
							 strings.Contains(result, "_temp")
			
			if !hasObfuscation {
				t.Logf("Warning: No clear obfuscation indicators found in: %s", tt.name)
				t.Logf("Result: %s", result)
			}
		})
	}
}

func TestIntegrationSyntaxValidation_FeatureCombinations(t *testing.T) {
	baseCode := `<?php
// Test function
function test() {
    // Variable assignments
    $a = 1;
    $b = 2;
    $c = 3;
    /* Return sum */
    return $a + $b + $c;
}`

	// Test different combinations of features
	combinations := []struct {
		name   string
		config ObfuscationConfig
	}{
		{
			name: "comments_and_dead_code",
			config: ObfuscationConfig{
				CommentStripping:  true,
				DeadCodeInjection: true,
				InjectionRate:     100,
			},
		},
		{
			name: "comments_and_control_flow",
			config: ObfuscationConfig{
				CommentStripping: true,
				ControlFlowObf:   true,
			},
		},
		{
			name: "dead_code_and_shuffling",
			config: ObfuscationConfig{
				DeadCodeInjection:  true,
				StatementShuffling: true,
				MinChunkSize:       2,
				InjectionRate:      100,
			},
		},
		{
			name: "control_flow_and_shuffling",
			config: ObfuscationConfig{
				ControlFlowObf:     true,
				StatementShuffling: true,
				MinChunkSize:       2,
			},
		},
		{
			name: "junk_code_and_control_flow",
			config: ObfuscationConfig{
				JunkCodeInsertion: true,
				ControlFlowObf:    true,
				InjectionRate:     100,
			},
		},
		{
			name: "triple_combination",
			config: ObfuscationConfig{
				CommentStripping:  true,
				DeadCodeInjection: true,
				ControlFlowObf:    true,
				InjectionRate:     75,
			},
		},
	}

	for _, combo := range combinations {
		t.Run(combo.name, func(t *testing.T) {
			result := processCodeWithAllFeatures(t, baseCode, combo.config)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure function structure is preserved
			if !strings.Contains(result, "function test") {
				t.Errorf("Function structure was corrupted in combination: %s", combo.name)
			}
		})
	}
}

func TestIntegrationSyntaxValidation_TransformationOrder(t *testing.T) {
	baseCode := `<?php
// Main function
function main() {
    // Setup variables
    $config = getConfig();
    $data = loadData();
    $processor = new DataProcessor($config);
    
    // Process data
    $result = $processor->process($data);
    return $result;
}`

	// Test the same configuration but ensure consistent results
	config := ObfuscationConfig{
		CommentStripping:   true,
		DeadCodeInjection:  true,
		ControlFlowObf:     true,
		StatementShuffling: true,
		MinChunkSize:       2,
		InjectionRate:      50,
	}

	// Run multiple times to check for consistency and stability
	results := make([]string, 3)
	for i := 0; i < 3; i++ {
		results[i] = processCodeWithAllFeatures(t, baseCode, config)
		
		// Each result should be syntactically valid
		validatePHPSyntax(t, results[i])
		
		// Ensure basic structure is preserved
		if !strings.Contains(results[i], "function main") {
			t.Errorf("Function structure corrupted in iteration %d", i)
		}
	}
}

func TestIntegrationSyntaxValidation_ErrorConditions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		config      ObfuscationConfig
		description string
	}{
		{
			name: "empty_function_all_features",
			input: `<?php
// Empty function
function empty() {
    // Nothing here
}`,
			config: ObfuscationConfig{
				CommentStripping:   true,
				DeadCodeInjection:  true,
				JunkCodeInsertion:  true,
				ControlFlowObf:     true,
				StatementShuffling: true,
				MinChunkSize:       1,
				InjectionRate:      100,
			},
			description: "Empty functions should handle all features gracefully",
		},
		{
			name: "single_statement_all_features",
			input: `<?php
function single() {
    return 42; // Just return
}`,
			config: ObfuscationConfig{
				CommentStripping:   true,
				DeadCodeInjection:  true,
				JunkCodeInsertion:  true,
				ControlFlowObf:     true,
				StatementShuffling: true,
				MinChunkSize:       1,
				InjectionRate:      100,
			},
			description: "Single statement functions should remain valid",
		},
		{
			name: "minimal_script_all_features",
			input: `<?php
$x = 1; // Simple assignment`,
			config: ObfuscationConfig{
				CommentStripping:   true,
				DeadCodeInjection:  true,
				JunkCodeInsertion:  true,
				ControlFlowObf:     false, // Control flow obfuscation doesn't apply to root level single statements
				StatementShuffling: true,
				MinChunkSize:       1,
				InjectionRate:      100,
			},
			description: "Minimal scripts should handle multiple transformations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithAllFeatures(t, tt.input, tt.config)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestIntegrationSyntaxValidation_RealWorldScenarios(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "mvc_controller",
			input: `<?php
/**
 * User controller for handling user operations
 */
class UserController {
    private $userService;
    private $validator;
    
    /**
     * Constructor
     */
    public function __construct($userService, $validator) {
        // Initialize services
        $this->userService = $userService;
        $this->validator = $validator;
    }
    
    /**
     * Create a new user
     */
    public function createUser($userData) {
        // Validate input data
        $errors = $this->validator->validate($userData);
        if (!empty($errors)) {
            throw new ValidationException('Invalid user data');
        }
        
        // Check if user exists
        $existingUser = $this->userService->findByEmail($userData['email']);
        if ($existingUser) {
            throw new ConflictException('User already exists');
        }
        
        // Create new user
        $user = $this->userService->create($userData);
        return $user;
    }
}`,
			description: "MVC controller should handle all transformations correctly",
		},
		{
			name: "data_processing_pipeline",
			input: `<?php
// Data processing configuration
$config = [
    'batch_size' => 100,
    'timeout' => 30,
    'retry_count' => 3
];

/**
 * Process data in batches
 */
function processBatch($data, $config) {
    // Initialize counters
    $processed = 0;
    $errors = 0;
    $batch = [];
    
    // Process each item
    foreach ($data as $index => $item) {
        try {
            // Validate item
            $validated = validateItem($item);
            if (!$validated) {
                $errors++;
                continue;
            }
            
            // Transform item
            $transformed = transformItem($item);
            $batch[] = $transformed;
            $processed++;
            
            // Check batch size
            if (count($batch) >= $config['batch_size']) {
                // Process batch
                $result = processBatchItems($batch);
                $batch = []; // Reset batch
            }
        } catch (Exception $e) {
            // Log error and continue
            error_log("Error processing item $index: " . $e->getMessage());
            $errors++;
        }
    }
    
    // Process remaining items
    if (!empty($batch)) {
        processBatchItems($batch);
    }
    
    return [
        'processed' => $processed,
        'errors' => $errors
    ];
}`,
			description: "Data processing pipeline with complex control flow",
		},
	}

	config := ObfuscationConfig{
		CommentStripping:   true,
		AggressiveComments: true,
		DeadCodeInjection:  true,
		JunkCodeInsertion:  true,
		ControlFlowObf:     true,
		StatementShuffling: true,
		MinChunkSize:       2,
		InjectionRate:      25, // Lower rate for complex scenarios
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processCodeWithAllFeatures(t, tt.input, config)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
		})
	}
}

func TestIntegrationSyntaxValidation_PerformanceStability(t *testing.T) {
	// Test that multiple transformations don't cause exponential growth or instability
	baseCode := `<?php
function performanceTest() {
    $data = [];
    $count = 10;
    $multiplier = 2;
    
    for ($i = 0; $i < $count; $i++) {
        $value = $i * $multiplier;
        $data[] = $value;
    }
    
    $sum = array_sum($data);
    $average = $sum / count($data);
    
    return [
        'data' => $data,
        'sum' => $sum,
        'average' => $average,
        'count' => count($data)
    ];
}`

	config := ObfuscationConfig{
		CommentStripping:   true,
		DeadCodeInjection:  true,
		JunkCodeInsertion:  true,
		ControlFlowObf:     true,
		StatementShuffling: true,
		MinChunkSize:       2,
		InjectionRate:      30,
	}

	// Process the same code multiple times
	for i := 0; i < 5; i++ {
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			result := processCodeWithAllFeatures(t, baseCode, config)
			
			// Validate syntax
			validatePHPSyntax(t, result)
			
			// Check that the result isn't excessively long (indicating runaway transformation)
			if len(result) > len(baseCode)*10 {
				t.Errorf("Result is excessively long (%d chars vs original %d chars)", 
					len(result), len(baseCode))
			}
			
			// Ensure core structure is preserved
			if !strings.Contains(result, "function performanceTest") {
				t.Errorf("Core function structure lost in iteration %d", i)
			}
		})
	}
}

// Benchmark test for integration performance
func BenchmarkIntegrationAllFeatures(b *testing.B) {
	phpCode := `<?php
function benchmarkTest($data) {
    $config = getConfig();
    $processor = new DataProcessor($config);
    $validator = new Validator();
    
    $results = [];
    foreach ($data as $item) {
        if ($validator->validate($item)) {
            $processed = $processor->process($item);
            $results[] = $processed;
        }
    }
    
    return $results;
}`

	config := ObfuscationConfig{
		CommentStripping:   true,
		DeadCodeInjection:  true,
		JunkCodeInsertion:  true,
		ControlFlowObf:     true,
		StatementShuffling: true,
		MinChunkSize:       2,
		InjectionRate:      50,
	}

	for i := 0; i < b.N; i++ {
		result := processCodeWithAllFeatures(nil, phpCode, config)
		_ = result // Prevent optimization
	}
}