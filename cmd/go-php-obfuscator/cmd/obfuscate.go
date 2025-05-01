package cmd

import (
	"github.com/spf13/cobra"
)

// obfuscateCmd represents the base command for obfuscation actions
var obfuscateCmd = &cobra.Command{
	Use:   "obfuscate",
	Short: "Obfuscates PHP code using various methods",
	Long: `Provides subcommands to obfuscate individual files or entire directories.

Example:
  go-php-obfuscator obfuscate file input.php -o output.php
  go-php-obfuscator obfuscate dir ./src -o ./dist --clean`,
	// Aliases can be added if desired, e.g., Aliases: []string{"ob"},
}

func init() {
	rootCmd.AddCommand(obfuscateCmd)

	// Add flags specific to the obfuscate command and its children here, if any.
	// Example: obfuscateCmd.PersistentFlags().IntP("scramble-length", "l", 10, "Target length for scrambled names")
}
