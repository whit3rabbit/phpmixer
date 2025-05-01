// [File Begins] cmd\go-php-obfuscator\cmd\whatis.go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
	"github.com/whit3rabbit/phpmixer/internal/scrambler"
)

var (
	whatisTargetDir string
	whatisType      string
)

// whatisCmd represents the whatis command
var whatisCmd = &cobra.Command{
	Use:   "whatis <scrambled_name>",
	Short: "Looks up the original name for a given scrambled name",
	Long: `Loads the saved obfuscation context from a previous run's target directory
and attempts to find the original identifier corresponding to the provided scrambled name.

You must specify the target directory where the context was saved using --target-dir (-t).
You can optionally specify the type of identifier (--type) to narrow the search.`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		// Check if target-dir is provided
		if whatisTargetDir == "" {
			return fmt.Errorf("--target-dir (-t) flag is required")
		}
		// Check if target-dir exists
		info, err := os.Stat(whatisTargetDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("target directory '%s' not found", whatisTargetDir)
			}
			return fmt.Errorf("error checking target directory '%s': %w", whatisTargetDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("target path '%s' is not a directory", whatisTargetDir)
		}

		// Validate --type if provided
		if whatisType != "" {
			if _, err := scrambler.ParseScrambleType(whatisType); err != nil {
				return err // Return error from ParseScrambleType
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		scrambledName := args[0]
		cmd.SilenceUsage = true // Prevent usage print on expected errors (like not found)

		// Create a minimal config (or load defaults?)
		// We primarily need it to initialize the context structure.
		// Loading defaults might be safer if NewObfuscationContext relies on them.
		// Let's use LoadConfig with no path, which loads defaults.
		dummyCfg, err := config.LoadConfig("") // Load defaults
		if err != nil {
			// This should generally not fail if defaults are okay
			return fmt.Errorf("failed to load default config for context init: %w", err)
		}
		// Inherit silent setting if needed, though whatis is usually verbose
		// dummyCfg.Silent = cfg.Silent // Or maybe whatis shouldn't be silent?

		// Initialize context structure
		octx, err := obfuscator.NewObfuscationContext(dummyCfg)
		if err != nil {
			return fmt.Errorf("failed to initialize obfuscation context structure: %w", err)
		}

		// Load the actual context from the specified directory
		if err := octx.Load(whatisTargetDir); err != nil {
			// Treat loading errors as potentially serious, as the context might be incomplete
			return fmt.Errorf("error loading obfuscation context from %s: %w", whatisTargetDir, err)
		}
		if !octx.Silent { // Use silent flag from loaded context/defaults if desired
			fmt.Printf("Searching for original name of '%s' in context %s\n", scrambledName, whatisTargetDir)
		}

		// Determine types to check
		typesToCheck := make(map[scrambler.ScrambleType]bool)
		if whatisType != "" {
			sType, _ := scrambler.ParseScrambleType(whatisType) // Ignore error, already checked in PreRunE
			typesToCheck[sType] = true
			if !octx.Silent {
				fmt.Printf("Limiting search to type: %s\n", sType)
			}
		} else {
			// Check all types known to the context
			for sType := range octx.Scramblers {
				typesToCheck[sType] = true
			}
			if !octx.Silent {
				fmt.Println("Searching across all known types...")
			}
		}

		// Search scramblers
		found := false
		for sType, check := range typesToCheck {
			if !check {
				continue
			}
			scramblerInstance, exists := octx.Scramblers[sType]
			if !exists || scramblerInstance == nil {
				continue
			} // Skip if scrambler missing

			originalName, ok := scramblerInstance.Unscramble(scrambledName)
			if ok {
				fmt.Printf("Found: '%s' (Type: %s)\n", originalName, sType)
				found = true
				break // Stop after first match
			}
		}

		if !found {
			fmt.Fprintf(os.Stderr, "Error: Scrambled name '%s' not found in the loaded context.\n", scrambledName)
			return fmt.Errorf("name not found") // Return specific error for scripting
		}

		return nil // Success
	},
}

func init() {
	rootCmd.AddCommand(whatisCmd)
	whatisCmd.Flags().StringVarP(&whatisTargetDir, "target-dir", "t", "", "Target directory of a previous obfuscate run (required)")
	whatisCmd.Flags().StringVar(&whatisType, "type", "", "Specific identifier type (e.g., variable, function, class, method)")
	// Mark target-dir as required? Cobra doesn't enforce globally required flags easily, handled in PreRunE.
}

// [File Ends] cmd\go-php-obfuscator\cmd\whatis.go
