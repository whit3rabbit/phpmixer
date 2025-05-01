package transformer

import (
	"fmt"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

// ArrayAccessCollectorVisitor collects array access expressions and creates replacements
// This visitor doesn't modify the AST directly but creates a list of replacements
// that will be applied later by a NodeReplacer
type ArrayAccessCollectorVisitor struct {
	visitor.Null
	Replacements   *[]*NodeReplacement
	processedNodes map[ast.Vertex]bool
	DebugMode      bool
	Count          int
}

// Make ArrayAccessCollectorVisitor implement the NodeReplacer interface
// This allows it to be used with ReplaceTraverser for more direct control

// EnterNode is called when entering a node during traversal
func (v *ArrayAccessCollectorVisitor) EnterNode(node ast.Vertex) bool {
	// Skip already processed nodes to avoid infinite recursion
	if v.processedNodes == nil {
		v.processedNodes = make(map[ast.Vertex]bool)
	}

	if v.processedNodes[node] {
		return false
	}

	// Add debug output
	if v.DebugMode {
		fmt.Printf("ArrayAccessCollector EnterNode: %T\n", node)
	}

	// Check if this is an array access expression
	if arrFetch, ok := node.(*ast.ExprArrayDimFetch); ok {
		if v.DebugMode {
			fmt.Printf("Found array access node: %T\n", node)
		}

		// Check if we have a valid array and key
		if arrFetch.Var == nil || arrFetch.Dim == nil {
			if v.DebugMode {
				fmt.Printf("Skipping invalid array access: missing var or dim\n")
			}
			return true
		}

		// More debug output
		if v.DebugMode {
			fmt.Printf("Array: %T, Key: %T\n", arrFetch.Var, arrFetch.Dim)
		}

		// Create the replacement function call
		helperCall := &ast.ExprFunctionCall{
			Function: &ast.Name{
				Parts: []ast.Vertex{&ast.NamePart{Value: []byte("_phpmix_array_get")}},
			},
			Args: []ast.Vertex{
				&ast.Argument{Expr: arrFetch.Var},
				&ast.Argument{Expr: arrFetch.Dim},
			},
		}

		// Create a replacement
		replacement := &NodeReplacement{
			Original:    arrFetch,
			Replacement: helperCall,
		}

		// Add to our collection
		if v.Replacements != nil {
			*v.Replacements = append(*v.Replacements, replacement)
			v.Count++

			if v.DebugMode {
				fmt.Printf("Collected array access replacement #%d\n", v.Count)
			}
		}

		// Mark as processed
		v.processedNodes[arrFetch] = true
	}

	return true
}

// GetReplacement returns a replacement node if available
func (v *ArrayAccessCollectorVisitor) GetReplacement(node ast.Vertex) (ast.Vertex, bool) {
	// We don't actually replace in this pass, just collect
	return nil, false
}

// LeaveNode is called when leaving a node during traversal
func (v *ArrayAccessCollectorVisitor) LeaveNode(node ast.Vertex) {
	// Nothing to do here, replacements are collected in EnterNode
}

// AddArrayAccessHelper adds the necessary helper function to the PHP AST
func AddArrayAccessHelper(root ast.Vertex) {
	if root == nil {
		return
	}

	// Check if we're dealing with a proper Root node
	rootNode, ok := root.(*ast.Root)
	if !ok {
		return
	}

	// Check if the helper function already exists
	for _, stmt := range rootNode.Stmts {
		if funcStmt, ok := stmt.(*ast.StmtFunction); ok {
			if ident, ok := funcStmt.Name.(*ast.Identifier); ok {
				if string(ident.Value) == "_phpmix_array_get" {
					// Function already exists, don't add it again
					return
				}
			}
		}
	}

	// Create a function declaration for _phpmix_array_get
	funcDecl := &ast.StmtFunction{
		Name: &ast.Identifier{
			Value: []byte("_phpmix_array_get"),
		},
		Params: []ast.Vertex{
			&ast.Parameter{ // Array parameter
				Var: &ast.ExprVariable{
					Name: &ast.Identifier{Value: []byte("arr")},
				},
			},
			&ast.Parameter{ // Key parameter
				Var: &ast.ExprVariable{
					Name: &ast.Identifier{Value: []byte("key")},
				},
			},
			&ast.Parameter{ // Default value parameter (optional)
				Var: &ast.ExprVariable{
					Name: &ast.Identifier{Value: []byte("default")},
				},
				DefaultValue: &ast.ExprConstFetch{
					Const: &ast.Name{
						Parts: []ast.Vertex{
							&ast.NamePart{Value: []byte("null")},
						},
					},
				},
			},
		},
		Stmts: []ast.Vertex{
			// We'll implement this with a simple return statement for now
			// In practice, you'd want to add the full function body with checks
			&ast.StmtReturn{
				Expr: &ast.ExprArrayDimFetch{
					Var: &ast.ExprVariable{
						Name: &ast.Identifier{Value: []byte("arr")},
					},
					Dim: &ast.ExprVariable{
						Name: &ast.Identifier{Value: []byte("key")},
					},
				},
			},
		},
	}

	// Add the function declaration to the beginning of the AST
	rootNode.Stmts = append([]ast.Vertex{funcDecl}, rootNode.Stmts...)
}
