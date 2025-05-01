// Package transformer provides AST transformation utilities for PHP obfuscation.
package transformer

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/VKCOM/php-parser/pkg/ast"
	"github.com/VKCOM/php-parser/pkg/visitor"
)

/*
Statement Shuffling Overview:
----------------------------
This file implements a visitor pattern that shuffles independent statements within code blocks
to make the code more difficult to read while preserving its functionality.

The main challenge is correctly identifying "independent" statements that can be
safely reordered without changing program behavior. For example:

Original:
```php
$a = 1;
$b = 2;
$c = $a + $b;
echo $c;
```

Shuffled (possible output):
```php
$b = 2;
$a = 1;
$c = $a + $b;
echo $c;
```

The first two statements are independent and can be reordered, but the last two
statements depend on the first two and must maintain their relative positions.

Safe shuffling requires dependency analysis to ensure that statements are only
reordered if they don't have data or control dependencies.
*/

// StatementShufflerVisitor implements the NodeReplacer interface to shuffle independent
// statements within code blocks
type StatementShufflerVisitor struct {
	visitor.Null // Extends the Null visitor which provides default behavior
	replacements []*NodeReplacement
	tracker      *ParentTracker
	DebugMode    bool
	random       *rand.Rand

	// Configuration options
	MinChunkSize int    // Minimum number of statements to consider for shuffling
	ChunkMode    string // "fixed" or "ratio"
	ChunkRatio   int    // Percentage of statements to include in a chunk
}

// NewStatementShufflerVisitor creates a new visitor for shuffling statements
func NewStatementShufflerVisitor() *StatementShufflerVisitor {
	source := rand.NewSource(time.Now().UnixNano())
	return &StatementShufflerVisitor{
		replacements: make([]*NodeReplacement, 0),
		tracker:      NewParentTracker(),
		DebugMode:    false,
		random:       rand.New(source),
		MinChunkSize: 1,
		ChunkMode:    "fixed",
		ChunkRatio:   20,
	}
}

// GetParentTracker returns the parent tracker
func (v *StatementShufflerVisitor) GetParentTracker() *ParentTracker {
	return v.tracker
}

// SetParentTracker sets the parent tracker
func (v *StatementShufflerVisitor) SetParentTracker(tracker *ParentTracker) {
	v.tracker = tracker
}

// AddReplacement adds a node replacement
func (v *StatementShufflerVisitor) AddReplacement(original ast.Vertex, replacement ast.Vertex) {
	if original == nil || replacement == nil {
		return
	}
	v.replacements = append(v.replacements, &NodeReplacement{
		Original:    original,
		Replacement: replacement,
	})
}

// GetReplacements returns the list of replacements
func (v *StatementShufflerVisitor) GetReplacements() []*NodeReplacement {
	return v.replacements
}

// EnterNode is called before traversing children
func (v *StatementShufflerVisitor) EnterNode(n ast.Vertex) bool {
	if v.DebugMode && n == nil {
		fmt.Println("EnterNode called with nil node")
		return true
	}

	if v.DebugMode {
		fmt.Printf("EnterNode called with node type: %T\n", n)

		// Special debug for root node
		if root, ok := n.(*ast.Root); ok {
			fmt.Printf("Root node found with %d statements\n", len(root.Stmts))
			for i, stmt := range root.Stmts {
				fmt.Printf("  Root statement %d is type: %T\n", i, stmt)

				// Check for assignment statements
				if exprStmt, ok := stmt.(*ast.StmtExpression); ok {
					if assign, ok := exprStmt.Expr.(*ast.ExprAssign); ok {
						fmt.Printf("    Found assignment statement\n")
						if varExpr, ok := assign.Var.(*ast.ExprVariable); ok {
							if id, ok := varExpr.Name.(*ast.Identifier); ok {
								fmt.Printf("    Variable name: %s\n", string(id.Value))
							}
						}
					}
				}
			}
		}
	}

	// Always traverse children
	return true
}

