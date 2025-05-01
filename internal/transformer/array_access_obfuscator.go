/*
Package transformer provides AST transformation utilities for PHP obfuscation.

Implementation Plan for Array Access Obfuscation
-----------------------------------------------
The current implementation has identified a key limitation in the PHP parser visitor pattern when
it comes to replacing complex expression nodes during traversal. While the LeaveNode method can
return replacements, these aren't being properly integrated into parent expressions in all cases.

Steps to Implement a Complete Solution:

 1. Create a Two-Pass Approach:
    a. First pass: Use the current visitor pattern to identify array access nodes and create replacements
    b. Second pass: Apply the replacements to the AST

 2. Utilize existing ParentTracker:
    a. Build a parent map during traversal to track parent-child relationships
    b. Use this map to locate where replacements need to be made

 3. Implement Using NodeReplacer:
    a. Create an array of replacement requests (original node -> replacement node)
    b. Use parent relationships to update children in parent nodes

 4. Integration with Main Obfuscator:
    a. Modify ProcessFile to perform the parent tracking pass first
    b. Then perform the array access obfuscation pass
    c. Finally apply all replacements before output generation

 5. Extended Testing:
    a. Create tests for various array access scenarios:
    - Simple array access: $array['key']
    - Nested array access: $array['key1']['key2']
    - Mixed with properties: $obj->property['key']
    - Dynamic keys: $array[$variable]
    b. Verify that all access patterns are correctly obfuscated

By implementing this approach, we can overcome the limitations of the visitor pattern and
properly transform array access expressions throughout the codebase.
*/
package transformer

import (
	"fmt"

	"github.com/VKCOM/php-parser/pkg/ast" // Correct casing
	// Correct casing
	"github.com/VKCOM/php-parser/pkg/visitor" // Correct casing

	"github.com/whit3rabbit/phpmixer/internal/config"
	// No longer importing obfuscator package
)

// Define nodeReplacement locally to avoid import cycle with obfuscator package
type nodeReplacement struct {
	Original ast.Vertex
	New      ast.Vertex
}

// ArrayAccessObfuscatorVisitor transforms array access expressions ($arr[$key])
// into helper function calls (_phpmix_array_get($arr, $key)).
// It operates in two phases coordinated by the main obfuscator:
//  1. Analysis (LeaveNode): Identifies `ast.ExprArrayDimFetch` nodes and adds
//     `nodeReplacement` requests to a shared slice.
//  2. Replacement (Post-Traversal): The main obfuscator uses the parent map
//     and the collected replacements to modify the AST.
type ArrayAccessObfuscatorVisitor struct {
	visitor.Null // Embed visitor.Null to handle methods automatically
	Config       *config.Config
	// Pointer to the shared slice where replacement requests are stored.
	// Use the locally defined nodeReplacement type.
	Replacements *[]nodeReplacement
	// Pointer to the shared flag to signal if the helper function needs to be added.
	HelperNeeded *bool
	// Track nodes visited to prevent potential re-processing issues (though less critical now).
	processedNodes map[ast.Vertex]bool
	// Counter for debug/stats
	CountNodes int
	DebugMode  bool // Local debug flag for visitor-specific logging
	// Parent tracker to detect lvalue contexts
	ParentTracker *ParentTracker
}

// NewArrayAccessObfuscatorVisitor creates a new visitor for array access obfuscation.
// It requires pointers to the shared replacements slice and helper needed flag.
func NewArrayAccessObfuscatorVisitor(cfg *config.Config, replacements *[]nodeReplacement, helperNeeded *bool) *ArrayAccessObfuscatorVisitor {
	return &ArrayAccessObfuscatorVisitor{
		Config:         cfg,
		Replacements:   replacements,
		HelperNeeded:   helperNeeded,
		processedNodes: make(map[ast.Vertex]bool),
		DebugMode:      cfg.DebugMode, // Inherit debug mode from global config
	}
}

// NewArrayAccessObfuscatorVisitorWithNodeReplacement creates a new visitor that works with the exported NodeReplacement type
func NewArrayAccessObfuscatorVisitorWithNodeReplacement(cfg *config.Config, replacements *[]nodeReplacement, helperNeeded *bool, parentTracker *ParentTracker) *ArrayAccessObfuscatorVisitor {
	visitor := &ArrayAccessObfuscatorVisitor{
		Config:         cfg,
		Replacements:   replacements,
		HelperNeeded:   helperNeeded,
		processedNodes: make(map[ast.Vertex]bool),
		DebugMode:      cfg.DebugMode,
		ParentTracker:  parentTracker,
	}

	return visitor
}

