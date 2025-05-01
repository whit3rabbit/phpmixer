// Package transformer provides AST transformation utilities for PHP obfuscation.
package transformer

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/token"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

/*
Dead Code and Junk Code Obfuscation Overview:
--------------------------------------------
This file implements a visitor pattern to inject dead code (code that has no effect on program execution)
and junk code (non-functional code like unused variables and pointless calculations) into PHP code.

Examples of transformations:
- Inject if(false) {...} blocks with complex-looking but unreachable code
- Add unused variable declarations with calculations
- Insert useless function calls that don't affect program state
- Add no-op operations to existing code (like $x = $x * 1)

The goal is to make the code harder to understand by adding noise that has no functional impact.
*/

// DeadCodeInserterVisitor injects dead code and junk code to obfuscate the program
type DeadCodeInserterVisitor struct {
	visitor.Null // Extends the Null visitor which provides default behavior
	// Random source for code generation
	random *rand.Rand
	// Track processed nodes to avoid infinite recursion
	processedNodes map[ast.Vertex]bool
	// Whether to enable debug output
	DebugMode bool
	// Replacements to be applied
	replacements []*NodeReplacement
	// Rate of code injection (percentage chance to inject at each opportunity)
	InjectionRate int
	// Whether to inject dead code blocks (if(false){...})
	InjectDeadCodeBlocks bool
	// Whether to inject junk statements (unused variables, no-op calculations)
	InjectJunkStatements bool
	// Maximum depth to prevent excessive injection in nested structures
	MaxInjectionDepth int
	// Current depth during traversal
	currentDepth int
}

// NewDeadCodeInserterVisitor creates a new visitor for injecting dead code and junk statements
func NewDeadCodeInserterVisitor() *DeadCodeInserterVisitor {
	return &DeadCodeInserterVisitor{
		random:               rand.New(rand.NewSource(time.Now().UnixNano())),
		processedNodes:       make(map[ast.Vertex]bool),
		replacements:         make([]*NodeReplacement, 0),
		DebugMode:            false,
		InjectionRate:        50, // 50% chance to inject code at each opportunity by default
		InjectDeadCodeBlocks: true,
		InjectJunkStatements: true,
		MaxInjectionDepth:    3,
		currentDepth:         0,
	}
}

// EnterNode is called when entering a node during traversal
// Implements the NodeReplacer interface
func (v *DeadCodeInserterVisitor) EnterNode(n ast.Vertex) bool {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "DeadCodeInserter Enter: %T\n", n)
	}

	// Skip already processed nodes to avoid infinite recursion
	if v.processedNodes[n] {
		return false
	}

	// If we've reached max depth, don't inject further
	if v.currentDepth >= v.MaxInjectionDepth {
		return true
	}

	v.processedNodes[n] = true
	v.currentDepth++

	// Only inject at certain points in the AST
	switch node := n.(type) {
	case *ast.Root:
		// Handle root node's statements (which isn't a StmtStmtList but has Stmts)
		v.injectIntoRootStmts(node)
	case *ast.StmtStmtList:
		// Good place to inject statements between existing ones
		v.injectIntoStmtList(node)
	case *ast.StmtFunction:
		// We can inject at the beginning of function bodies
		v.injectIntoFunctionBody(node)
	case *ast.StmtClassMethod:
		// We can inject at the beginning of method bodies
		v.injectIntoMethodBody(node)
	case *ast.StmtIf:
		// We might want to inject in the if/else branches
		v.injectIntoIfStatement(node)
	case *ast.StmtFor, *ast.StmtForeach, *ast.StmtWhile, *ast.StmtDo:
		// We might want to inject into loop bodies
		v.injectIntoLoopBody(node)
	}

	return true
}

// GetReplacement checks if there's a replacement for the given node
// Implements the NodeReplacer interface
func (v *DeadCodeInserterVisitor) GetReplacement(n ast.Vertex) (ast.Vertex, bool) {
	for _, rep := range v.replacements {
		if rep.Original == n {
			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Found replacement for %T\n", n)
			}
			return rep.Replacement, true
		}
	}
	return nil, false
}