// LeaveNode is called after traversing children
func (v *StatementShufflerVisitor) LeaveNode(n ast.Vertex) {
	// Handle different types of statement blocks
	switch node := n.(type) {
	case *ast.StmtStmtList:
		v.shuffleStatementList(node)
	case *ast.Root:
		v.shuffleRootStatements(node)
	case *ast.StmtFunction:
		v.shuffleFunctionStatements(node)
	case *ast.StmtClassMethod:
		v.shuffleMethodStatements(node)
	case *ast.StmtNamespace:
		v.shuffleNamespaceStatements(node)
	}
}

// shuffleStatementList shuffles a list of statements if they meet the criteria
func (v *StatementShufflerVisitor) shuffleStatementList(node *ast.StmtStmtList) {
	if node == nil || len(node.Stmts) < 2 {
		return
	}

	if v.DebugMode {
		fmt.Printf("Examining statement list with %d statements\n", len(node.Stmts))
		for i, stmt := range node.Stmts {
			fmt.Printf("  Statement %d is type: %T\n", i, stmt)
		}
	}

	// In the first implementation, just identify independent blocks that can be shuffled
	// by looking for simple statement types that are typically safe to reorder
	chunks := v.identifyIndependentChunks(node.Stmts)
	if len(chunks) <= 1 {
		if v.DebugMode {
			fmt.Println("No chunks identified for shuffling in statement list")
		}
		return // Nothing to shuffle
	}

	// Create a shuffled version of the statement list
	shuffledList := v.shuffleChunks(node.Stmts, chunks)

	// Create a new StmtStmtList with the shuffled statements
	replacement := &ast.StmtStmtList{
		Stmts: shuffledList,
	}

	// Add the replacement
	v.AddReplacement(node, replacement)
}

// shuffleRootStatements shuffles independent statements in the root node
func (v *StatementShufflerVisitor) shuffleRootStatements(node *ast.Root) {
	if node == nil || len(node.Stmts) < 2 {
		return
	}

	if v.DebugMode {
		fmt.Printf("Examining root node with %d statements\n", len(node.Stmts))
		for i, stmt := range node.Stmts {
			fmt.Printf("  Statement %d is type: %T\n", i, stmt)
		}
	}

	// Identify chunks that can be safely shuffled
	chunks := v.identifyIndependentChunks(node.Stmts)
	if len(chunks) <= 1 {
		if v.DebugMode {
			fmt.Println("No chunks identified for shuffling in root node")
		}
		return // Nothing to shuffle
	}

	// Create a shuffled version of the statements
	shuffledStmts := v.shuffleChunks(node.Stmts, chunks)

	// Create a new Root with the shuffled statements
	replacement := &ast.Root{
		Stmts: shuffledStmts,
	}

	// Add the replacement
	v.AddReplacement(node, replacement)
}

// shuffleFunctionStatements shuffles statements in a function body
func (v *StatementShufflerVisitor) shuffleFunctionStatements(node *ast.StmtFunction) {
	if node == nil || len(node.Stmts) < 2 {
		return
	}

	if v.DebugMode {
		fmt.Printf("Examining function with %d statements\n", len(node.Stmts))
		fmt.Printf("Function name: %s\n", string(node.Name.(*ast.Identifier).Value))
		for i, stmt := range node.Stmts {
			fmt.Printf("  Statement %d is type: %T\n", i, stmt)
		}
	}

	// Identify chunks that can be safely shuffled
	chunks := v.identifyIndependentChunks(node.Stmts)
	if len(chunks) <= 1 {
		if v.DebugMode {
			fmt.Println("No chunks identified for shuffling in function")
		}
		return // Nothing to shuffle
	}

	// Create a shuffled version of the statements
	shuffledStmts := v.shuffleChunks(node.Stmts, chunks)

	// Create a new StmtFunction with the shuffled statements
	replacement := &ast.StmtFunction{
		Name:       node.Name,
		Params:     node.Params,
		Stmts:      shuffledStmts,
		ReturnType: node.ReturnType,
	}

	// Add the replacement
	v.AddReplacement(node, replacement)
}

