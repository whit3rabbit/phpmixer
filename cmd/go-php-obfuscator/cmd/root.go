// Package cmd implements the command line interface for the application.
package cmd

import (
	"fmt"
	"os"

	// Use correct relative path
	"github.com/whit3rabbit/phpmixer/internal/config"

	"github.com/spf13/cobra"
)

var (
	cfgFile string         // Variable to hold the config file path from the flag
	cfg     *config.Config // Global variable to hold the loaded configuration

	// Flag variables mapped to config fields for override
	silentMode       bool // -> cfg.Silent
	abortOnError     bool // -> cfg.AbortOnError
	obfuscateStrings bool // -> cfg.Obfuscation.Strings.Enabled
	stripComments    bool // -> cfg.Obfuscation.Comments.Strip
	shuffleStmts     bool // -> cfg.Obfuscation.StatementShuffling.Enabled
	controlFlow      bool // -> cfg.Obfuscation.ControlFlow.Enabled
	arrayAccess      bool // -> cfg.Obfuscation.ArrayAccess.Enabled
	deadCode         bool // -> cfg.Obfuscation.DeadCode.Enabled
	junkCode         bool // -> cfg.Obfuscation.JunkCode.Enabled
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "go-php-obfuscator",
	Short: "A CLI tool to obfuscate PHP code, inspired by yakpro-po.",
	Long: `go-php-obfuscator provides various techniques to make PHP code
harder to understand and reverse-engineer.`,
	// PersistentPreRunE runs before any subcommand's RunE.
	// Use this to load configuration early.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil { // Only load config once
			loadedCfg, err := config.LoadConfig(cfgFile) // cfgFile is set by PersistentFlags
			if err != nil {
				return fmt.Errorf("error loading configuration: %w", err) // Return error for Cobra to handle
			}
			cfg = loadedCfg

			// Apply command-line flag overrides *after* loading config file
			applyFlagOverrides(cfg, cmd)

			// Update silent mode based on final config value
			// Note: ObfuscationContext needs its own silent handling if necessary
			if cfg.Silent {
				// Potentially suppress Cobra's output? Usually not needed.
				// cmd.SilenceErrors = true // Example
				// cmd.SilenceUsage = true
			}
		}
		return nil
	},
	// Run: Executes if no subcommand is given. Print help.
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// applyFlagOverrides applies command-line flag values to the config struct.
// Only overrides if the flag was explicitly set by the user via cmd.Flags().Changed().
func applyFlagOverrides(cfg *config.Config, cmd *cobra.Command) {
	if cmd.Flags().Changed("silent") {
		cfg.Silent = silentMode
	}
	if cmd.Flags().Changed("abort-on-error") {
		cfg.AbortOnError = abortOnError
	}
	// Apply overrides for obfuscation flags
	if cmd.Flags().Changed("obfuscate-strings") {
		cfg.Obfuscation.Strings.Enabled = obfuscateStrings
	}
	if cmd.Flags().Changed("strip-comments") {
		cfg.Obfuscation.Comments.Strip = stripComments
	}
	if cmd.Flags().Changed("shuffle-statements") {
		cfg.Obfuscation.StatementShuffling.Enabled = shuffleStmts
	}
	if cmd.Flags().Changed("control-flow") {
		cfg.Obfuscation.ControlFlow.Enabled = controlFlow
	}
	if cmd.Flags().Changed("array-access") {
		cfg.Obfuscation.ArrayAccess.Enabled = arrayAccess
	}
	if cmd.Flags().Changed("dead-code") {
		cfg.Obfuscation.DeadCode.Enabled = deadCode
	}
	if cmd.Flags().Changed("junk-code") {
		cfg.Obfuscation.JunkCode.Enabled = junkCode
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		// Cobra usually prints the error. We just need to exit non-zero.
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here, will be global for your application.
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./config.yaml)")

	// Add flags for common config options
	rootCmd.PersistentFlags().BoolVarP(&silentMode, "silent", "s", false, "Suppress informational output (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&abortOnError, "abort-on-error", true, "Stop processing on the first error (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&obfuscateStrings, "obfuscate-strings", true, "Enable/disable string obfuscation (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&stripComments, "strip-comments", false, "Enable/disable comment stripping (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&shuffleStmts, "shuffle-statements", true, "Enable/disable statement shuffling (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&controlFlow, "control-flow", true, "Enable/disable control flow obfuscation (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&arrayAccess, "array-access", true, "Enable/disable array access obfuscation (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&deadCode, "dead-code", false, "Enable/disable dead code injection (overrides config)")
	rootCmd.PersistentFlags().BoolVar(&junkCode, "junk-code", false, "Enable/disable junk code insertion (overrides config)")

	// Add subcommands
	rootCmd.AddCommand(obfuscateCmd)
	rootCmd.AddCommand(whatisCmd) // Add the whatis command
	// Add other commands like config generate, etc.
}

// handleError is a utility for consistent error output
// (Potentially move to a shared utility package later)
/* Removed - Handle errors in RunE or return them
func handleError(err error, silent bool) {
    if err != nil && !silent {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
    }
}
*/
