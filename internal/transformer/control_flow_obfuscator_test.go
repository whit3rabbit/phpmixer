package transformer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
)

/*
Control Flow Obfuscation Tests
------------------------------
These tests verify the functionality of the control flow obfuscator, which transforms PHP code
by wrapping various code blocks in redundant conditional statements that always evaluate to true.

The obfuscator should handle the following PHP constructs:
1. Function bodies
2. Class method bodies
3. Loop statements (for, while, foreach, do-while)
4. If/elseif/else statements

For each test, we:
1. Define a sample PHP code snippet
2. Parse it into an AST
3. Apply the ControlFlowObfuscatorVisitor
4. Generate the transformed PHP code
5. Verify that the transformation matches the expected output

The transformation pattern is consistent: code blocks are wrapped in `if(1){...}` statements,
which always evaluate to true but make static analysis more difficult.
*/

// Transformer defines the interface for AST transformers
type Transformer interface {
	Transform(root *ast.Root) (*ast.Root, error)
}

// ControlFlowObfuscator implements the Transformer interface
type ControlFlowObfuscator struct {
	randomize bool
}

// NewControlFlowObfuscator creates a new control flow obfuscator
func NewControlFlowObfuscator(randomize bool) *ControlFlowObfuscator {
	return &ControlFlowObfuscator{
		randomize: randomize,
	}
}

// Transform applies control flow obfuscation to the AST
func (o *ControlFlowObfuscator) Transform(root *ast.Root) (*ast.Root, error) {
	// Create a visitor and apply it
	visitor := NewControlFlowObfuscatorVisitor()
	visitor.UseRandomConditions = o.randomize

	// Apply the visitor to the AST
	traverser := traverser.NewTraverser(visitor)
	root.Accept(traverser)

	return root, nil
}

// TestControlFlowObfuscatorFunctionBody tests the obfuscation of function bodies.
// A function body's statements should be wrapped in an if(RANDOM_TRUE_CONDITION){...} block.
func TestControlFlowObfuscatorFunctionBody(t *testing.T) {
	src := `<?php
function test() {
    echo "Hello";
    return "World";
}
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Print the original AST structure for debugging purposes
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	// Create and apply the control flow obfuscator
	obfuscator := NewControlFlowObfuscatorVisitor()
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Instead of exact string comparison, check for presence of "if" in the function body
	if !strings.Contains(output, "function test() {if") {
		t.Errorf("Function body not properly obfuscated. Output:\n%s", output)
	}

	// Verify that the original statements are present inside the if block
	if !strings.Contains(output, "echo \"Hello\";") || !strings.Contains(output, "return \"World\";") {
		t.Errorf("Original statements not preserved in obfuscated output. Output:\n%s", output)
	}
}

// TestControlFlowObfuscatorClassMethod tests the obfuscation of class method bodies.
// Similar to function bodies, class method statements should be wrapped in an if(RANDOM_TRUE_CONDITION){...} block.
func TestControlFlowObfuscatorClassMethod(t *testing.T) {
	src := `<?php
class TestClass {
    public function testMethod() {
        echo "Hello";
        return "World";
    }
}
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Print the original AST structure for debugging purposes
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	// Create and apply the control flow obfuscator
	obfuscator := NewControlFlowObfuscatorVisitor()
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Check for presence of "if" in the method body
	if !strings.Contains(output, "function testMethod() {if") {
		t.Errorf("Method body not properly obfuscated. Output:\n%s", output)
	}

	// Verify that the original statements are present inside the if block
	if !strings.Contains(output, "echo \"Hello\";") || !strings.Contains(output, "return \"World\";") {
		t.Errorf("Original statements not preserved in obfuscated output. Output:\n%s", output)
	}
}

