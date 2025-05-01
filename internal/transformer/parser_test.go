package transformer

import (
	"reflect"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/stretchr/testify/require"
)

func TestPhpParser(t *testing.T) {
	code := `<?php echo "Hello {$name}"; ?>`

	// Parse the PHP code
	rootNode, err := parser.Parse([]byte(code), conf.Config{})
	require.NoError(t, err, "Parser error")
	require.NotNil(t, rootNode, "Root node should not be nil")

	// Print the full AST structure
	dumpAST(t, rootNode, 0)
}

func dumpAST(t *testing.T, node ast.Vertex, level int) {
	if node == nil {
		return
	}

	indent := ""
	for i := 0; i < level; i++ {
		indent += "  "
	}

	t.Logf("%s%T", indent, node)

	// Recursively dump child nodes
	switch n := node.(type) {
	case *ast.Root:
		for _, stmt := range n.Stmts {
			dumpAST(t, stmt, level+1)
		}
	case *ast.StmtEcho:
		for _, expr := range n.Exprs {
			dumpAST(t, expr, level+1)
		}
	case *ast.StmtExpression:
		dumpAST(t, n.Expr, level+1)
	case *ast.StmtStmtList:
		for _, stmt := range n.Stmts {
			dumpAST(t, stmt, level+1)
		}
	case *ast.ScalarEncapsed:
		for _, part := range n.Parts {
			dumpAST(t, part, level+1)
		}
	case *ast.ScalarEncapsedStringPart:
		// Leaf node
	case *ast.ScalarEncapsedStringVar:
		dumpAST(t, n.Name, level+1)
	case *ast.ScalarEncapsedStringBrackets:
		dumpAST(t, n.Var, level+1)
	case *ast.ExprVariable:
		dumpAST(t, n.Name, level+1)
	default:
		// Use reflection to try to find fields that might contain nodes
		val := reflect.ValueOf(node)
		if val.Kind() == reflect.Ptr {
			val = val.Elem() // Dereference pointer
		}

		if val.Kind() == reflect.Struct {
			for i := 0; i < val.NumField(); i++ {
				field := val.Field(i)

				// Check if the field is a node, a slice of nodes, or a map with node values
				switch {
				case field.Type().AssignableTo(reflect.TypeOf((*ast.Vertex)(nil)).Elem()):
					if !field.IsNil() {
						node := field.Interface().(ast.Vertex)
						dumpAST(t, node, level+1)
					}
				case field.Kind() == reflect.Slice:
					for j := 0; j < field.Len(); j++ {
						item := field.Index(j)
						if item.Type().AssignableTo(reflect.TypeOf((*ast.Vertex)(nil)).Elem()) && !item.IsNil() {
							node := item.Interface().(ast.Vertex)
							dumpAST(t, node, level+1)
						}
					}
				}
			}
		}
	}
}
