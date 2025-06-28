// Package transformer provides AST transformation utilities for PHP obfuscation.
package transformer

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

/*
Control Flow Obfuscation Overview:
---------------------------------
This file implements a visitor pattern to transform PHP Abstract Syntax Tree (AST) nodes
for control flow obfuscation. The main approach is to wrap code blocks inside
redundant conditional blocks that always evaluate to true, such as:

Original:
```php
function foo() {
    echo "Hello";
    return "World";
}
```

Obfuscated:
```php
function foo() {
    if(1){
        echo "Hello";
        return "World";
    }
}
```

The AST transformation preserves functionality while making static analysis more difficult.
This obfuscator handles various PHP control flow constructs, including:
- Function bodies
- Class method bodies
- If/elseif/else statements
- For, foreach, while, and do-while loops

AST Structure Notes:
-------------------
The PHP AST used by this library has the following structure for key elements:

1. Function nodes (*ast.StmtFunction):
   - n.Stmts: []ast.Vertex - Contains the list of statements in the function body

2. Method nodes (*ast.StmtClassMethod):
   - n.Stmt: ast.Vertex - Contains the method body, usually a *ast.StmtStmtList

3. Loop nodes (*ast.StmtFor, *ast.StmtWhile, *ast.StmtForeach, *ast.StmtDo):
   - n.Stmt: ast.Vertex - Contains the loop body, either a *ast.StmtStmtList
     for multiple statements or a single statement

4. If statement nodes (*ast.StmtIf):
   - n.Cond: ast.Vertex - The condition expression
   - n.Stmt: ast.Vertex - The "then" branch body
   - n.ElseIf: []ast.Vertex - List of elseif branches
   - n.Else: ast.Vertex - The else branch

5. Statement lists (*ast.StmtStmtList):
   - stmtList.Stmts: []ast.Vertex - List of statements in a block

6. Literal values (*ast.ScalarLnumber for integers):
   - Used to create the "true" condition (1) in our generated if statements
*/

// ControlFlowObfuscatorVisitor wraps function/method bodies in if (true) {...} statements
// to make static analysis more difficult. It implements an AST visitor pattern that
// transforms specific statement types.
type ControlFlowObfuscatorVisitor struct {
	visitor.Null // Extends the Null visitor which provides default behavior
	DebugMode    bool
	// Track processed nodes to avoid infinite recursion during traversal
	processedNodes map[ast.Vertex]bool
	// Random source for condition generation
	random *rand.Rand
	// Flag to enable condition type randomization
	UseRandomConditions bool
	// Maximum nesting depth for if-true statements
	MaxNestingDepth int
	// Flag to enable adding bogus else branches
	AddDeadBranches bool
	// Flag to enable advanced loop obfuscation
	UseAdvancedLoopObfuscation bool
}

// NewControlFlowObfuscatorVisitor creates a new visitor for control flow obfuscation.
func NewControlFlowObfuscatorVisitor() *ControlFlowObfuscatorVisitor {
	// Use a time-based seed for true randomness
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &ControlFlowObfuscatorVisitor{
		DebugMode:                  false, // Default to no debug mode
		processedNodes:             make(map[ast.Vertex]bool),
		random:                     r,
		UseRandomConditions:        false, // Default to consistent conditions for testing
		MaxNestingDepth:            1,     // Default to single level nesting
		AddDeadBranches:            false, // Default to not adding bogus else branches
		UseAdvancedLoopObfuscation: false, // Default to not using advanced loop obfuscation
	}
}

// StmtFunction handles function nodes by wrapping their bodies in if(1){...} blocks.
// In the PHP AST, functions contain a slice of statements that form the function body.
// This method replaces that slice with a single if-statement containing the original statements.
func (v *ControlFlowObfuscatorVisitor) StmtFunction(n *ast.StmtFunction) {
	// Enhance debug output
	if v.DebugMode {
		funcName := "unknown"
		if n.Name != nil {
			if id, ok := n.Name.(*ast.Identifier); ok {
				funcName = string(id.Value)
			}
		}
		fmt.Fprintf(os.Stderr, "DEBUG: StmtFunction processing function '%s'\n", funcName)
	}

	// Avoid processing the same node multiple times (prevent infinite recursion)
	if v.processedNodes[n] {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Function already processed, skipping\n")
		}
		return
	}
	v.processedNodes[n] = true

	// Only process functions that have at least one statement
	if len(n.Stmts) > 0 {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Function '%s' has %d statements, wrapping with if (true)\n",
				func() string { // Inline function to safely get name
					if n.Name != nil {
						if id, ok := n.Name.(*ast.Identifier); ok {
							return string(id.Value)
						}
					}
					return "unknown"
				}(),
				len(n.Stmts))

			// Log the first few statements for debugging
			for i, stmt := range n.Stmts {
				if i >= 3 { // Only show first 3 statements
					break
				}
				fmt.Fprintf(os.Stderr, "DEBUG: Statement %d type: %T\n", i, stmt)
			}
		}

		// Store original statements for comparison
		originalStmtCount := len(n.Stmts)

		// Wrap the function body in if (true) {...}
		// This replaces the function's statement list with a single if statement
		// that contains all the original statements
		n.Stmts = wrapStmtsInIfTrueWithVisitor(n.Stmts, v)

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Function now has %d statements after wrapping (from %d)\n",
				len(n.Stmts), originalStmtCount)

			// Check if the wrapping worked
			if len(n.Stmts) == 1 {
				if ifStmt, ok := n.Stmts[0].(*ast.StmtIf); ok {
					fmt.Fprintf(os.Stderr, "DEBUG: Successfully wrapped in if statement with condition type: %T\n", ifStmt.Cond)
				} else {
					fmt.Fprintf(os.Stderr, "DEBUG: WARNING! First statement is not an if statement but %T\n", n.Stmts[0])
				}
			} else {
				fmt.Fprintf(os.Stderr, "DEBUG: WARNING! Expected 1 statement after wrapping, got %d\n", len(n.Stmts))
			}
		}
	} else {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Function has no statements, skipping wrapping\n")
		}
	}
}

