package transformer

import (
	"fmt"

	"github.com/VKCOM/php-parser/pkg/ast"
)

// NodeReplacer is an interface for visitors that can replace nodes during traversal
type NodeReplacer interface {
	EnterNode(n ast.Vertex) bool
	GetReplacement(n ast.Vertex) (ast.Vertex, bool)
	LeaveNode(n ast.Vertex)
}

// ReplaceTraverser is a custom traverser that supports node replacement
type ReplaceTraverser struct {
	visitor NodeReplacer
	debug   bool
}

// NewReplaceTraverser creates a new traverser with node replacement support
func NewReplaceTraverser(v NodeReplacer, debug bool) *ReplaceTraverser {
	return &ReplaceTraverser{
		visitor: v,
		debug:   debug,
	}
}

// Traverse starts traversal of the AST with node replacement support
func (t *ReplaceTraverser) Traverse(node ast.Vertex) ast.Vertex {
	if node == nil {
		return nil
	}

	if t.debug {
		fmt.Printf("DEBUG: Traversing node type: %T\n", node)
	}

	// Enter node and decide whether to traverse children
	traverseChildren := t.visitor.EnterNode(node)

	// Check if the node should be replaced directly
	if replacement, shouldReplace := t.visitor.GetReplacement(node); shouldReplace {
		if t.debug {
			fmt.Printf("DEBUG: Replacing node %T with %T\n", node, replacement)
		}
		return replacement
	}

	// If not traversing children, just return the original node
	if !traverseChildren {
		return node
	}

	// Based on node type, traverse children
	// Focus on the most important node types for string literals
	switch n := node.(type) {
	case *ast.Root:
		// Process the child nodes
		if n.Stmts != nil {
			for i, stmt := range n.Stmts {
				n.Stmts[i] = t.Traverse(stmt)
			}
		}
	case *ast.StmtStmtList:
		// Process each statement
		for i, stmt := range n.Stmts {
			n.Stmts[i] = t.Traverse(stmt)
		}
	case *ast.StmtEcho:
		// Process each expression
		for i, expr := range n.Exprs {
			n.Exprs[i] = t.Traverse(expr)
		}
	case *ast.StmtExpression:
		// Process the expression
		if n.Expr != nil {
			n.Expr = t.Traverse(n.Expr)
		}
	case *ast.ExprAssign:
		// Process variable and expression
		if n.Var != nil {
			n.Var = t.Traverse(n.Var)
		}
		if n.Expr != nil {
			n.Expr = t.Traverse(n.Expr)
		}
	case *ast.ExprArray:
		// Process array items
		for i, item := range n.Items {
			n.Items[i] = t.Traverse(item)
		}
	case *ast.ExprArrayItem:
		// Process key and value
		if n.Key != nil {
			n.Key = t.Traverse(n.Key)
		}
		if n.Val != nil {
			n.Val = t.Traverse(n.Val)
		}
	case *ast.ExprFunctionCall:
		// Process function and arguments
		if n.Function != nil {
			n.Function = t.Traverse(n.Function)
		}
		for i, arg := range n.Args {
			n.Args[i] = t.Traverse(arg)
		}
	case *ast.Argument:
		// Process expression
		if n.Expr != nil {
			n.Expr = t.Traverse(n.Expr)
		}
	case *ast.ExprVariable:
		// Process variable name
		if n.Name != nil {
			n.Name = t.Traverse(n.Name)
		}
	case *ast.ScalarString:
		// Leaf node - no children to process
		// This is the primary target for string obfuscation
	case *ast.ScalarEncapsed:
		// Process the expression parts within the encapsed string
		for i, part := range n.Parts {
			n.Parts[i] = t.Traverse(part)
		}
	case *ast.ExprBinaryConcat:
		// Process left and right operands of concatenation
		if n.Left != nil {
			n.Left = t.Traverse(n.Left)
		}
		if n.Right != nil {
			n.Right = t.Traverse(n.Right)
		}
	default:
		if t.debug {
			fmt.Printf("DEBUG: Unhandled node type: %T\n", node)
		}
	}

	// Leave node
	t.visitor.LeaveNode(node)

	// Return the same node or a replacement
	if replacement, shouldReplace := t.visitor.GetReplacement(node); shouldReplace {
		if t.debug {
			fmt.Printf("DEBUG: Replacing node %T with %T after traversal\n", node, replacement)
		}
		return replacement
	}

	return node
}
