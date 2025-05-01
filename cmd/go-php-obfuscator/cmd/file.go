// [File Begins] cmd\go-php-obfuscator\cmd\file.go
package cmd

import (
	// Need bytes.Buffer for printer output
	"fmt"
	"os"

	"github.com/spf13/cobra"

	// Use the correct relative path for internal packages

	"github.com/whit3rabbit/phpmixer/internal/obfuscator" // Import context
	// Import transformer
	// Use VKCOM parser
	// Correct Printer import based on the provided file list
)

var outputFile string // Flag variable for output file path

// fileCmd represents the obfuscate file command
var fileCmd = &cobra.Command{
	Use:   "file <php_file_path>",
	Short: "Obfuscate a single PHP file",
	Long: `Reads a single PHP file, applies the configured obfuscation
techniques, and outputs the result to stdout or a specified file.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}
		cmd.SilenceUsage = true
		filePath := args[0]
		targetFile := outputFile

		// --- Initialize Obfuscation Context (Fresh for single file) ---
		// Note: Single file command doesn't load/save context.
		octx, errCtx := obfuscator.NewObfuscationContext(cfg)
		if errCtx != nil {
			// Use handleError for consistency if silent is enabled
			if !cfg.Silent {
				fmt.Fprintf(os.Stderr, "Error initializing obfuscation context: %v\n", errCtx)
			}
			return fmt.Errorf("failed to initialize obfuscation context: %w", errCtx) // Return error to main handler
		}
		if !cfg.Silent {
			fmt.Println("Obfuscation context initialized for single file.")
		}

		// --- Process the File ---
		if !cfg.Silent {
			fmt.Printf("Processing file: %s\n", filePath)
			fmt.Printf("Debug - ObfuscateArrayAccess: %v\n", cfg.ObfuscateArrayAccess)
		}
		outputContent, err := obfuscator.ProcessFile(filePath, octx)
		if err != nil {
			// ProcessFile returns errors, let the main handler deal with printing/exiting
			return fmt.Errorf("error processing file %s: %w", filePath, err)
		}
		if !cfg.Silent {
			fmt.Println("File processed successfully.")
		}

		// --- Write Output ---
		if targetFile != "" {
			if !cfg.Silent {
				fmt.Printf("Info: Writing output to file: %s\n", targetFile)
			}
			err = os.WriteFile(targetFile, []byte(outputContent), 0644)
			if err != nil {
				return fmt.Errorf("error writing to output file %s: %w", targetFile, err)
			}
		} else { // Write to stdout
			if !cfg.Silent {
				fmt.Println("\n--- Output to stdout ---")
			}
			fmt.Print(outputContent) // Use Print not Println to avoid extra newline
		}

		if !cfg.Silent {
			fmt.Println("\nFile processing finished.")
		}
		return nil
	},
}

func init() {
	obfuscateCmd.AddCommand(fileCmd)
	fileCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path (default: stdout)")
}
