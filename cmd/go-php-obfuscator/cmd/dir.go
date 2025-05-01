package cmd

import (
	"fmt"
	"io"
	"io/fs" // Import the fs package
	"os"
	"path/filepath"
	"runtime"
	"strings" // Import strings for extension check
	"syscall" // For EvalSymlinks error handling

	"github.com/spf13/cobra"

	"github.com/whit3rabbit/phpmixer/internal/obfuscator" // Import context & ProcessFile
	// "github.com/spf13/viper" // Might be needed later
)

var (
	outputDir string // Flag variable for output directory
	cleanMode bool   // Flag variable for cleaning target directory
)

// dirCmd represents the obfuscate dir command
var dirCmd = &cobra.Command{
	Use:   "dir <source_directory>",
	Short: "Obfuscate PHP code in a directory recursively",
	Long: `Recursively scans the source directory for PHP files (based on configured extensions),
applies obfuscation, and outputs the results to the specified target directory,
preserving the original structure. Manages a shared context for name scrambling.`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		outputDirFromFlag, _ := cmd.Flags().GetString("output")
		if outputDirFromFlag == "" {
			return fmt.Errorf("output directory (-o, --output) is required for directory obfuscation")
		}
		// Check if source directory exists
		sourceDir := args[0]
		info, err := os.Stat(sourceDir)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("source directory '%s' not found", sourceDir)
			}
			return fmt.Errorf("error checking source directory '%s': %w", sourceDir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("source path '%s' is not a directory", sourceDir)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			return fmt.Errorf("configuration not loaded")
		}
		cmd.SilenceUsage = true

		sourceDir := args[0]
		targetDir := outputDir          // Use the flag variable
		cfg.TargetDirectory = targetDir // Update config struct

		if !cfg.Silent {
			fmt.Println("--- Directory Obfuscation ---")
			fmt.Printf("Source Directory: %s\n", sourceDir)
			fmt.Printf("Target Directory: %s\n", cfg.TargetDirectory)
			fmt.Printf("Clean Mode: %t\n", cleanMode)
			fmt.Println("---------------------------")
		}

		// --- Clean Target Directory ---
		if cleanMode {
			targetPath := cfg.TargetDirectory
			if targetPath == "" {
				return fmt.Errorf("cannot clean: target directory is not specified")
			}
			if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
				if !cfg.Silent {
					fmt.Printf("Info: Cleaning target directory: %s\n", targetPath)
				}
				// Refined safety check: block only root, current, or parent directory specifically
				isRoot := targetPath == filepath.VolumeName(targetPath)+"\\" // Check for C:\ etc. on Windows (Escaped backslash)
				if runtime.GOOS != "windows" {
					isRoot = targetPath == "/"
				}
				if isRoot || targetPath == "." || targetPath == ".." {
					return fmt.Errorf("refusing to clean potentially dangerous path: %s", targetPath)
				}
				// Consider adding marker file check for safety here
				err := os.RemoveAll(targetPath)
				if err != nil {
					return fmt.Errorf("failed to clean target directory %s: %w", targetPath, err)
				}
				if !cfg.Silent {
					fmt.Println("Info: Target directory cleaned.")
				}
			} else if !cfg.Silent {
				fmt.Printf("Info: Target directory %s does not exist, no cleaning needed.\n", targetPath)
			}
		}

		// --- Create Target Directory Structure ---
		if !cfg.Silent {
			fmt.Println("Info: Creating target directory structure (if needed)...")
		}
		obfuscatedPath := filepath.Join(cfg.TargetDirectory, "obfuscated")
		contextPath := filepath.Join(cfg.TargetDirectory, "context") // Context saved relative to target base
		if err := os.MkdirAll(obfuscatedPath, 0755); err != nil {
			return fmt.Errorf("failed to create obfuscated directory %s: %w", obfuscatedPath, err)
		}
		if err := os.MkdirAll(contextPath, 0755); err != nil {
			return fmt.Errorf("failed to create context directory %s: %w", contextPath, err)
		}

		// --- Initialize Obfuscation Context ---
		octx, err := obfuscator.NewObfuscationContext(cfg)
		if err != nil {
			return fmt.Errorf("failed to initialize obfuscation context: %w", err)
		}

		// --- Load Existing Context ---
		// Context is loaded/saved relative to the main target directory (not the 'obfuscated' subdir)
		if err := octx.Load(cfg.TargetDirectory); err != nil {
			fmt.Fprintf(os.Stderr, "Warning during context load: %v\n", err)
			// Decide if any Load errors should be fatal - currently just warning
		}

		// --- Visited Path Tracking (for loop prevention) ---
		// Stores canonical paths to prevent processing the same file/dir multiple times via links
		visitedPaths := make(map[string]bool)

		// --- Directory Walking and Processing ---
		if !cfg.Silent {
			fmt.Println("Info: Starting directory walk...")
		}
		var collectedErrors []error

		walkErr := filepath.WalkDir(sourceDir, func(entryPath string, d fs.DirEntry, err error) error {
			if err != nil {
				walkErr := fmt.Errorf("error accessing path %q: %w", entryPath, err)
				collectedErrors = append(collectedErrors, walkErr)
				if cfg.AbortOnError {
					return walkErr
				} // Abort on access error
				return nil // Skip this entry but continue walking
			}

			// --- Canonical Path & Loop Check ---
			canonicalPath, err := filepath.EvalSymlinks(entryPath)
			if err != nil {
				// Handle specific error cases
				if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ENOENT {
					// Likely a broken symlink - warn and skip
					if !cfg.Silent {
						fmt.Printf("Warning: Skipping broken symlink/path %q: %v\n", entryPath, err)
					}
					return nil
				} else {
					// More serious error evaluating symlinks
					evalErr := fmt.Errorf("error resolving path %q: %w", entryPath, err)
					collectedErrors = append(collectedErrors, evalErr)
					if cfg.AbortOnError {
						return evalErr
					}
					return nil // Skip entry on error
				}
			}

			// Check if we've already processed the canonical path
			if visitedPaths[canonicalPath] {
				if !cfg.Silent && cfg.DebugMode {
					fmt.Printf("Debug: Skipping already visited canonical path %q (from %q)\n", canonicalPath, entryPath)
				}
				if d.IsDir() {
					// Skip directory if already visited
					return filepath.SkipDir
				}
				return nil // Skip visited file/link
			}
			// Will mark as visited below before actual processing

			// Calculate relative path
			relPath, err := filepath.Rel(sourceDir, entryPath)
			if err != nil {
				relErr := fmt.Errorf("error calculating relative path for %q: %w", entryPath, err)
				collectedErrors = append(collectedErrors, relErr)
				if cfg.AbortOnError {
					return relErr
				} // Abort if cannot get relative path
				return nil // Skip this entry
			}

			// Skip root source directory itself
			if relPath == "." {
				return nil
			}

			// --- Check Skip Logic --- (Uses relative path)
			isSkipped, skipErr := checkPathAgainstPatterns(relPath, cfg.SkipPaths)
			if skipErr != nil {
				patternErr := fmt.Errorf("error matching skip pattern '%s': %w", relPath, skipErr)
				collectedErrors = append(collectedErrors, patternErr)
				if cfg.AbortOnError {
					return patternErr
				} // Abort on bad pattern
			} else if isSkipped {
				if !cfg.Silent {
					fmt.Printf("Skipping: %s\n", entryPath)
				}
				if d.IsDir() {
					return filepath.SkipDir // Don't walk into skipped directories
				} else {
					return nil // Skip this file
				}
			}

			// --- Check Keep Logic --- (Uses relative path, checked after Skip)
			targetEntryPathForKeep := filepath.Join(cfg.TargetDirectory, relPath) // Kept files go to root of target, not 'obfuscated'
			isKept, keepErr := checkPathAgainstPatterns(relPath, cfg.KeepPaths)
			if keepErr != nil {
				patternErr := fmt.Errorf("error matching keep pattern '%s': %w", relPath, keepErr)
				collectedErrors = append(collectedErrors, patternErr)
				if cfg.AbortOnError {
					return patternErr
				} // Abort on bad pattern
			} else if isKept {
				if !cfg.Silent {
					fmt.Printf("Keeping (Copying): %s -> %s\n", entryPath, targetEntryPathForKeep)
				}
				// Mark as visited before processing kept files
				visitedPaths[canonicalPath] = true

				if d.IsDir() {
					// Ensure target directory exists for kept directory structure
					if err := os.MkdirAll(targetEntryPathForKeep, 0755); err != nil {
						mkdirErr := fmt.Errorf("error creating directory for kept path %s: %w", targetEntryPathForKeep, err)
						collectedErrors = append(collectedErrors, mkdirErr)
						if cfg.AbortOnError {
							return mkdirErr
						}
						return nil // Skip if cannot create dir
					}
					// Don't return SkipDir, we need to walk into kept dirs to copy their contents
					return nil
				} else {
					// Keep file: Ensure parent dir exists and copy the file
					if err := os.MkdirAll(filepath.Dir(targetEntryPathForKeep), 0755); err != nil {
						mkdirErr := fmt.Errorf("error creating directory for kept file %s: %w", targetEntryPathForKeep, err)
						collectedErrors = append(collectedErrors, mkdirErr)
						if cfg.AbortOnError {
							return mkdirErr
						}
						return nil // Skip this file
					}
					if err := copyFile(entryPath, targetEntryPathForKeep); err != nil {
						copyErr := fmt.Errorf("error copying kept file %s to %s: %w", entryPath, targetEntryPathForKeep, err)
						collectedErrors = append(collectedErrors, copyErr)
						if cfg.AbortOnError {
							return copyErr
						}
						return nil // Skip this file on copy error
					}
					return nil // File processed (copied), stop further processing for this entry
				}
			}

			// --- Default Processing (If not Skipped or Kept) ---
			targetEntryPath := filepath.Join(obfuscatedPath, relPath) // Default target is inside 'obfuscated'

			// --- Symlink Handling ---
			if d.Type()&fs.ModeSymlink != 0 {
				// Mark original symlink path as visited to prevent re-evaluating it
				visitedPaths[canonicalPath] = true

				linkTarget, err := os.Readlink(entryPath)
				if err != nil {
					linkErr := fmt.Errorf("error reading symlink %q: %w", entryPath, err)
					collectedErrors = append(collectedErrors, linkErr)
					if cfg.AbortOnError {
						return linkErr
					}
					return nil // Skip this entry on read error
				}

				if !cfg.FollowSymlinks {
					// --- Copy Link ---
					if !cfg.Silent {
						fmt.Printf("Copying symlink: %s -> %s\n", entryPath, linkTarget)
					}
					// Ensure parent directory exists for the link
					if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0755); err != nil {
						mkdirErr := fmt.Errorf("error creating directory for symlink %s: %w", targetEntryPath, err)
						collectedErrors = append(collectedErrors, mkdirErr)
						if cfg.AbortOnError {
							return mkdirErr
						}
						return nil // Skip this symlink if cannot create parent dir
					}
					// Create the symlink in the target directory
					if err := os.Symlink(linkTarget, targetEntryPath); err != nil {
						// Check if it already exists
						if !os.IsExist(err) {
							symlinkCreateErr := fmt.Errorf("error creating symlink %s -> %s: %w", targetEntryPath, linkTarget, err)
							collectedErrors = append(collectedErrors, symlinkCreateErr)
							if cfg.AbortOnError {
								return symlinkCreateErr
							}
							return nil // Skip this link on creation error
						} else if !cfg.Silent {
							fmt.Printf("Info: Symlink %s already exists, skipping creation.\n", targetEntryPath)
						}
					}
				} else {
					// --- Follow Link ---
					resolvedTarget := linkTarget
					if !filepath.IsAbs(linkTarget) {
						resolvedTarget = filepath.Join(filepath.Dir(entryPath), linkTarget)
					}

					if !cfg.Silent {
						fmt.Printf("Following symlink: %s -> %s\n", entryPath, resolvedTarget)
					}

					// --- Canonical Path & Loop Check for TARGET ---
					targetCanonicalPath, err := filepath.EvalSymlinks(resolvedTarget)
					if err != nil {
						// Handle broken links or other errors for the target
						if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ENOENT {
							if !cfg.Silent {
								fmt.Printf("Warning: Skipping symlink %q because target %q does not exist or is broken link.\n", entryPath, resolvedTarget)
							}
							return nil // Skip broken target link
						} else {
							evalErr := fmt.Errorf("error resolving symlink target path %q (from %q): %w", resolvedTarget, entryPath, err)
							collectedErrors = append(collectedErrors, evalErr)
							if cfg.AbortOnError {
								return evalErr
							}
							return nil // Skip entry on error
						}
					}

					if visitedPaths[targetCanonicalPath] {
						// Skip if target already processed
						if !cfg.Silent && cfg.DebugMode {
							fmt.Printf("Debug: Skipping symlink %q because target canonical path %q was already visited.\n", entryPath, targetCanonicalPath)
						}
						return nil
					}
					// Mark target canonical path as visited before processing it
					visitedPaths[targetCanonicalPath] = true

					targetInfo, err := os.Stat(resolvedTarget) // Use Stat, not Lstat, to follow the link
					if err != nil {
						linkErr := fmt.Errorf("error following symlink %q -> %q: %w", entryPath, resolvedTarget, err)
						collectedErrors = append(collectedErrors, linkErr)
						if cfg.AbortOnError {
							return linkErr
						}
						return nil // Skip this entry on stat error
					}

					// Ensure parent directory for the *target* structure exists
					if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0755); err != nil {
						mkdirErr := fmt.Errorf("error creating directory for symlink target %q: %w", targetEntryPath, err)
						collectedErrors = append(collectedErrors, mkdirErr)
						if cfg.AbortOnError {
							return mkdirErr
						}
						return nil
					}

					if targetInfo.IsDir() {
						// --- Follow Directory Symlink ---
						if !cfg.Silent {
							fmt.Printf("Recursively copying directory contents from symlink target: %s -> %s\n", resolvedTarget, targetEntryPath)
						}

						// Create the target directory
						if err := os.MkdirAll(targetEntryPath, 0755); err != nil {
							mkdirErr := fmt.Errorf("error creating directory for symlink target %q: %w", targetEntryPath, err)
							collectedErrors = append(collectedErrors, mkdirErr)
							if cfg.AbortOnError {
								return mkdirErr
							}
							return nil
						}

						// Use the recursive copy function for linked directories
						err = copyDirectoryRecursively(resolvedTarget, targetEntryPath, visitedPaths, cfg.AbortOnError, cfg.Silent, cfg.DebugMode, &collectedErrors)
						if err != nil && cfg.AbortOnError {
							return err // Abort if recursive copy failed critically
						}
					} else {
						// --- Follow File Symlink ---
						if !cfg.Silent {
							fmt.Printf("Copying file from symlink target: %s -> %s\n", resolvedTarget, targetEntryPath)
						}

						// Copy the file from the symlink target
						err = copyFile(resolvedTarget, targetEntryPath)
						if err != nil {
							copyErr := fmt.Errorf("error copying symlink target file %q to %q: %w", resolvedTarget, targetEntryPath, err)
							collectedErrors = append(collectedErrors, copyErr)
							if cfg.AbortOnError {
								return copyErr
							}
						}
					}
				}
				return nil // Symlink handled (copied or followed), continue walk
			}

			// --- Regular Directory or File Handling ---
			// Mark canonical path as visited *before* copy/mkdir actions for this item
			visitedPaths[canonicalPath] = true

			if d.IsDir() {
				// Create corresponding directory structure in target 'obfuscated' path
				if !cfg.Silent && cfg.DebugMode {
					fmt.Printf("Ensuring dir: %s\n", targetEntryPath)
				}
				if err := os.MkdirAll(targetEntryPath, 0755); err != nil {
					mkdirErr := fmt.Errorf("error creating directory %q: %w", targetEntryPath, err)
					collectedErrors = append(collectedErrors, mkdirErr)
					if cfg.AbortOnError {
						return mkdirErr
					}
				}
				return nil // Continue walking into directory
			}

			// It's a regular file

			// --- Timestamp Check --- (Only if target exists)
			sourceInfo, err := d.Info() // Get source file info from DirEntry
			if err != nil {
				statErr := fmt.Errorf("error getting source file info for %s: %w", entryPath, err)
				collectedErrors = append(collectedErrors, statErr)
				if cfg.AbortOnError {
					return statErr
				}
				return nil // Skip this file if we can't get info
			}

			targetInfo, err := os.Stat(targetEntryPath)
			if err == nil {
				// Target exists, compare modification times
				if targetInfo.ModTime().After(sourceInfo.ModTime()) {
					if !cfg.Silent {
						fmt.Printf("Skipping (target newer): %s\n", entryPath)
					}
					return nil // Skip processing/copying
				}
			} else if !os.IsNotExist(err) {
				// Error stating target file (other than NotExist)
				statErr := fmt.Errorf("error stating target file %s: %w", targetEntryPath, err)
				collectedErrors = append(collectedErrors, statErr)
				if cfg.AbortOnError {
					return statErr
				}
				return nil // Skip this file
			}
			// If target doesn't exist (os.IsNotExist), proceed normally.

			// Check if it's a PHP file
			isPhp := false
			ext := strings.ToLower(filepath.Ext(entryPath))
			for _, phpExt := range cfg.ObfuscatePhpExtensions {
				compareTo := "." + strings.ToLower(phpExt)
				if ext == compareTo {
					isPhp = true
					break
				}
			}

			// Ensure target directory exists
			if err := os.MkdirAll(filepath.Dir(targetEntryPath), 0755); err != nil {
				mkdirErr := fmt.Errorf("error creating directory for file %s: %w", targetEntryPath, err)
				collectedErrors = append(collectedErrors, mkdirErr)
				if cfg.AbortOnError {
					return mkdirErr
				}
				return nil // Skip this file
			}

			if isPhp {
				// Process PHP file
				if !cfg.Silent {
					fmt.Printf("Processing PHP: %s -> %s\n", entryPath, targetEntryPath)
				}

				// Process the file using the refactored logic
				outputContent, processErr := obfuscator.ProcessFile(entryPath, octx)
				if processErr != nil {
					procErr := fmt.Errorf("error processing file %s: %w", entryPath, processErr)
					collectedErrors = append(collectedErrors, procErr)
					if cfg.AbortOnError {
						return procErr
					} // Abort on processing error
					return nil // Skip this file
				}

				// Write the processed content
				writeErr := os.WriteFile(targetEntryPath, []byte(outputContent), 0644)
				if writeErr != nil {
					wrErr := fmt.Errorf("error writing output file %s: %w", targetEntryPath, writeErr)
					collectedErrors = append(collectedErrors, wrErr)
					if cfg.AbortOnError {
						return wrErr
					} // Abort on write error
					return nil // Skip this file
				}
			} else {
				// Handle non-PHP files (and not skipped/kept) -> Copy
				if !cfg.Silent {
					fmt.Printf("Copying file: %s -> %s\n", entryPath, targetEntryPath)
				}

				// Call the copy function
				if err := copyFile(entryPath, targetEntryPath); err != nil {
					copyErr := fmt.Errorf("error copying file %s to %s: %w", entryPath, targetEntryPath, err)
					collectedErrors = append(collectedErrors, copyErr)
					if cfg.AbortOnError {
						return copyErr
					} // Abort on copy error
					return nil // Skip this file
				}
			}

			return nil // Continue walking
		}) // End of WalkDir callback

		if walkErr != nil {
			// This error comes directly from WalkDir (e.g., permissions) or if a callback returned non-nil
			// It might have already been added to collectedErrors by the callback, but add it again just in case?
			// Or perhaps just return it directly, as it likely signifies a more major walk issue.
			finalWalkErr := fmt.Errorf("error during directory walk of %s: %w", sourceDir, walkErr)
			collectedErrors = append(collectedErrors, finalWalkErr) // Add it to the list
			// If AbortOnError is true, walkErr would have been returned already. If false, we collected it.
			// We'll return a general error later if collectedErrors is not empty.
		}

		// --- Save Updated Context ---
		if err := octx.Save(cfg.TargetDirectory); err != nil {
			// Saving errors are usually more critical
			saveErr := fmt.Errorf("failed to save obfuscation context: %w", err)
			collectedErrors = append(collectedErrors, saveErr)
			// Don't necessarily return here, report all errors at the end
		}

		// --- Final Error Reporting ---
		if len(collectedErrors) > 0 {
			fmt.Fprintf(os.Stderr, "\n--- Errors Encountered (%d) ---\n", len(collectedErrors))
			for i, e := range collectedErrors {
				fmt.Fprintf(os.Stderr, "  %d: %v\n", i+1, e)
			}
			fmt.Fprintln(os.Stderr, "-----------------------------")
			// Return a generic error indicating failures occurred
			return fmt.Errorf("directory processing finished with %d errors", len(collectedErrors))
		}

		if !cfg.Silent {
			// fmt.Printf("Directory processing finished. Encountered %d errors.\n", errorCount)
			fmt.Println("Directory processing finished successfully.")
		}
		/* Removed old error check
		if errorCount > 0 && cfg.AbortOnError {
			// Although WalkDir might not have returned the error, we check our count if AbortOnError is set.
			return fmt.Errorf("processing aborted due to %d errors", errorCount)
		}
		*/

		return nil // Return nil for overall success if no errors were collected
	},
}

