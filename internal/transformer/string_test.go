package transformer

import (
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/stretchr/testify/require"
)

// TestPhpStrings directly analyzes the PHP AST structure for string expressions
func TestPhpStrings(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{
			name: "String Literals",
			code: `<?php echo "Hello"; echo 'World';`,
		},
		{
			name: "String Concatenation",
			code: `<?php echo "Hello" . " " . "World";`,
		},
		{
			name: "Variable String",
			code: `<?php $name = "World"; echo "Hello $name";`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse PHP code
			rootNode, err := parser.Parse([]byte(tt.code), conf.Config{})
			require.NoError(t, err)
			require.NotNil(t, rootNode)

			// Direct AST traversal without visitors
			t.Logf("AST for: %s", tt.name)
			if root, ok := rootNode.(*ast.Root); ok {
				analyzeRoot(t, root)
			} else {
				t.Errorf("Expected *ast.Root but got %T", rootNode)
			}
		})
	}
}

// Recursively analyze the AST directly
func analyzeRoot(t *testing.T, root *ast.Root) {
	if root == nil || root.Stmts == nil {
		t.Log("Root node is nil or has no statements")
		return
	}

	t.Logf("Root node has %d statements", len(root.Stmts))
	for i, stmt := range root.Stmts {
		t.Logf("Statement %d is type: %T", i, stmt)

		// Check for echo statements
		if echo, ok := stmt.(*ast.StmtEcho); ok {
			t.Logf("  Echo statement with %d expressions", len(echo.Exprs))
			for j, expr := range echo.Exprs {
				analyzeExpr(t, expr, 2, j)
			}
		}

		// Check for expression statements (like assignments)
		if exprStmt, ok := stmt.(*ast.StmtExpression); ok {
			t.Logf("  Expression statement type: %T", exprStmt.Expr)
			if assign, ok := exprStmt.Expr.(*ast.ExprAssign); ok {
				t.Logf("  Assignment expression")
				analyzeExpr(t, assign.Expr, 2, 0)
			}
		}
	}
}

// Analyze an expression node
func analyzeExpr(t *testing.T, expr ast.Vertex, indent int, index int) {
	indentStr := ""
	for i := 0; i < indent; i++ {
		indentStr += "  "
	}

	t.Logf("%sExpression %d is type: %T", indentStr, index, expr)

	// Check various types of expressions
	switch e := expr.(type) {
	case *ast.ScalarString:
		t.Logf("%s  String literal: %s", indentStr, string(e.Value))

	case *ast.ScalarEncapsed:
		t.Logf("%s  Encapsed string with %d parts", indentStr, len(e.Parts))
		for i, part := range e.Parts {
			t.Logf("%s    Part %d is type: %T", indentStr, i, part)
			if sp, ok := part.(*ast.ScalarEncapsedStringPart); ok {
				t.Logf("%s      Text content: %s", indentStr, string(sp.Value))
			}
			if v, ok := part.(*ast.ExprVariable); ok && v.Name != nil {
				if id, ok := v.Name.(*ast.Identifier); ok {
					t.Logf("%s      Variable: $%s", indentStr, string(id.Value))
				}
			}
		}

	case *ast.ExprBinaryConcat:
		t.Logf("%s  Concatenation expression", indentStr)
		t.Logf("%s    Left part:", indentStr)
		analyzeExpr(t, e.Left, indent+2, 0)
		t.Logf("%s    Right part:", indentStr)
		analyzeExpr(t, e.Right, indent+2, 1)

	case *ast.ExprVariable:
		if e.Name != nil {
			if id, ok := e.Name.(*ast.Identifier); ok {
				t.Logf("%s  Variable: $%s", indentStr, string(id.Value))
			} else {
				t.Logf("%s  Variable with complex name: %T", indentStr, e.Name)
			}
		}
	}
}
