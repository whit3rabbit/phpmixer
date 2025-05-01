package obfuscator

import (
	"bytes"
	"io"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
)

// CommentRemovingPrinterVisitor is a custom visitor that implements a basic printer
// that doesn't print comments, only the essential code.
type CommentRemovingPrinterVisitor struct {
	visitor.Null
	w io.Writer
}

// NewCommentRemovingPrinterVisitor creates a new printer that won't print comments
func NewCommentRemovingPrinterVisitor(w io.Writer) *CommentRemovingPrinterVisitor {
	return &CommentRemovingPrinterVisitor{
		w: w,
	}
}

// EnterNode handles the node when entering it
func (v *CommentRemovingPrinterVisitor) EnterNode(n ast.Vertex) bool {
	if n == nil {
		return false
	}

	// Handle root node - print PHP opening tag
	if _, ok := n.(*ast.Root); ok {
		io.WriteString(v.w, "<?php\n")
		return true
	}

	// Handle function declaration
	if fn, ok := n.(*ast.StmtFunction); ok {
		io.WriteString(v.w, "function ")
		if fn.Name != nil {
			if ident, ok := fn.Name.(*ast.Identifier); ok {
				io.WriteString(v.w, string(ident.Value))
			}
		}
		io.WriteString(v.w, "(")
		// Print parameters if any
		for i, param := range fn.Params {
			if i > 0 {
				io.WriteString(v.w, ", ")
			}
			if p, ok := param.(*ast.Parameter); ok && p.Var != nil {
				if varExpr, ok := p.Var.(*ast.ExprVariable); ok && varExpr.Name != nil {
					if id, ok := varExpr.Name.(*ast.Identifier); ok {
						io.WriteString(v.w, string(id.Value))
					}
				}
			}
		}
		io.WriteString(v.w, ") {\n")
		return true
	}

	// Handle echo statement
	if _, ok := n.(*ast.StmtEcho); ok {
		io.WriteString(v.w, "echo ")
		return true
	}

	// Handle return statement
	if _, ok := n.(*ast.StmtReturn); ok {
		io.WriteString(v.w, "return ")
		return true
	}

	// Handle string literal
	if str, ok := n.(*ast.ScalarString); ok {
		val := string(str.Value)
		if len(val) > 0 {
			io.WriteString(v.w, val)
		} else {
			// Fallback if Value is empty
			io.WriteString(v.w, "\"\"")
		}
		return true
	}

	// Handle true/false literals
	if _, ok := n.(*ast.ScalarLnumber); ok {
		io.WriteString(v.w, "true")
		return true
	}

	// Handle variables
	if varExpr, ok := n.(*ast.ExprVariable); ok {
		if varExpr.Name != nil {
			if id, ok := varExpr.Name.(*ast.Identifier); ok {
				io.WriteString(v.w, string(id.Value))
			}
		}
		return true
	}

	// Handle concatenation
	if _, ok := n.(*ast.ExprBinaryConcat); ok {
		// Concatenation operator will be handled in LeaveNode
		return true
	}

	// Handle function calls
	if call, ok := n.(*ast.ExprFunctionCall); ok {
		if call.Function != nil {
			if name, ok := call.Function.(*ast.Name); ok {
				if len(name.Parts) > 0 {
					if part, ok := name.Parts[0].(*ast.NamePart); ok {
						io.WriteString(v.w, string(part.Value))
					}
				}
			}
		}
		io.WriteString(v.w, "(")
		return true
	}

	// Handle arguments
	if _, ok := n.(*ast.Argument); ok {
		// Arguments are handled by visiting their children
		return true
	}

	return true
}

// LeaveNode handles the node when leaving it
func (v *CommentRemovingPrinterVisitor) LeaveNode(n ast.Vertex) (ast.Vertex, bool) {
	switch n.(type) {
	case *ast.StmtEcho:
		io.WriteString(v.w, ";\n")
	case *ast.StmtReturn:
		io.WriteString(v.w, ";\n")
	case *ast.StmtFunction:
		io.WriteString(v.w, "}\n")
	case *ast.Root:
		io.WriteString(v.w, "?>")
	case *ast.ExprBinaryConcat:
		io.WriteString(v.w, ".")
	case *ast.ExprFunctionCall:
		io.WriteString(v.w, ")")
		// Handle the semicolon in the statement context
	}
	return n, false
}

// PrintWithoutComments takes an AST node and outputs it without any comments
func PrintWithoutComments(root ast.Vertex) string {
	if root == nil {
		return ""
	}
	buf := &bytes.Buffer{}
	printer := NewCommentRemovingPrinterVisitor(buf)
	t := traverser.NewTraverser(printer)
	root.Accept(t)
	return buf.String()
}