func init() {
	obfuscateCmd.AddCommand(dirCmd)
	dirCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory path (required)")
	dirCmd.Flags().BoolVar(&cleanMode, "clean", false, "Remove the target directory before obfuscating")
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %w", src, err)
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer source.Close()

	// Create destination file with same permissions, truncating if exists
	destination, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, sourceFileStat.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dst, err)
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	if err != nil {
		return fmt.Errorf("failed to copy data from %s to %s: %w", src, dst, err)
	}

	return nil
}

// checkPathAgainstPatterns checks if a given relative path matches any of the glob patterns.
// Normalizes path to use forward slashes for matching.
// Assumes patterns use forward slashes or OS-specific separators compatible with filepath.Match.
func checkPathAgainstPatterns(relPath string, patterns []string) (bool, error) {
	pathNormalized := filepath.ToSlash(relPath) // Convert path separators to /
	for _, pattern := range patterns {
		// Assume pattern is a glob pattern
		matched, err := filepath.Match(pattern, pathNormalized)
		if err != nil {
			// Invalid pattern in config - return error to let caller decide how critical it is
			return false, fmt.Errorf("invalid pattern '%s': %w", pattern, err)
		}
		if matched {
			return true, nil // Found a match
		}
	}
	return false, nil // No match found
}