// shuffleMethodStatements shuffles statements in a method body
func (v *StatementShufflerVisitor) shuffleMethodStatements(node *ast.StmtClassMethod) {
	if node == nil {
		return
	}

	// Method body is typically a StmtStmtList
	stmtList, ok := node.Stmt.(*ast.StmtStmtList)
	if !ok || len(stmtList.Stmts) < 2 {
		return
	}

	if v.DebugMode {
		fmt.Printf("Examining method with %d statements\n", len(stmtList.Stmts))
		fmt.Printf("Method name: %s\n", string(node.Name.(*ast.Identifier).Value))
		for i, stmt := range stmtList.Stmts {
			fmt.Printf("  Statement %d is type: %T\n", i, stmt)
		}
	}

	// Identify chunks that can be safely shuffled
	chunks := v.identifyIndependentChunks(stmtList.Stmts)
	if len(chunks) <= 1 {
		if v.DebugMode {
			fmt.Println("No chunks identified for shuffling in method")
		}
		return // Nothing to shuffle
	}

	// Create a shuffled version of the statements
	shuffledStmts := v.shuffleChunks(stmtList.Stmts, chunks)

	// Create a new StmtStmtList with the shuffled statements
	replacementStmtList := &ast.StmtStmtList{
		Stmts: shuffledStmts,
	}

	// Create a new StmtClassMethod with the shuffled statement list
	replacement := &ast.StmtClassMethod{
		Name:       node.Name,
		Modifiers:  node.Modifiers,
		Params:     node.Params,
		ReturnType: node.ReturnType,
		Stmt:       replacementStmtList,
	}

	// Add the replacement
	v.AddReplacement(node, replacement)
}

// shuffleNamespaceStatements shuffles statements in a namespace
func (v *StatementShufflerVisitor) shuffleNamespaceStatements(node *ast.StmtNamespace) {
	if node == nil || node.Stmts == nil || len(node.Stmts) < 2 {
		return
	}

	if v.DebugMode {
		fmt.Printf("Examining namespace with %d statements\n", len(node.Stmts))
		if node.Name != nil {
			fmt.Printf("Namespace name: %s\n", string(node.Name.(*ast.Name).Parts[0].(*ast.NamePart).Value))
		} else {
			fmt.Println("Namespace with no name (global namespace)")
		}
		for i, stmt := range node.Stmts {
			fmt.Printf("  Statement %d is type: %T\n", i, stmt)
		}
	}

	// Identify chunks that can be safely shuffled
	chunks := v.identifyIndependentChunks(node.Stmts)
	if len(chunks) <= 1 {
		if v.DebugMode {
			fmt.Println("No chunks identified for shuffling in namespace")
		}
		return // Nothing to shuffle
	}

	// Create a shuffled version of the statements
	shuffledStmts := v.shuffleChunks(node.Stmts, chunks)

	// Create a new StmtNamespace with the shuffled statements
	replacement := &ast.StmtNamespace{
		Name:  node.Name,
		Stmts: shuffledStmts,
	}

	// Add the replacement
	v.AddReplacement(node, replacement)
}