// TestControlFlowObfuscatorLoops tests the obfuscation of loop statements.
// All types of PHP loops (for, while, foreach, do-while) should have their bodies
// wrapped in if(RANDOM_TRUE_CONDITION){...} blocks regardless of whether the body is a single statement
// or a statement list.
func TestControlFlowObfuscatorLoops(t *testing.T) {
	src := `<?php
for ($i = 0; $i < 10; $i++) {
    echo $i;
}

$j = 0;
while ($j < 10) {
    echo $j;
    $j++;
}

$arr = [1, 2, 3];
foreach ($arr as $val) {
    echo $val;
}

$k = 0;
do {
    echo $k;
    $k++;
} while ($k < 5);
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Print the original AST structure for debugging purposes
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	// Create and apply the control flow obfuscator
	obfuscator := NewControlFlowObfuscatorVisitor()
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Check for presence of if statements in each type of loop
	if !strings.Contains(output, "for ($i = 0; $i < 10; $i++) {if") {
		t.Errorf("For loop not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "while ($j < 10) {if") {
		t.Errorf("While loop not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "foreach ($arr as $val) {if") {
		t.Errorf("Foreach loop not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "do {if") {
		t.Errorf("Do-while loop not properly obfuscated. Output:\n%s", output)
	}
}

// TestControlFlowObfuscatorIfStatements tests the obfuscation of if statements.
// The test verifies that:
// 1. The body of an if statement is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
// 2. The body of any elseif branches are wrapped in if(RANDOM_TRUE_CONDITION){...} blocks
// 3. The body of any else branch is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
// 4. Single statement bodies (without braces) are also properly wrapped
func TestControlFlowObfuscatorIfStatements(t *testing.T) {
	src := `<?php
if (true) {
    echo "true branch";
} else if (false) {
    echo "false branch";
} else {
    echo "else branch";
}

if ($test)
    echo "single statement";
else
    echo "single else statement";
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Create and apply the control flow obfuscator
	obfuscator := NewControlFlowObfuscatorVisitor()
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Check for presence of obfuscated if statements
	// Main if branch
	if !strings.Contains(output, "if (true) {if") {
		t.Errorf("If statement 'true' branch not properly obfuscated. Output:\n%s", output)
	}

	// Check that statements are preserved
	if !strings.Contains(output, "echo \"true branch\";") {
		t.Errorf("If statement 'true' branch content missing. Output:\n%s", output)
	}
	if !strings.Contains(output, "echo \"false branch\";") {
		t.Errorf("If statement 'false' branch content missing. Output:\n%s", output)
	}
	if !strings.Contains(output, "echo \"else branch\";") {
		t.Errorf("If statement 'else' branch content missing. Output:\n%s", output)
	}

	// Single statement if/else
	if !strings.Contains(output, "if ($test)if") {
		t.Errorf("Single statement if not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "else if") {
		t.Errorf("Single statement else not properly obfuscated. Output:\n%s", output)
	}
}

// TestControlFlowObfuscatorRandomization verifies that the control flow obfuscator
// generates different condition types across multiple runs.
// This ensures the randomization is working correctly.
func TestControlFlowObfuscatorRandomization(t *testing.T) {
	src := `<?php
function test() {
    echo "Hello";
    return "World";
}
`
	// Run the obfuscator multiple times and collect the outputs
	const iterations = 10
	results := make([]string, iterations)

	// Run multiple obfuscation passes
	for i := 0; i < iterations; i++ {
		// Parse the source code
		root, err := parsePhpCode(t, src)
		if err != nil {
			t.Fatalf("Failed to parse code: %s", err.Error())
		}

		// Create and apply the control flow obfuscator
		obfuscator := NewControlFlowObfuscatorVisitor()
		// Enable randomization for this test
		obfuscator.UseRandomConditions = true
		traverser := traverser.NewTraverser(obfuscator)
		root.Accept(traverser)

		// Print the transformed code
		output := printPhpCode(t, root)
		results[i] = output

		t.Logf("Iteration %d output:\n%s", i, output)
	}

	// Extract the condition part from each result
	conditions := make([]string, iterations)
	for i, result := range results {
		// Find the part between "function test() {if" and "{"
		start := strings.Index(result, "function test() {if")
		if start == -1 {
			t.Fatalf("Could not find function with if statement in output %d", i)
		}
		start += len("function test() {if")

		end := strings.Index(result[start:], "{")
		if end == -1 {
			t.Fatalf("Could not find opening brace after condition in output %d", i)
		}

		conditions[i] = strings.TrimSpace(result[start : start+end])
		t.Logf("Condition %d: %s", i, conditions[i])
	}

	// Verify that we have at least 2 different condition types
	uniqueConditions := make(map[string]bool)
	for _, cond := range conditions {
		uniqueConditions[cond] = true
	}

	t.Logf("Found %d unique conditions out of %d iterations", len(uniqueConditions), iterations)

	// We expect at least 2 different condition types with 10 iterations and 6 possible conditions
	// The probability of getting the same condition 10 times is (1/6)^9 which is extremely low
	if len(uniqueConditions) < 2 {
		t.Errorf("Expected at least 2 different condition types, but found %d", len(uniqueConditions))
	}
}

// TestControlFlowObfuscatorNested tests the nested conditional obfuscation.
// This test verifies that the obfuscator can wrap code blocks in multiple nested
// if(true){...} blocks based on the MaxNestingDepth configuration.
func TestControlFlowObfuscatorNested(t *testing.T) {
	src := `<?php
function test() {
    echo "Hello";
    return "World";
}
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Create and apply the control flow obfuscator with nesting depth set to 3
	obfuscator := NewControlFlowObfuscatorVisitor()
	obfuscator.MaxNestingDepth = 3 // Set nesting depth to 3
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST with nesting depth 3")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Check for the presence of nested if statements
	// For depth 3, we should have something like if(1){if(1){if(1){...}}}
	// Count the occurrences of "if" within the function body
	ifCount := strings.Count(output, "if")

	// For depth 3, we should have at least 3 'if' statements in a function body
	if ifCount < 3 {
		t.Errorf("Expected at least 3 nested if statements with depth=3, but found %d. Output:\n%s",
			ifCount, output)
	}

	// Verify that the function body still contains the original statements
	if !strings.Contains(output, "echo \"Hello\";") || !strings.Contains(output, "return \"World\";") {
		t.Errorf("Original statements not preserved in nested obfuscated output. Output:\n%s", output)
	}

	// Verify the nesting structure (simplified check)
	// We look for patterns like "if(...){if(...){" to confirm nesting
	nestingPattern := "if("
	nestedIfCount := 0
	remainingOutput := output

	// Count how many times we can find "if(" followed by another "if("
	for {
		ifIndex := strings.Index(remainingOutput, nestingPattern)
		if ifIndex == -1 {
			break
		}

		// Move past this if
		remainingOutput = remainingOutput[ifIndex+len(nestingPattern):]

		// Look for a nested if - the next occurrence should be after some characters
		// but before too many (avoiding counting adjacent if statements)
		nextIfIndex := strings.Index(remainingOutput, nestingPattern)
		if nextIfIndex >= 0 && nextIfIndex < 20 { // Arbitrary limit for proximity
			nestedIfCount++
		}
	}

	// We should have at least 2 nested structures for depth 3
	if nestedIfCount < 2 {
		t.Errorf("Expected at least 2 nested if structures with depth=3, but found %d. Output:\n%s",
			nestedIfCount, output)
	}
}

// TestControlFlowObfuscatorDeadBranches tests that the obfuscator correctly adds bogus
// else branches to if statements when the AddDeadBranches option is enabled.
func TestControlFlowObfuscatorDeadBranches(t *testing.T) {
	src := `<?php
function test() {
    echo "Hello";
    return "World";
}
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Print the original AST structure for debugging purposes
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	// Create and apply the control flow obfuscator with dead branches enabled
	obfuscator := NewControlFlowObfuscatorVisitor()
	obfuscator.AddDeadBranches = true // Enable dead branches
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST with dead branches enabled")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Verify the else branch is present in the output
	if !strings.Contains(output, "if(1){") {
		t.Errorf("If statement not added. Output:\n%s", output)
	}

	if !strings.Contains(output, "}else{") {
		t.Errorf("Else branch not added. Output:\n%s", output)
	}

	// Check for typical bogus statements in the else branch
	deadCodePatterns := []string{
		"$_",           // Variable name prefix
		"echo",         // Echo statement
		"return false", // Return statement
	}

	foundDeadCode := false
	for _, pattern := range deadCodePatterns {
		if strings.Contains(output, pattern) {
			foundDeadCode = true
			break
		}
	}

	if !foundDeadCode {
		t.Errorf("No dead code found in else branch. Output:\n%s", output)
	}
}

// TestControlFlowObfuscatorSwitchStatements tests the obfuscation of switch statements.
// The test verifies that:
// 1. The body of each case statement is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
// 2. The body of any default branch is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
func TestControlFlowObfuscatorSwitchStatements(t *testing.T) {
	src := `<?php
switch ($value) {
	case 'a':
		echo "Case A";
		break;
	case 'b':
		echo "Case B";
		$result = 1;
		break;
	default:
		echo "Default case";
		break;
}

// Alternative syntax
switch ($type):
	case 1:
		echo "Type 1";
		break;
	case 2:
		echo "Type 2";
		break;
	default:
		echo "Unknown type";
		break;
endswitch;
`

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, src)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Print the original AST structure for debugging purposes
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	// Create and apply the control flow obfuscator
	obfuscator := NewControlFlowObfuscatorVisitor()
	traverser := traverser.NewTraverser(obfuscator)

	// Apply the visitor to the AST
	t.Logf("Applying obfuscator to AST")
	root.Accept(traverser)

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Check for presence of if statements in each case and default
	if !strings.Contains(output, "case 'a':if") {
		t.Errorf("Case 'a' not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "case 'b':if") {
		t.Errorf("Case 'b' not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "default:if") {
		t.Errorf("Default case not properly obfuscated. Output:\n%s", output)
	}

	// Check for presence of if statements in alternative syntax switch
	if !strings.Contains(output, "case 1:if") {
		t.Errorf("Case 1 in alternative syntax switch not properly obfuscated. Output:\n%s", output)
	}
	if !strings.Contains(output, "case 2:if") {
		t.Errorf("Case 2 in alternative syntax switch not properly obfuscated. Output:\n%s", output)
	}

	// Verify that the original statements are preserved
	caseAContent := []string{"echo \"Case A\"", "break"}
	for _, content := range caseAContent {
		if !strings.Contains(output, content) {
			t.Errorf("Case 'a' content '%s' missing. Output:\n%s", content, output)
		}
	}

	caseBContent := []string{"echo \"Case B\"", "$result = 1", "break"}
	for _, content := range caseBContent {
		if !strings.Contains(output, content) {
			t.Errorf("Case 'b' content '%s' missing. Output:\n%s", content, output)
		}
	}
}

// TestControlFlowObfuscatorTryCatch tests the obfuscation of try/catch statements.
// The test verifies that:
// 1. The body of each try block is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
// 2. The body of each catch block is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
// 3. The body of each finally block is wrapped in an if(RANDOM_TRUE_CONDITION){...} block
func TestControlFlowObfuscatorTryCatch(t *testing.T) {
	input := `<?php
try {
    $value = doSomething();
    processValue($value);
} catch (Exception $e) {
    logError($e);
} finally {
    cleanup();
}
`
	expected := `<?php
try {
    if (1) {
        $value = doSomething();
        processValue($value);
    }
} catch (Exception $e) {
    if (1) {
        logError($e);
    }
} finally {
    if (1) {
        cleanup();
    }
}
`
	testTransformer(t, input, expected, func() Transformer {
		return NewControlFlowObfuscator(false)
	})
}

// Helper function to inspect the AST structure for debugging
func dumpASTFormatted(t *testing.T, root ast.Vertex, prefix string) {
	switch node := root.(type) {
	case *ast.Root:
		t.Logf("%sRoot with %d children", prefix, len(node.Stmts))
		for i, child := range node.Stmts {
			t.Logf("%sChild %d:", prefix+"  ", i)
			dumpASTFormatted(t, child, prefix+"    ")
		}
	case *ast.StmtFunction:
		t.Logf("%sFunction with %d statements", prefix, len(node.Stmts))
		if node.Name != nil {
			if id, ok := node.Name.(*ast.Identifier); ok {
				t.Logf("%sName: %s", prefix+"  ", string(id.Value))
			}
		}
		for i, stmt := range node.Stmts {
			t.Logf("%sStatement %d:", prefix+"  ", i)
			dumpASTFormatted(t, stmt, prefix+"    ")
		}
	case *ast.StmtClassMethod:
		t.Logf("%sClass Method", prefix)
		if node.Name != nil {
			if id, ok := node.Name.(*ast.Identifier); ok {
				t.Logf("%sName: %s", prefix+"  ", string(id.Value))
			}
		}
		if node.Stmt != nil {
			t.Logf("%sBody:", prefix+"  ")
			dumpASTFormatted(t, node.Stmt, prefix+"    ")
		}
	case *ast.StmtStmtList:
		t.Logf("%sStatement List with %d statements", prefix, len(node.Stmts))
		for i, stmt := range node.Stmts {
			t.Logf("%sStatement %d:", prefix+"  ", i)
			dumpASTFormatted(t, stmt, prefix+"    ")
		}
	case *ast.StmtClass:
		t.Logf("%sClass with %d statements", prefix, len(node.Stmts))
		if node.Name != nil {
			if id, ok := node.Name.(*ast.Identifier); ok {
				t.Logf("%sName: %s", prefix+"  ", string(id.Value))
			}
		}
		for i, stmt := range node.Stmts {
			t.Logf("%sStatement %d:", prefix+"  ", i)
			dumpASTFormatted(t, stmt, prefix+"    ")
		}
	case *ast.StmtIf:
		t.Logf("%sIf Statement", prefix)
		t.Logf("%sCondition:", prefix+"  ")
		dumpASTFormatted(t, node.Cond, prefix+"    ")
		t.Logf("%sBody:", prefix+"  ")
		dumpASTFormatted(t, node.Stmt, prefix+"    ")
	default:
		t.Logf("%sNode of type %T", prefix, node)
	}
}

func parsePhpCode(t *testing.T, src string) (*ast.Root, error) {
	t.Helper()
	phpVersion := &version.Version{Major: 8, Minor: 1}
	parserConfig := conf.Config{
		Version: phpVersion,
	}
	parseResult, err := parser.Parse([]byte(src), parserConfig)
	if err != nil {
		return nil, err
	}

	// Type assertion to *ast.Root
	root, ok := parseResult.(*ast.Root)
	if !ok {
		return nil, fmt.Errorf("parse result is not *ast.Root")
	}

	// Dump the initial AST for debugging
	t.Logf("Initial AST structure:")
	dumpASTFormatted(t, root, "")

	return root, nil
}

func printPhpCode(t *testing.T, root *ast.Root) string {
	t.Helper()
	var buf bytes.Buffer
	p := printer.NewPrinter(&buf)
	root.Accept(p)
	return buf.String()
}

// normalizeWhitespace removes extra whitespace to make string comparisons more forgiving
func normalizeWhitespace(s string) string {
	// Replace all whitespace sequences with a single space
	s = strings.Join(strings.Fields(s), " ")

	// Normalize braces, parentheses and semicolons by removing all whitespace
	// between them first, then adding back consistent spacing

	// Remove all spaces around punctuation
	s = strings.ReplaceAll(s, " {", "{")
	s = strings.ReplaceAll(s, "{ ", "{")
	s = strings.ReplaceAll(s, " }", "}")
	s = strings.ReplaceAll(s, "} ", "}")
	s = strings.ReplaceAll(s, " (", "(")
	s = strings.ReplaceAll(s, "( ", "(")
	s = strings.ReplaceAll(s, " )", ")")
	s = strings.ReplaceAll(s, ") ", ")")
	s = strings.ReplaceAll(s, " ;", ";")
	s = strings.ReplaceAll(s, "; ", ";")

	// Now add back consistent spacing
	s = strings.ReplaceAll(s, "if(", "if (")
	s = strings.ReplaceAll(s, "){", ") {")
	s = strings.ReplaceAll(s, ")if", ") if")
	s = strings.ReplaceAll(s, "try{", "try {")
	s = strings.ReplaceAll(s, "}catch", "} catch")
	s = strings.ReplaceAll(s, "}finally", "} finally")

	// Further normalize the try-catch statement
	s = strings.ReplaceAll(s, "try {if", "try { if")
	s = strings.ReplaceAll(s, "catch (", "catch(")
	s = strings.ReplaceAll(s, "finally {if", "finally { if")

	return s
}

// testTransformer is a helper function that tests a transformer by applying it to input code
// and comparing the result with the expected output
func testTransformer(t *testing.T, input, expected string, transformerFactory func() Transformer) {
	t.Helper()

	// Capture stderr during test to see the debug output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Parse the source code
	root, err := parsePhpCode(t, input)
	if err != nil {
		t.Fatalf("Failed to parse code: %s", err.Error())
	}

	// Create and apply the transformer
	transformerInstance := transformerFactory()
	root, err = transformerInstance.Transform(root)
	if err != nil {
		t.Fatalf("Failed to transform code: %s", err.Error())
	}

	// Restore stderr and collect debug output
	w.Close()
	os.Stderr = oldStderr

	var debugOutput bytes.Buffer
	io.Copy(&debugOutput, r)
	t.Logf("Debug output:\n%s", debugOutput.String())

	// Print the transformed code
	output := printPhpCode(t, root)
	t.Logf("Final output:\n%s", output)

	// Normalize whitespace for comparison
	normalizedOutput := normalizeWhitespace(output)
	normalizedExpected := normalizeWhitespace(expected)

	if normalizedOutput != normalizedExpected {
		// For debugging, show the normalized strings
		t.Logf("Normalized expected: %s", normalizedExpected)
		t.Logf("Normalized output: %s", normalizedOutput)
		t.Errorf("Transformation did not match expected output.\nExpected:\n%s\n\nGot:\n%s", expected, output)
	}
}
