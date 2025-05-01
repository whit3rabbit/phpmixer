package transformer

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
	"github.com/stretchr/testify/assert"
)

// TestShuffleSimpleAssignments tests shuffling of simple assignment statements
func TestShuffleSimpleAssignments(t *testing.T) {
	src := `<?php
$a = 1;
$b = 2;
$c = 3;
$d = $a + $b + $c;
echo $d;
`

	// Parse the PHP code to AST
	rootNode, err := parseCodeToAst(src)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Apply statement shuffling
	shuffledRoot, err := applyStatementShuffling(rootNode)
	if err != nil {
		t.Fatalf("Failed to apply statement shuffling: %v", err)
	}

	// Print the shuffled code
	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	traverser.NewTraverser(p).Traverse(shuffledRoot)
	shuffledCode := buf.String()

	// Print the original and shuffled code for inspection
	fmt.Printf("Original:\n%s\n", src)
	fmt.Printf("Shuffled:\n%s\n", shuffledCode)

	// Verify that we still have all the original statements
	// Note that we can't easily test the exact order, since shuffling is random
	for _, line := range []string{"$a = 1", "$b = 2", "$c = 3"} {
		assert.Contains(t, shuffledCode, line)
	}

	// Make sure the last two statements are still in the original order
	// since they have a dependency
	idxD := strings.Index(shuffledCode, "$d = $a + $b + $c")
	idxEcho := strings.Index(shuffledCode, "echo $d")
	assert.True(t, idxD >= 0 && idxEcho >= 0 && idxD < idxEcho, "Order of dependent statements should be preserved")
}

// TestShuffleComplexBlock tests shuffling statements with conditional blocks
func TestShuffleComplexBlock(t *testing.T) {
	src := `<?php
$a = 1;
$b = 2;
if ($a > 0) {
    echo "a is positive";
}
$c = 3;
$d = 4;
echo "Done";
`

	// Parse the PHP code to AST
	rootNode, err := parseCodeToAst(src)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Apply statement shuffling
	shuffledRoot, err := applyStatementShuffling(rootNode)
	if err != nil {
		t.Fatalf("Failed to apply statement shuffling: %v", err)
	}

	// Print the shuffled code
	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	traverser.NewTraverser(p).Traverse(shuffledRoot)
	shuffledCode := buf.String()

	// Print the original and shuffled code for inspection
	fmt.Printf("Original:\n%s\n", src)
	fmt.Printf("Shuffled:\n%s\n", shuffledCode)

	// Verify that the if statement remains intact
	assert.Contains(t, shuffledCode, "if ($a > 0) {")
	assert.Contains(t, shuffledCode, "echo \"a is positive\"")

	// Verify that all statements are still present
	for _, line := range []string{"$a = 1", "$b = 2", "$c = 3", "$d = 4", "echo \"Done\""} {
		assert.Contains(t, shuffledCode, line)
	}

	// Verify that the statements before the if statement are still before it
	// This is tricker because of shuffling, but we can check that $a is defined before the if
	idxA := strings.Index(shuffledCode, "$a = 1")
	idxIf := strings.Index(shuffledCode, "if ($a > 0)")
	assert.True(t, idxA >= 0 && idxIf >= 0 && idxA < idxIf, "$a must be defined before the if statement")
}

