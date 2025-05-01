package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Constants for string obfuscation techniques
const (
	StringObfuscationTechniqueBase64 = "base64"
	StringObfuscationTechniqueRot13  = "rot13"
	StringObfuscationTechniqueXOR    = "xor"
)

// --- Nested Configuration Structs ---

type StringObfuscationTechnique string

const (
	StringObfuscationTechniqueBase64Typed StringObfuscationTechnique = "base64"
	StringObfuscationTechniqueROT13Typed  StringObfuscationTechnique = "rot13"
	StringObfuscationTechniqueXORTyped    StringObfuscationTechnique = "xor"
)

type StatementShufflingChunkMode string

const (
	StatementShufflingChunkModeFixed StatementShufflingChunkMode = "fixed"
	StatementShufflingChunkModeRatio StatementShufflingChunkMode = "ratio"
)

// StringsConfig defines settings for string obfuscation
type StringsConfig struct {
	Enabled   bool   `yaml:"enabled" mapstructure:"enabled"`
	Technique string `yaml:"technique" mapstructure:"technique"`
	XorKey    string `yaml:"xor_key,omitempty" mapstructure:"xor_key,omitempty"`
}

// ScramblingConfig defines settings for name scrambling
type ScramblingConfig struct {
	Mode   string `yaml:"mode" mapstructure:"mode"`
	Length int    `yaml:"length" mapstructure:"length"`
}

// CommentsConfig defines settings for comment handling
type CommentsConfig struct {
	Strip bool `yaml:"strip" mapstructure:"strip"`
}

// NameToggleConfig defines settings for toggling name scrambling
type NameToggleConfig struct {
	Scramble bool `yaml:"scramble" mapstructure:"scramble"`
}

// ControlFlowConfig defines settings for control flow obfuscation
type ControlFlowConfig struct {
	Enabled          bool `yaml:"enabled" mapstructure:"enabled"`
	MaxNestingDepth  int  `yaml:"max_nesting_depth" mapstructure:"max_nesting_depth"`
	RandomConditions bool `yaml:"random_conditions" mapstructure:"random_conditions"`
	AddDeadBranches  bool `yaml:"add_dead_branches" mapstructure:"add_dead_branches"`
}

// AdvancedLoopsConfig defines settings for advanced loop obfuscation
type AdvancedLoopsConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

// ArrayAccessConfig defines settings for array access obfuscation
type ArrayAccessConfig struct {
	Enabled             bool `yaml:"enabled" mapstructure:"enabled"`
	ForceHelperFunction bool `yaml:"force_helper_function,omitempty" mapstructure:"force_helper_function,omitempty"`
}

// ArithmeticConfig defines settings for arithmetic expression obfuscation
type ArithmeticConfig struct {
	Enabled            bool `yaml:"enabled" mapstructure:"enabled"`
	ComplexityLevel    int  `yaml:"complexity_level" mapstructure:"complexity_level"`
	TransformationRate int  `yaml:"transformation_rate" mapstructure:"transformation_rate"`
}

// CodeInjectionConfig defines settings for code injection
type CodeInjectionConfig struct {
	Enabled       bool `yaml:"enabled" mapstructure:"enabled"`
	InjectionRate int  `yaml:"injection_rate" mapstructure:"injection_rate"`
}

// JunkCodeConfig defines settings for junk code insertion
type JunkCodeConfig struct {
	Enabled           bool `yaml:"enabled" mapstructure:"enabled"`
	InjectionRate     int  `yaml:"injection_rate" mapstructure:"injection_rate"`
	MaxInjectionDepth int  `yaml:"max_injection_depth" mapstructure:"max_injection_depth"`
}

// StatementShufflingConfig defines settings for statement shuffling
type StatementShufflingConfig struct {
	Enabled      bool   `yaml:"enabled" mapstructure:"enabled"`
	MinChunkSize int    `yaml:"min_chunk_size" mapstructure:"min_chunk_size"`
	ChunkMode    string `yaml:"chunk_mode" mapstructure:"chunk_mode"`
	ChunkRatio   int    `yaml:"chunk_ratio" mapstructure:"chunk_ratio"`
}

