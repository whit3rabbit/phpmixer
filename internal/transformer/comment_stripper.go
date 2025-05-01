// Package transformer holds implementations for specific obfuscation transformations.
package transformer

import (
	"fmt"
	"reflect" // Used for reflection to access token fields dynamically

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/token"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

// tokenGetter is an interface for nodes that have a single primary token.
type tokenGetter interface {
	GetToken() *token.Token
}

// tokensGetter is an interface for nodes that have multiple significant tokens.
type tokensGetter interface {
	GetTokens() []*token.Token
}

// CommentStripperVisitor removes comment tokens from the FreeFloating slices of all tokens in the AST.
type CommentStripperVisitor struct {
	visitor.Null  // Embed Null visitor for default traversal
	DebugMode     bool
	commentsFound int
	// Add a flag to completely remove comment tokens
	AggressiveMode bool
}

// NewCommentStripperVisitor creates a new visitor instance.
func NewCommentStripperVisitor() *CommentStripperVisitor {
	return &CommentStripperVisitor{
		AggressiveMode: true, // Default to aggressive mode
	}
}

// EnterNode is called for each node. We inspect its fields for tokens.
func (v *CommentStripperVisitor) EnterNode(n ast.Vertex) bool {
	if n == nil {
		return false // Skip nil nodes
	}

	// 1. Check common token fields manually using reflection (more robust than just methods)
	//    We use reflection here because there's no single interface covering all these fields.
	rv := reflect.ValueOf(n)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		rv = rv.Elem() // Get the struct value
		if rv.Kind() == reflect.Struct {
			for i := 0; i < rv.NumField(); i++ {
				field := rv.Field(i)
				// Check if the field is a *token.Token
				if field.Kind() == reflect.Ptr && field.Type() == reflect.TypeOf((*token.Token)(nil)) {
					if !field.IsNil() {
						if t, ok := field.Interface().(*token.Token); ok && t != nil {
							v.clearComments(t)
						}
					}
				}
				// Check if the field is a []*token.Token
				if field.Kind() == reflect.Slice && field.Type() == reflect.TypeOf(([]*token.Token)(nil)) {
					if !field.IsNil() {
						if tokens, ok := field.Interface().([]*token.Token); ok {
							for _, t := range tokens {
								if t != nil {
									v.clearComments(t)
								}
							}
						}
					}
				}
			}
		}
	}

	// 2. Check for GetToken method
	if tg, ok := n.(tokenGetter); ok {
		t := tg.GetToken()
		if t != nil {
			v.clearComments(t)
		}
	}

	// 3. Check for GetTokens method
	if tgs, ok := n.(tokensGetter); ok {
		tokens := tgs.GetTokens()
		for _, t := range tokens {
			if t != nil {
				v.clearComments(t)
			}
		}
	}

	return true // Continue traversal
}

// LeaveNode is called when leaving a node. We'll return the node unchanged.
func (v *CommentStripperVisitor) LeaveNode(n ast.Vertex) (ast.Vertex, bool) {
	if v.DebugMode && v.commentsFound > 0 {
		fmt.Printf("CommentStripperVisitor: Removed %d comments in total\n", v.commentsFound)
	}
	return n, false // Return the node unchanged
}

// clearComments removes T_COMMENT and T_DOC_COMMENT from a token's FreeFloating slice.
func (v *CommentStripperVisitor) clearComments(t *token.Token) {
	if t == nil || len(t.FreeFloating) == 0 {
		return
	}

	// Create a new slice with only non-comment tokens
	newFreeFloating := make([]*token.Token, 0, len(t.FreeFloating))
	removedCount := 0

	for _, ffToken := range t.FreeFloating {
		if ffToken == nil {
			// Skip nil tokens
			continue
		}

		// Check if it's a comment token - php-parser uses these token IDs for comments
		if ffToken.ID == token.T_COMMENT || ffToken.ID == token.T_DOC_COMMENT {
			removedCount++
			if v.DebugMode {
				fmt.Printf("Removing comment: %s\n", ffToken.Value)
			}

			// In aggressive mode, nullify the comment token's contents
			if v.AggressiveMode {
				// Replace the comment content with empty to ensure it doesn't get printed
				ffToken.Value = []byte{}
			} else {
				// In standard mode, we'll just filter it out of the FreeFloating slice
				// (This is what the original method did)
				continue
			}
		}

		// In aggressive mode with empty comments, skip adding them
		if v.AggressiveMode && ffToken.ID == token.T_COMMENT && len(ffToken.Value) == 0 {
			continue
		}

		// Add non-comment tokens to the new slice
		newFreeFloating = append(newFreeFloating, ffToken)
	}

	if removedCount > 0 {
		v.commentsFound += removedCount
		// Replace the original slice with the filtered one
		t.FreeFloating = newFreeFloating
	}
}
