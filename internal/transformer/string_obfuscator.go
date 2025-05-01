package transformer

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/token"
	"github.com/VKCOM/php-parser/pkg/visitor"
	"github.com/whit3rabbit/phpmixer/internal/config"
)

// StringObfuscatorVisitor transforms string literals into obfuscated function calls using LeaveNode.
type StringObfuscatorVisitor struct {
	visitor.Null // Embed Null visitor for default traversal
	DebugMode    bool
	// Technique to use for obfuscation
	technique string
	// Track if XOR was used to inject helper function later (alternative: check config in ProcessFile)
	xorUsed bool
}

// NewStringObfuscatorVisitor creates a new visitor instance.
func NewStringObfuscatorVisitor(technique string) *StringObfuscatorVisitor {
	return &StringObfuscatorVisitor{
		DebugMode: false, // Default to no debug mode
		technique: technique,
	}
}

// HandleEscapedSequences processes common PHP escaped sequences in string literals
func HandleEscapedSequences(str []byte) []byte {
	result := bytes.ReplaceAll(str, []byte{'\\', 'n'}, []byte{'\n'})
	result = bytes.ReplaceAll(result, []byte{'\\', 'r'}, []byte{'\r'})
	result = bytes.ReplaceAll(result, []byte{'\\', 't'}, []byte{'\t'})
	result = bytes.ReplaceAll(result, []byte{'\\', '"'}, []byte{'"'})
	result = bytes.ReplaceAll(result, []byte{'\\', '\''}, []byte{'\''})
	result = bytes.ReplaceAll(result, []byte{'\\', '\\'}, []byte{'\\'})
	result = bytes.ReplaceAll(result, []byte{'\\', '$'}, []byte{'$'})
	return result
}

// EnterNode is called before traversing children.
func (v *StringObfuscatorVisitor) EnterNode(n ast.Vertex) bool {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "EnterNode called for %T\n", n)
	}

	switch n.(type) {
	case *ast.ScalarString:
		// Handle in LeaveNode since we need to wait for children to be processed
		return true
	case *ast.ScalarEncapsed:
		// Handle in LeaveNode since we need to wait for children to be processed
		return true
	}

	return true // Default: continue traversal
}

// createObfuscatedCallNode generates the AST node for the obfuscated function call.
// It handles Base64, ROT13, and XOR.
func (v *StringObfuscatorVisitor) createObfuscatedCallNode(value []byte) ast.Vertex {
	// Skip empty strings
	if len(value) == 0 {
		return nil
	}

	switch v.technique {
	case config.StringObfuscationTechniqueBase64:
		encoded := base64.StdEncoding.EncodeToString(value)
		return &ast.ExprFunctionCall{
			Function: &ast.Name{Parts: []ast.Vertex{&ast.NamePart{Value: []byte("base64_decode")}}},
			Args: []ast.Vertex{&ast.Argument{Expr: &ast.ScalarString{
				StringTkn: &token.Token{Value: []byte("'" + encoded + "'")},
				Value:     []byte("'" + encoded + "'"),
			}}},
		}
	case config.StringObfuscationTechniqueRot13:
		v.xorUsed = false
		// Apply ROT13 transformation to get the encoded value to decode
		rot13Content := make([]byte, len(value))
		for i, c := range value {
			switch {
			case c >= 'a' && c <= 'z':
				rot13Content[i] = 'a' + (c-'a'+13)%26
			case c >= 'A' && c <= 'Z':
				rot13Content[i] = 'A' + (c-'A'+13)%26
			default:
				rot13Content[i] = c
			}
		}

		return &ast.ExprFunctionCall{
			Function: &ast.Name{Parts: []ast.Vertex{&ast.NamePart{Value: []byte("str_rot13")}}},
			Args: []ast.Vertex{&ast.Argument{Expr: &ast.ScalarString{
				// PHP str_rot13 expects the already rot13-encoded string
				StringTkn: &token.Token{Value: []byte("'" + string(value) + "'")},
				Value:     []byte("'" + string(value) + "'"),
			}}},
		}
	case config.StringObfuscationTechniqueXOR:
		key := make([]byte, 8)
		_, err := rand.Read(key)
		if err != nil {
			if v.DebugMode {
				fmt.Printf("Error generating XOR key, falling back to base64 for this string: %v\n", err)
			}
			encoded := base64.StdEncoding.EncodeToString(value)
			return &ast.ExprFunctionCall{
				Function: &ast.Name{Parts: []ast.Vertex{&ast.NamePart{Value: []byte("base64_decode")}}},
				Args: []ast.Vertex{&ast.Argument{Expr: &ast.ScalarString{
					StringTkn: &token.Token{Value: []byte("'" + encoded + "'")},
					Value:     []byte("'" + encoded + "'"),
				}}},
			}
		}
		xorEncoded := make([]byte, len(value))
		for i := 0; i < len(value); i++ {
			xorEncoded[i] = value[i] ^ key[i%len(key)]
		}
		encodedData := base64.StdEncoding.EncodeToString(xorEncoded)
		encodedKey := base64.StdEncoding.EncodeToString(key)
		v.xorUsed = true
		return &ast.ExprFunctionCall{
			Function: &ast.Name{Parts: []ast.Vertex{&ast.NamePart{Value: []byte("_obfuscated_xor_decode")}}},
			Args: []ast.Vertex{
				&ast.Argument{Expr: &ast.ScalarString{
					StringTkn: &token.Token{Value: []byte("'" + encodedData + "'")},
					Value:     []byte("'" + encodedData + "'"),
				}},
				&ast.Argument{Expr: &ast.ScalarString{
					StringTkn: &token.Token{Value: []byte("'" + encodedKey + "'")},
					Value:     []byte("'" + encodedKey + "'"),
				}},
			},
		}
	default:
		if v.DebugMode {
			fmt.Printf("Unknown string technique '%s', using base64 for this string.\n", v.technique)
		}
		encoded := base64.StdEncoding.EncodeToString(value)
		return &ast.ExprFunctionCall{
			Function: &ast.Name{Parts: []ast.Vertex{&ast.NamePart{Value: []byte("base64_decode")}}},
			Args: []ast.Vertex{&ast.Argument{Expr: &ast.ScalarString{
				StringTkn: &token.Token{Value: []byte("'" + encoded + "'")},
				Value:     []byte("'" + encoded + "'"),
			}}},
		}
	}
}