// LeaveNode is called when leaving a node during traversal
// Implements the NodeReplacer interface
func (v *DeadCodeInserterVisitor) LeaveNode(n ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "DeadCodeInserter Leave: %T\n", n)
	}
	v.currentDepth--
}

// AddReplacement adds a node replacement
func (v *DeadCodeInserterVisitor) AddReplacement(original ast.Vertex, replacement ast.Vertex) {
	v.replacements = append(v.replacements, &NodeReplacement{
		Original:    original,
		Replacement: replacement,
	})
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Added replacement: %T -> %T\n", original, replacement)
	}
}

// injectIntoStmtList injects dead code and junk statements into a statement list
func (v *DeadCodeInserterVisitor) injectIntoStmtList(stmtList *ast.StmtStmtList) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	if stmtList == nil || len(stmtList.Stmts) == 0 {
		return
	}

	newStmts := make([]ast.Vertex, 0, len(stmtList.Stmts)+2) // Reserve space for potential injections

	// Have a chance to inject at the beginning of the list
	if v.random.Intn(100) < v.InjectionRate {
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
	}

	// Insert the original statements, possibly with injections between them
	for i, stmt := range stmtList.Stmts {
		newStmts = append(newStmts, stmt)

		// Don't inject after the last statement to prevent issues with control flow
		if i < len(stmtList.Stmts)-1 && v.random.Intn(100) < v.InjectionRate {
			newStmts = append(newStmts, v.generateJunkOrDeadCode())
		}
	}

	// Replace the original statement list if we've made changes
	if len(newStmts) > len(stmtList.Stmts) {
		newStmtList := &ast.StmtStmtList{
			Stmts: newStmts,
		}
		v.AddReplacement(stmtList, newStmtList)

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into statement list\n")
		}
	}
}

// injectIntoFunctionBody injects dead code at the beginning of a function body
func (v *DeadCodeInserterVisitor) injectIntoFunctionBody(function *ast.StmtFunction) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	// If the function has a statement list body, we can manipulate it
	if function.Stmts != nil && len(function.Stmts) > 0 {
		// Insert a junk statement at the beginning
		newStmts := make([]ast.Vertex, 0, len(function.Stmts)+1)
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
		newStmts = append(newStmts, function.Stmts...)

		function.Stmts = newStmts

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into function body\n")
		}
	}
}

// injectIntoMethodBody injects dead code at the beginning of a method body
func (v *DeadCodeInserterVisitor) injectIntoMethodBody(method *ast.StmtClassMethod) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	// For methods, the body is typically a single StmtStmtList
	if stmtList, ok := method.Stmt.(*ast.StmtStmtList); ok {
		// Insert a junk statement at the beginning
		newStmts := make([]ast.Vertex, 0, len(stmtList.Stmts)+1)
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
		newStmts = append(newStmts, stmtList.Stmts...)

		newStmtList := &ast.StmtStmtList{
			Stmts: newStmts,
		}

		v.AddReplacement(method.Stmt, newStmtList)

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into method body\n")
		}
	}
}

// injectIntoIfStatement injects dead code into if/else branches
func (v *DeadCodeInserterVisitor) injectIntoIfStatement(ifStmt *ast.StmtIf) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	// For if statements, we can inject into the 'if' branch and potentially 'else' branch
	if stmtList, ok := ifStmt.Stmt.(*ast.StmtStmtList); ok {
		// Insert a junk statement into the if branch
		newStmts := make([]ast.Vertex, 0, len(stmtList.Stmts)+1)
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
		newStmts = append(newStmts, stmtList.Stmts...)

		newStmtList := &ast.StmtStmtList{
			Stmts: newStmts,
		}

		v.AddReplacement(ifStmt.Stmt, newStmtList)

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into if branch\n")
		}
	}

	// Also inject into the else branch if it exists
	if ifStmt.Else != nil {
		if elseStmtList, ok := ifStmt.Else.(*ast.StmtStmtList); ok {
			// Insert a junk statement into the else branch
			newStmts := make([]ast.Vertex, 0, len(elseStmtList.Stmts)+1)
			newStmts = append(newStmts, v.generateJunkOrDeadCode())
			newStmts = append(newStmts, elseStmtList.Stmts...)

			newStmtList := &ast.StmtStmtList{
				Stmts: newStmts,
			}

			v.AddReplacement(ifStmt.Else, newStmtList)

			if v.DebugMode {
				fmt.Fprintf(os.Stderr, "Injected code into else branch\n")
			}
		}
	}
}

