package transformer

import (
	"fmt"
	"log"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
	// Import Go's AST package with an alias
)

// DebugMode is a global flag to enable debug output for node replacers
var DebugMode bool

// NodeReplacement represents a node to be replaced and its replacement
type NodeReplacement struct {
	Original    ast.Vertex
	Replacement ast.Vertex
}

// ASTNodeReplacer replaces nodes in the AST based on a list of replacements using parent tracking
type ASTNodeReplacer struct {
	visitor.Null // Embed visitor.Null to satisfy visitor interface if needed
	tracker      *ParentTracker
	replacements []*NodeReplacement
	debug        bool
}

// NewASTNodeReplacer creates a new AST node replacer
func NewASTNodeReplacer(tracker *ParentTracker, debug bool) *ASTNodeReplacer {
	return &ASTNodeReplacer{
		tracker:      tracker,
		replacements: make([]*NodeReplacement, 0),
		debug:        debug,
	}
}

// NewNodeReplacer creates a new AST node replacer with a fresh parent tracker
func NewNodeReplacer() *ASTNodeReplacer {
	tracker := NewParentTracker()
	return NewASTNodeReplacer(tracker, DebugMode)
}

// AddReplacement adds a node replacement request
func (r *ASTNodeReplacer) AddReplacement(original ast.Vertex, replacement ast.Vertex) {
	if original == nil {
		log.Println("Warning: Attempted to add replacement for nil original node")
		return
	}
	if replacement == nil {
		log.Println("Warning: Attempted to add nil replacement node")
		return
	}
	// Check if we already have a replacement for this original node
	for _, rep := range r.replacements {
		if rep.Original == original {
			if r.debug {
				log.Printf("Warning: Duplicate replacement request for node %T. Overwriting.", original)
			}
			rep.Replacement = replacement // Overwrite existing request
			return
		}
	}
	r.replacements = append(r.replacements, &NodeReplacement{
		Original:    original,
		Replacement: replacement,
	})
	if r.debug {
		log.Printf("Added replacement request: %T -> %T\n", original, replacement)
	}
}

// AddNodeReplacement adds a NodeReplacement to the list of replacements
func (r *ASTNodeReplacer) AddNodeReplacement(replacement *NodeReplacement) {
	if replacement == nil {
		log.Println("Warning: Attempted to add nil NodeReplacement")
		return
	}
	if replacement.Original == nil {
		log.Println("Warning: NodeReplacement has nil Original node")
		return
	}
	if replacement.Replacement == nil {
		log.Println("Warning: NodeReplacement has nil Replacement node")
		return
	}

	// Check if we already have a replacement for this original node
	for _, rep := range r.replacements {
		if rep.Original == replacement.Original {
			if r.debug {
				log.Printf("Warning: Duplicate replacement request for node %T. Overwriting.", replacement.Original)
			}
			rep.Replacement = replacement.Replacement // Overwrite existing request
			return
		}
	}

	r.replacements = append(r.replacements, replacement)
	if r.debug {
		log.Printf("Added replacement request: %T -> %T\n", replacement.Original, replacement.Replacement)
	}
}

// Apply replaces all scheduled nodes by traversing the tree using the standard visitor traverser
func (r *ASTNodeReplacer) Apply(root ast.Vertex) {
	// The ASTNodeReplacer itself acts as the visitor for the traversal
	if r.debug {
		log.Printf("Starting node replacement with %d pending replacements", len(r.replacements))
	}
	root.Accept(r)
	if r.debug {
		log.Printf("Finished node replacement, %d replacements still pending", len(r.replacements))
	}
}

// HasReplacements returns true if this replacer has any pending replacements
func (r *ASTNodeReplacer) HasReplacements() bool {
	return len(r.replacements) > 0
}

// ApplyReplacements applies all pending replacements to the given AST
// This is an alias of Apply with a more descriptive name
func (r *ASTNodeReplacer) ApplyReplacements(root ast.Vertex) {
	r.Apply(root)
}

