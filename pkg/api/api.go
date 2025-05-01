// Package api provides the public API for using the PHP obfuscator as a library.
//
// This package allows users to obfuscate PHP code programmatically using the
// same techniques available in the command-line interface. The API provides
// methods for obfuscating PHP code strings, files, and directories.
//
// Basic usage example:
//
//	obf, err := api.NewObfuscator(api.Options{ConfigPath: "config.yaml"})
//	if err != nil {
//	    log.Fatalf("Failed to create obfuscator: %v", err)
//	}
//
//	result, err := obf.ObfuscateCode("<?php echo 'Hello World'; ?>")
//	if err != nil {
//	    log.Fatalf("Failed to obfuscate code: %v", err)
//	}
//
//	fmt.Println(result) // Prints obfuscated PHP code
package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/whit3rabbit/phpmixer/internal/config"
	"github.com/whit3rabbit/phpmixer/internal/obfuscator"
	"github.com/whit3rabbit/phpmixer/internal/scrambler"
)

// PrintInfo prints formatted information to stdout, respecting the Testing flag.
// If Testing mode is active, no output will be generated.
// This function forwards to the internal config.PrintInfo function.
func PrintInfo(format string, args ...interface{}) {
	config.PrintInfo(format, args...)
}

// Obfuscator represents the main obfuscation engine that can be used to obfuscate PHP code.
// It encapsulates the configuration and context needed for obfuscation operations.
type Obfuscator struct {
	// Context holds the obfuscation context including scramblers and state
	Context *obfuscator.ObfuscationContext
	// Config holds the configuration settings for obfuscation
	Config *config.Config
}

// Options represents configuration options for creating a new Obfuscator instance.
type Options struct {
	// ConfigPath is the path to a YAML configuration file
	// If empty, default configuration will be used
	ConfigPath string

	// Silent suppresses informational messages during obfuscation
	// Set to true to make the obfuscator operate silently
	Silent bool

	// ConfigOverrides allows overriding specific config options
	// This is reserved for future use and not currently implemented
	ConfigOverrides map[string]interface{}
}

// NewObfuscator creates a new Obfuscator instance using the provided options.
//
// If ConfigPath is empty, default configuration will be used.
// If Silent is true, informational messages will be suppressed.
// ConfigOverrides is reserved for future use and not currently implemented.
//
// Returns an error if the configuration cannot be loaded or the context cannot be created.
func NewObfuscator(options Options) (*Obfuscator, error) {
	// Load configuration
	cfg, err := config.LoadConfig(options.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Apply options
	if options.Silent {
		cfg.Silent = true
	}

	// Apply any config overrides (future enhancement)
	// TODO: Implement config override functionality

	// Create obfuscation context
	ctx, err := obfuscator.NewObfuscationContext(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create obfuscation context: %w", err)
	}

	return &Obfuscator{
		Context: ctx,
		Config:  cfg,
	}, nil
}

// ObfuscateCode obfuscates a string of PHP code and returns the obfuscated code.
// Uses a temporary file approach since the underlying obfuscator works with files.
//
// Returns the obfuscated PHP code as a string, or an error if obfuscation fails.
func (o *Obfuscator) ObfuscateCode(code string) (string, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "php-code-*.php")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file

	// Write the code to the temp file
	if _, err := tmpFile.Write([]byte(code)); err != nil {
		return "", fmt.Errorf("failed to write to temporary file: %w", err)
	}

	// Close the file to ensure content is flushed
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Process the temporary file
	result, err := obfuscator.ProcessFile(tmpFile.Name(), o.Context)
	if err != nil {
		return "", fmt.Errorf("failed to obfuscate code: %w", err)
	}

	return result, nil
}

// ObfuscateFile obfuscates a PHP file and returns the obfuscated code.
//
// Parameters:
//   - filePath: The path to the PHP file to obfuscate
//
// Returns the obfuscated PHP code as a string, or an error if obfuscation fails.
func (o *Obfuscator) ObfuscateFile(filePath string) (string, error) {
	// Process the file
	result, err := obfuscator.ProcessFile(filePath, o.Context)
	if err != nil {
		return "", fmt.Errorf("failed to obfuscate file %s: %w", filePath, err)
	}
	return result, nil
}

