package transformer

import (
	"fmt"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/stretchr/testify/require"
)

// A proper PHP parser visitor that counts nodes by type
type phpParserVisitor struct {
	visitor.Null
	nodeCount map[string]int
}

func newPhpParserVisitor() *phpParserVisitor {
	return &phpParserVisitor{
		nodeCount: make(map[string]int),
	}
}

func (v *phpParserVisitor) countNode(name string) {
	v.nodeCount[name]++
	fmt.Printf("Visited: %s\n", name)
}

// Implement specific node visitor methods
func (v *phpParserVisitor) Root(n *ast.Root) {
	v.countNode("*ast.Root")
}

func (v *phpParserVisitor) StmtEcho(n *ast.StmtEcho) {
	v.countNode("*ast.StmtEcho")
	fmt.Printf("  Echo statement with %d expressions\n", len(n.Exprs))
}

func (v *phpParserVisitor) ExprVariable(n *ast.ExprVariable) {
	v.countNode("*ast.ExprVariable")
	if id, ok := n.Name.(*ast.Identifier); ok {
		fmt.Printf("  Variable: $%s\n", id.Value)
	}
}

func (v *phpParserVisitor) Identifier(n *ast.Identifier) {
	v.countNode("*ast.Identifier")
	fmt.Printf("  Identifier: %s\n", n.Value)
}

func (v *phpParserVisitor) ScalarString(n *ast.ScalarString) {
	v.countNode("*ast.ScalarString")
	fmt.Printf("  String: %s\n", n.Value)
}

func (v *phpParserVisitor) ScalarEncapsed(n *ast.ScalarEncapsed) {
	v.countNode("*ast.ScalarEncapsed")
	fmt.Printf("  Encapsed string with %d parts\n", len(n.Parts))
}

func (v *phpParserVisitor) ScalarEncapsedStringPart(n *ast.ScalarEncapsedStringPart) {
	v.countNode("*ast.ScalarEncapsedStringPart")
	fmt.Printf("  Encapsed string part: %s\n", n.Value)
}

func (v *phpParserVisitor) ScalarEncapsedStringBrackets(n *ast.ScalarEncapsedStringBrackets) {
	v.countNode("*ast.ScalarEncapsedStringBrackets")
	fmt.Printf("  Encapsed string brackets\n")
}

func TestDebugPhpParser(t *testing.T) {
	// Multiple test cases to identify what's happening
	testCases := []struct {
		name string
		code string
	}{
		{
			name: "Simple echo with variable",
			code: `<?php echo $name; ?>`,
		},
		{
			name: "String literal echo",
			code: `<?php echo "Hello"; ?>`,
		},
		{
			name: "Echo with encapsed string",
			code: `<?php echo "Hello {$name}"; ?>`,
		},
		{
			name: "Echo with simpler variable interpolation",
			code: `<?php echo "Hello $name"; ?>`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse the PHP code
			rootNode, err := parser.Parse([]byte(tc.code), conf.Config{})
			require.NoError(t, err, "Parser error")
			require.NotNil(t, rootNode, "Root node should not be nil")

			// Create properly implemented visitor
			phpVisitor := newPhpParserVisitor()

			// Print test case info
			fmt.Printf("\n\nAST for: %s\n", tc.code)
			fmt.Println("================================")

			// Traverse using the PHP parser's visitor
			rootNode.Accept(phpVisitor)

			// Check if we found a ScalarEncapsed node
			count, hasEncapsed := phpVisitor.nodeCount["*ast.ScalarEncapsed"]
			if hasEncapsed {
				fmt.Printf("\nFound %d *ast.ScalarEncapsed nodes\n", count)
			} else {
				fmt.Println("\nNo *ast.ScalarEncapsed nodes found")
			}

			// Print summary
			fmt.Println("\nVisited node types:")
			for nodeType, count := range phpVisitor.nodeCount {
				fmt.Printf("  - %s: %d\n", nodeType, count)
			}
		})
	}
}