// StmtClassMethod handles class method nodes by wrapping their bodies in if(1){...} blocks.
// In the PHP AST, class methods have a body represented by a single statement, typically a
// statement list (*ast.StmtStmtList). We modify this list to contain our obfuscated control flow.
func (v *ControlFlowObfuscatorVisitor) StmtClassMethod(n *ast.StmtClassMethod) {
	// Enhance debug output
	if v.DebugMode {
		methodName := "unknown"
		if n.Name != nil {
			if id, ok := n.Name.(*ast.Identifier); ok {
				methodName = string(id.Value)
			}
		}
		fmt.Fprintf(os.Stderr, "DEBUG: StmtClassMethod processing method '%s'\n", methodName)
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Method already processed, skipping\n")
		}
		return
	}
	v.processedNodes[n] = true

	// Class methods have a Stmt field which is usually a StmtStmtList
	if n.Stmt != nil {
		if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
			if len(stmtList.Stmts) > 0 {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "DEBUG: Method '%s' has %d statements in StmtList, wrapping with if (true)\n",
						func() string { // Inline function to safely get name
							if n.Name != nil {
								if id, ok := n.Name.(*ast.Identifier); ok {
									return string(id.Value)
								}
							}
							return "unknown"
						}(),
						len(stmtList.Stmts))

					// Log the first few statements for debugging
					for i, stmt := range stmtList.Stmts {
						if i >= 3 { // Only show first 3 statements
							break
						}
						fmt.Fprintf(os.Stderr, "DEBUG: Statement %d type: %T\n", i, stmt)
					}
				}

				// Store original statements for comparison
				originalStmtCount := len(stmtList.Stmts)

				// Wrap the method body in if (true) {...}
				// This modifies the statement list to contain a single if statement
				stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)

				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "DEBUG: Method now has %d statements after wrapping (from %d)\n",
						len(stmtList.Stmts), originalStmtCount)

					// Check if the wrapping worked
					if len(stmtList.Stmts) == 1 {
						if ifStmt, ok := stmtList.Stmts[0].(*ast.StmtIf); ok {
							fmt.Fprintf(os.Stderr, "DEBUG: Successfully wrapped in if statement with condition type: %T\n", ifStmt.Cond)
						} else {
							fmt.Fprintf(os.Stderr, "DEBUG: WARNING! First statement is not an if statement but %T\n", stmtList.Stmts[0])
						}
					} else {
						fmt.Fprintf(os.Stderr, "DEBUG: WARNING! Expected 1 statement after wrapping, got %d\n", len(stmtList.Stmts))
					}
				}
			} else {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "DEBUG: Method has empty statement list, skipping wrapping\n")
				}
			}
		} else {
			// This case is hit when the method body is a single statement, not a statement list
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "DEBUG: Method has single statement of type %T, not a StmtStmtList\n", n.Stmt)
			}

			// Replace the single statement with an if-true containing the original statement
			originalStmt := n.Stmt
			n.Stmt = wrapSingleStmtInIfTrueWithVisitor(originalStmt, v)

			if v.DebugMode {
				if _, ok := n.Stmt.(*ast.StmtIf); ok {
					fmt.Fprintf(os.Stderr, "DEBUG: Successfully wrapped single statement in if statement\n")
				} else {
					fmt.Fprintf(os.Stderr, "DEBUG: WARNING! Statement was not successfully wrapped, type is now: %T\n", n.Stmt)
				}
			}
		}
	} else {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Method has nil statement, skipping wrapping\n")
		}
	}
}