// LeaveNode performs the replacements after visiting a node's children.
// This ensures children are processed before the parent attempts replacement.
func (r *ASTNodeReplacer) LeaveNode(currentNode ast.Vertex) {
	if currentNode == nil {
		return
	}

	// Check if any children of this currentNode need replacement
	// Iterate safely as replaceChildNode might alter the slice implicitly via parent modification
	newReplacements := make([]*NodeReplacement, 0, len(r.replacements))
	modified := false

	if r.debug {
		log.Printf("LeaveNode for node type %T with %d pending replacements", currentNode, len(r.replacements))
	}

	for _, rep := range r.replacements {
		parent := r.tracker.GetParent(rep.Original)
		if parent == currentNode {
			if r.debug {
				log.Printf("LeaveNode: Found parent %T for replacement: %T -> %T\n", parent, rep.Original, rep.Replacement)
			}
			if r.replaceChildNode(parent, rep.Original, rep.Replacement) {
				modified = true
				// Don't add this replacement to newReplacements, it's done
				if r.debug {
					log.Printf("Replacement successfully applied: %T -> %T in parent %T", rep.Original, rep.Replacement, parent)
				}
			} else {
				// Replacement failed, keep it for potential retry or logging
				newReplacements = append(newReplacements, rep)
				if r.debug {
					log.Printf("Replacement FAILED: %T -> %T in parent %T", rep.Original, rep.Replacement, parent)
				}
			}
		} else {
			// This replacement doesn't belong to the current node, keep it
			newReplacements = append(newReplacements, rep)
			if r.debug && parent != nil {
				log.Printf("Skipping replacement for now, parent is %T (not current node %T)", parent, currentNode)
			} else if r.debug {
				log.Printf("Skipping replacement for now, NO PARENT FOUND for %T", rep.Original)
			}
		}
	}

	// Update the list of replacements
	r.replacements = newReplacements

	// If we modified children, we might need to update the parent map for the new child
	// Rebuilding the parent map after all replacements might be safer if needed.
	if modified && r.debug {
		log.Printf("Node %T had children replaced.", currentNode)
	}
}