// IgnoreConfig defines lists of names to ignore during obfuscation
type IgnoreConfig struct {
	Functions []string `yaml:"functions" mapstructure:"functions"`
	Variables []string `yaml:"variables" mapstructure:"variables"`
	Classes   []string `yaml:"classes" mapstructure:"classes"`
	// Add fields for other types (properties, methods, constants) as needed
}

// ObfuscationConfig holds all obfuscation-specific settings
type ObfuscationConfig struct {
	Strings            StringsConfig            `yaml:"strings" mapstructure:"strings"`
	Scrambling         ScramblingConfig         `yaml:"scrambling" mapstructure:"scrambling"`
	Comments           CommentsConfig           `yaml:"comments" mapstructure:"comments"`
	Variables          NameToggleConfig         `yaml:"variables" mapstructure:"variables"`
	Functions          NameToggleConfig         `yaml:"functions" mapstructure:"functions"`
	Classes            NameToggleConfig         `yaml:"classes" mapstructure:"classes"`
	ControlFlow        ControlFlowConfig        `yaml:"control_flow" mapstructure:"control_flow"`
	AdvancedLoops      AdvancedLoopsConfig      `yaml:"advanced_loops" mapstructure:"advanced_loops"`
	ArrayAccess        ArrayAccessConfig        `yaml:"array_access" mapstructure:"array_access"`
	Arithmetic         ArithmeticConfig         `yaml:"arithmetic_expressions,omitempty" mapstructure:"arithmetic_expressions,omitempty"`
	DeadCode           CodeInjectionConfig      `yaml:"dead_code" mapstructure:"dead_code"`
	JunkCode           JunkCodeConfig           `yaml:"junk_code" mapstructure:"junk_code"`
	StatementShuffling StatementShufflingConfig `yaml:"statement_shuffling" mapstructure:"statement_shuffling"`
	Ignore             IgnoreConfig             `yaml:"ignore" mapstructure:"ignore"`
}