// StmtFor handles for loop nodes by wrapping their bodies in if(1){...} blocks.
// For loops in PHP AST have their body in the Stmt field, which can be either a
// statement list or a single statement.
func (v *ControlFlowObfuscatorVisitor) StmtFor(n *ast.StmtFor) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtFor called for a for loop\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Apply advanced loop obfuscation if enabled
	if v.UseAdvancedLoopObfuscation && v.UseRandomConditions {
		v.applyAdvancedLoopObfuscation(n)
		return
	}

	// For loops can have either a statement list or a single statement as their body
	// Handle both cases differently
	if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
		// Case 1: Loop body is a statement list
		if len(stmtList.Stmts) > 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  For loop has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
			}
			stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
		}
	} else if n.Stmt != nil {
		// Case 2: Loop body is a single statement
		// Wrap it in a statement list inside an if (true) block
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  For loop has a single statement, wrapping with if (true)\n")
		}
		n.Stmt = wrapSingleStmtInIfTrueWithVisitor(n.Stmt, v)
	}
}

// StmtWhile handles while loop nodes by wrapping their bodies in if(1){...} blocks.
// Similar to for loops, while loops have their body in the Stmt field.
func (v *ControlFlowObfuscatorVisitor) StmtWhile(n *ast.StmtWhile) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtWhile called for a while loop\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Apply advanced loop obfuscation if enabled
	if v.UseAdvancedLoopObfuscation && v.UseRandomConditions {
		v.applyAdvancedLoopObfuscation(n)
		return
	}

	// Handle both statement list and single statement cases
	if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
		if len(stmtList.Stmts) > 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  While loop has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
			}
			stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
		}
	} else if n.Stmt != nil {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  While loop has a single statement, wrapping with if (true)\n")
		}
		n.Stmt = wrapSingleStmtInIfTrueWithVisitor(n.Stmt, v)
	}
}

// StmtForeach handles foreach loop nodes by wrapping their bodies in if(1){...} blocks.
// Foreach loops in PHP iterate over arrays/collections and have a similar structure to other loops.
func (v *ControlFlowObfuscatorVisitor) StmtForeach(n *ast.StmtForeach) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtForeach called for a foreach loop\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Apply advanced loop obfuscation if enabled
	if v.UseAdvancedLoopObfuscation && v.UseRandomConditions {
		v.applyAdvancedLoopObfuscation(n)
		return
	}

	// Handle both statement list and single statement cases
	if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
		if len(stmtList.Stmts) > 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  Foreach loop has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
			}
			stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
		}
	} else if n.Stmt != nil {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Foreach loop has a single statement, wrapping with if (true)\n")
		}
		n.Stmt = wrapSingleStmtInIfTrueWithVisitor(n.Stmt, v)
	}
}

// StmtDo handles do-while loop nodes by wrapping their bodies in if(1){...} blocks.
// Do-while loops execute the body first, then check the condition, but the AST structure
// is similar to other loop types.
func (v *ControlFlowObfuscatorVisitor) StmtDo(n *ast.StmtDo) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtDo called for a do-while loop\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Apply advanced loop obfuscation if enabled
	if v.UseAdvancedLoopObfuscation && v.UseRandomConditions {
		v.applyAdvancedLoopObfuscation(n)
		return
	}

	// Handle both statement list and single statement cases
	if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
		if len(stmtList.Stmts) > 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  Do-while loop has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
			}
			stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
		}
	} else if n.Stmt != nil {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Do-while loop has a single statement, wrapping with if (true)\n")
		}
		n.Stmt = wrapSingleStmtInIfTrueWithVisitor(n.Stmt, v)
	}
}

// isObfuscationIfStatement checks if the if statement is already our generated obfuscation (if(1){}).
// This helps prevent duplicate obfuscation of already processed nodes.
// The function examines the condition to see if it's an always-true condition.
func isObfuscationIfStatement(ifStmt *ast.StmtIf) bool {
	// Check if the condition is a numeric literal (e.g., 1, 2, etc.)
	if _, ok := ifStmt.Cond.(*ast.ScalarLnumber); ok {
		return true
	}

	// Check for boolean true literal
	if scalar, ok := ifStmt.Cond.(*ast.ScalarString); ok {
		if string(scalar.Value) == "true" {
			return true
		}
	}

	// Check for !0 pattern
	if not, ok := ifStmt.Cond.(*ast.ExprBooleanNot); ok {
		if scalar, ok := not.Expr.(*ast.ScalarLnumber); ok {
			if string(scalar.Value) == "0" {
				return true
			}
		}
	}

	// Check for comparison expressions like 1==1, 2>=1, etc.
	if _, ok := ifStmt.Cond.(*ast.ExprBinaryEqual); ok {
		return true
	}
	if _, ok := ifStmt.Cond.(*ast.ExprBinaryGreater); ok {
		return true
	}
	if _, ok := ifStmt.Cond.(*ast.ExprBinaryGreaterOrEqual); ok {
		return true
	}
	if _, ok := ifStmt.Cond.(*ast.ExprBinaryLogicalOr); ok {
		return true
	}

	return false
}

