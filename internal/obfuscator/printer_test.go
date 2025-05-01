package obfuscator_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/conf"
	"github.com/VKCOM/php-parser/pkg/parser"
	"github.com/VKCOM/php-parser/pkg/token"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/VKCOM/php-parser/pkg/visitor/printer"
	"github.com/VKCOM/php-parser/pkg/visitor/traverser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
	"github.com/whit3rabbit/phpmixer/internal/transformer"
)

// TokenInspectorVisitor counts and logs all tokens with comments
type TokenInspectorVisitor struct {
	visitor.Null
	CommentTokenCount int
}

func (v *TokenInspectorVisitor) EnterNode(n ast.Vertex) bool {
	if n == nil {
		return false
	}

	// Try to get token from node
	var nodeToken *token.Token
	if tnode, ok := n.(interface{ GetToken() *token.Token }); ok {
		nodeToken = tnode.GetToken()
		if nodeToken != nil && len(nodeToken.FreeFloating) > 0 {
			for _, ff := range nodeToken.FreeFloating {
				if ff != nil && (ff.ID == token.T_COMMENT || ff.ID == token.T_DOC_COMMENT) {
					v.CommentTokenCount++
					fmt.Printf("Found comment in token: %s\n", ff.Value)
				}
			}
		}
	}

	return true
}

func (v *TokenInspectorVisitor) LeaveNode(n ast.Vertex) (ast.Vertex, bool) {
	return n, false
}

func TestCommentStrippingAndPrinting(t *testing.T) {
	// Test code with comments
	phpCode := `<?php
// This is a line comment
function test() {
    /* This is a block comment */
    echo "Hello";
    /**
     * This is a doc comment
     */
    return true;
}
?>`

	// Parse the code
	rootNode, err := parser.Parse([]byte(phpCode), conf.Config{})
	require.NoError(t, err)
	require.NotNil(t, rootNode)

	// Create a buffer for the printer output
	originalBuf := &bytes.Buffer{}
	p := printer.NewPrinter(originalBuf)
	rootNode.Accept(p)
	originalOutput := originalBuf.String()

	// Check that the original output contains comments
	assert.Contains(t, originalOutput, "// This is a line comment", "Original output should contain line comment")
	assert.Contains(t, originalOutput, "/* This is a block comment */", "Original output should contain block comment")
	assert.Contains(t, originalOutput, "* This is a doc comment", "Original output should contain doc comment")

	// Count comment tokens in the original AST
	originalInspector := &TokenInspectorVisitor{}
	originalTraverser := traverser.NewTraverser(originalInspector)
	rootNode.Accept(originalTraverser)
	t.Logf("Original AST contains %d comment tokens", originalInspector.CommentTokenCount)

	// Strip comments
	commentStripper := transformer.NewCommentStripperVisitor()
	// Enable debug to see what's happening
	commentStripper.DebugMode = true
	strippingTraverser := traverser.NewTraverser(commentStripper)
	rootNode.Accept(strippingTraverser)

	// Count comment tokens after stripping
	strippedInspector := &TokenInspectorVisitor{}
	strippedTraverser := traverser.NewTraverser(strippedInspector)
	rootNode.Accept(strippedTraverser)
	t.Logf("After stripping, AST contains %d comment tokens", strippedInspector.CommentTokenCount)

	// Create a buffer for the post-stripping printer output
	strippedBuf := &bytes.Buffer{}
	pStripped := printer.NewPrinter(strippedBuf)
	rootNode.Accept(pStripped)
	strippedOutput := strippedBuf.String()

	// Try our custom printer that ignores comments
	customOutput := obfuscator.PrintWithoutComments(rootNode)
	t.Logf("Custom printer output: %s", customOutput)

	// Output the result for debugging
	t.Logf("Original output: %s", originalOutput)
	t.Logf("Stripped output: %s", strippedOutput)

	// Check that comments are not in the output from the custom printer
	assert.NotContains(t, customOutput, "// This is a line comment", "Custom printer output should not contain line comment")
	assert.NotContains(t, customOutput, "/* This is a block comment */", "Custom printer output should not contain block comment")
	assert.NotContains(t, customOutput, "* This is a doc comment", "Custom printer output should not contain doc comment")
}

