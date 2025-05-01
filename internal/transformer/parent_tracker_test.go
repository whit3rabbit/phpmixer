package transformer

import (
	"reflect"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nodeFinderVisitor helps find a node of a specific type
type nodeFinderVisitor struct {
	visitor.Null
	findNodeType reflect.Type
	foundNode    ast.Vertex
	t            *testing.T
	allNodes     []string // Store all node types for debugging
	visitCount   int      // Count of nodes visited
}

// Check if a node matches the target type
func (v *nodeFinderVisitor) checkNode(node ast.Vertex) {
	v.visitCount++

	if v.foundNode != nil {
		return // Already found
	}

	if node != nil {
		nodeType := reflect.TypeOf(node).String()
		v.allNodes = append(v.allNodes, nodeType)

		// Check if this is the type we're looking for
		if reflect.TypeOf(node) == v.findNodeType {
			v.foundNode = node
		}
	}
}

// Now implement the specific visitor methods to find nodes
func (v *nodeFinderVisitor) Root(n *ast.Root) {
	v.checkNode(n)

	// Continue traversal if we haven't found the node yet
	if v.foundNode == nil {
		for _, stmt := range n.Stmts {
			stmt.Accept(v)
		}
	}
}

func (v *nodeFinderVisitor) StmtEcho(n *ast.StmtEcho) {
	v.checkNode(n)

	// Continue traversal if we haven't found the node yet
	if v.foundNode == nil {
		for _, expr := range n.Exprs {
			expr.Accept(v)
		}
	}
}

func (v *nodeFinderVisitor) StmtExpression(n *ast.StmtExpression) {
	v.checkNode(n)

	// Continue traversal if we haven't found the node yet
	if v.foundNode == nil && n.Expr != nil {
		n.Expr.Accept(v)
	}
}

func (v *nodeFinderVisitor) ExprAssign(n *ast.ExprAssign) {
	v.checkNode(n)

	if v.foundNode == nil {
		if n.Var != nil {
			n.Var.Accept(v)
		}

		if v.foundNode == nil && n.Expr != nil {
			n.Expr.Accept(v)
		}
	}
}

func (v *nodeFinderVisitor) ScalarString(n *ast.ScalarString) {
	v.checkNode(n)
}

func (v *nodeFinderVisitor) ScalarEncapsed(n *ast.ScalarEncapsed) {
	v.checkNode(n)

	if v.foundNode == nil {
		for _, part := range n.Parts {
			part.Accept(v)
			if v.foundNode != nil {
				break
			}
		}
	}
}

func (v *nodeFinderVisitor) ExprVariable(n *ast.ExprVariable) {
	v.checkNode(n)

	if v.foundNode == nil && n.Name != nil {
		n.Name.Accept(v)
	}
}

func (v *nodeFinderVisitor) Identifier(n *ast.Identifier) {
	v.checkNode(n)
}

func (v *nodeFinderVisitor) Argument(n *ast.Argument) {
	v.checkNode(n)

	if v.foundNode == nil && n.Expr != nil {
		n.Expr.Accept(v)
	}
}

func (v *nodeFinderVisitor) ExprFunctionCall(n *ast.ExprFunctionCall) {
	v.checkNode(n)

	if v.foundNode == nil {
		if n.Function != nil {
			n.Function.Accept(v)
		}

		if v.foundNode == nil {
			for _, arg := range n.Args {
				arg.Accept(v)
				if v.foundNode != nil {
					break
				}
			}
		}
	}
}

func (v *nodeFinderVisitor) ExprArrayDimFetch(n *ast.ExprArrayDimFetch) {
	v.checkNode(n)

	if v.foundNode == nil {
		if n.Var != nil {
			n.Var.Accept(v)
		}

		if v.foundNode == nil && n.Dim != nil {
			n.Dim.Accept(v)
		}
	}
}

func (v *nodeFinderVisitor) ScalarEncapsedStringPart(n *ast.ScalarEncapsedStringPart) {
	v.checkNode(n)
}

func (v *nodeFinderVisitor) ScalarEncapsedStringBrackets(n *ast.ScalarEncapsedStringBrackets) {
	v.checkNode(n)

	if v.foundNode == nil && n.Var != nil {
		n.Var.Accept(v)
	}
}

func (v *nodeFinderVisitor) ExprBinaryConcat(n *ast.ExprBinaryConcat) {
	v.checkNode(n)

	if v.foundNode == nil {
		if n.Left != nil {
			n.Left.Accept(v)
		}

		if v.foundNode == nil && n.Right != nil {
			n.Right.Accept(v)
		}
	}
}

// Add a debug visitor for printing node types
type debugVisitor struct {
	visitor.Null
	t *testing.T
}

func (v *debugVisitor) EnterNode(n ast.Vertex) bool {
	if n != nil {
		v.t.Logf("DEBUG: Found node of type %T", n)
	}
	return true
}

func TestParentTracker(t *testing.T) {
	// Test cases
	testCases := []struct {
		name             string
		code             string
		findNodeType     reflect.Type
		expectParentType reflect.Type
	}{
		{
			name:             "Echo with encapsed string",
			code:             `<?php echo "Hello {$name}"; ?>`,
			findNodeType:     reflect.TypeOf((*ast.ScalarEncapsed)(nil)),
			expectParentType: reflect.TypeOf((*ast.StmtEcho)(nil)),
		},
		{
			name:             "Assignment with string",
			code:             `<?php $a = "Hello"; ?>`,
			findNodeType:     reflect.TypeOf((*ast.ScalarString)(nil)),
			expectParentType: reflect.TypeOf((*ast.ExprAssign)(nil)),
		},
		{
			name:             "Function call with string argument",
			code:             `<?php foo("bar"); ?>`,
			findNodeType:     reflect.TypeOf((*ast.ScalarString)(nil)),
			expectParentType: reflect.TypeOf((*ast.Argument)(nil)),
		},
		{
			name:             "Variable inside encapsed string",
			code:             `<?php echo "User: $username"; ?>`,
			findNodeType:     reflect.TypeOf((*ast.ExprVariable)(nil)),
			expectParentType: reflect.TypeOf((*ast.ScalarEncapsed)(nil)),
		},
		{
			name:             "Nested expression in encapsed string",
			code:             `<?php echo "Result: {$data[0]}"; ?>`,
			findNodeType:     reflect.TypeOf((*ast.ExprArrayDimFetch)(nil)),
			expectParentType: reflect.TypeOf((*ast.ScalarEncapsedStringBrackets)(nil)),
		},
		{
			name:             "Root node parent",
			code:             `<?php echo "test"; ?>`,
			findNodeType:     reflect.TypeOf((*ast.StmtEcho)(nil)),
			expectParentType: reflect.TypeOf((*ast.Root)(nil)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the PHP code
			rootNode, err := parser.Parse([]byte(tc.code), conf.Config{})
			require.NoError(t, err, "Parser error")
			require.NotNil(t, rootNode, "Root node should not be nil")

			// Build parent map
			parentTracker := BuildParentMap(rootNode, true)

			// Find target node using our custom visitor
			finder := &nodeFinderVisitor{
				findNodeType: tc.findNodeType,
				t:            t,
				allNodes:     make([]string, 0),
			}
			rootNode.Accept(finder)

			// Debug: Print all node types we found
			t.Logf("Looking for node type: %v", tc.findNodeType)
			t.Logf("Visited %d nodes during traversal", finder.visitCount)
			t.Logf("Found %d node types during traversal", len(finder.allNodes))
			if len(finder.allNodes) > 0 {
				t.Logf("All node types found:")
				for _, nodeType := range finder.allNodes {
					t.Logf("  - %s", nodeType)
				}
			}

			// Verify we found the target node
			targetNode := finder.foundNode
			require.NotNil(t, targetNode, "Failed to find target node of type %v", tc.findNodeType)

			// Verify parent relationship
			parent := parentTracker.GetParent(targetNode)
			require.NotNil(t, parent, "Parent of target node (%v) should not be nil", tc.findNodeType)

			// Verify parent type
			actualParentType := reflect.TypeOf(parent)
			assert.Equal(t, tc.expectParentType, actualParentType,
				"Parent of %v should be %v, got %v", tc.findNodeType, tc.expectParentType, actualParentType)
		})
	}
}