// StmtIf handles if statement nodes by wrapping their bodies in if(1){...} blocks.
// If statements in PHP have more complex structure with possible elseif/else branches,
// all of which need to be obfuscated separately.
func (v *ControlFlowObfuscatorVisitor) StmtIf(n *ast.StmtIf) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtIf called for an if statement\n")
	}

	// Skip if this node has already been processed
	if v.processedNodes[n] {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  If statement already processed, skipping\n")
		}
		return
	}

	// Skip if this is already one of our obfuscation if statements (if(1){...})
	// This prevents "double-obfuscating" already transformed code
	if isObfuscationIfStatement(n) {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Skipping already obfuscated if statement (if(1))\n")
		}
		return
	}

	v.processedNodes[n] = true

	// Obfuscate the main "if" body (the "then" branch)
	if stmtList, ok := n.Stmt.(*ast.StmtStmtList); ok {
		if len(stmtList.Stmts) > 0 {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  If statement has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
			}
			stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
		}
	} else if n.Stmt != nil {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  If statement has a single statement, wrapping with if (true)\n")
		}
		n.Stmt = wrapSingleStmtInIfTrueWithVisitor(n.Stmt, v)
	}

	// Obfuscate the "else" branch if present
	if n.Else != nil {
		if elseStmt, ok := n.Else.(*ast.StmtElse); ok && elseStmt.Stmt != nil {
			if stmtList, ok := elseStmt.Stmt.(*ast.StmtStmtList); ok {
				if len(stmtList.Stmts) > 0 {
					if v.DebugMode {
						fmt.Fprintf(os.Stderr, "  Else statement has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
					}
					stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
				}
			} else {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "  Else statement has a single statement, wrapping with if (true)\n")
				}
				elseStmt.Stmt = wrapSingleStmtInIfTrueWithVisitor(elseStmt.Stmt, v)
			}
		}
	}

	// Process all "elseif" branches if any
	if n.ElseIf != nil {
		for i := range n.ElseIf {
			if elseIfStmt, ok := n.ElseIf[i].(*ast.StmtElseIf); ok && elseIfStmt.Stmt != nil {
				if stmtList, ok := elseIfStmt.Stmt.(*ast.StmtStmtList); ok {
					if len(stmtList.Stmts) > 0 {
						if v.DebugMode {
							fmt.Fprintf(os.Stderr, "  ElseIf statement has statement list with %d statements, wrapping with if (true)\n", len(stmtList.Stmts))
						}
						stmtList.Stmts = wrapStmtsInIfTrueWithVisitor(stmtList.Stmts, v)
					}
				} else {
					if v.DebugMode {
						fmt.Fprintf(os.Stderr, "  ElseIf statement has a single statement, wrapping with if (true)\n")
					}
					elseIfStmt.Stmt = wrapSingleStmtInIfTrueWithVisitor(elseIfStmt.Stmt, v)
				}
			}
		}
	}
}

// StmtSwitch handles switch statement nodes by wrapping the body of each case in if(1){...} blocks.
// In PHP AST, switch statements have Cases field containing a list of case statements.
func (v *ControlFlowObfuscatorVisitor) StmtSwitch(n *ast.StmtSwitch) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtSwitch called for a switch statement\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Process each case in the switch statement
	if n.Cases != nil && len(n.Cases) > 0 {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Switch has %d cases, processing each case\n", len(n.Cases))
		}

		// Iterate through each case/default statement
		for i, caseNode := range n.Cases {
			if caseStmt, ok := caseNode.(*ast.StmtCase); ok {
				if len(caseStmt.Stmts) > 0 {
					if v.DebugMode {
						fmt.Fprintf(os.Stderr, "  Case %d has %d statements, wrapping with if (true)\n",
							i, len(caseStmt.Stmts))
					}
					// Wrap the case body statements in if (true) {...}
					caseStmt.Stmts = wrapStmtsInIfTrueWithVisitor(caseStmt.Stmts, v)
				}
			} else if defaultStmt, ok := caseNode.(*ast.StmtDefault); ok {
				if len(defaultStmt.Stmts) > 0 {
					if v.DebugMode {
						fmt.Fprintf(os.Stderr, "  Default case has %d statements, wrapping with if (true)\n",
							len(defaultStmt.Stmts))
					}
					// Wrap the default case body statements in if (true) {...}
					defaultStmt.Stmts = wrapStmtsInIfTrueWithVisitor(defaultStmt.Stmts, v)
				}
			}
		}
	}
}

// StmtTry handles try statement nodes by wrapping the bodies of try, catch, and finally blocks
// in if(1){...} blocks. This preserves the error handling structure while making the control
// flow more complex for static analysis.
func (v *ControlFlowObfuscatorVisitor) StmtTry(n *ast.StmtTry) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "StmtTry called for a try-catch-finally block\n")
	}

	// Avoid processing the same node multiple times
	if v.processedNodes[n] {
		return
	}
	v.processedNodes[n] = true

	// Handle the try block
	if len(n.Stmts) > 0 {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Try block has %d statements, wrapping with if (true)\n", len(n.Stmts))
		}
		// Wrap the try block in if(true) {...}
		n.Stmts = wrapStmtsInIfTrueWithVisitor(n.Stmts, v)
	}

	// Handle all catch blocks
	for i, catchNode := range n.Catches {
		if catchStmt, ok := catchNode.(*ast.StmtCatch); ok {
			if len(catchStmt.Stmts) > 0 {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "  Catch block %d has %d statements, wrapping with if (true)\n",
						i, len(catchStmt.Stmts))
				}
				// Wrap the catch block in if(true) {...}
				catchStmt.Stmts = wrapStmtsInIfTrueWithVisitor(catchStmt.Stmts, v)
			}
		}
	}

	// Handle the finally block if present
	if n.Finally != nil {
		if finallyStmt, ok := n.Finally.(*ast.StmtFinally); ok {
			if len(finallyStmt.Stmts) > 0 {
				if v.DebugMode {
					fmt.Fprintf(os.Stderr, "  Finally block has %d statements, wrapping with if (true)\n",
						len(finallyStmt.Stmts))
				}
				// Wrap the finally block in if(true) {...}
				finallyStmt.Stmts = wrapStmtsInIfTrueWithVisitor(finallyStmt.Stmts, v)
			}
		}
	}
}

