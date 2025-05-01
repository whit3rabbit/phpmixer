package transformer

import (
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
	"github.com/stretchr/testify/require"
)

// DebugVisitor is a visitor that prints node information
type DebugVisitor struct {
	visitor.Null
	t         *testing.T
	nodeCount int
}

func (v *DebugVisitor) EnterNode(n ast.Vertex) bool {
	v.nodeCount++
	if n == nil {
		return false
	}

	v.t.Logf("Node %d: %T", v.nodeCount, n)

	switch node := n.(type) {
	case *ast.ScalarString:
		v.t.Logf("  ScalarString Value: '%s'", string(node.Value))
	case *ast.ScalarEncapsed:
		v.t.Logf("  ScalarEncapsed with %d parts", len(node.Parts))
		for i, part := range node.Parts {
			v.t.Logf("    Part %d: %T", i, part)
			if stringPart, ok := part.(*ast.ScalarEncapsedStringPart); ok {
				v.t.Logf("      Text content: '%s'", string(stringPart.Value))
			}
		}
	case *ast.ScalarEncapsedStringPart:
		v.t.Logf("  ScalarEncapsedStringPart Value: '%s'", string(node.Value))
	case *ast.ExprBinaryConcat:
		v.t.Logf("  ExprBinaryConcat (string concatenation)")
	case *ast.ExprVariable:
		if id, ok := node.Name.(*ast.Identifier); ok {
			v.t.Logf("  ExprVariable: $%s", string(id.Value))
		}
	}

	return true
}

// FunctionCallVisitor is a visitor that counts function calls
type FunctionCallVisitor struct {
	visitor.Null
	t     *testing.T
	count int
}

func (v *FunctionCallVisitor) EnterNode(n ast.Vertex) bool {
	if funcCall, ok := n.(*ast.ExprFunctionCall); ok {
		v.count++
		if name, ok := funcCall.Function.(*ast.Name); ok && len(name.Parts) > 0 {
			if part, ok := name.Parts[0].(*ast.NamePart); ok {
				v.t.Logf("Function call: %s with %d args", string(part.Value), len(funcCall.Args))
			}
		}
	}
	return true
}

// Simple test to understand the structure of the PHP AST with string literals and expressions
func TestPhpAstStringDetection(t *testing.T) {
	// Test code with string concatenation
	phpCode := `<?php 
	$name = "World";
	echo "Hello " . $name . "!";
	`

	// Parse the PHP code
	rootNode, err := parser.Parse([]byte(phpCode), conf.Config{})
	require.NoError(t, err)
	require.NotNil(t, rootNode)

	// Use a visitor to print node information
	debugVisitor := &DebugVisitor{t: t}
	astTraverser := traverser.NewTraverser(debugVisitor)
	rootNode.Accept(astTraverser)

	t.Logf("Total nodes: %d", debugVisitor.nodeCount)
}

// Test to focus specifically on encapsed strings (strings with variables)
func TestPhpAstEncapsedStringDetection(t *testing.T) {
	// Test code with encapsed string (string interpolation)
	phpCode := `<?php 
	$name = "World";
	echo "Hello $name!";
	`

	// Parse the PHP code
	rootNode, err := parser.Parse([]byte(phpCode), conf.Config{})
	require.NoError(t, err)
	require.NotNil(t, rootNode)

	// Use a visitor to print node information
	debugVisitor := &DebugVisitor{t: t}
	astTraverser := traverser.NewTraverser(debugVisitor)
	rootNode.Accept(astTraverser)

	t.Logf("Total nodes: %d", debugVisitor.nodeCount)

	// Also check for function call nodes to understand if they're properly detected
	funcVisitor := &FunctionCallVisitor{t: t}
	funcTraverser := traverser.NewTraverser(funcVisitor)
	rootNode.Accept(funcTraverser)

	t.Logf("Total function calls: %d", funcVisitor.count)
}