// injectIntoLoopBody injects dead code into loop bodies
func (v *DeadCodeInserterVisitor) injectIntoLoopBody(loopStmt ast.Vertex) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	var stmtNode ast.Vertex

	// Extract the statement node based on loop type
	switch node := loopStmt.(type) {
	case *ast.StmtFor:
		stmtNode = node.Stmt
	case *ast.StmtForeach:
		stmtNode = node.Stmt
	case *ast.StmtWhile:
		stmtNode = node.Stmt
	case *ast.StmtDo:
		stmtNode = node.Stmt
	default:
		return // Unknown loop type
	}

	// If the loop body is a statement list, we can inject into it
	if stmtList, ok := stmtNode.(*ast.StmtStmtList); ok {
		// Insert a junk statement at the beginning of the loop body
		newStmts := make([]ast.Vertex, 0, len(stmtList.Stmts)+1)
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
		newStmts = append(newStmts, stmtList.Stmts...)

		newStmtList := &ast.StmtStmtList{
			Stmts: newStmts,
		}

		v.AddReplacement(stmtNode, newStmtList)

		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into loop body\n")
		}
	}
}

// injectIntoRootStmts injects dead code and junk statements into Root.Stmts
func (v *DeadCodeInserterVisitor) injectIntoRootStmts(root *ast.Root) {
	// Only inject if we're below the injection rate threshold
	if v.random.Intn(100) >= v.InjectionRate {
		return
	}

	if root == nil || len(root.Stmts) == 0 {
		return
	}

	newStmts := make([]ast.Vertex, 0, len(root.Stmts)+2) // Reserve space for potential injections

	// Have a chance to inject at the beginning
	if v.random.Intn(100) < v.InjectionRate {
		newStmts = append(newStmts, v.generateJunkOrDeadCode())
	}

	// Insert the original statements, possibly with injections between them
	for i, stmt := range root.Stmts {
		newStmts = append(newStmts, stmt)

		// Don't inject after the last statement to prevent issues with control flow
		if i < len(root.Stmts)-1 && v.random.Intn(100) < v.InjectionRate {
			newStmts = append(newStmts, v.generateJunkOrDeadCode())
		}
	}

	// Replace the original statements if we've made changes
	if len(newStmts) > len(root.Stmts) {
		root.Stmts = newStmts
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Injected code into root statement list\n")
		}
	}
}

// generateJunkOrDeadCode creates either a dead code block or junk statements based on configuration
func (v *DeadCodeInserterVisitor) generateJunkOrDeadCode() ast.Vertex {
	// Randomly choose between dead code blocks and junk statements
	if v.InjectDeadCodeBlocks && v.InjectJunkStatements {
		if v.random.Intn(2) == 0 {
			return v.generateDeadCodeBlock()
		} else {
			return v.generateJunkStatement()
		}
	} else if v.InjectDeadCodeBlocks {
		return v.generateDeadCodeBlock()
	} else {
		return v.generateJunkStatement()
	}
}