// createAlwaysTrueCondition generates a random PHP expression that always evaluates to true.
// It selects from various patterns to make the obfuscated code less predictable.
func (v *ControlFlowObfuscatorVisitor) createAlwaysTrueCondition() ast.Vertex {
	// Unless randomization is explicitly enabled, always return the numeric literal '1'
	// This ensures test consistency while allowing randomization when needed
	if !v.UseRandomConditions {
		return &ast.ScalarLnumber{
			Value: []byte("1"),
		}
	}

	// Choose a random condition type when randomization is enabled
	conditionType := v.random.Intn(6)

	switch conditionType {
	case 0:
		// Simple numeric literal (1, 2, etc.)
		return &ast.ScalarLnumber{
			Value: []byte(fmt.Sprintf("%d", v.random.Intn(10)+1)), // Random number 1-10
		}

	case 1:
		// Boolean true as numeric literal
		return &ast.ScalarLnumber{
			Value: []byte("1"),
		}

	case 2:
		// Logical NOT of 0: !0
		return &ast.ExprBooleanNot{
			Expr: &ast.ScalarLnumber{
				Value: []byte("0"),
			},
		}

	case 3:
		// Equality comparison: 1==1, 2==2, etc.
		num := v.random.Intn(10) + 1 // Random number 1-10
		return &ast.ExprBinaryEqual{
			Left: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", num)),
			},
			Right: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", num)),
			},
		}

	case 4:
		// Greater than comparison: 2>1, 5>3, etc.
		right := v.random.Intn(5) + 1        // Random number 1-5
		left := right + v.random.Intn(5) + 1 // Random number (right+1) to (right+5)
		return &ast.ExprBinaryGreater{
			Left: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", left)),
			},
			Right: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", right)),
			},
		}

	default: // case 5
		// Logical OR with true: 1 || 0
		return &ast.ExprBinaryLogicalOr{
			Left: &ast.ScalarLnumber{
				Value: []byte("1"),
			},
			Right: &ast.ScalarLnumber{
				Value: []byte("0"),
			},
		}
	}
}

// wrapStmtsInIfTrue wraps a list of statements in if (true) {...}.
// This helper function creates a new if statement node with a condition that always
// evaluates to true and puts the provided statements inside it.
//
// AST Transformation:
// [stmt1, stmt2, ...] -> [if(RANDOM_TRUE_CONDITION){ stmt1; stmt2; ... }]
func wrapStmtsInIfTrue(stmts []ast.Vertex) []ast.Vertex {
	// Create a visitor instance for condition generation
	v := NewControlFlowObfuscatorVisitor()

	return wrapStmtsInIfTrueWithVisitor(stmts, v)
}

// wrapStmtsInIfTrueWithVisitor is an internal helper that wraps statements using the provided visitor
// This allows the randomization settings to be preserved and supports nested wrapping based on MaxNestingDepth
func wrapStmtsInIfTrueWithVisitor(stmts []ast.Vertex, v *ControlFlowObfuscatorVisitor) []ast.Vertex {
	return wrapStmtsInIfTrueWithVisitorAndDepth(stmts, v, 1)
}

