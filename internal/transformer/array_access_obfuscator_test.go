package transformer

import (
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/stretchr/testify/assert"
)

// TestArrayAccessCollectorVisitor_EnterNode tests the analysis phase
// where the visitor queues replacements but doesn't modify the node directly.
func TestArrayAccessCollectorVisitor_EnterNode(t *testing.T) {
	// Basic test case setup
	var replacements []*NodeReplacement // Use the exported type

	// Initialize visitor with necessary arguments
	visitor := &ArrayAccessCollectorVisitor{
		Replacements: &replacements,
		DebugMode:    false,
	}

	// Create a sample AST node for testing (e.g., a simple $a['b'])
	// Use correct field names from vkcom/php-parser/pkg/ast
	sampleNode := &ast.ExprArrayDimFetch{
		Var: &ast.ExprVariable{Name: &ast.Identifier{Value: []byte("a")}},
		Dim: &ast.ScalarString{Value: []byte("'b'")},
	}

	// Call EnterNode
	result := visitor.EnterNode(sampleNode)

	// Assertions
	assert.True(t, result, "EnterNode should return true to continue traversal")
	assert.Len(t, replacements, 1, "One replacement should be queued")

	if len(replacements) == 1 {
		rep := replacements[0]
		assert.Same(t, sampleNode, rep.Original, "Original node in replacement should match sample pointer")

		// Check if the replacement node is the expected function call
		helpCall, ok := rep.Replacement.(*ast.ExprFunctionCall)
		assert.True(t, ok, "Replacement node should be an ExprFunctionCall")
		if ok {
			funcName, nameOk := helpCall.Function.(*ast.Name)
			assert.True(t, nameOk, "Function call name should be *ast.Name")
			if nameOk && len(funcName.Parts) == 1 {
				part, partOk := funcName.Parts[0].(*ast.NamePart)
				assert.True(t, partOk, "Function name should have one NamePart")
				assert.Equal(t, "_phpmix_array_get", string(part.Value), "Function name should be _phpmix_array_get")
			}

			// Check arguments of the generated helper call
			assert.NotNil(t, helpCall.Args, "Helper call should have Args")
			args := helpCall.Args
			assert.Len(t, args, 2, "Helper call should have 2 arguments")

			// Check argument types and values
			if len(args) == 2 {
				arg0, ok0 := args[0].(*ast.Argument)
				arg1, ok1 := args[1].(*ast.Argument)
				assert.True(t, ok0 && ok1, "Arguments should be *ast.Argument type")

				if ok0 && ok1 {
					// Check if the first argument is the original variable node
					assert.Same(t, sampleNode.Var, arg0.Expr, "First arg should be original variable")
					// Check if the second argument is the original dimension node
					assert.Same(t, sampleNode.Dim, arg1.Expr, "Second arg should be original dimension")
				}
			}
		}
	}

	// TODO: Add more test cases for different scenarios:
	// - Nil dimension (should use ast.ScalarNull in replacement args)
	// - Non-variable base (e.g., function call result)
	// - L-value context (visitor should still queue replacement; skipping happens later)
}