// processEncapsedString processes an encapsed string (like "Hello $name")
// by transforming it into a concatenation of obfuscated literals and preserved variables
func (v *StringObfuscatorVisitor) processEncapsedString(node *ast.ScalarEncapsed) ast.Vertex {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Processing encapsed string with %d parts\n", len(node.Parts))
	}

	// If there are no parts, return the original node
	if len(node.Parts) == 0 {
		return node
	}

	// If there's only one part and it's a string literal, obfuscate it directly
	if len(node.Parts) == 1 {
		if stringPart, ok := node.Parts[0].(*ast.ScalarEncapsedStringPart); ok {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Single part encapsed string, obfuscating directly: %s\n",
					string(stringPart.Value))
			}
			return v.createObfuscatedCallNode(stringPart.Value)
		}
	}

	// For multiple parts or mixed content, build a concatenation expression
	var resultExpr ast.Vertex

	for _, part := range node.Parts {
		var partExpr ast.Vertex

		// Process based on part type
		switch p := part.(type) {
		case *ast.ScalarEncapsedStringPart:
			// Obfuscate string parts
			if len(p.Value) > 0 {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "Obfuscating encapsed string part: %s\n", string(p.Value))
				}
				partExpr = v.createObfuscatedCallNode(p.Value)
			} else {
				// Empty string part, skip
				continue
			}
		case *ast.ExprVariable:
			// Preserve variables as-is
			if v.DebugMode {
				if id, ok := p.Name.(*ast.Identifier); ok {
					fmt.Fprintf(os.Stderr, "Preserving variable in encapsed string: $%s\n", string(id.Value))
				} else {
					fmt.Fprintf(os.Stderr, "Preserving complex variable in encapsed string\n")
				}
			}
			partExpr = p
		default:
			// Other parts, preserve as-is
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Unknown part type in encapsed string: %T\n", p)
			}
			partExpr = p
		}

		// Skip if this part resulted in no expression
		if partExpr == nil {
			continue
		}

		// Build the concatenation expression
		if resultExpr == nil {
			resultExpr = partExpr
		} else {
			resultExpr = &ast.ExprBinaryConcat{
				Left:  resultExpr,
				Right: partExpr,
			}
		}
	}

	// If we couldn't build any expression, return the original
	if resultExpr == nil {
		return node
	}

	return resultExpr
}