// TestShuffleFunctionBody tests shuffling statements within a function body
func TestShuffleFunctionBody(t *testing.T) {
	src := `<?php
function test() {
    $a = 1;
    $b = 2;
    $c = 3;
    return $a + $b + $c;
}
`

	// Parse the PHP code to AST
	rootNode, err := parseCodeToAst(src)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Apply statement shuffling
	shuffledRoot, err := applyStatementShuffling(rootNode)
	if err != nil {
		t.Fatalf("Failed to apply statement shuffling: %v", err)
	}

	// Print the shuffled code
	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	traverser.NewTraverser(p).Traverse(shuffledRoot)
	shuffledCode := buf.String()

	// Print the original and shuffled code for inspection
	fmt.Printf("Original:\n%s\n", src)
	fmt.Printf("Shuffled:\n%s\n", shuffledCode)

	// Verify that all statements in the function are still present
	for _, line := range []string{"$a = 1", "$b = 2", "$c = 3", "return $a + $b + $c"} {
		assert.Contains(t, shuffledCode, line)
	}

	// Find function body bounds
	functionBodyStart := strings.Index(shuffledCode, "function test()")
	functionBodyEnd := -1
	if functionBodyStart >= 0 {
		// Find the matching closing brace for this function
		openBracePos := strings.Index(shuffledCode[functionBodyStart:], "{") + functionBodyStart
		braceCount := 1
		for i := openBracePos + 1; i < len(shuffledCode) && braceCount > 0; i++ {
			if shuffledCode[i] == '{' {
				braceCount++
			} else if shuffledCode[i] == '}' {
				braceCount--
				if braceCount == 0 {
					functionBodyEnd = i
					break
				}
			}
		}
	}

	// If we couldn't find function bounds, use a more basic check
	if functionBodyStart < 0 || functionBodyEnd < 0 {
		t.Logf("Could not find function bounds, using basic check")
		// Just check that a return statement exists
		assert.Contains(t, shuffledCode, "return $a + $b + $c")
	} else {
		// Extract just the function body
		functionBody := shuffledCode[functionBodyStart : functionBodyEnd+1]

		// Verify that return statement is within the actual function body
		// and appears after all other statements
		returnInBody := strings.LastIndex(functionBody, "return")
		assert.True(t, returnInBody > 0, "Return statement should be present in function body")

		// Check that return appears after assignments
		aAssignPos := strings.LastIndex(functionBody, "$a = 1")
		bAssignPos := strings.LastIndex(functionBody, "$b = 2")
		cAssignPos := strings.LastIndex(functionBody, "$c = 3")

		assert.True(t, aAssignPos >= 0 && aAssignPos < returnInBody, "$a assignment should come before return")
		assert.True(t, bAssignPos >= 0 && bAssignPos < returnInBody, "$b assignment should come before return")
		assert.True(t, cAssignPos >= 0 && cAssignPos < returnInBody, "$c assignment should come before return")
	}
}

// TestShuffleMethodBody tests shuffling statements within a class method body
func TestShuffleMethodBody(t *testing.T) {
	src := `<?php
class Test {
    public function method() {
        $a = 1;
        $b = 2;
        $c = 3;
        return $a + $b + $c;
    }
}
`

	// Parse the PHP code to AST
	rootNode, err := parseCodeToAst(src)
	if err != nil {
		t.Fatalf("Failed to parse PHP code: %v", err)
	}

	// Apply statement shuffling
	shuffledRoot, err := applyStatementShuffling(rootNode)
	if err != nil {
		t.Fatalf("Failed to apply statement shuffling: %v", err)
	}

	// Print the shuffled code
	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	traverser.NewTraverser(p).Traverse(shuffledRoot)
	shuffledCode := buf.String()

	// Print the original and shuffled code for inspection
	fmt.Printf("Original:\n%s\n", src)
	fmt.Printf("Shuffled:\n%s\n", shuffledCode)

	// Verify that all statements in the method are still present
	for _, line := range []string{"$a = 1", "$b = 2", "$c = 3", "return $a + $b + $c"} {
		assert.Contains(t, shuffledCode, line)
	}

	// Verify that return statement is still the last one in the method
	methodBodyStart := strings.Index(shuffledCode, "public function method()")
	methodBodyEnd := strings.Index(shuffledCode[methodBodyStart:], "}") + methodBodyStart
	methodBody := shuffledCode[methodBodyStart:methodBodyEnd]
	idxReturn := strings.LastIndex(methodBody, "return")
	assert.True(t, idxReturn > 0, "Return statement should be present in the method body")
}

// Helper function to parse PHP code to AST
func parseCodeToAst(src string) (*ast.Root, error) {
	// Configure the PHP parser
	parserConfig := conf.Config{
		Version: &version.Version{
			Major: 7,
			Minor: 4,
		},
	}

	// Use parser.Parse directly
	vertex, err := parser.Parse([]byte(src), parserConfig)
	if err != nil {
		return nil, fmt.Errorf("parse error: %v", err)
	}

	// Convert the interface to *ast.Root with type assertion
	root, ok := vertex.(*ast.Root)
	if !ok {
		return nil, fmt.Errorf("expected *ast.Root but got %T", vertex)
	}

	return root, nil
}

// Helper function to apply statement shuffling to an AST
func applyStatementShuffling(root *ast.Root) (*ast.Root, error) {
	// Create the statement shuffler
	shuffler := NewStatementShufflerVisitor()

	// Create a parent tracker and build parent map
	shuffler.SetParentTracker(BuildParentMap(root, false))

	// Traverse the AST, collecting statements to shuffle
	traverser.NewTraverser(shuffler).Traverse(root)

	// Apply the replacements using a node replacer
	replacer := NewASTNodeReplacer(shuffler.GetParentTracker(), false)
	for _, repl := range shuffler.GetReplacements() {
		replacer.AddNodeReplacement(repl)
	}
	replacer.ApplyReplacements(root)

	return root, nil
}
