package transformer

import (
	"bytes"
	"testing"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/token"
	"github.com/stretchr/testify/assert"
)

func TestStringObfuscatorVisitor_HeredocStrings(t *testing.T) {
	testCases := []struct {
		name       string
		inputValue []byte
		shouldSkip bool
	}{
		{
			name:       "Regular string",
			inputValue: []byte("'hello world'"),
			shouldSkip: false,
		},
		{
			name:       "Heredoc string",
			inputValue: []byte("<<<EOT\nHello World\nEOT"),
			shouldSkip: true,
		},
		{
			name:       "Nowdoc string",
			inputValue: []byte("<<<'EOT'\nHello World\nEOT"),
			shouldSkip: true,
		},
		{
			name:       "String with special characters",
			inputValue: []byte("'Hello \\' World'"),
			shouldSkip: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a string obfuscator with a known technique
			visitor := NewStringObfuscatorVisitor("base64")
			visitor.DebugMode = true

			// Create an AST node representing the string
			node := &ast.ScalarString{
				StringTkn: &token.Token{Value: tc.inputValue},
				Value:     tc.inputValue,
			}

			// Process the node
			result, replaced := visitor.LeaveNode(node)

			// Verify the result
			if tc.shouldSkip {
				assert.False(t, replaced, "Should not replace heredoc/nowdoc strings")
				assert.Equal(t, node, result, "Should return the original node")
			} else {
				// For non-heredoc strings, some processing should occur
				// Note: actual replacement may or may not happen based on other criteria
				// We're only checking that the heredoc check is working correctly
				if len(tc.inputValue) > 2 {
					// If it's a properly formatted string, it should be processed
					// The actual result depends on the node's content
					if tc.inputValue[0] == '\'' && tc.inputValue[len(tc.inputValue)-1] == '\'' {
						content := tc.inputValue[1 : len(tc.inputValue)-1]
						if len(content) > 0 {
							// Only assert it was replaced if the string has content
							if !bytes.Contains(content, []byte{'$'}) {
								// Only if there are no variables
								if !assert.True(t, replaced, "Should have replaced the regular string") {
									t.Logf("Input: %s", tc.inputValue)
								}
							}
						}
					}
				}
			}
		})
	}
}
