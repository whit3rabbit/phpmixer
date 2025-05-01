package transformer

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/stretchr/testify/assert"
)

// TestArithmeticObfuscatorVisitor tests the arithmetic expression obfuscation
func TestArithmeticObfuscatorVisitor(t *testing.T) {
	testCases := []struct {
		input    string
		hasNodes bool // Whether we expect to find nodes to obfuscate
	}{
		{
			input:    "<?php $x = 1 + 2;",
			hasNodes: true,
		},
		{
			input:    "<?php $x = 5 - 3;",
			hasNodes: true,
		},
		{
			input:    "<?php $x = 4 * 2;",
			hasNodes: true,
		},
		{
			input:    "<?php $x = 8 / 2;",
			hasNodes: true,
		},
		{
			input:    "<?php $x = 'string';", // No arithmetic expressions
			hasNodes: false,
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("TestCase-%d", i), func(t *testing.T) {
			// Parse PHP code to AST
			src := []byte(tc.input)
			rootNode, err := parser.Parse(src, conf.Config{})
			if err != nil {
				t.Fatalf("Error parsing PHP code: %v", err)
			}

			// Create the visitor with a fixed seed for reproducible tests
			visitor := NewArithmeticObfuscatorVisitor()
			visitor.random = rand.New(rand.NewSource(42)) // Fixed seed for deterministic output
			visitor.DebugMode = true

			// Create the traverser
			traverser := NewReplaceTraverser(visitor, true)

			// Traverse the AST with the visitor
			rootNode = traverser.Traverse(rootNode)

			// Convert the AST to a simple string representation for debugging
			transformedCode := simpleAstToString(rootNode)
			t.Logf("Transformed code: %s", transformedCode)

			// Verify there are (or are not) transformations
			if tc.hasNodes {
				assert.NotEmpty(t, visitor.replacements, "Expected replacements but none were found")
			} else {
				assert.Empty(t, visitor.replacements, "Found unexpected replacements")
			}
		})
	}
}

// simpleAstToString converts an AST to a simplified string representation
// This is just for testing purposes
func simpleAstToString(node ast.Vertex) string {
	if node == nil {
		return ""
	}

	var result strings.Builder

	switch n := node.(type) {
	case *ast.Root:
		result.WriteString("<?php ")
		for _, stmt := range n.Stmts {
			result.WriteString(simpleAstToString(stmt))
		}
	case *ast.StmtExpression:
		result.WriteString(simpleAstToString(n.Expr))
		result.WriteString(";")
	case *ast.ExprAssign:
		result.WriteString(simpleAstToString(n.Var))
		result.WriteString(" = ")
		result.WriteString(simpleAstToString(n.Expr))
	case *ast.ExprVariable:
		result.WriteString("$")
		result.WriteString(simpleAstToString(n.Name))
	case *ast.Identifier:
		result.WriteString(string(n.Value))
	case *ast.ScalarLnumber:
		result.WriteString(string(n.Value))
	case *ast.ScalarString:
		result.WriteString("'" + string(n.Value) + "'")
	case *ast.ExprBinaryPlus:
		result.WriteString(simpleAstToString(n.Left))
		result.WriteString(" + ")
		result.WriteString(simpleAstToString(n.Right))
	case *ast.ExprBinaryMinus:
		result.WriteString(simpleAstToString(n.Left))
		result.WriteString(" - ")
		result.WriteString(simpleAstToString(n.Right))
	case *ast.ExprBinaryMul:
		result.WriteString(simpleAstToString(n.Left))
		result.WriteString(" * ")
		result.WriteString(simpleAstToString(n.Right))
	case *ast.ExprBinaryDiv:
		result.WriteString(simpleAstToString(n.Left))
		result.WriteString(" / ")
		result.WriteString(simpleAstToString(n.Right))
	}

	return result.String()
}