// identifyIndependentChunks identifies independent chunks of statements that can be shuffled
func (v *StatementShufflerVisitor) identifyIndependentChunks(stmts []ast.Vertex) [][]int {
	if len(stmts) < 2 { // Always require at least 2 statements for shuffling
		return nil
	}

	// Initial implementation: just group simple independent statements
	// and preserve ordering of complex statements
	var chunks [][]int
	var currentChunk []int
	var inSafeRegion bool = true

	// First identify any return statements - they need to remain at the end
	var hasReturnStmt bool
	var returnStmtIndex int
	for i, stmt := range stmts {
		if _, isReturn := stmt.(*ast.StmtReturn); isReturn {
			hasReturnStmt = true
			returnStmtIndex = i
			break
		}
	}

	// If we have a return statement, we only process statements before it
	stmtsToProcess := stmts
	if hasReturnStmt {
		stmtsToProcess = stmts[:returnStmtIndex]
	}

	for i, stmt := range stmtsToProcess {
		// Helper function to check if a statement is an expression statement with a simple assignment
		isSimpleAssignment := func(s ast.Vertex) bool {
			if exprStmt, ok := s.(*ast.StmtExpression); ok {
				if assign, ok := exprStmt.Expr.(*ast.ExprAssign); ok {
					// Check if the left side is a simple variable
					if _, isVar := assign.Var.(*ast.ExprVariable); isVar {
						return true
					}
				}
			}
			return false
		}

		// Helper function to check if a statement is an echo with a simple string
		isSimpleEcho := func(s ast.Vertex) bool {
			if echoStmt, ok := s.(*ast.StmtEcho); ok {
				if len(echoStmt.Exprs) == 1 {
					// If it's a simple string or variable, consider it safe
					if _, isString := echoStmt.Exprs[0].(*ast.ScalarString); isString {
						return true
					}
					if _, isVar := echoStmt.Exprs[0].(*ast.ExprVariable); isVar {
						return true
					}
				}
			}
			return false
		}

		switch {
		// Statements that are typically safe to reorder
		case stmt.(*ast.StmtGlobal) != nil,
			stmt.(*ast.StmtStatic) != nil,
			isSimpleAssignment(stmt),
			isSimpleEcho(stmt):
			if inSafeRegion {
				// Continue or start a new safe chunk
				currentChunk = append(currentChunk, i)
			} else {
				// We've transitioned back to a safe region
				inSafeRegion = true
				if len(currentChunk) > 0 {
					chunks = append(chunks, currentChunk)
				}
				currentChunk = []int{i}
			}

		// Statements that should maintain their relative positions
		case stmt.(*ast.StmtReturn) != nil,
			stmt.(*ast.StmtThrow) != nil,
			stmt.(*ast.StmtBreak) != nil,
			stmt.(*ast.StmtContinue) != nil,
			stmt.(*ast.StmtIf) != nil,
			stmt.(*ast.StmtSwitch) != nil,
			stmt.(*ast.StmtTry) != nil,
			stmt.(*ast.StmtFunction) != nil,
			stmt.(*ast.StmtClass) != nil,
			stmt.(*ast.StmtForeach) != nil,
			stmt.(*ast.StmtFor) != nil,
			stmt.(*ast.StmtWhile) != nil,
			stmt.(*ast.StmtDo) != nil:
			if inSafeRegion {
				// Transition out of a safe region
				inSafeRegion = false
				if len(currentChunk) >= v.MinChunkSize {
					chunks = append(chunks, currentChunk)
				}
				currentChunk = []int{}
			}
			// Treat this statement as its own "chunk" that won't be shuffled
			chunks = append(chunks, []int{i})

		// Default: treat as a boundary by default until we have more sophisticated analysis
		default:
			if v.DebugMode {
				fmt.Printf("Unknown statement type (treating as boundary): %T\n", stmt)
			}
			if inSafeRegion {
				inSafeRegion = false
				if len(currentChunk) >= v.MinChunkSize {
					chunks = append(chunks, currentChunk)
				}
				currentChunk = []int{}
			}
			chunks = append(chunks, []int{i})
		}
	}

	// Add the last chunk if it exists and meets the minimum size
	if len(currentChunk) >= v.MinChunkSize {
		chunks = append(chunks, currentChunk)
	}

	// If we have a return statement, add it as a separate chunk at the end
	if hasReturnStmt {
		// Add all statements from the return to the end as a single chunk
		var finalChunk []int
		for i := returnStmtIndex; i < len(stmts); i++ {
			finalChunk = append(finalChunk, i)
		}
		chunks = append(chunks, finalChunk)
	}

	// Filter chunks to exclude any that don't meet minimum size
	var filteredChunks [][]int
	for _, chunk := range chunks {
		if len(chunk) >= v.MinChunkSize || len(chunk) == 1 {
			// Keep chunks that meet minimum size for shuffling or are single statements
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	// We need at least 2 chunks that meet the minimum size for shuffling
	shufflableChunks := 0
	for _, chunk := range filteredChunks {
		if len(chunk) >= v.MinChunkSize {
			shufflableChunks++
		}
	}

	if shufflableChunks < 2 {
		// Not enough chunks to shuffle effectively
		if v.DebugMode {
			fmt.Printf("Not enough shufflable chunks found (need at least 2, found %d)\n", shufflableChunks)
		}
		return nil
	}

	if v.DebugMode {
		fmt.Printf("Found %d chunks, %d of which are shufflable\n", len(filteredChunks), shufflableChunks)

		for i, chunk := range filteredChunks {
			fmt.Printf("Chunk %d: size=%d, positions=%v\n", i, len(chunk), chunk)
			if len(chunk) >= v.MinChunkSize {
				fmt.Printf("  This chunk is shufflable\n")
				for _, pos := range chunk {
					fmt.Printf("    Statement %d: %T\n", pos, stmts[pos])
				}
			}
		}
	}

	return filteredChunks
}

// shuffleChunks shuffles chunks of statements while preserving order within each chunk
func (v *StatementShufflerVisitor) shuffleChunks(stmts []ast.Vertex, chunks [][]int) []ast.Vertex {
	if len(chunks) <= 1 {
		return stmts
	}

	// Make a copy of the original statements
	result := make([]ast.Vertex, len(stmts))
	copy(result, stmts)

	// Filter out chunks that should be shuffled (those with more than one statement)
	var shufflableChunks [][]int
	var nonShufflableChunks [][]int

	for _, chunk := range chunks {
		if len(chunk) >= v.MinChunkSize {
			shufflableChunks = append(shufflableChunks, chunk)
		} else {
			nonShufflableChunks = append(nonShufflableChunks, chunk)
		}
	}

	// If no shufflable chunks, return original
	if len(shufflableChunks) <= 1 {
		if v.DebugMode {
			fmt.Printf("No shufflable chunks found (need at least 2, found %d)\n", len(shufflableChunks))
		}
		return stmts
	}

	if v.DebugMode {
		fmt.Printf("Shuffling %d chunks\n", len(shufflableChunks))
		// Print before shuffling
		fmt.Println("Before shuffling:")
		for i, chunk := range shufflableChunks {
			fmt.Printf("  Chunk %d: positions %v\n", i, chunk)
		}
	}

	// Shuffle the array of shufflable chunks (not the statements inside them)
	v.random.Shuffle(len(shufflableChunks), func(i, j int) {
		shufflableChunks[i], shufflableChunks[j] = shufflableChunks[j], shufflableChunks[i]
	})

	if v.DebugMode {
		// Print after shuffling
		fmt.Println("After shuffling:")
		for i, chunk := range shufflableChunks {
			fmt.Printf("  Chunk %d: positions %v\n", i, chunk)
		}
	}

	// Build the shuffled result
	resultMap := make(map[int]ast.Vertex, len(stmts))

	// First handle non-shufflable chunks - they stay in place
	for _, chunk := range nonShufflableChunks {
		for _, pos := range chunk {
			resultMap[pos] = stmts[pos]
		}
	}

	// Now handle shufflable chunks
	// Create a copy of shufflable chunks for mapping
	shuffledChunks := make([][]int, len(shufflableChunks))
	copy(shuffledChunks, shufflableChunks)

	// For each original shufflable chunk
	for i, originalChunk := range shufflableChunks {
		// Get the corresponding shuffled chunk
		shuffledChunk := shuffledChunks[i]

		// Map original positions to shuffled positions
		for j, pos := range originalChunk {
			if j < len(shuffledChunk) {
				shuffledPos := shuffledChunk[j]
				resultMap[shuffledPos] = stmts[pos]
			}
		}
	}

	// Build the final result array
	for i := 0; i < len(stmts); i++ {
		if stmt, ok := resultMap[i]; ok {
			result[i] = stmt
		}
	}

	return result
}

// GetReplacement implements the NodeReplacer interface
func (v *StatementShufflerVisitor) GetReplacement(node ast.Vertex) ast.Vertex {
	for _, repl := range v.replacements {
		if repl.Original == node {
			return repl.Replacement
		}
	}
	return nil
}