// wrapStmtsInIfTrueWithVisitorAndDepth is an internal helper that wraps statements using the provided visitor
// with support for nested if statements up to the specified max depth.
// This recursively wraps the statements in if(true){...} blocks based on the current depth and max depth.
func wrapStmtsInIfTrueWithVisitorAndDepth(stmts []ast.Vertex, v *ControlFlowObfuscatorVisitor, currentDepth int) []ast.Vertex {
	// Create a random true condition
	trueCondition := v.createAlwaysTrueCondition()

	// If we've reached the maximum nesting depth, just wrap once and return
	if currentDepth >= v.MaxNestingDepth {
		// Create the if statement with the true condition and original statements
		ifStmt := &ast.StmtIf{
			Cond:   trueCondition,                   // Random condition that's always true
			Stmt:   &ast.StmtStmtList{Stmts: stmts}, // Body: original statements
			ElseIf: nil,                             // No elseif branches
			Else:   nil,                             // No else branch
		}

		// Add bogus else branch if enabled
		if v.AddDeadBranches {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "  Adding bogus else branch to if statement\n")
			}

			// Create a random bogus code block for the else branch
			bogusCode := v.createBogusCode()

			// Create else statement with the bogus code
			elseStmt := &ast.StmtElse{
				Stmt: bogusCode,
			}

			// Add the else branch to the if statement
			ifStmt.Else = elseStmt
		}

		// Return a new statement list with just the if statement
		return []ast.Vertex{ifStmt}
	}

	// For nested wrapping, first wrap the statements in an if block
	stmtList := &ast.StmtStmtList{Stmts: stmts}

	// Then, recursively wrap this if block in another if block for the next level of nesting
	nestedIfStmt := &ast.StmtIf{
		Cond:   trueCondition, // Random condition that's always true
		Stmt:   stmtList,      // Body: the original statements
		ElseIf: nil,           // No elseif branches
		Else:   nil,           // No else branch
	}

	// Add bogus else branch to the innermost if statement if enabled
	if v.AddDeadBranches && currentDepth == 1 {
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "  Adding bogus else branch to innermost nested if statement\n")
		}

		// Create a random bogus code block for the else branch
		bogusCode := v.createBogusCode()

		// Create else statement with the bogus code
		elseStmt := &ast.StmtElse{
			Stmt: bogusCode,
		}

		// Add the else branch to the if statement
		nestedIfStmt.Else = elseStmt
	}

	// If we need more nesting levels, recursively wrap this if statement
	if currentDepth < v.MaxNestingDepth {
		// Recursively wrap the if statement in more if statements
		return wrapStmtsInIfTrueWithVisitorAndDepth([]ast.Vertex{nestedIfStmt}, v, currentDepth+1)
	}

	// Return a new statement list with just the if statement
	return []ast.Vertex{nestedIfStmt}
}

// wrapSingleStmtInIfTrue wraps a single statement in if (true) {...}.
// Similar to wrapStmtsInIfTrue but designed for a single statement.
// It creates a statement list containing just that statement.
//
// AST Transformation:
// stmt -> if(RANDOM_TRUE_CONDITION){ stmt; }
func wrapSingleStmtInIfTrue(stmt ast.Vertex) ast.Vertex {
	// Create a visitor instance for condition generation
	v := NewControlFlowObfuscatorVisitor()

	return wrapSingleStmtInIfTrueWithVisitor(stmt, v)
}

// wrapSingleStmtInIfTrueWithVisitor is an internal helper that wraps a single statement using the provided visitor
// This allows the randomization settings to be preserved
func wrapSingleStmtInIfTrueWithVisitor(stmt ast.Vertex, v *ControlFlowObfuscatorVisitor) ast.Vertex {
	// Wrap the single statement in a statement list, then wrap that list in if(true)
	wrappedStmts := wrapStmtsInIfTrueWithVisitor([]ast.Vertex{stmt}, v)

	// Return the first (and only) statement from the wrapped list, which should be an if statement
	return wrappedStmts[0]
}

// createBogusCode generates a syntactically valid but unreachable code block
// for use in else branches of always-true if statements.
// This adds complexity to the code without affecting functionality.
func (v *ControlFlowObfuscatorVisitor) createBogusCode() ast.Vertex {
	// Create a statement list to hold our bogus statements
	stmts := make([]ast.Vertex, 0)

	// Choose a random number of statements to generate (1-3)
	numStatements := 1
	if v.UseRandomConditions {
		numStatements = v.random.Intn(3) + 1
	}

	// Generate the specified number of bogus statements
	for i := 0; i < numStatements; i++ {
		// Choose a random statement type
		stmtType := 0
		if v.UseRandomConditions {
			stmtType = v.random.Intn(5)
		}

		var stmt ast.Vertex
		switch stmtType {
		case 0:
			// Variable assignment: $x = 1;
			variableName := "$_" + randomVariable(v)
			stmt = &ast.StmtExpression{
				Expr: &ast.ExprAssign{
					Var: &ast.ExprVariable{
						Name: &ast.Identifier{
							Value: []byte(variableName),
						},
					},
					Expr: &ast.ScalarLnumber{
						Value: []byte(fmt.Sprintf("%d", v.random.Intn(100))),
					},
				},
			}

		case 1:
			// Echo statement: echo "Dead code";
			messageIdx := v.random.Intn(5)
			messages := []string{
				"Unreachable code",
				"Dead branch",
				"This will never execute",
				"Bogus code path",
				"Dummy statement",
			}
			message := messages[messageIdx]

			stmt = &ast.StmtEcho{
				Exprs: []ast.Vertex{
					&ast.ScalarString{
						Value: []byte("\"" + message + "\""),
					},
				},
			}

		case 2:
			// Return statement with a value: return false;
			stmt = &ast.StmtReturn{
				Expr: &ast.ScalarString{
					Value: []byte("false"),
				},
			}

		case 3:
			// Break statement
			stmt = &ast.StmtBreak{}

		case 4:
			// Continue statement
			stmt = &ast.StmtContinue{}
		}

		if stmt != nil {
			stmts = append(stmts, stmt)
		}
	}

	// Return a statement list with our bogus statements
	return &ast.StmtStmtList{
		Stmts: stmts,
	}
}

