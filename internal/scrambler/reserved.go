package scrambler

import "strings"

// ScrambleType defines the category of identifier being scrambled.
type ScrambleType string

const (
	TypeVariable      ScrambleType = "variable"
	TypeFunction      ScrambleType = "function" // Also used for classes, interfaces, traits, namespaces due to case-insensitivity and shared collision potential
	TypeClass         ScrambleType = "class"    // Distinction might be needed for specific reserved words/ignore lists? Revisit if necessary.
	TypeInterface     ScrambleType = "interface"
	TypeTrait         ScrambleType = "trait"
	TypeNamespace     ScrambleType = "namespace"
	TypeProperty      ScrambleType = "property"
	TypeMethod        ScrambleType = "method"
	TypeClassConstant ScrambleType = "class_constant"
	TypeConstant      ScrambleType = "constant"
	TypeLabel         ScrambleType = "label"
)

// Note: Using string constants allows easy use in maps/config.

// --- Reserved PHP Keywords and Constructs ---
// (From Yakpro-PO & PHP docs - case-insensitive matching needed for these)
var reservedKeywords = map[string]bool{
	"__halt_compiler": true, "__class__": true, "__dir__": true, "__file__": true,
	"__function__": true, "__line__": true, "__method__": true, "__namespace__": true,
	"__trait__": true, "abstract": true, "and": true, "array": true, "as": true,
	"break": true, "callable": true, "case": true, "catch": true, "class": true,
	"clone": true, "const": true, "continue": true, "declare": true, "default": true,
	"die": true, "do": true, "echo": true, "else": true, "elseif": true, "empty": true,
	"enddeclare": true, "endfor": true, "endforeach": true, "endif": true,
	"endswitch": true, "endwhile": true, "eval": true, "exit": true, "extends": true,
	"final": true, "finally": true, "fn": true, "for": true, "foreach": true,
	"function": true, "global": true, "goto": true, "if": true, "implements": true,
	"include": true, "include_once": true, "instanceof": true, "insteadof": true,
	"interface": true, "isset": true, "list": true, "match": true, "namespace": true,
	"new": true, "or": true, "print": true, "private": true, "protected": true,
	"public": true, "readonly": true, "require": true, "require_once": true,
	"return": true, "static": true, "switch": true, "throw": true, "trait": true,
	"try": true, "unset": true, "use": true, "var": true, "while": true, "xor": true,
	"yield": true, "yield from": true,
	// PHP built-in types (lowercase) - cannot be used as class/interface/trait names
	"int": true, "float": true, "bool": true, "string": true, "true": true,
	"false": true, "null": true, "void": true, "iterable": true, "object": true,
	"resource": true, "mixed": true, "never": true, // Added never for PHP 8.1+
	// "numeric" and "scalar" are often mentioned but aren't keywords in the same way.
	// Added 'scalar' as per yakpro-po
	"scalar": true, "numeric": true,
}

// --- Reserved Variable Names (Superglobals, etc.) ---
// (Case-sensitive matching needed)
var reservedVariables = map[string]bool{
	"this":                 true, // Special object context variable
	"GLOBALS":              true,
	"_SERVER":              true,
	"_GET":                 true,
	"_POST":                true,
	"_FILES":               true,
	"_COOKIE":              true,
	"_SESSION":             true,
	"_REQUEST":             true,
	"_ENV":                 true,
	"php_errormsg":         true, // If track_errors is enabled
	"HTTP_RAW_POST_DATA":   true, // Deprecated but might exist
	"http_response_header": true, // If network functions used
	"argc":                 true, // If register_argc_argv enabled
	"argv":                 true, // If register_argc_argv enabled
}

// --- Reserved Class/Interface/Trait Names ---
// (Case-insensitive matching needed)
var reservedClasses = map[string]bool{
	"parent": true,
	"self":   true,
	"static": true, // Also a keyword, but important here
	// Add built-in types again as they can't be class names
	"int": true, "float": true, "bool": true, "string": true, "true": true,
	"false": true, "null": true, "void": true, "iterable": true, "object": true,
	"resource": true, "mixed": true, "never": true,
}

// --- Reserved Method Names (Magic Methods) ---
// (Case-insensitive matching needed)
var reservedMethods = map[string]bool{
	"__construct":   true,
	"__destruct":    true,
	"__call":        true,
	"__callstatic":  true,
	"__get":         true,
	"__set":         true,
	"__isset":       true,
	"__unset":       true,
	"__sleep":       true,
	"__wakeup":      true,
	"__serialize":   true, // PHP 7.4+
	"__unserialize": true, // PHP 7.4+
	"__tostring":    true,
	"__invoke":      true,
	"__set_state":   true,
	"__clone":       true,
	"__debuginfo":   true, // PHP 5.6+
}

// --- Reserved Constants (Magic Constants + true/false/null) ---
// (Most are case-insensitive, handled by reservedKeywords, but check specific usage)
var reservedConstants = map[string]bool{
	// true, false, null are handled by keyword check usually
	"true":  true,
	"false": true,
	"null":  true,
	// Magic constants handled during parsing/visiting, not renaming usually.
}

// isReserved checks if a name is reserved for a given scramble type.
// It handles case sensitivity appropriately.
func isReserved(name string, sType ScrambleType) bool {
	lowerName := strings.ToLower(name)

	// Check keywords first (always case-insensitive)
	if reservedKeywords[lowerName] {
		return true
	}

	switch sType {
	case TypeVariable, TypeProperty:
		// Case-sensitive check for specific variables
		return reservedVariables[name]
	case TypeFunction: // Combined check for function/class/interface/trait/namespace
		// Case-insensitive check
		if reservedClasses[lowerName] {
			return true
		}
		// Also check function names that might collide (e.g., internal PHP functions)
		// Note: A full list of internal functions isn't maintained here, rely on ignore list instead.
		return false
	case TypeMethod:
		// Case-insensitive check
		return reservedMethods[lowerName]
	case TypeConstant, TypeClassConstant:
		// Case-insensitive check
		return reservedConstants[lowerName]
	case TypeLabel:
		// Labels generally don't collide with PHP keywords directly,
		// but could collide with generated goto targets. Check keywords anyway.
		return false // Rely on keyword check above
	default:
		return false
	}
}
