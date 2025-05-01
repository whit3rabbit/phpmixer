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
Arithmetic Expression Obfuscation Overview:
-----------------------------------------
This file implements a visitor pattern to transform PHP arithmetic expressions
into more complex but equivalent expressions. The goal is to make the code
harder to understand and reverse engineer while maintaining the same functionality.

Examples of transformations:
- `1 + 2` → `(3 - 1) + (8 / 4)`
- `$x - 5` → `$x - (2 + 3)`
- `$a * $b` → `($a * $b * 1) + (0 * $b)`

The transformations are applied randomly to avoid predictable patterns.
*/

// ArithmeticObfuscatorVisitor obfuscates arithmetic expressions by replacing them
// with equivalent but more complex expressions
type ArithmeticObfuscatorVisitor struct {
	visitor.Null // Extends the Null visitor which provides default behavior
	replacements []*NodeReplacement
	DebugMode    bool
	// Random source for expression generation
	random *rand.Rand
	// Flag to enable/disable obfuscation of binary operations
	ObfuscateBinaryOperations bool
	// Flag to enable/disable obfuscation of integer literals
	ObfuscateIntegerLiterals bool
	// Maximum depth of obfuscation nesting to prevent overly complex expressions
	MaxObfuscationDepth int
	// Tracks current obfuscation depth during traversal
	currentDepth int
	// Track processed nodes to avoid infinite recursion
	processedNodes map[ast.Vertex]bool
}

// NewArithmeticObfuscatorVisitor creates a new ArithmeticObfuscatorVisitor
func NewArithmeticObfuscatorVisitor() *ArithmeticObfuscatorVisitor {
	return &ArithmeticObfuscatorVisitor{
		replacements:              make([]*NodeReplacement, 0),
		DebugMode:                 false,
		random:                    rand.New(rand.NewSource(time.Now().UnixNano())),
		ObfuscateBinaryOperations: true,
		ObfuscateIntegerLiterals:  true,
		MaxObfuscationDepth:       2,
		currentDepth:              0,
		processedNodes:            make(map[ast.Vertex]bool),
	}
}

// EnterNode is called when entering a node during traversal
// Implements the NodeReplacer interface
func (v *ArithmeticObfuscatorVisitor) EnterNode(n ast.Vertex) bool {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "ArithmeticObfuscator Enter: %T\n", n)
	}

	// Skip already processed nodes to avoid infinite recursion
	if v.processedNodes[n] {
		return false
	}

	// If we've reached max depth, don't obfuscate further
	if v.currentDepth >= v.MaxObfuscationDepth {
		return true
	}

	switch node := n.(type) {
	case *ast.ExprBinaryPlus:
		if v.ObfuscateBinaryOperations {
			v.obfuscateAddition(node)
		}
	case *ast.ExprBinaryMinus:
		if v.ObfuscateBinaryOperations {
			v.obfuscateSubtraction(node)
		}
	case *ast.ExprBinaryMul:
		if v.ObfuscateBinaryOperations {
			v.obfuscateMultiplication(node)
		}
	case *ast.ExprBinaryDiv:
		if v.ObfuscateBinaryOperations {
			v.obfuscateDivision(node)
		}
	case *ast.ScalarLnumber:
		if v.ObfuscateIntegerLiterals {
			v.obfuscateIntegerLiteral(node)
		}
	}

	v.processedNodes[n] = true
	v.currentDepth++
	return true
}

// GetReplacement checks if there's a replacement for the given node
// Implements the NodeReplacer interface
func (v *ArithmeticObfuscatorVisitor) GetReplacement(n ast.Vertex) (ast.Vertex, bool) {
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
func (v *ArithmeticObfuscatorVisitor) LeaveNode(n ast.Vertex) {
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "ArithmeticObfuscator Leave: %T\n", n)
	}
	v.currentDepth--
}

// AddReplacement adds a node replacement
func (v *ArithmeticObfuscatorVisitor) AddReplacement(original ast.Vertex, replacement ast.Vertex) {
	v.replacements = append(v.replacements, &NodeReplacement{
		Original:    original,
		Replacement: replacement,
	})
	if v.DebugMode {
		fmt.Fprintf(os.Stderr, "Added replacement: %T -> %T\n", original, replacement)
	}
}