// randomVariable generates a random variable name for use in bogus code
func randomVariable(v *ControlFlowObfuscatorVisitor) string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	length := 5
	if v.UseRandomConditions {
		length = v.random.Intn(5) + 3 // Random length between 3-7
	}

	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[v.random.Intn(len(chars))]
	}
	return string(result)
}

// applyAdvancedLoopObfuscation applies more complex obfuscation techniques to loop structures
// This function handles all types of loops (for, while, foreach, do-while)
func (v *ControlFlowObfuscatorVisitor) applyAdvancedLoopObfuscation(loop ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Applying advanced loop obfuscation to %T\n", loop)
	}

	// Determine which type of loop we're dealing with and extract its statement
	var stmt ast.Vertex
	var cond ast.Vertex

	switch n := loop.(type) {
	case *ast.StmtFor:
		stmt = n.Stmt
		if len(n.Cond) > 0 {
			cond = n.Cond[0] // For simplicity, use the first condition if there are multiple
		}
	case *ast.StmtWhile:
		stmt = n.Stmt
		cond = n.Cond
	case *ast.StmtForeach:
		stmt = n.Stmt
	case *ast.StmtDo:
		stmt = n.Stmt
		cond = n.Cond
	default:
		// Unknown loop type, return without changes
		return
	}

	// Apply one of several obfuscation techniques randomly
	obfuscationType := v.random.Intn(3) // Choose between 0-2 obfuscation types

	switch obfuscationType {
	case 0:
		// Technique 1: Insert continue inside nested conditional
		v.insertConditionalContinue(stmt)
	case 1:
		// Technique 2: Add condition check redundancy
		v.addRedundantConditionCheck(stmt, cond)
	case 2:
		// Technique 3: Add sentinel variable
		v.addSentinelVariable(stmt)
	}
}

// insertConditionalContinue inserts a conditional continue statement that never executes
// This makes the control flow analysis more complex
func (v *ControlFlowObfuscatorVisitor) insertConditionalContinue(stmt ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Inserting conditional continue in loop body\n")
	}

	var stmtList *ast.StmtStmtList
	var stmts []ast.Vertex

	// Extract the statement list from the loop body
	if sl, ok := stmt.(*ast.StmtStmtList); ok {
		stmtList = sl
		stmts = sl.Stmts
	} else if stmt != nil {
		// If the statement is not a list, create a single-element list
		stmtList = &ast.StmtStmtList{}
		stmts = []ast.Vertex{stmt}
	} else {
		// No statement to modify
		return
	}

	// Create a condition that always evaluates to false
	falseCondition := v.createAlwaysFalseCondition()

	// Create an if statement with the false condition and a continue statement
	ifStmt := &ast.StmtIf{
		Cond: falseCondition,
		Stmt: &ast.StmtStmtList{
			Stmts: []ast.Vertex{
				&ast.StmtContinue{},
			},
		},
		ElseIf: nil,
		Else:   nil,
	}

	// Insert the if-continue statement at a random position in the statement list
	if len(stmts) > 0 {
		position := v.random.Intn(len(stmts) + 1) // +1 to allow insertion at the end

		if position == len(stmts) {
			// Append at the end
			stmts = append(stmts, ifStmt)
		} else {
			// Insert in the middle
			stmts = append(stmts[:position+1], stmts[position:]...)
			stmts[position] = ifStmt
		}
	} else {
		// Empty statement list, just add our if-continue
		stmts = []ast.Vertex{ifStmt}
	}

	// Update the statement list
	stmtList.Stmts = stmts

	// If the original statement was not a list, update the reference
	if _, ok := stmt.(*ast.StmtStmtList); !ok {
		switch n := stmt.(type) {
		case *ast.StmtFor:
			n.Stmt = stmtList
		case *ast.StmtWhile:
			n.Stmt = stmtList
		case *ast.StmtForeach:
			n.Stmt = stmtList
		case *ast.StmtDo:
			n.Stmt = stmtList
		}
	}
}

// addRedundantConditionCheck adds a redundant check of the loop condition
// inside the loop body
func (v *ControlFlowObfuscatorVisitor) addRedundantConditionCheck(stmt ast.Vertex, cond ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Adding redundant condition check in loop body\n")
	}

	// If no condition to duplicate, fall back to basic obfuscation
	if cond == nil {
		v.insertConditionalContinue(stmt)
		return
	}

	var stmtList *ast.StmtStmtList
	var stmts []ast.Vertex

	// Extract the statement list from the loop body
	if sl, ok := stmt.(*ast.StmtStmtList); ok {
		stmtList = sl
		stmts = sl.Stmts
	} else if stmt != nil {
		// If the statement is not a list, create a single-element list
		stmtList = &ast.StmtStmtList{}
		stmts = []ast.Vertex{stmt}
	} else {
		// No statement to modify
		return
	}

	// Create a copy of the original condition
	// We'll use this in an if statement that always executes
	// If condition is complex, this makes analysis harder

	// Create an if statement with the condition and the original statements
	ifStmt := &ast.StmtIf{
		Cond: cond,
		Stmt: &ast.StmtStmtList{
			Stmts: stmts,
		},
		ElseIf: nil,
		Else:   nil,
	}

	// Add a redundant else branch that never executes
	elseStmt := &ast.StmtElse{
		Stmt: &ast.StmtBreak{}, // Add a break to make it look like it might execute
	}
	ifStmt.Else = elseStmt

	// Replace the statement list with our if statement
	stmtList.Stmts = []ast.Vertex{ifStmt}
}