// SetParentTracker sets the parent tracker for the visitor
func (v *ArrayAccessObfuscatorVisitor) SetParentTracker(tracker *ParentTracker) {
	v.ParentTracker = tracker
}

// isLValueContext determines if the given array access node is in an L-value context
// (e.g., left side of assignment, reference parameter, etc.)
func (v *ArrayAccessObfuscatorVisitor) isLValueContext(node ast.Vertex) bool {
	if v.ParentTracker == nil {
		if v.DebugMode {
			fmt.Println("  [ArrayAccess Warning] No parent tracker available, cannot determine L-value context")
		}
		return false
	}

	parent := v.ParentTracker.GetParent(node)
	if parent == nil {
		return false
	}

	// Check for assignment (array access is the var being assigned to)
	if assign, ok := parent.(*ast.ExprAssign); ok && assign.Var == node {
		if v.DebugMode {
			fmt.Println("  [ArrayAccess Debug] Skip obfuscation - L-value in assignment")
		}
		return true
	}

	// Check for list assignment
	if list, ok := parent.(*ast.ExprList); ok {
		for _, item := range list.Items {
			if item == node {
				if v.DebugMode {
					fmt.Println("  [ArrayAccess Debug] Skip obfuscation - L-value in list assignment")
				}
				return true
			}
		}
	}

	// Check for reference in function call
	// The PHP parser doesn't expose a direct IsReference field, but we can check the function context
	// A parent of type *ast.Argument might indicate a function call parameter
	// We need to check if this argument is used in a reference context
	if _, ok := parent.(*ast.Argument); ok {
		// Check if the parent function call has ampersand in parameter list
		// This would require context from the parent of the argument
		grandparent := v.ParentTracker.GetParent(parent)
		if funcCall, ok := grandparent.(*ast.ExprFunctionCall); ok {
			// At this point, we know the array is an argument to a function call
			// We would need to examine the original PHP source to know if it was passed by reference
			// Since we can't easily determine this without parser support, we'll take a conservative approach
			// and avoid obfuscating array access in function call arguments that might be references

			// We can check specific functions known to use references
			if function, ok := funcCall.Function.(*ast.Name); ok && len(function.Parts) > 0 {
				if part, ok := function.Parts[0].(*ast.NamePart); ok {
					// List of PHP functions that commonly take reference arguments
					refFunctions := map[string]bool{
						"sort":                 true,
						"usort":                true,
						"rsort":                true,
						"asort":                true,
						"arsort":               true,
						"ksort":                true,
						"krsort":               true,
						"array_walk":           true,
						"array_walk_recursive": true,
						"array_reduce":         true,
						"array_filter":         true,
						"array_map":            true,
						"preg_match":           true,
						"preg_match_all":       true,
						"next":                 true,
						"prev":                 true,
						"end":                  true,
						"each":                 true,
						"current":              true,
						"key":                  true,
					}

					funcName := string(part.Value)
					if refFunctions[funcName] {
						if v.DebugMode {
							fmt.Printf("  [ArrayAccess Debug] Skip obfuscation - argument to function %s that may use references\n", funcName)
						}
						return true
					}
				}
			}
		}
	}

	// Check for unset or similar operations
	if funCall, ok := parent.(*ast.ExprFunctionCall); ok {
		if funcName, ok := funCall.Function.(*ast.Name); ok {
			if len(funcName.Parts) > 0 {
				if part, ok := funcName.Parts[0].(*ast.NamePart); ok {
					name := string(part.Value)
					if name == "unset" || name == "isset" {
						if v.DebugMode {
							fmt.Printf("  [ArrayAccess Debug] Skip obfuscation - array used in %s\n", name)
						}
						return true
					}
				}
			}
		}
	}

	// Check for array append ($array[] = value)
	if arrFetch, ok := node.(*ast.ExprArrayDimFetch); ok {
		if arrFetch.Dim == nil {
			if v.DebugMode {
				fmt.Println("  [ArrayAccess Debug] Skip obfuscation - array append operation (empty dim)")
			}
			return true
		}
	}

	return false
}