// obfuscateAddition transforms a + b into equivalent expressions like:
// (a + c) + (b - c), a + (b + 0), etc.
func (v *ArithmeticObfuscatorVisitor) obfuscateAddition(n *ast.ExprBinaryPlus) {
	// Don't always obfuscate to maintain some naturalness in the code
	if v.random.Intn(100) > 80 {
		return
	}

	// Choose a random obfuscation technique
	technique := v.random.Intn(3)

	var replacement ast.Vertex

	switch technique {
	case 0:
		// a + b => (a - c) + (b + c) where c is a random number
		c := v.random.Intn(10) + 1

		// (a - c)
		left := &ast.ExprBinaryMinus{
			Left:  n.Left,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (b + c)
		right := &ast.ExprBinaryPlus{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (a - c) + (b + c)
		replacement = &ast.ExprBinaryPlus{
			Left:  left,
			Right: right,
		}

	case 1:
		// a + b => a + (b * 1)
		right := &ast.ExprBinaryMul{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte("1")},
		}

		replacement = &ast.ExprBinaryPlus{
			Left:  n.Left,
			Right: right,
		}

	case 2:
		// a + b => (a + b + c) - c where c is a random number
		c := v.random.Intn(10) + 1

		// (a + b + c)
		innerSum := &ast.ExprBinaryPlus{
			Left: &ast.ExprBinaryPlus{
				Left:  n.Left,
				Right: n.Right,
			},
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (a + b + c) - c
		replacement = &ast.ExprBinaryMinus{
			Left:  innerSum,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}
	}

	if replacement != nil {
		v.AddReplacement(n, replacement)
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Obfuscated addition\n")
		}
	}
}

// obfuscateSubtraction transforms a - b into equivalent expressions
func (v *ArithmeticObfuscatorVisitor) obfuscateSubtraction(n *ast.ExprBinaryMinus) {
	// Don't always obfuscate to maintain some naturalness in the code
	if v.random.Intn(100) > 80 {
		return
	}

	// Choose a random obfuscation technique
	technique := v.random.Intn(2)

	var replacement ast.Vertex

	switch technique {
	case 0:
		// a - b => (a + c) - (b + c) where c is a random number
		c := v.random.Intn(10) + 1

		// (a + c)
		left := &ast.ExprBinaryPlus{
			Left:  n.Left,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (b + c)
		right := &ast.ExprBinaryPlus{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (a + c) - (b + c)
		replacement = &ast.ExprBinaryMinus{
			Left:  left,
			Right: right,
		}

	case 1:
		// a - b => a + (-1 * b)
		negB := &ast.ExprBinaryMul{
			Left:  &ast.ScalarLnumber{Value: []byte("-1")},
			Right: n.Right,
		}

		replacement = &ast.ExprBinaryPlus{
			Left:  n.Left,
			Right: negB,
		}
	}

	if replacement != nil {
		v.AddReplacement(n, replacement)
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Obfuscated subtraction\n")
		}
	}
}

// obfuscateMultiplication transforms a * b into equivalent expressions
func (v *ArithmeticObfuscatorVisitor) obfuscateMultiplication(n *ast.ExprBinaryMul) {
	// Don't always obfuscate to maintain some naturalness in the code
	if v.random.Intn(100) > 80 {
		return
	}

	// Choose a random obfuscation technique
	technique := v.random.Intn(2)

	var replacement ast.Vertex

	switch technique {
	case 0:
		// a * b => (a * c) * (b / c) where c is a random number
		c := v.random.Intn(5) + 2 // avoid small numbers like 1

		// (a * c)
		left := &ast.ExprBinaryMul{
			Left:  n.Left,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (b / c)
		right := &ast.ExprBinaryDiv{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (a * c) * (b / c)
		replacement = &ast.ExprBinaryMul{
			Left:  left,
			Right: right,
		}

	case 1:
		// a * b => (a / 2) * (b * 2)
		left := &ast.ExprBinaryDiv{
			Left:  n.Left,
			Right: &ast.ScalarLnumber{Value: []byte("2")},
		}

		right := &ast.ExprBinaryMul{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte("2")},
		}

		replacement = &ast.ExprBinaryMul{
			Left:  left,
			Right: right,
		}
	}

	if replacement != nil {
		v.AddReplacement(n, replacement)
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Obfuscated multiplication\n")
		}
	}
}

// obfuscateDivision transforms a / b into equivalent expressions
func (v *ArithmeticObfuscatorVisitor) obfuscateDivision(n *ast.ExprBinaryDiv) {
	// Don't always obfuscate to maintain some naturalness in the code
	if v.random.Intn(100) > 80 {
		return
	}

	// Division is more sensitive to obfuscation due to potential floating point issues
	// and division by zero errors, so we're more careful here

	// Choose a random obfuscation technique
	technique := v.random.Intn(1)

	var replacement ast.Vertex

	switch technique {
	case 0:
		// a / b => (a * c) / (b * c) where c is a random number
		c := v.random.Intn(5) + 2 // avoid small numbers like 1

		// (a * c)
		left := &ast.ExprBinaryMul{
			Left:  n.Left,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (b * c)
		right := &ast.ExprBinaryMul{
			Left:  n.Right,
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", c))},
		}

		// (a * c) / (b * c)
		replacement = &ast.ExprBinaryDiv{
			Left:  left,
			Right: right,
		}
	}

	if replacement != nil {
		v.AddReplacement(n, replacement)
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Obfuscated division\n")
		}
	}
}

// obfuscateIntegerLiteral transforms integer literals into expressions
// For example: 5 => 10/2, 5 => 2+3, etc.
func (v *ArithmeticObfuscatorVisitor) obfuscateIntegerLiteral(n *ast.ScalarLnumber) {
	// Don't always obfuscate to maintain some naturalness in the code
	if v.random.Intn(100) > 70 {
		return
	}

	// Parse the original number
	var num int
	fmt.Sscanf(string(n.Value), "%d", &num)

	// Don't obfuscate small numbers or 0
	if num < 3 || num == 0 {
		return
	}

	// Choose a random obfuscation technique
	technique := v.random.Intn(4)

	var replacement ast.Vertex

	switch technique {
	case 0:
		// n => n+1-1
		replacement = &ast.ExprBinaryMinus{
			Left: &ast.ExprBinaryPlus{
				Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", num+1))},
				Right: &ast.ScalarLnumber{Value: []byte("1")},
			},
			Right: &ast.ScalarLnumber{Value: []byte("1")},
		}

	case 1:
		// n => (n*2)/2
		replacement = &ast.ExprBinaryDiv{
			Left: &ast.ExprBinaryMul{
				Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", num))},
				Right: &ast.ScalarLnumber{Value: []byte("2")},
			},
			Right: &ast.ScalarLnumber{Value: []byte("2")},
		}

	case 2:
		// n => a+b where a+b=n
		// Find a random number between 1 and n-1
		a := v.random.Intn(num-1) + 1
		b := num - a

		replacement = &ast.ExprBinaryPlus{
			Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", a))},
			Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", b))},
		}

	case 3:
		// n => a*b where a*b=n (only if n has factors)
		factors := findFactors(num)
		if len(factors) > 2 { // at least two factors besides 1 and n
			// Pick two random factors that multiply to give n
			a := factors[v.random.Intn(len(factors)-2)+1] // Skip 1
			b := num / a

			replacement = &ast.ExprBinaryMul{
				Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", a))},
				Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", b))},
			}
		} else {
			// Fallback to addition
			a := v.random.Intn(num-1) + 1
			b := num - a

			replacement = &ast.ExprBinaryPlus{
				Left:  &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", a))},
				Right: &ast.ScalarLnumber{Value: []byte(fmt.Sprintf("%d", b))},
			}
		}
	}

	if replacement != nil {
		v.AddReplacement(n, replacement)
		if v.DebugMode {
			fmt.Fprintf(os.Stderr, "Obfuscated integer literal %d\n", num)
		}
	}
}

// findFactors returns all factors of n
func findFactors(n int) []int {
	var factors []int
	for i := 1; i <= n; i++ {
		if n%i == 0 {
			factors = append(factors, i)
		}
	}
	return factors
}
