package transformer

import (
	"fmt"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/whit3rabbit/phpmixer/internal/config"
)

// Helper to check if a node is an obfuscated function call
func isObfuscatedFunctionCall(n ast.Vertex) bool {
	funcCall, ok := n.(*ast.ExprFunctionCall)
	if !ok {
		return false
	}

	name, ok := funcCall.Function.(*ast.Name)
	if !ok || len(name.Parts) == 0 {
		return false
	}

	part, ok := name.Parts[0].(*ast.NamePart)
	if !ok {
		return false
	}

	funcName := string(part.Value)
	return funcName == "base64_decode" || funcName == "str_rot13" || funcName == "_obfuscated_xor_decode"
}

// Helper to check if a node is a string concatenation
func isConcatenation(n ast.Vertex) bool {
	_, ok := n.(*ast.ExprBinaryConcat)
	return ok
}

// Helper to check if a node is a variable
func isVariable(n ast.Vertex) bool {
	_, ok := n.(*ast.ExprVariable)
	return ok
}

// Test that encapsed strings are properly handled by the StringObfuscatorVisitor
func TestEncapsedStringObfuscation(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		technique      string
		expectedResult func(node ast.Vertex) bool
	}{
		{
			name:      "Simple variable string",
			code:      `<?php echo "Hello $name";`,
			technique: config.StringObfuscationTechniqueBase64,
			// Should be transformed into a concatenation with an obfuscated part and a variable
			expectedResult: func(node ast.Vertex) bool {
				// The echo statement should contain a concatenation
				echo, ok := node.(*ast.Root)
				if !ok || len(echo.Stmts) == 0 {
					return false
				}

				stmt, ok := echo.Stmts[0].(*ast.StmtEcho)
				if !ok || len(stmt.Exprs) == 0 {
					return false
				}

				expr := stmt.Exprs[0]
				concat, ok := expr.(*ast.ExprBinaryConcat)
				if !ok {
					return false
				}

				// Left side should be an obfuscated call, right side should be a variable
				return isObfuscatedFunctionCall(concat.Left) && isVariable(concat.Right)
			},
		},
		{
			name:      "Complex variable string",
			code:      `<?php echo "User: {$user->name}, Age: {$user->age}";`,
			technique: config.StringObfuscationTechniqueBase64,
			// Should be transformed into a complex concatenation
			expectedResult: func(node ast.Vertex) bool {
				// Should be a root with an echo statement
				root, ok := node.(*ast.Root)
				if !ok || len(root.Stmts) == 0 {
					return false
				}

				stmt, ok := root.Stmts[0].(*ast.StmtEcho)
				if !ok || len(stmt.Exprs) == 0 {
					return false
				}

				// The echo expression should be a concatenation
				return isConcatenation(stmt.Exprs[0])
			},
		},
		{
			name:      "String with no variables",
			code:      `<?php echo "Hello World";`,
			technique: config.StringObfuscationTechniqueBase64,
			// Should be transformed into a simple obfuscated call
			expectedResult: func(node ast.Vertex) bool {
				// The echo statement should contain an obfuscated function call
				root, ok := node.(*ast.Root)
				if !ok || len(root.Stmts) == 0 {
					return false
				}

				stmt, ok := root.Stmts[0].(*ast.StmtEcho)
				if !ok || len(stmt.Exprs) == 0 {
					return false
				}

				return isObfuscatedFunctionCall(stmt.Exprs[0])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the PHP code
			rootNode, err := parser.Parse([]byte(tt.code), conf.Config{})
			require.NoError(t, err, "Failed to parse PHP code")
			require.NotNil(t, rootNode, "Root node should not be nil")

			// Print the original AST for debugging
			t.Log("Original AST:")
			root, ok := rootNode.(*ast.Root)
			if ok {
				dumpEncapsedAST(t, root, 0)
			}

			// Create and apply the visitor
			visitor := NewStringObfuscatorVisitor(tt.technique)
			visitor.DebugMode = true

			// Define a recursive function to process the AST
			var processNode func(ast.Vertex) ast.Vertex
			processNode = func(node ast.Vertex) ast.Vertex {
				switch n := node.(type) {
				case *ast.Root:
					for i, stmt := range n.Stmts {
						n.Stmts[i] = processNode(stmt)
					}
				case *ast.StmtEcho:
					for i, expr := range n.Exprs {
						n.Exprs[i] = processNode(expr)
					}
				case *ast.ScalarEncapsed:
					processed := visitor.processEncapsedString(n)
					return processed
				case *ast.ScalarString:
					return visitor.createObfuscatedCallNode(n.Value)
				}
				return node
			}

			// Process the AST
			rootNode = processNode(rootNode)

			// Print the transformed AST for debugging
			t.Log("Transformed AST:")
			if root, ok := rootNode.(*ast.Root); ok {
				dumpEncapsedAST(t, root, 0)
			}

			// Verify the transformation
			assert.True(t, tt.expectedResult(rootNode),
				fmt.Sprintf("Expected transformation not found for test case: %s", tt.name))
		})
	}
}

// Helper function to dump the AST for debugging
func dumpEncapsedAST(t *testing.T, node ast.Vertex, level int) {
	if node == nil {
		return
	}

	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}

	t.Logf("%s%T", indent, node)

	switch n := node.(type) {
	case *ast.Root:
		for _, stmt := range n.Stmts {
			dumpEncapsedAST(t, stmt, level+1)
		}
	case *ast.StmtEcho:
		for _, expr := range n.Exprs {
			dumpEncapsedAST(t, expr, level+1)
		}
	case *ast.ExprBinaryConcat:
		t.Logf("%s  Left:", indent)
		dumpEncapsedAST(t, n.Left, level+2)
		t.Logf("%s  Right:", indent)
		dumpEncapsedAST(t, n.Right, level+2)
	case *ast.ExprFunctionCall:
		if name, ok := n.Function.(*ast.Name); ok && len(name.Parts) > 0 {
			if part, ok := name.Parts[0].(*ast.NamePart); ok {
				t.Logf("%s  Function: %s", indent, string(part.Value))
			}
		}
		for i, arg := range n.Args {
			t.Logf("%s  Arg %d:", indent, i)
			dumpEncapsedAST(t, arg, level+2)
		}
	case *ast.Argument:
		dumpEncapsedAST(t, n.Expr, level+1)
	case *ast.ScalarString:
		t.Logf("%s  Value: %s", indent, string(n.Value))
	case *ast.ScalarEncapsed:
		t.Logf("%s  Parts:", indent)
		for i, part := range n.Parts {
			t.Logf("%s    Part %d:", indent, i)
			dumpEncapsedAST(t, part, level+3)
		}
	case *ast.ScalarEncapsedStringPart:
		t.Logf("%s  Value: %s", indent, string(n.Value))
	case *ast.ExprVariable:
		if id, ok := n.Name.(*ast.Identifier); ok {
			t.Logf("%s  Variable: $%s", indent, string(id.Value))
		}
	}
}