// ObfuscateFileToFile obfuscates a PHP file and writes the result to another file.
//
// Parameters:
//   - inputPath: The path to the PHP file to obfuscate
//   - outputPath: The path where the obfuscated code will be written
//
// Returns an error if obfuscation or file operations fail.
func (o *Obfuscator) ObfuscateFileToFile(inputPath, outputPath string) error {
	// Process the file
	result, err := obfuscator.ProcessFile(inputPath, o.Context)
	if err != nil {
		return fmt.Errorf("failed to obfuscate file %s: %w", inputPath, err)
	}

	// Create output directory if needed
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Write the result to the output file
	if err := os.WriteFile(outputPath, []byte(result), 0644); err != nil {
		return fmt.Errorf("failed to write to output file %s: %w", outputPath, err)
	}

	return nil
}

// ObfuscateDirectory obfuscates all PHP files in a directory and writes the results to another directory.
//
// Parameters:
//   - inputDir: The source directory containing PHP files to obfuscate
//   - outputDir: The target directory where obfuscated files will be written
//
// The function will:
// 1. Load any existing context from the output directory
// 2. Create the output directory if it doesn't exist
// 3. Process all PHP files recursively, preserving directory structure
// 4. Copy non-PHP files to the output directory
// 5. Skip files that match patterns in the configuration's skip list
// 6. Save the obfuscation context to the output directory
//
// Returns an error if directory operations or obfuscation fail.
func (o *Obfuscator) ObfuscateDirectory(inputDir, outputDir string) error {
	// We need to implement directory processing logic here since the internal function
	// is not readily available

	// Get stats for the input directory
	inputInfo, err := os.Stat(inputDir)
	if err != nil {
		return fmt.Errorf("failed to stat input directory %s: %w", inputDir, err)
	}

	if !inputInfo.IsDir() {
		return fmt.Errorf("input path %s is not a directory", inputDir)
	}

	// Make sure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// First, load any existing context if available
	if err := o.Context.Load(outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load existing context: %v\n", err)
		fmt.Fprintf(os.Stderr, "Starting with fresh context.\n")
	}

	// Update the config with the target directory
	o.Config.TargetDirectory = outputDir

	// Process the directory recursively
	if err := o.processDirectoryRecursive(inputDir, outputDir); err != nil {
		return err
	}

	// Save the updated context to the output directory
	if err := o.Context.Save(outputDir); err != nil {
		return fmt.Errorf("failed to save obfuscation context: %w", err)
	}

	return nil
}

// processDirectoryRecursive is a helper function for recursive directory processing
func (o *Obfuscator) processDirectoryRecursive(inputDir, outputDir string) error {
	// Read directory entries
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", inputDir, err)
	}

	// Process each entry
	for _, entry := range entries {
		inputPath := filepath.Join(inputDir, entry.Name())
		outputPath := filepath.Join(outputDir, entry.Name())

		// Skip files/dirs based on config
		// Get relative path for matching in skiplist
		relPath, err := filepath.Rel(inputDir, inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to get relative path for %s: %v\n", inputPath, err)
			relPath = entry.Name() // Fallback to just the name
		}

		// Check if file should be skipped
		skip := shouldSkipPath(relPath, o.Config.SkipPaths)
		if skip {
			PrintInfo("Skipping path (matches skiplist): %s\n", relPath)
			continue
		}

		// Handle directories
		if entry.IsDir() {
			// Create corresponding output directory
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fmt.Errorf("failed to create output directory %s: %w", outputPath, err)
			}

			// Recursively process subdirectory
			if err := o.processDirectoryRecursive(inputPath, outputPath); err != nil {
				return err
			}
			continue
		}

		// Handle PHP files
		if isPhpFile(entry.Name()) {
			// Process the file
			obfuscated, err := obfuscator.ProcessFile(inputPath, o.Context)
			if err != nil {
				if o.Config.AbortOnError {
					return fmt.Errorf("failed to process %s: %w", inputPath, err)
				}
				// Just log and continue
				fmt.Fprintf(os.Stderr, "Warning: Failed to process %s: %v\n", inputPath, err)
			} else {
				// Write obfuscated content to output file
				if err := os.WriteFile(outputPath, []byte(obfuscated), 0644); err != nil {
					if o.Config.AbortOnError {
						return fmt.Errorf("failed to write output to %s: %w", outputPath, err)
					}
					// Just log and continue
					fmt.Fprintf(os.Stderr, "Warning: Failed to write output to %s: %v\n", outputPath, err)
				} else {
					PrintInfo("Processed: %s -> %s\n", inputPath, outputPath)
				}
			}
		} else {
			// Just copy non-PHP files
			// Read the file
			content, err := os.ReadFile(inputPath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", inputPath, err)
			}

			// Write to output
			if err := os.WriteFile(outputPath, content, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", outputPath, err)
			}

			PrintInfo("Copied: %s -> %s\n", inputPath, outputPath)
		}
	}

	return nil
}

