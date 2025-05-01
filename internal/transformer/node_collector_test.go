package transformer

import (
	"fmt"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/version"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// NodeCollectorVisitor collects all unique node types during traversal
type NodeCollectorVisitor struct {
	visitor.Null
	FoundNodes map[string]bool
}

func NewNodeCollectorVisitor() *NodeCollectorVisitor {
	return &NodeCollectorVisitor{
		FoundNodes: make(map[string]bool),
	}
}

// Helper method to record a node type
func (v *NodeCollectorVisitor) recordNode(n ast.Vertex) {
	if n == nil {
		return
	}
	typeName := fmt.Sprintf("%T", n)
	v.FoundNodes[typeName] = true
}

// Root nodes
func (v *NodeCollectorVisitor) Root(n *ast.Root) {
	v.recordNode(n)
}

// Statement nodes
func (v *NodeCollectorVisitor) StmtEcho(n *ast.StmtEcho) {
	v.recordNode(n)
}

// Expression nodes
func (v *NodeCollectorVisitor) ExprVariable(n *ast.ExprVariable) {
	v.recordNode(n)
}

// Scalar nodes
func (v *NodeCollectorVisitor) ScalarString(n *ast.ScalarString) {
	v.recordNode(n)
}

func (v *NodeCollectorVisitor) ScalarEncapsed(n *ast.ScalarEncapsed) {
	v.recordNode(n)
}

func (v *NodeCollectorVisitor) ScalarEncapsedStringPart(n *ast.ScalarEncapsedStringPart) {
	v.recordNode(n)
}

func (v *NodeCollectorVisitor) ScalarEncapsedStringBrackets(n *ast.ScalarEncapsedStringBrackets) {
	v.recordNode(n)
}

// Identifier nodes
func (v *NodeCollectorVisitor) Identifier(n *ast.Identifier) {
	v.recordNode(n)
}

// Array access nodes
func (v *NodeCollectorVisitor) ExprArrayDimFetch(n *ast.ExprArrayDimFetch) {
	v.recordNode(n)
}

func TestPhpParserFindsNodes(t *testing.T) {
	// Test code with a mix of language features to hit different node types
	code := `<?php 
	echo "Hello {$name}"; 
	$arr["key"] = "value";
	?>`

	// Parse the PHP code with a specific PHP version
	phpVersion := &version.Version{Major: 8, Minor: 1}
	parserConfig := conf.Config{
		Version: phpVersion,
	}
	rootNode, err := parser.Parse([]byte(code), parserConfig)
	require.NoError(t, err, "Parser error")
	require.NotNil(t, rootNode, "Root node should not be nil")

	// Use our visitor to collect node types
	visitor := NewNodeCollectorVisitor()
	traverser := traverser.NewTraverser(visitor)
	rootNode.Accept(traverser)

	// Print all the node types we found
	nodeCount := len(visitor.FoundNodes)
	t.Logf("Found %d unique node types:", nodeCount)
	for typeName := range visitor.FoundNodes {
		t.Logf("  - %s", typeName)
	}

	// Check that we found at least some nodes
	assert.Greater(t, nodeCount, 0, "Should find at least some nodes")

	// Check for specific node types we're interested in
	expectedNodes := []string{
		"*ast.Root",
		"*ast.StmtEcho",
		"*ast.ScalarEncapsed",
		"*ast.ScalarEncapsedStringPart",
		"*ast.ScalarEncapsedStringBrackets",
		"*ast.ExprVariable",
		"*ast.Identifier",
		"*ast.ScalarString",
		"*ast.ExprArrayDimFetch",
	}

	// List which ones we found and which ones we didn't
	for _, typeName := range expectedNodes {
		if visitor.FoundNodes[typeName] {
			t.Logf("FOUND: %s", typeName)
		} else {
			t.Logf("NOT FOUND: %s", typeName)
		}
	}

	// Print all available interfaces in the AST package to help debugging
	t.Log("Available interfaces in PHP parser's AST package:")
	t.Log("- ast.Vertex (the base interface for all nodes)")
	t.Logf("Root type: %T", rootNode)
}
