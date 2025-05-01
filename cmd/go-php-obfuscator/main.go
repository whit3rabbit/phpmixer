/*
PHP Parser Tool (Entry Point)

This tool parses PHP source code files and generates an Abstract Syntax Tree (AST).
It can be used for static analysis, refactoring, and code quality checks.

The parser supports PHP versions from 5.x through 8.x, depending on the configuration.
*/
package main

import (
	// Use the correct relative path for the cmd package
	"github.com/whit3rabbit/phpmixer/cmd/go-php-obfuscator/cmd"
)

// main is the entry point of the application.
func main() {
	// Execute the root command from the cmd package
	cmd.Execute()
}