// EnterNode is called before traversing children.
func (v *ArrayAccessObfuscatorVisitor) EnterNode(node ast.Vertex) (bool, error) {
	// Prevent processing nodes we might have added (less likely now, but safe)
	if v.processedNodes[node] {
		return false, nil // Do not traverse children of already processed nodes
	}
	return true, nil // Continue traversal
}

// LeaveNode is called after traversing children. This is where we identify
// array accesses and queue them for replacement in the post-processing step.
func (v *ArrayAccessObfuscatorVisitor) LeaveNode(node ast.Vertex) (ast.Vertex, error) {
	if v.processedNodes[node] {
		return node, nil // Skip already processed
	}

	// We are interested only in array dimension fetch expressions: $var[...] or func()[...]
	arrFetch, ok := node.(*ast.ExprArrayDimFetch)
	if !ok {
		return node, nil // Not an array dim fetch, ignore.
	}

	// Skip array access nodes that are in L-value contexts
	if v.isLValueContext(node) {
		if v.DebugMode {
			fmt.Println("  [ArrayAccess Info] Skipping array access in L-value context")
		}
		return node, nil
	}

	// --- Construct the Replacement Node (Helper Function Call) ---
	// Try without explicit type assertion for diagnosis
	arrayVarExpr := arrFetch.Var // Directly assign, check type later if needed
	// ok := true // Assume ok for now, or add a type check
	if arrayVarExpr == nil { // Check if Var is nil instead of type assertion failure
		if v.DebugMode {
			fmt.Printf("  [ArrayAccess Warning] Skipping ArrayDimFetch replacement because base variable is nil or not an expression.\n")
		}
		return node, nil
	}

	dimExpr := arrFetch.Dim
	/* // Temporarily comment out ScalarNull usage for diagnosis
	if dimExpr == nil {
		dimExpr = &ast.ScalarNull{}
		if v.DebugMode {
			fmt.Println("  [ArrayAccess Debug] ArrayDimFetch dimension is nil, using null literal for helper call.")
		}
	}
	*/

	// Create the actual function call to replace the array access expression
	helperCall := &ast.ExprFunctionCall{
		Function: &ast.Name{
			Parts: []ast.Vertex{&ast.NamePart{Value: []byte("_phpmix_array_get")}},
		},
		Args: []ast.Vertex{
			&ast.Argument{Expr: arrayVarExpr},
			&ast.Argument{Expr: dimExpr},
		},
	}

	// --- Queue the Replacement ---
	replacement := nodeReplacement{
		Original: arrFetch,
		New:      helperCall, // Use placeholder
	}

	*v.Replacements = append(*v.Replacements, replacement)
	*v.HelperNeeded = true
	v.CountNodes++

	v.processedNodes[arrFetch] = true

	if v.DebugMode {
		fmt.Printf("  [ArrayAccess Queued] Queued replacement for %T node (Total Queued: %d)\n", arrFetch, len(*v.Replacements))
	}

	return node, nil
}

// HelperFunctionCode returns the PHP code for the array access helper function.
func (v *ArrayAccessObfuscatorVisitor) HelperFunctionCode() string {
	return `
if (!function_exists('_phpmix_array_get')) {
    function _phpmix_array_get($arr, $key, $default = null) {
        if (!is_array($arr) && !($arr instanceof ArrayAccess)) {
            @trigger_error(sprintf('Warning: Trying to access array offset on value of type %s in %s on line %d', gettype($arr), __FILE__, __LINE__), E_USER_WARNING);
            return $default;
        }
        $exists = is_array($arr) ? array_key_exists($key, $arr) : $arr->offsetExists($key);
        if ($exists) {
            return $arr[$key];
        } else {
            if (is_int($key)) {
                 @trigger_error(sprintf('Warning: Undefined array key %d in %s on line %d', $key, __FILE__, __LINE__), E_USER_WARNING);
            } elseif (is_string($key)) {
                 @trigger_error(sprintf('Warning: Undefined array key "%s" in %s on line %d', $key, __FILE__, __LINE__), E_USER_WARNING);
            } else {
                 @trigger_error(sprintf('Warning: Undefined array key of type %s in %s on line %d', gettype($key), __FILE__, __LINE__), E_USER_WARNING);
            }
            return $default;
        }
    }
}`
}