// generateDeadCodeBlock creates a block of code that will never execute (if(false){...})
func (v *DeadCodeInserterVisitor) generateDeadCodeBlock() ast.Vertex {
	// Create a false condition for the if statement
	var condition ast.Vertex

	// Choose a random way to represent 'false'
	conditionType := v.random.Intn(4)
	switch conditionType {
	case 0:
		// false
		condition = &ast.ScalarString{
			Value: []byte("false"),
		}
	case 1:
		// 0
		condition = &ast.ScalarLnumber{Value: []byte("0")}
	case 2:
		// 1 == 2
		condition = &ast.ExprBinaryEqual{
			Left:  &ast.ScalarLnumber{Value: []byte("1")},
			Right: &ast.ScalarLnumber{Value: []byte("2")},
		}
	case 3:
		// "a" === "b"
		condition = &ast.ExprBinaryIdentical{
			Left: &ast.ScalarString{
				Value: []byte("'a'"),
			},
			Right: &ast.ScalarString{
				Value: []byte("'b'"),
			},
		}
	}

	// Create the dead code statements to put inside the if block
	stmts := make([]ast.Vertex, 0, v.random.Intn(3)+1) // 1-3 statements

	for i := 0; i < cap(stmts); i++ {
		stmts = append(stmts, v.generateJunkStatement())
	}

	// Create the if statement with the false condition and dead code
	return &ast.StmtIf{
		Cond: condition,
		Stmt: &ast.StmtStmtList{
			Stmts: stmts,
		},
	}
}

// generateJunkStatement creates a statement that has no effect on program execution
func (v *DeadCodeInserterVisitor) generateJunkStatement() ast.Vertex {
	// Choose a random type of junk statement
	stmtType := v.random.Intn(3)

	switch stmtType {
	case 0:
		// Unused variable assignment: $junk123 = [random expression];
		return v.generateUnusedVariableAssignment()
	case 1:
		// No-op calculation: $x = $x * 1; or $x = $x + 0;
		return v.generateNoopCalculation()
	case 2:
		// Useless function call: time(); or rand();
		return v.generateUselessFunctionCall()
	default:
		return v.generateUnusedVariableAssignment()
	}
}

// generateUnusedVariableAssignment creates an assignment to a variable that won't be used
func (v *DeadCodeInserterVisitor) generateUnusedVariableAssignment() ast.Vertex {
	// Generate a random variable name
	varName := fmt.Sprintf("_junk%d", v.random.Intn(10000))

	// Create the variable expression
	variable := &ast.ExprVariable{
		Name: &ast.Identifier{
			Value: []byte(varName),
		},
	}

	// Create a random expression for the right side
	var expr ast.Vertex
	exprType := v.random.Intn(5)

	switch exprType {
	case 0:
		// Random number
		expr = &ast.ScalarLnumber{
			Value: []byte(fmt.Sprintf("%d", v.random.Intn(1000))),
		}
	case 1:
		// Random string
		randomString := fmt.Sprintf("junk%d", v.random.Intn(1000))
		expr = &ast.ScalarString{
			Value: []byte(fmt.Sprintf("'%s'", randomString)),
			StringTkn: &token.Token{
				Value: []byte(fmt.Sprintf("'%s'", randomString)),
			},
		}
	case 2:
		// Random array
		expr = &ast.ExprArray{
			Items: []ast.Vertex{
				&ast.ExprArrayItem{
					Val: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", v.random.Intn(100)))},
				},
				&ast.ExprArrayItem{
					Val: &ast.ScalarString{
						Value: []byte(fmt.Sprintf("'item%d'", v.random.Intn(100))),
						StringTkn: &token.Token{
							Value: []byte(fmt.Sprintf("'item%d'", v.random.Intn(100))),
						},
					},
				},
			},
		}
	case 3:
		// time() call
		expr = &ast.ExprFunctionCall{
			Function: &ast.Name{
				Parts: []ast.Vertex{
					&ast.NamePart{Value: []byte("time")},
				},
			},
			Args: []ast.Vertex{},
		}
	case 4:
		// Binary expression
		num1 := v.random.Intn(100)
		num2 := v.random.Intn(100)

		expr = &ast.ExprBinaryPlus{
			Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", num1))},
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", num2))},
		}
	}

	// Create the assignment expression
	assignment := &ast.ExprAssign{
		Var:  variable,
		Expr: expr,
	}

	// Wrap in a statement
	return &ast.StmtExpression{
		Expr: assignment,
	}
}