// copyDirectoryRecursively copies contents from src to dst, checking visited paths.
func copyDirectoryRecursively(src, dst string, visitedPaths map[string]bool, abortOnError, silent, debug bool, collectedErrors *[]error) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Handle errors accessing files within the source directory being copied
			walkErr := fmt.Errorf("error accessing path %q during recursive copy from %q: %w", path, src, err)
			*collectedErrors = append(*collectedErrors, walkErr)
			if abortOnError {
				return walkErr
			}
			return nil // Skip entry
		}

		// --- Canonical Path & Loop Check (within the recursive copy) ---
		canonicalPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			// Handle potential errors resolving paths *within* the linked dir
			if pathErr, ok := err.(*os.PathError); ok && pathErr.Err == syscall.ENOENT {
				if !silent {
					fmt.Printf("Warning: Skipping broken link/path %q during recursive copy.\n", path)
				}
				return nil
			} else {
				evalErr := fmt.Errorf("error resolving path %q during recursive copy: %w", path, err)
				*collectedErrors = append(*collectedErrors, evalErr)
				if abortOnError {
					return evalErr
				}
				return nil
			}
		}

		if visitedPaths[canonicalPath] {
			if !silent && debug {
				fmt.Printf("Debug: Recursive copy skipping already visited canonical path %q (from %q)\n", canonicalPath, path)
			}
			if d.IsDir() {
				return filepath.SkipDir // Prevent descending further
			}
			return nil
		}

		// Calculate relative path within the source *being copied*
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			relErr := fmt.Errorf("error calculating relative path for %q during recursive copy: %w", path, err)
			*collectedErrors = append(*collectedErrors, relErr)
			if abortOnError {
				return relErr
			}
			return nil
		}
		if relPath == "." {
			return nil // Skip the root of the source itself
		}

		targetPath := filepath.Join(dst, relPath)

		// Mark canonical path as visited *before* copy/mkdir actions for this item
		visitedPaths[canonicalPath] = true

		// Handle symlinks encountered *during* the recursive copy
		if d.Type()&fs.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				linkErr := fmt.Errorf("error reading symlink %q during recursive copy: %w", path, err)
				*collectedErrors = append(*collectedErrors, linkErr)
				if abortOnError {
					return linkErr
				}
				return nil
			}
			// Always copy nested symlinks as links, do not follow recursively within the copy function
			if !silent {
				fmt.Printf("Copying nested symlink: %s -> %s\n", path, targetPath)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				mkdirErr := fmt.Errorf("error creating directory for nested symlink %q: %w", targetPath, err)
				*collectedErrors = append(*collectedErrors, mkdirErr)
				if abortOnError {
					return mkdirErr
				}
				return nil
			}
			if err := os.Symlink(linkTarget, targetPath); err != nil && !os.IsExist(err) {
				symlinkErr := fmt.Errorf("error creating nested symlink %q -> %q: %w", targetPath, linkTarget, err)
				*collectedErrors = append(*collectedErrors, symlinkErr)
				if abortOnError {
					return symlinkErr
				}
				return nil
			}
			return nil // Nested symlink copied
		}

		if d.IsDir() {
			if !silent && debug {
				fmt.Printf("Recursive copy ensuring dir: %s\n", targetPath)
			}
			if err := os.MkdirAll(targetPath, 0755); err != nil {
				mkdirErr := fmt.Errorf("error creating directory %q during recursive copy: %w", targetPath, err)
				*collectedErrors = append(*collectedErrors, mkdirErr)
				if abortOnError {
					return mkdirErr
				}
				return nil
			}
		} else {
			if !silent && debug {
				fmt.Printf("Recursive copy copying file: %s -> %s\n", path, targetPath)
			}
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				mkdirErr := fmt.Errorf("error creating directory for file %q during recursive copy: %w", targetPath, err)
				*collectedErrors = append(*collectedErrors, mkdirErr)
				if abortOnError {
					return mkdirErr
				}
				return nil
			}
			if err := copyFile(path, targetPath); err != nil {
				copyErr := fmt.Errorf("error copying file %q to %q during recursive copy: %w", path, targetPath, err)
				*collectedErrors = append(*collectedErrors, copyErr)
				if abortOnError {
					return copyErr
				}
				return nil
			}
		}
		return nil // Continue walking within the recursive copy source
	})
}