// replaceChildNode replaces a child node within its parent.
// Returns true if replacement was successful, false otherwise.
func (r *ASTNodeReplacer) replaceChildNode(parent ast.Vertex, original ast.Vertex, replacement ast.Vertex) bool {
	success := false
	defer func() {
		if success && r.debug {
			log.Printf("Successfully replaced child %T in parent %T\n", original, parent)
		}
	}()

	switch p := parent.(type) {
	case *ast.Root:
		for i, stmt := range p.Stmts {
			if stmt == original {
				if repStmt, ok := replacement.(ast.Vertex); ok {
					p.Stmts[i] = repStmt
					success = true
				} else if r.debug {
					log.Printf("Error: Cannot replace Root statement %T with non-vertex %T\n", original, replacement)
				}
				return success
			}
		}
	case *ast.StmtExpression:
		if p.Expr == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Expr = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace StmtExpression expression %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
	case *ast.StmtEcho:
		for i, expr := range p.Exprs {
			if expr == original {
				if repExpr, ok := replacement.(ast.Vertex); ok {
					p.Exprs[i] = repExpr
					success = true
				} else if r.debug {
					log.Printf("Error: Cannot replace StmtEcho expression %T with non-vertex %T\n", original, replacement)
				}
				return success
			}
		}
	case *ast.ExprAssign:
		if p.Expr == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Expr = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprAssign expression %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
		if p.Var == original {
			if repVar, ok := replacement.(ast.Vertex); ok { // General check for LHS
				p.Var = repVar
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprAssign variable %T with %T\n", original, replacement)
			}
			return success
		}
	case *ast.ExprArrayItem:
		if p.Key == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Key = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprArrayItem key %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
		if p.Val == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Val = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprArrayItem value %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
	case *ast.Argument:
		if p.Expr == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Expr = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace Argument expression %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
	case *ast.ExprBinaryConcat:
		if p.Left == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Left = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprBinaryConcat left operand %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
		if p.Right == original {
			if repExpr, ok := replacement.(ast.Vertex); ok {
				p.Right = repExpr
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprBinaryConcat right operand %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
	case *ast.ExprFunctionCall:
		for i, arg := range p.Args {
			if arg == original {
				if repArg, ok := replacement.(*ast.Argument); ok {
					p.Args[i] = repArg
					success = true
				} else if r.debug {
					log.Printf("Error: Cannot replace function call argument %T with non-argument %T\n", original, replacement)
				}
				return success
			}
		}

	case *ast.ExprArrayDimFetch:
		if p.Var == original {
			if repVar, ok := replacement.(ast.Vertex); ok {
				p.Var = repVar
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprArrayDimFetch variable %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
		if p.Dim == original {
			if repDim, ok := replacement.(ast.Vertex); ok {
				p.Dim = repDim
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprArrayDimFetch dimension %T with non-vertex %T\n", original, replacement)
			}
			return success
		}

	case *ast.ExprPropertyFetch:
		if p.Var == original {
			if repVar, ok := replacement.(ast.Vertex); ok {
				p.Var = repVar
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprPropertyFetch variable %T with non-vertex %T\n", original, replacement)
			}
			return success
		}
		if p.Prop == original {
			if repProp, ok := replacement.(ast.Vertex); ok {
				p.Prop = repProp
				success = true
			} else if r.debug {
				log.Printf("Error: Cannot replace ExprPropertyFetch property %T with non-vertex %T\n", original, replacement)
			}
			return success
		}

	// Add more parent types as needed...

	default:
		if r.debug {
			fmt.Printf("Warning: Unhandled parent type %T in replaceChildNode. Attempting reflection fallback (not implemented).\n", parent)
			// Dump more information about the nodes
			fmt.Printf("Original node type: %T\n", original)
			fmt.Printf("Replacement node type: %T\n", replacement)
			// List the handled parent types
			fmt.Println("Handled parent types: Root, StmtExpression, StmtEcho, ExprAssign, ExprArrayItem, Argument, ExprBinaryConcat, ExprFunctionCall, ExprArrayDimFetch, ExprPropertyFetch")

			// Check if we need to add support for more types
			if arrFetch, ok := parent.(*ast.ExprArrayDimFetch); ok {
				fmt.Printf("Found ExprArrayDimFetch parent: Var=%T, Dim=%T\n", arrFetch.Var, arrFetch.Dim)
				if arrFetch.Var == original {
					fmt.Println("Original is the Var field of ExprArrayDimFetch")
					fmt.Println("Need to add support for ExprArrayDimFetch.Var replacement")
				}
				if arrFetch.Dim == original {
					fmt.Println("Original is the Dim field of ExprArrayDimFetch")
					fmt.Println("Need to add support for ExprArrayDimFetch.Dim replacement")
				}
			} else if propFetch, ok := parent.(*ast.ExprPropertyFetch); ok {
				fmt.Printf("Found ExprPropertyFetch parent: Var=%T, Prop=%T\n", propFetch.Var, propFetch.Prop)
				fmt.Println("Need to add support for ExprPropertyFetch replacement")
			}
		}
		// success = r.replaceChildViaReflection(parent, original, replacement)
	}

	if !success && r.debug {
		log.Printf("Failed to replace child %T in parent %T\n", original, parent)
	}
	return success
}

// replaceChildViaReflection uses reflection to replace a child node
// WARNING: This is complex, potentially slow, and error-prone.
func (r *ASTNodeReplacer) replaceChildViaReflection(parent ast.Vertex, original ast.Vertex, replacement ast.Vertex) bool {
	if r.debug {
		log.Println("replaceChildViaReflection is not implemented yet.")
	}
	return false
}