// Config holds all configuration settings for the obfuscator.
// Struct tags control how Viper maps config file keys and environment variables.
type Config struct {
	// Input/Output settings
	SourceDirectory string `mapstructure:"source_directory"`
	TargetDirectory string `mapstructure:"target_directory"`

	// General behavior
	Silent         bool   `mapstructure:"silent"`          // Suppress informational messages
	AbortOnError   bool   `mapstructure:"abort_on_error"`  // Stop processing on the first error
	Confirm        bool   `mapstructure:"confirm"`         // Ask for confirmation (currently unused placeholder)
	ParserMode     string `mapstructure:"parser_mode"`     // PHP Parser version preference (e.g., PREFER_PHP7)
	FollowSymlinks bool   `mapstructure:"follow_symlinks"` // Whether to follow symbolic links during directory processing
	DebugMode      bool   `mapstructure:"debug_mode"`      // Enable verbose debug logging

	// File Handling
	ObfuscatePhpExtensions []string `mapstructure:"obfuscatephpextensions"` // File extensions to treat as PHP
	SkipPaths              []string `mapstructure:"skip"`                   // Full paths or directories to completely ignore
	KeepPaths              []string `mapstructure:"keep"`                   // Full paths or directories to copy without obfuscating
	AllowEmptyFiles        bool     `mapstructure:"allow_and_overwrite_empty_files"`

	// Add the Obfuscation field to connect all the nested structs
	Obfuscation ObfuscationConfig `mapstructure:"obfuscation" yaml:"obfuscation"`

	// Scrambling Options
	ScrambleMode   string `mapstructure:"scramble_mode"`   // 'identifier', 'hexa', 'numeric'
	ScrambleLength int    `mapstructure:"scramble_length"` // Target length for scrambled names

	// Obfuscation Feature Toggles (What to obfuscate)
	StripIndentation            bool   `mapstructure:"strip_indentation"`
	StripComments               bool   `mapstructure:"strip_comments"` // Removes comments from the PHP code
	ObfuscateStringLiteral      bool   `mapstructure:"obfuscate_string_literal"`
	StringObfuscationTechnique  string `mapstructure:"string_obfuscation_technique"` // 'base64' or 'rot13'
	ObfuscateLoopStatement      bool   `mapstructure:"obfuscate_loop_statement"`
	ObfuscateIfStatement        bool   `mapstructure:"obfuscate_if_statement"`
	ObfuscateControlFlow        bool   `mapstructure:"obfuscate_control_flow"`         // Wrap blocks in if(true){} statements
	ControlFlowMaxNestingDepth  int    `mapstructure:"control_flow_max_nesting_depth"` // Maximum depth for nested control flow obfuscation
	ControlFlowRandomConditions bool   `mapstructure:"control_flow_random_conditions"` // Whether to use random conditions in control flow obfuscation
	ControlFlowAddDeadBranches  bool   `mapstructure:"control_flow_add_dead_branches"` // Whether to add bogus else branches to if statements
	UseAdvancedLoopObfuscation  bool   `mapstructure:"use_advanced_loop_obfuscation"`  // Whether to use advanced loop obfuscation techniques
	ObfuscateArrayAccess        bool   `mapstructure:"obfuscate_array_access"`         // Obfuscate how arrays/objects are accessed

	// Arithmetic Expression Obfuscation
	ObfuscateArithmeticExpressions bool `mapstructure:"obfuscate_arithmetic_expressions"` // Whether to obfuscate arithmetic expressions
	ArithmeticComplexityLevel      int  `mapstructure:"arithmetic_complexity_level"`      // Complexity level for arithmetic obfuscation (1-3)
	ArithmeticTransformationRate   int  `mapstructure:"arithmetic_transformation_rate"`   // Percentage of eligible expressions to transform (0-100)

	// Dead Code and Junk Code Insertion
	InjectDeadCode        bool `mapstructure:"inject_dead_code"`         // Whether to inject dead code blocks (if(false){...})
	InjectJunkCode        bool `mapstructure:"inject_junk_code"`         // Whether to inject junk statements (unused vars, no-op calcs)
	DeadJunkInjectionRate int  `mapstructure:"dead_junk_injection_rate"` // Percentage chance to inject at each opportunity (0-100)
	MaxInjectionDepth     int  `mapstructure:"max_injection_depth"`      // Maximum depth for code injection

	// Array Access Obfuscation Options
	ArrayAccess struct {
		ForceHelperFunction bool `mapstructure:"array_access_force_helper"` // Force including the helper function even if no replacements
	}

	ObfuscateConstantName      bool `mapstructure:"obfuscate_constant_name"`
	ObfuscateVariableName      bool `mapstructure:"obfuscate_variable_name"`
	ObfuscateFunctionName      bool `mapstructure:"obfuscate_function_name"`
	ObfuscateClassName         bool `mapstructure:"obfuscate_class_name"`
	ObfuscateInterfaceName     bool `mapstructure:"obfuscate_interface_name"`
	ObfuscateTraitName         bool `mapstructure:"obfuscate_trait_name"`
	ObfuscateClassConstantName bool `mapstructure:"obfuscate_class_constant_name"`
	ObfuscatePropertyName      bool `mapstructure:"obfuscate_property_name"`
	ObfuscateMethodName        bool `mapstructure:"obfuscate_method_name"`
	ObfuscateNamespaceName     bool `mapstructure:"obfuscate_namespace_name"`
	ObfuscateLabelName         bool `mapstructure:"obfuscate_label_name"`

	// Statement Shuffling Options
	ShuffleStmts             bool   `mapstructure:"shuffle_stmts"`
	ShuffleStmtsMinChunkSize int    `mapstructure:"shuffle_stmts_min_chunk_size"`
	ShuffleStmtsChunkMode    string `mapstructure:"shuffle_stmts_chunk_mode"` // 'fixed' or 'ratio'
	ShuffleStmtsChunkRatio   int    `mapstructure:"shuffle_stmts_chunk_ratio"`

	// Ignore Lists (Symbols NOT to obfuscate)
	IgnorePreDefinedClasses string   `mapstructure:"ignore_pre_defined_classes"` // 'all', 'none', or list TBD
	IgnoreConstants         []string `mapstructure:"ignore_constants"`
	IgnoreVariables         []string `mapstructure:"ignore_variables"`
	IgnoreFunctions         []string `mapstructure:"ignore_functions"`
	IgnoreClassConstants    []string `mapstructure:"ignore_class_constants"`
	IgnoreMethods           []string `mapstructure:"ignore_methods"`
	IgnoreProperties        []string `mapstructure:"ignore_properties"`
	IgnoreClasses           []string `mapstructure:"ignore_classes"`
	IgnoreInterfaces        []string `mapstructure:"ignore_interfaces"`
	IgnoreTraits            []string `mapstructure:"ignore_traits"`
	IgnoreNamespaces        []string `mapstructure:"ignore_namespaces"`
	IgnoreLabels            []string `mapstructure:"ignore_labels"`

	// Ignore Prefixes (Symbols starting with these prefixes NOT to obfuscate)
	IgnoreConstantsPrefix      []string `mapstructure:"ignore_constants_prefix"`
	IgnoreVariablesPrefix      []string `mapstructure:"ignore_variables_prefix"`
	IgnoreFunctionsPrefix      []string `mapstructure:"ignore_functions_prefix"`
	IgnoreClassConstantsPrefix []string `mapstructure:"ignore_class_constants_prefix"`
	IgnorePropertiesPrefix     []string `mapstructure:"ignore_properties_prefix"`
	IgnoreMethodsPrefix        []string `mapstructure:"ignore_methods_prefix"`
	IgnoreClassesPrefix        []string `mapstructure:"ignore_classes_prefix"`
	IgnoreInterfacesPrefix     []string `mapstructure:"ignore_interfaces_prefix"`
	IgnoreTraitsPrefix         []string `mapstructure:"ignore_traits_prefix"`
	IgnoreNamespacesPrefix     []string `mapstructure:"ignore_namespaces_prefix"`
	IgnoreLabelsPrefix         []string `mapstructure:"ignore_labels_prefix"`

	// User Comment Options
	UserComment             string `mapstructure:"user_comment"`
	ExtractCommentFromLine  int    `mapstructure:"extract_comment_from_line"` // 0 means disabled
	ExtractCommentToLine    int    `mapstructure:"extract_comment_to_line"`   // 0 means disabled
	MaxNestedDirectoryLevel int    `mapstructure:"max_nested_directory"`      // Protection against symlink loops

	// -- Internal/Derived fields (not loaded directly) --
	// TargetPhpVersion string // Maybe parse from parser_mode later?
}

