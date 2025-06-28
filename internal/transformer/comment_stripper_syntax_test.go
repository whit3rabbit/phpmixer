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
Comment Stripper Syntax Validation Tests
========================================
These tests specifically validate that the comment stripping functionality
generates syntactically valid PHP code. They would have caught the original
token handling issues that caused malformed AST structures.

The tests cover:
1. Standard mode comment stripping
2. Aggressive mode comment stripping  
3. Various comment types (line, block, hash)
4. Empty comment token handling
5. Edge cases with mixed content

Each test validates PHP syntax using `php -l` to ensure no lint errors.
*/


// processCodeWithCommentStripper applies comment stripping and returns the result
func processCodeWithCommentStripper(t *testing.T, phpCode string, aggressiveMode bool) string {
	t.Helper()
	
	// Parse the PHP code
	parserConfig := conf.Config{
		Version: &version.Version{Major: 8, Minor: 1},
	}
	
	rootNode, err := parser.Parse([]byte(phpCode), parserConfig)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Create and configure comment stripper
	visitor := NewCommentStripperVisitor()
	visitor.AggressiveMode = aggressiveMode
	visitor.DebugMode = false // Disable debug to avoid test noise

	// Apply comment stripping
	traverser := traverser.NewTraverser(visitor)
	rootNode.Accept(traverser)

	// Generate the output
	var output strings.Builder
	p := printer.NewPrinter(&output)
	rootNode.Accept(p)

	return output.String()
}

func TestCommentStripperSyntaxValidation_StandardMode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "line_comments",
			input: `<?php
// This is a line comment
function test() {
    $x = 1; // Inline comment
    return $x;
}`,
			description: "Line comments should be properly stripped",
		},
		{
			name: "block_comments",
			input: `<?php
/* This is a block comment */
function test() {
    /* Multi-line
       block comment */
    $x = 1;
    return $x;
}`,
			description: "Block comments should be properly stripped",
		},
		{
			name: "hash_comments",
			input: `<?php
# Hash comment
function test() {
    $x = 1; # Inline hash comment
    return $x;
}`,
			description: "Hash comments should be properly stripped",
		},
		{
			name: "mixed_comments",
			input: `<?php
// Line comment
/* Block comment */
# Hash comment
function test() {
    $x = 1; // Inline
    /* Block */ $y = 2; # Hash
    return $x + $y;
}`,
			description: "Mixed comment types should be handled correctly",
		},
		{
			name: "empty_comments",
			input: `<?php
//
/**/
#
function test() {
    $x = 1;
    return $x;
}`,
			description: "Empty comments should not cause token issues",
		},
		{
			name: "docblock_comments",
			input: `<?php
/**
 * This is a docblock comment
 * @param int $x
 * @return int
 */
function test($x) {
    return $x * 2;
}`,
			description: "Docblock comments should be stripped correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process with standard mode (non-aggressive)
			result := processCodeWithCommentStripper(t, tt.input, false)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// Ensure comments were actually stripped
			if strings.Contains(result, "//") || strings.Contains(result, "/*") || strings.Contains(result, "#") {
				// Allow some cases where comments might be in strings
				if !strings.Contains(result, `"`) && !strings.Contains(result, `'`) {
					t.Errorf("Comments were not properly stripped from: %s", tt.name)
				}
			}
		})
	}
}

func TestCommentStripperSyntaxValidation_AggressiveMode(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "aggressive_line_comments",
			input: `<?php
// This should be completely removed
function test() {
    $x = 1; // This too
    return $x;
}`,
			description: "Aggressive mode should completely remove comment tokens",
		},
		{
			name: "aggressive_empty_comments",
			input: `<?php
//
/**/
function test() {
    $x = 1; //
    return $x; /**/
}`,
			description: "Empty comments in aggressive mode should not cause token issues",
		},
		{
			name: "complex_nested_structure",
			input: `<?php
// Top level comment
class TestClass {
    /* Class comment */
    private $prop; // Property comment
    
    /**
     * Method comment
     */
    public function method() {
        // Method body comment
        if (true) { // Condition comment
            /* Block comment */
            return 1;
        }
    }
}`,
			description: "Complex nested structures with comments should remain valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Process with aggressive mode
			result := processCodeWithCommentStripper(t, tt.input, true)
			
			// Validate the result is syntactically correct PHP
			validatePHPSyntax(t, result)
			
			// In aggressive mode, ensure all comments are gone
			if strings.Contains(result, "//") || strings.Contains(result, "/*") || strings.Contains(result, "#") {
				// Allow some cases where comments might be in strings
				if !strings.Contains(result, `"`) && !strings.Contains(result, `'`) {
					t.Errorf("Aggressive mode failed to strip all comments from: %s", tt.name)
				}
			}
		})
	}
}

func TestCommentStripperSyntaxValidation_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		description string
	}{
		{
			name: "comments_in_strings",
			input: `<?php
// Real comment
function test() {
    $str1 = "This // is not a comment";
    $str2 = 'This /* is not a comment */';
    $str3 = "This # is not a comment";
    return $str1 . $str2 . $str3;
}`,
			description: "Comments inside strings should not be stripped",
		},
		{
			name: "comment_like_operators",
			input: `<?php
// Real comment
function test() {
    $a = 5; // Comment
    $b = $a /* comment */ * 2;
    return $b;
}`,
			description: "Comments mixed with operators should be handled correctly",
		},
		{
			name: "heredoc_with_comments",
			input: `<?php
// Comment before heredoc
function test() {
    $text = <<<EOD
This // is not a comment
This /* is not a comment */
This # is not a comment
EOD;
    return $text; // Comment after
}`,
			description: "Heredoc content should not be affected by comment stripping",
		},
		{
			name: "only_comments_file",
			input: `<?php
// Just comments
/* Only comments */
# Nothing but comments
`,
			description: "File with only comments should remain valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test both modes
			for _, aggressive := range []bool{false, true} {
				modeName := "standard"
				if aggressive {
					modeName = "aggressive"
				}
				
				t.Run(modeName, func(t *testing.T) {
					result := processCodeWithCommentStripper(t, tt.input, aggressive)
					validatePHPSyntax(t, result)
				})
			}
		})
	}
}

// Benchmark test to ensure performance doesn't degrade
func BenchmarkCommentStripperSyntaxValidation(b *testing.B) {
	phpCode := `<?php
// Comment 1
/* Comment 2 */
# Comment 3
class TestClass {
    // Property comment
    private $prop;
    
    /* Method comment */
    public function method() {
        // Body comment
        return true;
    }
}
`

	for i := 0; i < b.N; i++ {
		result := processCodeWithCommentStripper(nil, phpCode, true)
		_ = result // Prevent optimization
	}
}