// Package astutil provides utilities for working with the PHP Abstract Syntax Tree (AST),
// including traversal mechanisms like visitors for obfuscation.
package astutil

import (
	"bytes" // Need bytes for comparison
	"fmt"   // For potential error reporting if token is nil
	"os"
	"strings"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"

	// No longer import obfuscator directly
	// "github.com/whit3rabbit/phpmixer/internal/obfuscator"
	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/scrambler" // Need this for ScrambleType
)

// --- Interfaces to break import cycle ---

// ScramblerInterface defines the methods needed from a scrambler by the visitor.
type ScramblerInterface interface {
	Scramble(originalName string) string
	ShouldIgnore(name string) bool
}

// Context defines the methods needed from the obfuscation context by the visitor.
type Context interface {
	GetConfig() *config.Config
	GetScrambler(sType scrambler.ScrambleType) ScramblerInterface
}

// --- Obfuscation Visitor ---

// ObfuscationVisitor implements the visitor.Visitor interface to traverse and modify the PHP AST.
// Modifications are done in-place.
type ObfuscationVisitor struct {
	visitor.Null // Embed Null visitor for default traversal

	ctx Context // Use the interface type

	// --- Visitor State (Context during traversal) ---
	currentClass ast.Vertex // Track if inside a class definition
	// Add more state if needed (namespace, function, method)
}

// NewObfuscationVisitor creates a new visitor instance.
func NewObfuscationVisitor(ctx Context) *ObfuscationVisitor { // Accept the interface
	return &ObfuscationVisitor{
		ctx: ctx,
	}
}

// Helper to modify an identifier node and its token value in place
func modifyIdentifier(ident *ast.Identifier, newValue []byte) {
	// Only modify if the new value is actually different
	if !bytes.Equal(ident.Value, newValue) {
		ident.Value = newValue // Update AST node value (for dumper/semantic checks)
		if ident.IdentifierTkn != nil {
			ident.IdentifierTkn.Value = newValue // Update TOKEN value (for printer)
		} else {
			// This case should be rare for identifiers involved in names,
			// but log a warning if it happens.
			fmt.Printf("Warning: Identifier node '%s' has nil IdentifierTkn, printer output might be incorrect.\n", string(newValue))
		}
	}
}

// Helper to get identifier string value
func getIdentifierValue(n ast.Vertex) string {
	if ident, ok := n.(*ast.Identifier); ok && ident != nil {
		return string(ident.Value)
	}
	return ""
}

// Helper to create a new Identifier node with a new value
func newIdentifier(value string) *ast.Identifier {
	// Note: We lose original token and position info here.
	// A more robust solution might involve cloning the token and updating its value.
	return &ast.Identifier{Value: []byte(value)}
}

// Helper to create a new NamePart node with a new value
func newNamePart(value string) *ast.NamePart {
	// Similar loss of token/position info.
	return &ast.NamePart{Value: []byte(value)}
}

// --- Implementing visitor.Visitor methods that MODIFY the AST ---
// Methods modify nodes in-place and do not return ast.Vertex

func (v *ObfuscationVisitor) EnterNode(n ast.Vertex) bool {
	// Track current context (simplified for now)
	if _, ok := n.(*ast.StmtClass); ok {
		v.currentClass = n
	}
	// Allow traversal into all nodes by default
	return true
}

func (v *ObfuscationVisitor) LeaveNode(n ast.Vertex) {
	// Reset context when leaving
	if _, ok := n.(*ast.StmtClass); ok && v.currentClass == n {
		v.currentClass = nil
	}
	// The Null visitor handles default LeaveNode actions
	// We could potentially move the modification logic here to ensure children are processed first,
	// but for direct name modifications, modifying on the specific type method might be clearer.
}

// Variable node modification (in-place)
func (v *ObfuscationVisitor) ExprVariable(n *ast.ExprVariable) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateVariableName {
		// Handle simple variables: $foo
		if ident, ok := n.Name.(*ast.Identifier); ok {
			originalName := string(ident.Value)
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeVariable) // Use interface method
			if scramblerInstance == nil {
				// Handle case where scrambler might be nil if GetScrambler returned nil
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in ExprVariable\n", scrambler.TypeVariable)
				return // Skip obfuscation if scrambler is missing
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(ident, []byte(scrambledName))
		}
		// Handle complex variables like ${"foo"} or ${$bar}
		// We need to recursively check if the expression inside {} resolves to an identifier we can scramble.
		// This requires more complex logic, possibly a separate pass or deeper analysis.
		// For now, we focus on simple identifiers.
	}
	// Note: Children (if variable is complex like ${...}) are handled by traversal
}