// Helper to determine if a file should be skipped based on patterns
func shouldSkipPath(path string, patterns []string) bool {
	if patterns == nil {
		return false
	}

	// Check against the skiplist
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, path)
		if err != nil {
			// If pattern is invalid, report warning but don't error out
			fmt.Fprintf(os.Stderr, "Warning: Invalid skip pattern '%s': %v\n", pattern, err)
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// Helper to check if a file is a PHP file
func isPhpFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".php" || ext == ".phtml" || ext == ".php5" || ext == ".php7" || ext == ".phps"
}

// LoadContext loads an existing obfuscation context from a directory.
//
// This is useful when you want to reuse the same obfuscation context (including name mappings)
// across multiple runs.
//
// Returns an error if the context cannot be loaded.
func (o *Obfuscator) LoadContext(baseDir string) error {
	return o.Context.Load(baseDir)
}

// SaveContext saves the current obfuscation context to a directory.
//
// This saves the current name mappings and other state to be loaded later.
//
// Returns an error if the context cannot be saved.
func (o *Obfuscator) SaveContext(baseDir string) error {
	return o.Context.Save(baseDir)
}

// LookupObfuscatedName looks up an obfuscated name from the context.
//
// Parameters:
//   - name: The original name to look up
//   - typeStr: The type of the name, one of: "variable", "function", "property",
//     "method", "class_const", "constant", "label"
//
// Returns the obfuscated name and nil if the name is found,
// or an empty string and an error if the name is not found.
func (o *Obfuscator) LookupObfuscatedName(name string, typeStr string) (string, error) {
	// Convert string type to ScrambleType
	var sType scrambler.ScrambleType
	switch typeStr {
	case "variable":
		sType = scrambler.TypeVariable
	case "function":
		sType = scrambler.TypeFunction
	case "property":
		sType = scrambler.TypeProperty
	case "method":
		sType = scrambler.TypeMethod
	case "class_const":
		sType = scrambler.TypeClassConstant
	case "constant":
		sType = scrambler.TypeConstant
	case "label":
		sType = scrambler.TypeLabel
	default:
		return "", fmt.Errorf("unknown scramble type: %s", typeStr)
	}

	// Get the scrambler
	s := o.Context.GetScrambler(sType)
	if s == nil {
		return "", fmt.Errorf("scrambler not found for type: %s", typeStr)
	}

	// Convert to *scrambler.Scrambler to access methods
	scr, ok := s.(*scrambler.Scrambler)
	if !ok {
		return "", fmt.Errorf("failed to convert scrambler interface to concrete type")
	}

	// Lookup the name
	obfName, found := scr.LookupObfuscated(name)
	if !found {
		return "", fmt.Errorf("name not found in context: %s", name)
	}

	return obfName, nil
}