// generateNoopCalculation creates a calculation that has no effect (e.g., $x = $x * 1)
func (v *DeadCodeInserterVisitor) generateNoopCalculation() ast.Vertex {
	// Generate a random variable name that looks like a temp var
	varName := fmt.Sprintf("_temp%d", v.random.Intn(1000))

	// Create the variable expression
	variable := &ast.ExprVariable{
		Name: &ast.Identifier{
			Value: []byte(varName),
		},
	}

	// First create an initialization for the variable
	init := &ast.StmtExpression{
		Expr: &ast.ExprAssign{
			Var: variable,
			Expr: &ast.ScalarLnumber{
				Value: []byte(fmt.Sprintf("%d", v.random.Intn(100))),
			},
		},
	}

	// Then create a no-op calculation with the variable
	var noop ast.Vertex
	noopType := v.random.Intn(4)

	switch noopType {
	case 0:
		// $x = $x * 1
		noop = &ast.StmtExpression{
			Expr: &ast.ExprAssign{
				Var: variable,
				Expr: &ast.ExprBinaryMul{
					Left:  variable,
					Right: &ast.ScalarLnumber{Value: []byte("1")},
				},
			},
		}
	case 1:
		// $x = $x + 0
		noop = &ast.StmtExpression{
			Expr: &ast.ExprAssign{
				Var: variable,
				Expr: &ast.ExprBinaryPlus{
					Left:  variable,
					Right: &ast.ScalarLnumber{Value: []byte("0")},
				},
			},
		}
	case 2:
		// $x = $x - 0
		noop = &ast.StmtExpression{
			Expr: &ast.ExprAssign{
				Var: variable,
				Expr: &ast.ExprBinaryMinus{
					Left:  variable,
					Right: &ast.ScalarLnumber{Value: []byte("0")},
				},
			},
		}
	case 3:
		// $x = $x / 1
		noop = &ast.StmtExpression{
			Expr: &ast.ExprAssign{
				Var: variable,
				Expr: &ast.ExprBinaryDiv{
					Left:  variable,
					Right: &ast.ScalarLnumber{Value: []byte("1")},
				},
			},
		}
	}

	// Return a statement list with both the initialization and the no-op
	return &ast.StmtStmtList{
		Stmts: []ast.Vertex{init, noop},
	}
}

// generateUselessFunctionCall creates a call to a function that has no side effects
func (v *DeadCodeInserterVisitor) generateUselessFunctionCall() ast.Vertex {
	// Choose a random PHP function with minimal side effects
	funcs := []string{"time", "rand", "microtime", "memory_get_usage", "gettype", "count"}
	funcName := funcs[v.random.Intn(len(funcs))]

	// Create the function call
	call := &ast.ExprFunctionCall{
		Function: &ast.Name{
			Parts: []ast.Vertex{
				&ast.NamePart{Value: []byte(funcName)},
			},
		},
		Args: []ast.Vertex{},
	}

	// Add arguments for certain functions
	if funcName == "rand" {
		call.Args = []ast.Vertex{
			&ast.Argument{
				Expr: &ast.ScalarLnumber{Value: []byte("1")},
			},
			&ast.Argument{
				Expr: &ast.ScalarLnumber{Value: []byte("100")},
			},
		}
	} else if funcName == "gettype" || funcName == "count" {
		// Create a simple array for gettype or count
		call.Args = []ast.Vertex{
			&ast.Argument{
				Expr: &ast.ExprArray{
					Items: []ast.Vertex{},
				},
			},
		}
	}

	// Wrap in a statement that discards the result
	return &ast.StmtExpression{
		Expr: call,
	}
}