// LeaveNode performs the replacement after children have been visited.
func (v *StringObfuscatorVisitor) LeaveNode(n ast.Vertex) (ast.Vertex, bool) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "LeaveNode called for %T\n", n)
	}

	switch node := n.(type) {
	case *ast.ScalarString:
		// Skip strings that are too short (basically empty strings)
		if len(node.Value) <= 2 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Skipping string that's too short: %s\n", string(node.Value))
			}
			return node, false
		}

		quotedValue := node.Value

		// Skip heredoc/nowdoc strings (identified by <<<)
		if bytes.Contains(quotedValue, []byte("<<<")) {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Skipping heredoc/nowdoc string: %s\n", string(quotedValue))
			}
			return node, false
		}

		var unquotedValue []byte
		var quoteChar byte

		// Debug the exact token value
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Processing string token: %s\n", string(quotedValue))
		}

		// Check if it's a quoted string and extract the content
		if len(quotedValue) >= 2 {
			if quotedValue[0] == '\'' && quotedValue[len(quotedValue)-1] == '\'' {
				quoteChar = '\''
				unquotedValue = quotedValue[1 : len(quotedValue)-1]
			} else if quotedValue[0] == '"' && quotedValue[len(quotedValue)-1] == '"' {
				quoteChar = '"'
				unquotedValue = quotedValue[1 : len(quotedValue)-1]
			} else {
				// Handle the case where the value isn't properly quoted
				// This can happen with certain parser configurations
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "String not properly quoted: %s\n", string(quotedValue))
				}
				// Try to infer quote type from first character
				if len(quotedValue) > 0 {
					if quotedValue[0] == '\'' {
						quoteChar = '\''
						// Try to extract until the last quote or use as is
						lastQuote := bytes.LastIndexByte(quotedValue, '\'')
						if lastQuote > 0 {
							unquotedValue = quotedValue[1:lastQuote]
						} else {
							unquotedValue = quotedValue
						}
					} else if quotedValue[0] == '"' {
						quoteChar = '"'
						// Try to extract until the last quote or use as is
						lastQuote := bytes.LastIndexByte(quotedValue, '"')
						if lastQuote > 0 {
							unquotedValue = quotedValue[1:lastQuote]
						} else {
							unquotedValue = quotedValue
						}
					} else {
						// No clear quotes, use as is
						unquotedValue = quotedValue
						quoteChar = 0 // No specific quote char
					}
				} else {
					unquotedValue = quotedValue
					quoteChar = 0
				}
			}
		} else {
			// Too short to be a proper quoted string
			return node, false
		}

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Extracted unquoted value: %s\n", string(unquotedValue))
		}

		// Skip double-quoted strings containing variables
		if quoteChar == '"' && bytes.Contains(unquotedValue, []byte{'$'}) {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Skipping string with variables: %s\n", string(unquotedValue))
			}
			return node, false
		}

		// Process the value based on quote type
		processingValue := unquotedValue
		if quoteChar == '\'' {
			// Handle single-quoted string escapes
			processingValue = bytes.ReplaceAll(processingValue, []byte("\\'"), []byte("'"))
			processingValue = bytes.ReplaceAll(processingValue, []byte("\\\\"), []byte("\\"))
		} else if quoteChar == '"' {
			// Handle double-quoted string escapes
			processingValue = HandleEscapedSequences(processingValue)
		}

		// Skip empty strings after processing escapes
		if len(processingValue) == 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Skipping empty string after processing\n")
			}
			return node, false
		}

		// Create the obfuscated function call node
		obfuscatedNode := v.createObfuscatedCallNode(processingValue)
		if obfuscatedNode == nil {
			return node, false
		}

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Replacing string: '%s' with obfuscated call\n", string(unquotedValue))
		}

		return obfuscatedNode, true

	case *ast.ScalarEncapsed:
		// Process encapsed strings (strings with variables like "Hello $name")
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Processing encapsed string node\n")
		}

		processed := v.processEncapsedString(node)
		if processed != node {
			return processed, true
		}
	}

	return n, false // Default: return original node, no replacement
}

// XORWasUsed returns true if XOR obfuscation was used.
func (v *StringObfuscatorVisitor) XORWasUsed() bool {
	// Debug output
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StringObfuscatorVisitor.XORWasUsed called, returning %v\n", v.xorUsed)
	}

	// If the technique is XOR, we should always return true, even if no strings were actually processed
	if v.technique == config.StringObfuscationTechniqueXOR {
		return true
	}

	return v.xorUsed
}