// Function Definition modification (in-place)
func (v *ObfuscationVisitor) StmtFunction(n *ast.StmtFunction) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateFunctionName {
		if ident, ok := n.Name.(*ast.Identifier); ok {
			originalName := string(ident.Value)
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeFunction) // Use interface method
			if scramblerInstance == nil {
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtFunction\n", scrambler.TypeFunction)
				return
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(ident, []byte(scrambledName))
		}
	}
	// Children (Params, ReturnType, Stmts) are handled by traversal
}

// Function Call modification (in-place)
func (v *ObfuscationVisitor) ExprFunctionCall(n *ast.ExprFunctionCall) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateFunctionName {
		// Only handle direct function calls like foo() or \foo(), not dynamic calls like $fn() or $obj->meth()
		if nameNode, ok := n.Function.(*ast.Name); ok && len(nameNode.Parts) == 1 {
			if partNode, ok := nameNode.Parts[0].(*ast.NamePart); ok {
				originalName := string(partNode.Value)
				lowerOrig := strings.ToLower(originalName)
				functionScrambler := v.ctx.GetScrambler(scrambler.TypeFunction) // Use interface method
				if functionScrambler == nil {
					fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in ExprFunctionCall (Name)\n", scrambler.TypeFunction)
					return
				}
				// Skip renaming certain built-ins here (more robust handling might be needed)
				// Also check the global scrambler ignore list
				if lowerOrig != "define" && lowerOrig != "defined" && lowerOrig != "function_exists" &&
					!functionScrambler.ShouldIgnore(originalName) { // Use interface method
					scrambledName := functionScrambler.Scramble(originalName) // Use interface method
					scrambledNameBytes := []byte(scrambledName)
					// Modify the NamePart's value and token
					if !bytes.Equal(partNode.Value, scrambledNameBytes) {
						partNode.Value = scrambledNameBytes
						if partNode.StringTkn != nil {
							partNode.StringTkn.Value = scrambledNameBytes
						} else {
							fmt.Printf("Warning: NamePart node '%s' has nil StringTkn, printer output might be incorrect.\n", string(scrambledNameBytes))
						}
					}
				}
			}
		} else if fqNameNode, ok := n.Function.(*ast.NameFullyQualified); ok && len(fqNameNode.Parts) == 1 {
			// Handle fully qualified single-part names like \foo()
			if partNode, ok := fqNameNode.Parts[0].(*ast.NamePart); ok {
				originalName := string(partNode.Value)
				functionScrambler := v.ctx.GetScrambler(scrambler.TypeFunction) // Use interface method
				if functionScrambler == nil {
					fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in ExprFunctionCall (FQName)\n", scrambler.TypeFunction)
					return
				}
				if !functionScrambler.ShouldIgnore(originalName) { // Use interface method
					scrambledName := functionScrambler.Scramble(originalName) // Use interface method
					scrambledNameBytes := []byte(scrambledName)
					if !bytes.Equal(partNode.Value, scrambledNameBytes) {
						partNode.Value = scrambledNameBytes
						if partNode.StringTkn != nil {
							partNode.StringTkn.Value = scrambledNameBytes
						} else {
							fmt.Printf("Warning: NamePart node '%s' has nil StringTkn, printer output might be incorrect.\n", string(scrambledNameBytes))
						}
					}
				}
			}
		}
		// TODO: Handle namespaced calls (Name nodes with multiple parts) if needed for obfuscation
	}
	// Children (Args) are handled by traversal
}

// Class Definition modification (in-place)
func (v *ObfuscationVisitor) StmtClass(n *ast.StmtClass) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateClassName && n.Name != nil {
		if ident, ok := n.Name.(*ast.Identifier); ok {
			originalName := string(ident.Value)
			// Use TypeFunction scrambler for Classes/Interfaces/Traits for simplicity
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeFunction) // Use interface method
			if scramblerInstance == nil {
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtClass\n", scrambler.TypeFunction)
				return
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(ident, []byte(scrambledName))
		}
	}
	// TODO: Scramble n.Extends and n.Implements if enabled (requires traversing Name nodes)
	// Children (Stmts) are handled by traversal
}

// Property Fetch modification (in-place)
func (v *ObfuscationVisitor) ExprPropertyFetch(n *ast.ExprPropertyFetch) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscatePropertyName {
		if propIdent, ok := n.Prop.(*ast.Identifier); ok {
			originalName := string(propIdent.Value)
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeProperty) // Use interface method
			if scramblerInstance == nil {
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in ExprPropertyFetch\n", scrambler.TypeProperty)
				return
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(propIdent, []byte(scrambledName))
		}
		// TODO: Handle dynamic property fetches like $obj->{$propName} if needed
	}
	// Children (Var) are handled by traversal
}

// Method Call modification (in-place)
func (v *ObfuscationVisitor) ExprMethodCall(n *ast.ExprMethodCall) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateMethodName {
		if methodIdent, ok := n.Method.(*ast.Identifier); ok {
			originalName := string(methodIdent.Value)
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeMethod) // Use interface method
			if scramblerInstance == nil {
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in ExprMethodCall\n", scrambler.TypeMethod)
				return
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(methodIdent, []byte(scrambledName))
		}
		// TODO: Handle dynamic method calls like $obj->{$methName}() if needed
	}
	// Children (Var, Args) are handled by traversal
}