// Default values for the configuration
// Viper requires keys to be lowercase for automatic env var binding.
var defaults = map[string]interface{}{
	"silent":                         false,
	"abortonerror":                   true,
	"confirm":                        true, // Default from yakpro, though unused currently
	"parsermode":                     "PREFER_PHP7",
	"followsymlinks":                 false,
	"obfuscatephpextensions":         []string{"php"},
	"skip":                           nil,
	"keep":                           nil,
	"allowemptyfiles":                true,
	"scramblemode":                   "identifier",
	"scramblelength":                 5,
	"stripindentation":               true,
	"stripcomments":                  false,
	"obfuscatestringliteral":         true,
	"stringobfuscationtechnique":     StringObfuscationTechniqueBase64,
	"obfuscateloopstatement":         true,
	"obfuscateifstatement":           true,
	"obfuscatecontrolflow":           false, // Disabled by default
	"controlflowmaxnestingdepth":     1,     // Default to single level nesting
	"controlflowrandomconditions":    true,  // Default to random conditions for better obfuscation
	"controlflowadddeadbranches":     true,  // Default to adding dead branches for better obfuscation
	"useadvancedloopobfuscation":     false, // Disabled by default - use simpler loop obfuscation
	"obfuscatearrayaccess":           false, // Disabled by default
	"obfuscatearithmeticexpressions": false, // Disabled by default
	"arithmeticcomplexitylevel":      1,     // Default to lowest complexity level (1-3)
	"arithmetictransformationrate":   80,    // Default to 80% transformation rate
	"injectdeadcode":                 false, // Disabled by default
	"injectjunkcode":                 false, // Disabled by default
	"deadjunkinjectionrate":          30,    // Default to 30% injection rate
	"maxinjectiondepth":              3,     // Default to 3 levels of nesting
	"array_access_force_helper":      false, // Default to not forcing helper function
	"obfuscateconstantname":          true,
	"obfuscatevariablename":          true,
	"obfuscatefunctionname":          true,
	"obfuscateclassname":             true,
	"obfuscateinterfacename":         true,
	"obfuscatetraitname":             true,
	"obfuscateclassconstantname":     true,
	"obfuscatepropertyname":          true,
	"obfuscatemethodname":            true,
	"obfuscatenamespacename":         true,
	"obfuscatelabelname":             true,
	"shufflestmts":                   true,
	"shufflestmtsminchunksize":       1,
	"shufflestmtschunkmode":          "fixed",
	"shufflestmtschunkratio":         20,
	"ignorepredefinedclasses":        "all",
	"ignoreconstants":                nil,
	"ignorevariables":                nil,
	"ignorefunctions":                nil,
	"ignoreclassconstants":           nil,
	"ignoremethods":                  nil,
	"ignoreproperties":               nil,
	"ignoreclasses":                  nil,
	"ignoreinterfaces":               nil,
	"ignoretraits":                   nil,
	"ignorenamespaces":               nil,
	"ignorelabels":                   nil,
	"ignoreconstantsprefix":          nil,
	"ignorevariablesprefix":          nil,
	"ignorefunctionsprefix":          nil,
	"ignoreclassconstantsprefix":     nil,
	"ignorepropertiesprefix":         nil,
	"ignoremethodsprefix":            nil,
	"ignoreclassesprefix":            nil,
	"ignoreinterfacesprefix":         nil,
	"ignoretraitsprefix":             nil,
	"ignorenamespacesprefix":         nil,
	"ignorelabelsprefix":             nil,
	"usercomment":                    "",
	"extractcommentfromline":         0,
	"extractcommenttoline":           0,
	"maxnesteddirectorylevel":        99,
	// --- Input/Output defaults ---
	// source_directory and target_directory often come from flags or specific logic,
	// so no default here. They might be set in a config file.
	"sourcedirectory": "",
	"targetdirectory": "",
	"debugmode":       false, // Default for DebugMode
}

