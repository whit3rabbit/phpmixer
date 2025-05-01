package transformer

import (
	"log" // Added for debug logging
	"reflect"
	"sync"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

// ParentTracker maintains a mapping of child nodes to their parent nodes
type ParentTracker struct {
	parentMap map[ast.Vertex]ast.Vertex
	mu        sync.RWMutex // For thread safety
}

// NewParentTracker creates a new parent tracker
func NewParentTracker() *ParentTracker {
	return &ParentTracker{
		parentMap: make(map[ast.Vertex]ast.Vertex),
	}
}

// SetParent records the parent-child relationship
func (pt *ParentTracker) SetParent(child ast.Vertex, parent ast.Vertex) {
	if child == nil || parent == nil {
		return
	}
	pt.mu.Lock()
	defer pt.mu.Unlock()
	pt.parentMap[child] = parent
}

// GetParent returns the parent of a node, or nil if not found
func (pt *ParentTracker) GetParent(child ast.Vertex) ast.Vertex {
	if child == nil {
		return nil
	}
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	parent, exists := pt.parentMap[child]
	if !exists {
		return nil
	}
	return parent
}

// FindAncestorOfType traverses up the hierarchy until it finds a node of the specified type
func (pt *ParentTracker) FindAncestorOfType(node ast.Vertex, targetType reflect.Type) ast.Vertex {
	current := node
	for current != nil {
		// Handle potential pointer types from reflection
		nodeType := reflect.TypeOf(current)
		if nodeType.Kind() == reflect.Ptr {
			nodeType = nodeType.Elem()
		}
		if nodeType == targetType {
			return current
		}
		current = pt.GetParent(current)
	}
	return nil
}

// IsAncestor checks if potentialAncestor is an ancestor of node
func (pt *ParentTracker) IsAncestor(node ast.Vertex, potentialAncestor ast.Vertex) bool {
	current := node
	for current != nil {
		if current == potentialAncestor {
			return true
		}
		current = pt.GetParent(current)
	}
	return false
}

// ParentTrackerVisitor is a visitor that builds a parent-child relationship map
type ParentTrackerVisitor struct {
	visitor.Null // Embed Null visitor for default implementations
	tracker      *ParentTracker
	parents      []ast.Vertex // Stack to keep track of parents
	debug        bool
}

// NewParentTrackerVisitor creates a new visitor for tracking parent-child relationships
func NewParentTrackerVisitor(tracker *ParentTracker, debug bool) *ParentTrackerVisitor {
	return &ParentTrackerVisitor{
		tracker: tracker,
		parents: make([]ast.Vertex, 0),
		debug:   debug,
	}
}

// getCurrentParent gets the current parent from the stack
func (v *ParentTrackerVisitor) getCurrentParent() ast.Vertex {
	if len(v.parents) == 0 {
		return nil
	}
	return v.parents[len(v.parents)-1]
}

// pushParent adds a new parent to the stack
func (v *ParentTrackerVisitor) pushParent(node ast.Vertex) {
	if node == nil {
		return
	}

	// Set parent relationship
	parent := v.getCurrentParent()
	if parent != nil {
		v.tracker.SetParent(node, parent)
		if v.debug {
			log.Printf("Parent: %T -> Child: %T\n", parent, node)
		}
	}

	// Push current node as new parent
	v.parents = append(v.parents, node)
}

// popParent removes the current parent from the stack
func (v *ParentTrackerVisitor) popParent() {
	if len(v.parents) > 0 {
		v.parents = v.parents[:len(v.parents)-1]
	}
}

// Now implement all the specific visitor methods to track parents
// Root tracks the root node
func (v *ParentTrackerVisitor) Root(n *ast.Root) {
	v.pushParent(n)

	// Visit children
	for _, stmt := range n.Stmts {
		stmt.Accept(v)
	}

	v.popParent()
}

// StmtEcho handles echo statements
func (v *ParentTrackerVisitor) StmtEcho(n *ast.StmtEcho) {
	v.pushParent(n)

	// Visit expressions
	for _, expr := range n.Exprs {
		expr.Accept(v)
	}

	v.popParent()
}

// StmtExpression handles expression statements
func (v *ParentTrackerVisitor) StmtExpression(n *ast.StmtExpression) {
	v.pushParent(n)

	if n.Expr != nil {
		n.Expr.Accept(v)
	}

	v.popParent()
}

// ExprAssign handles assignments
func (v *ParentTrackerVisitor) ExprAssign(n *ast.ExprAssign) {
	v.pushParent(n)

	if n.Var != nil {
		n.Var.Accept(v)
	}

	if n.Expr != nil {
		n.Expr.Accept(v)
	}

	v.popParent()
}

// ScalarString handles string literals
func (v *ParentTrackerVisitor) ScalarString(n *ast.ScalarString) {
	v.pushParent(n)
	// No children to visit
	v.popParent()
}

// ScalarEncapsed handles string interpolation
func (v *ParentTrackerVisitor) ScalarEncapsed(n *ast.ScalarEncapsed) {
	v.pushParent(n)

	// Visit all parts
	for _, part := range n.Parts {
		part.Accept(v)
	}

	v.popParent()
}

// ExprVariable handles variables
func (v *ParentTrackerVisitor) ExprVariable(n *ast.ExprVariable) {
	v.pushParent(n)

	if n.Name != nil {
		n.Name.Accept(v)
	}

	v.popParent()
}

// Identifier handles identifiers
func (v *ParentTrackerVisitor) Identifier(n *ast.Identifier) {
	v.pushParent(n)
	// No children to visit
	v.popParent()
}

// Argument handles function call arguments
func (v *ParentTrackerVisitor) Argument(n *ast.Argument) {
	v.pushParent(n)

	if n.Expr != nil {
		n.Expr.Accept(v)
	}

	v.popParent()
}

// ExprFunctionCall handles function calls
func (v *ParentTrackerVisitor) ExprFunctionCall(n *ast.ExprFunctionCall) {
	v.pushParent(n)

	if n.Function != nil {
		n.Function.Accept(v)
	}

	for _, arg := range n.Args {
		arg.Accept(v)
	}

	v.popParent()
}

// ExprArrayDimFetch handles array access
func (v *ParentTrackerVisitor) ExprArrayDimFetch(n *ast.ExprArrayDimFetch) {
	v.pushParent(n)

	if n.Var != nil {
		n.Var.Accept(v)
	}

	if n.Dim != nil {
		n.Dim.Accept(v)
	}

	v.popParent()
}

// ScalarEncapsedStringPart handles parts of encapsed strings
func (v *ParentTrackerVisitor) ScalarEncapsedStringPart(n *ast.ScalarEncapsedStringPart) {
	v.pushParent(n)
	// No children to visit
	v.popParent()
}

// ScalarEncapsedStringBrackets handles bracketed expressions in encapsed strings
func (v *ParentTrackerVisitor) ScalarEncapsedStringBrackets(n *ast.ScalarEncapsedStringBrackets) {
	v.pushParent(n)

	if n.Var != nil {
		n.Var.Accept(v)
	}

	v.popParent()
}

// ExprBinaryConcat handles string concatenation
func (v *ParentTrackerVisitor) ExprBinaryConcat(n *ast.ExprBinaryConcat) {
	v.pushParent(n)

	if n.Left != nil {
		n.Left.Accept(v)
	}

	if n.Right != nil {
		n.Right.Accept(v)
	}

	v.popParent()
}

// GetVisitor returns a new visitor for tracking parent-child relationships
func (pt *ParentTracker) GetVisitor() *ParentTrackerVisitor {
	return NewParentTrackerVisitor(pt, DebugMode)
}

// BuildParentMap creates a new ParentTracker and builds a parent map for the given AST
func BuildParentMap(root ast.Vertex, debug bool) *ParentTracker {
	tracker := NewParentTracker()
	visitor := NewParentTrackerVisitor(tracker, debug)
	root.Accept(visitor)
	return tracker
}