// addSentinelVariable adds an additional variable that tracks loop execution
// but doesn't affect the actual logic
func (v *ControlFlowObfuscatorVisitor) addSentinelVariable(stmt ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Adding sentinel variable in loop body\n")
	}

	var stmtList *ast.StmtStmtList
	var stmts []ast.Vertex

	// Extract the statement list from the loop body
	if sl, ok := stmt.(*ast.StmtStmtList); ok {
		stmtList = sl
		stmts = sl.Stmts
	} else if stmt != nil {
		// If the statement is not a list, create a single-element list
		stmtList = &ast.StmtStmtList{}
		stmts = []ast.Vertex{stmt}
	} else {
		// No statement to modify
		return
	}

	// Generate a random variable name for our sentinel
	sentinelVar := "$_" + randomVariable(v)

	// Create a variable initialization statement at the beginning
	initStmt := &ast.StmtExpression{
		Expr: &ast.ExprAssign{
			Var: &ast.ExprVariable{
				Name: &ast.Identifier{
					Value: []byte(sentinelVar),
				},
			},
			Expr: &ast.ScalarLnumber{
				Value: []byte("0"),
			},
		},
	}

	// Create an increment statement for the end of each iteration
	incrStmt := &ast.StmtExpression{
		Expr: &ast.ExprAssign{
			Var: &ast.ExprVariable{
				Name: &ast.Identifier{
					Value: []byte(sentinelVar),
				},
			},
			Expr: &ast.ExprBinaryPlus{
				Left: &ast.ExprVariable{
					Name: &ast.Identifier{
						Value: []byte(sentinelVar),
					},
				},
				Right: &ast.ScalarLnumber{
					Value: []byte("1"),
				},
			},
		},
	}

	// Create a conditional that uses the sentinel but doesn't affect execution
	checkStmt := &ast.StmtIf{
		Cond: &ast.ExprBinaryGreater{
			Left: &ast.ExprVariable{
				Name: &ast.Identifier{
					Value: []byte(sentinelVar),
				},
			},
			Right: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", 1000)), // Some large number that won't be reached
			},
		},
		Stmt: &ast.StmtStmtList{
			Stmts: []ast.Vertex{
				&ast.StmtBreak{},
			},
		},
	}

	// Add our statements to the statement list
	newStmts := []ast.Vertex{initStmt}
	newStmts = append(newStmts, stmts...)

	// Insert the check statement at a random position in the middle of the loop
	if len(stmts) > 1 {
		position := 1 // Default position after the first statement
		maxPos := len(stmts) - 1
		if maxPos > 0 {
			position = v.random.Intn(maxPos) + 1 // Ensure it's not at the beginning
		}

		// Fix the append operation to correctly insert at position
		// Create a new slice with the correct insertion
		result := make([]ast.Vertex, 0, len(newStmts)+1)
		result = append(result, newStmts[:position]...)
		result = append(result, checkStmt)
		result = append(result, newStmts[position:]...)
		newStmts = result
	} else {
		// If there's only one statement, add it after that statement
		newStmts = append(newStmts, checkStmt)
	}

	// Add the increment at the end
	newStmts = append(newStmts, incrStmt)

	// Update the statement list
	stmtList.Stmts = newStmts
}

// createAlwaysFalseCondition creates a condition that always evaluates to false
// Used for creating conditional branches that never execute
func (v *ControlFlowObfuscatorVisitor) createAlwaysFalseCondition() ast.Vertex {
	// Choose a random type of condition based on the visitor's randomization setting
	conditionType := 0
	if v.UseRandomConditions {
		conditionType = v.random.Intn(4)
	}

	var condition ast.Vertex

	switch conditionType {
	case 0:
		// Simple false: 0
		condition = &ast.ScalarLnumber{
			Value: []byte("0"),
		}

	case 1:
		// Numeric zero: 0
		condition = &ast.ScalarLnumber{
			Value: []byte("0"),
		}

	case 2:
		// Equality: 1 == 2
		condition = &ast.ExprBinaryEqual{
			Left: &ast.ScalarLnumber{
				Value: []byte("1"),
			},
			Right: &ast.ScalarLnumber{
				Value: []byte("2"),
			},
		}

	case 3:
		// Less than: 1 < 0
		condition = &ast.ExprBinarySmaller{
			Left: &ast.ScalarLnumber{
				Value: []byte("1"),
			},
			Right: &ast.ScalarLnumber{
				Value: []byte("0"),
			},
		}
	}

	return condition
}