var (
	// Testing controls whether output is suppressed for testing purposes
	Testing bool
)

// Example function to print info with respect for Testing mode
func PrintInfo(format string, args ...interface{}) {
	if !Testing {
		fmt.Printf(format, args...)
	}
}

// LoadConfig reads configuration from file, environment variables,
// and command-line flags, then returns a filled Config struct.
func LoadConfig(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	if configPath == "" {
		configPath = "config.yaml" // Default path
	}

	if _, err := os.Stat(configPath); err == nil {
		yamlFile, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("error reading config file %s: %w", configPath, err)
		}

		// Strict unmarshalling can help catch typos in the config file
		// err = yaml.UnmarshalStrict(yamlFile, cfg) // Consider using this
		err = yaml.Unmarshal(yamlFile, cfg) // Standard unmarshalling
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling config file %s: %w", configPath, err)
		}
		// Use PrintInfo to respect Testing flag
		if !cfg.Silent {
			PrintInfo("Info: Loaded configuration from %s\n", configPath)
		}

		// Optional: Add legacy key detection here if desired
		// var raw map[string]interface{}
		// if yaml.Unmarshal(yamlFile, &raw) == nil {
		//     // check raw for keys like "strip_comments" and print warnings
		// }

	} else if os.IsNotExist(err) {
		if configPath != "config.yaml" {
			return nil, fmt.Errorf("specified config file not found: %s", configPath)
		}
		// Use PrintInfo to respect Testing flag
		PrintInfo("Info: Configuration file 'config.yaml' not found, using default settings.\n")
	} else {
		return nil, fmt.Errorf("error checking config file %s: %w", configPath, err)
	}

	cfg.TargetDirectory = filepath.Clean(cfg.TargetDirectory)
	return cfg, nil
}