// Add implementations for other relevant nodes similarly, modifying in-place:
// StmtInterface, StmtTrait, StmtClassMethod, StmtPropertyList, StmtConstList,
// StmtClassConstList, StmtLabel, StmtGoto, StmtUseList, StmtGroupUseList, StmtUse,
// ExprConstFetch, ExprClassConstFetch, ExprNew, ExprStaticCall, ExprStaticPropertyFetch

// Example: Method Definition
func (v *ObfuscationVisitor) StmtClassMethod(n *ast.StmtClassMethod) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateMethodName {
		if ident, ok := n.Name.(*ast.Identifier); ok {
			originalName := string(ident.Value)
			scramblerInstance := v.ctx.GetScrambler(scrambler.TypeMethod) // Use interface method
			if scramblerInstance == nil {
				fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtClassMethod\n", scrambler.TypeMethod)
				return
			}
			scrambledName := scramblerInstance.Scramble(originalName)
			modifyIdentifier(ident, []byte(scrambledName))
		}
	}
	// Children handled by traversal
}

// Example: Property Definition
func (v *ObfuscationVisitor) StmtProperty(n *ast.StmtProperty) {
	// Note: StmtProperty holds the variable node ($foo), not just the name (foo).
	// StmtPropertyList handles the whole declaration like `public $foo = 1, $bar;`
	// We need to handle the individual property declarations inside StmtPropertyList
	// For now, this specific method might not be the right place. Let's handle it in StmtPropertyList.
}

// Example: Property List (e.g., `public $prop1, $prop2 = null;`)
func (v *ObfuscationVisitor) StmtPropertyList(n *ast.StmtPropertyList) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscatePropertyName {
		propertyScrambler := v.ctx.GetScrambler(scrambler.TypeProperty) // Use interface method
		if propertyScrambler == nil {
			fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtPropertyList\n", scrambler.TypeProperty)
			return // Skip if scrambler missing
		}
		for _, propElement := range n.Props {
			if propStmt, ok := propElement.(*ast.StmtProperty); ok {
				if propVar, ok := propStmt.Var.(*ast.ExprVariable); ok {
					if ident, ok := propVar.Name.(*ast.Identifier); ok {
						originalName := string(ident.Value)
						scrambledName := propertyScrambler.Scramble(originalName)
						modifyIdentifier(ident, []byte(scrambledName))
					}
				}
			}
		}
	}
	// Children (Type, individual property default values) are handled by traversal
}

// Example: Class Constant Definition List (e.g., `const FOO = 1, BAR = 2;`)
func (v *ObfuscationVisitor) StmtClassConstList(n *ast.StmtClassConstList) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateClassConstantName {
		constScrambler := v.ctx.GetScrambler(scrambler.TypeClassConstant) // Use interface method
		if constScrambler == nil {
			fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtClassConstList\n", scrambler.TypeClassConstant)
			return // Skip if scrambler missing
		}
		for _, constElement := range n.Consts {
			if constStmt, ok := constElement.(*ast.StmtConstant); ok {
				if ident, ok := constStmt.Name.(*ast.Identifier); ok {
					originalName := string(ident.Value)
					scrambledName := constScrambler.Scramble(originalName)
					modifyIdentifier(ident, []byte(scrambledName))
				}
			}
		}
	}
	// Children (Constant values) are handled by traversal
}

// Example: Global Constant Definition List (e.g., `const FOO = 1, BAR = 2;` outside class)
func (v *ObfuscationVisitor) StmtConstList(n *ast.StmtConstList) {
	cfg := v.ctx.GetConfig() // Use renamed interface method
	if cfg.ObfuscateConstantName {
		constScrambler := v.ctx.GetScrambler(scrambler.TypeConstant) // Use interface method
		if constScrambler == nil {
			fmt.Fprintf(os.Stderr, "Warning: No scrambler found for type %s in StmtConstList\n", scrambler.TypeConstant)
			return // Skip if scrambler missing
		}
		for _, constElement := range n.Consts {
			if constStmt, ok := constElement.(*ast.StmtConstant); ok {
				if ident, ok := constStmt.Name.(*ast.Identifier); ok {
					originalName := string(ident.Value)
					scrambledName := constScrambler.Scramble(originalName)
					modifyIdentifier(ident, []byte(scrambledName))
				}
			}
		}
	}
	// Children (Constant values) are handled by traversal
}

// TODO: Add more specific visitor methods here for all node types you want to modify...
// e.g., ExprStaticCall, ExprClassConstFetch, StmtInterface, StmtTrait, etc.

