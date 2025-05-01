package main

import (
	"fmt"
	"log"
	"os"

	"github.com/whit3rabbit/phpmixer/pkg/api"
)

func main() {
	// Create a new obfuscator with default configuration
	obf, err := api.NewObfuscator(api.Options{
		ConfigPath: "config.yaml", // Path to your config file
		Silent:     false,         // Set to true to suppress informational messages
	})
	if err != nil {
		log.Fatalf("Failed to create obfuscator: %v", err)
	}

	// Example 1: Obfuscate a PHP code string
	phpCode := `<?php
// This is a sample PHP file
function greet($name) {
    echo "Hello, " . $name . "!";
}

$user = "World";
greet($user);
?>`

	obfuscated, err := obf.ObfuscateCode(phpCode)
	if err != nil {
		log.Fatalf("Failed to obfuscate code: %v", err)
	}
	fmt.Println("Obfuscated code:")
	fmt.Println(obfuscated)

	// Example 2: Obfuscate a PHP file
	inputFile := "input.php"
	outputFile := "output.php"

	// First, create a sample input file
	err = os.WriteFile(inputFile, []byte(phpCode), 0644)
	if err != nil {
		log.Fatalf("Failed to create input file: %v", err)
	}
	defer os.Remove(inputFile) // Clean up

	// Obfuscate the file
	err = obf.ObfuscateFileToFile(inputFile, outputFile)
	if err != nil {
		log.Fatalf("Failed to obfuscate file: %v", err)
	}
	defer os.Remove(outputFile) // Clean up

	fmt.Printf("File obfuscated: %s -> %s\n", inputFile, outputFile)

	// Example 3: Obfuscate a directory
	// Uncomment and use if needed:
	/*
		inputDir := "input_dir"
		outputDir := "output_dir"

		// Create the input directory if it doesn't exist
		if err := os.MkdirAll(inputDir, 0755); err != nil {
			log.Fatalf("Failed to create input directory: %v", err)
		}
		// Create a sample PHP file in the input directory
		err = os.WriteFile(filepath.Join(inputDir, "sample.php"), []byte(phpCode), 0644)
		if err != nil {
			log.Fatalf("Failed to create sample file: %v", err)
		}

		// Obfuscate the directory
		err = obf.ObfuscateDirectory(inputDir, outputDir)
		if err != nil {
			log.Fatalf("Failed to obfuscate directory: %v", err)
		}

		fmt.Printf("Directory obfuscated: %s -> %s\n", inputDir, outputDir)
	*/

	// Example 4: Look up an obfuscated name (after obfuscation has been done)
	// For this to work, you'd need to have processed files and saved context
	/*
		originalName := "greet"
		obfuscatedName, err := obf.LookupObfuscatedName(originalName, "function")
		if err != nil {
			log.Printf("Failed to look up obfuscated name: %v", err)
		} else {
			fmt.Printf("Original function 'greet' was obfuscated to: %s\n", obfuscatedName)
		}
	*/
}