// SaveConfig saves the default configuration to a file.
func SaveConfig(configPath string) error {
	// Implementation unchanged from previous step, ensure it uses DefaultConfig()
	cfg := DefaultConfig()
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("error marshalling default config: %w", err)
	}
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating directory for config file %s: %w", configPath, err)
	}
	err = os.WriteFile(configPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("error writing config file %s: %w", configPath, err)
	}
	// Use PrintInfo to respect Testing flag
	PrintInfo("Info: Saved default configuration to %s\n", configPath)
	return nil
}

// DefaultConfig returns a configuration with default settings.
func DefaultConfig() *Config {
	return &Config{
		Silent:                 false,
		AbortOnError:           true,
		DebugMode:              false,
		ParserMode:             "PREFER_PHP7",
		SkipPaths:              []string{"vendor/*", "*.git*", "*.svn*", "*.bak"},
		KeepPaths:              []string{},
		ObfuscatePhpExtensions: []string{"php", "php5", "phtml"},
		FollowSymlinks:         false,
		TargetDirectory:        "", // Set via CLI

		Obfuscation: ObfuscationConfig{
			Strings: StringsConfig{
				Enabled:   true,
				Technique: StringObfuscationTechniqueBase64,
			},
			Scrambling: ScramblingConfig{
				Mode:   "identifier",
				Length: 5,
			},
			Comments: CommentsConfig{
				Strip: false,
			},
			Variables: NameToggleConfig{Scramble: false},
			Functions: NameToggleConfig{Scramble: false},
			Classes:   NameToggleConfig{Scramble: false},
			ControlFlow: ControlFlowConfig{
				Enabled:          true,
				MaxNestingDepth:  2,
				RandomConditions: true,
				AddDeadBranches:  false,
			},
			AdvancedLoops: AdvancedLoopsConfig{
				Enabled: false,
			},
			ArrayAccess: ArrayAccessConfig{
				Enabled: true,
			},
			Arithmetic: ArithmeticConfig{
				Enabled:            false,
				ComplexityLevel:    1,
				TransformationRate: 50,
			},
			DeadCode: CodeInjectionConfig{
				Enabled:       false,
				InjectionRate: 30,
			},
			JunkCode: JunkCodeConfig{
				Enabled:           false,
				InjectionRate:     30,
				MaxInjectionDepth: 3,
			},
			StatementShuffling: StatementShufflingConfig{
				Enabled:      true,
				MinChunkSize: 2,
				ChunkMode:    "fixed", // Use string literal instead of constant
				ChunkRatio:   20,
			},
			Ignore: IgnoreConfig{
				Functions: []string{},
				Variables: []string{},
				Classes:   []string{},
			},
		},
	}
}

// Helper to explicitly bind environment variables, handling potential key mismatches
func bindEnv(v *viper.Viper, key string) {
	envKey := strings.ToUpper(strings.ReplaceAll(key, "-", "_"))
	_ = v.BindEnv(key, "GOPHO_"+envKey)
}

// ObfuscateControlFlow enables control flow obfuscation which wraps code blocks
// inside redundant conditional blocks that always evaluate to true (if(1){...}).
// When enabled, it transforms:
// - Function bodies
// - Class method bodies
// - If/elseif/else statements
// - Loop statements (for, while, foreach, do-while)
//
// With UseAdvancedLoopObfuscation enabled, loop structures are transformed using more
// complex patterns that make it harder to follow the execution flow.
//
// Example:
// Original:
// ```php
// function foo() {
//     echo "Hello";
//     return "World";
// }
// ```
//
// Obfuscated:
// ```php
// function foo() {
//     if(1){
//         echo "Hello";
//         return "World";
//     }
// }
// ```
//
// This transformation preserves functionality while making static analysis
// and reverse engineering more difficult. It's particularly effective when
// combined with other obfuscation techniques.

// mapNestedToFlatConfig maps values from the new nested configuration structure
// to the flat structure used by the current implementation.
func mapNestedToFlatConfig(v *viper.Viper) {
	// Check if we have a nested structure
	if v.IsSet("obfuscation") {
		// Map string obfuscation
		if v.IsSet("obfuscation.strings.enabled") {
			v.Set("obfuscate_string_literal", v.GetBool("obfuscation.strings.enabled"))
		}
		if v.IsSet("obfuscation.strings.technique") {
			v.Set("string_obfuscation_technique", v.GetString("obfuscation.strings.technique"))
		}

		// Map comment stripping
		if v.IsSet("obfuscation.comments.strip") {
			v.Set("strip_comments", v.GetBool("obfuscation.comments.strip"))
		}

		// Map variable scrambling
		if v.IsSet("obfuscation.variables.scramble") {
			v.Set("obfuscate_variable_name", v.GetBool("obfuscation.variables.scramble"))
		}

		// Map function scrambling
		if v.IsSet("obfuscation.functions.scramble") {
			v.Set("obfuscate_function_name", v.GetBool("obfuscation.functions.scramble"))
		}

		// Map class scrambling
		if v.IsSet("obfuscation.classes.scramble") {
			v.Set("obfuscate_class_name", v.GetBool("obfuscation.classes.scramble"))
		}

		// Map control flow obfuscation
		if v.IsSet("obfuscation.control_flow.enabled") {
			v.Set("obfuscate_control_flow", v.GetBool("obfuscation.control_flow.enabled"))
		}
		if v.IsSet("obfuscation.control_flow.max_nesting_depth") {
			v.Set("control_flow_max_nesting_depth", v.GetInt("obfuscation.control_flow.max_nesting_depth"))
		}
		if v.IsSet("obfuscation.control_flow.random_conditions") {
			v.Set("control_flow_random_conditions", v.GetBool("obfuscation.control_flow.random_conditions"))
		}
		if v.IsSet("obfuscation.control_flow.add_dead_branches") {
			v.Set("control_flow_add_dead_branches", v.GetBool("obfuscation.control_flow.add_dead_branches"))
		}
		if v.IsSet("obfuscation.control_flow.advanced_loop_obfuscation") {
			v.Set("use_advanced_loop_obfuscation", v.GetBool("obfuscation.control_flow.advanced_loop_obfuscation"))
		}

		// Map array access obfuscation
		if v.IsSet("obfuscation.array_access.enabled") {
			v.Set("obfuscate_array_access", v.GetBool("obfuscation.array_access.enabled"))
		}

		// Map arithmetic expression obfuscation
		if v.IsSet("obfuscation.arithmetic.enabled") {
			v.Set("obfuscate_arithmetic_expressions", v.GetBool("obfuscation.arithmetic.enabled"))
		}
		if v.IsSet("obfuscation.arithmetic.complexity_level") {
			v.Set("arithmetic_complexity_level", v.GetInt("obfuscation.arithmetic.complexity_level"))
		}
		if v.IsSet("obfuscation.arithmetic.transformation_rate") {
			v.Set("arithmetic_transformation_rate", v.GetInt("obfuscation.arithmetic.transformation_rate"))
		}

		// Map dead code injection
		if v.IsSet("obfuscation.dead_code.enabled") {
			v.Set("inject_dead_code", v.GetBool("obfuscation.dead_code.enabled"))
		}
		if v.IsSet("obfuscation.dead_code.injection_rate") {
			v.Set("dead_junk_injection_rate", v.GetInt("obfuscation.dead_code.injection_rate"))
		}

		// Map junk code insertion
		if v.IsSet("obfuscation.junk_code.enabled") {
			v.Set("inject_junk_code", v.GetBool("obfuscation.junk_code.enabled"))
		}
		if v.IsSet("obfuscation.junk_code.injection_rate") {
			v.Set("dead_junk_injection_rate", v.GetInt("obfuscation.junk_code.injection_rate"))
		}
		if v.IsSet("obfuscation.junk_code.max_injection_depth") {
			v.Set("max_injection_depth", v.GetInt("obfuscation.junk_code.max_injection_depth"))
		}
	}
}
