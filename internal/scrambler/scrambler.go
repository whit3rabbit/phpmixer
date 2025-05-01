// Package scrambler handles name scrambling logic and context persistence.
package scrambler

import (
	"bytes" // For gob encoding/decoding
	"crypto/rand"
	"encoding/gob" // For saving/loading state
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/whit3rabbit/phpmixer/internal/config" // Adjust import path
)

const (
	// Characters for different scramble modes
	firstCharsIdentifier = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	allCharsIdentifier   = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ_"
	firstCharsHex        = "abcdefABCDEF"
	allCharsHex          = "0123456789abcdefABCDEF"
	firstCharsNumeric    = "O"
	allCharsNumeric      = "0123456789"

	// Limits
	maxIdentifierLen = 16
	maxHexNumericLen = 32
	minScrambleLen   = 2
	maxRegenAttempts = 50

	// Context serialization version
	contextVersion = "gopho-scramble-v1.0"
)

// scramblerState holds the data that needs to be persisted.
// Use exported fields for gob encoding.
type scramblerState struct {
	Version      string
	ScrambleMap  map[string]string // originalLower -> scrambledLower
	RScrambleMap map[string]string // scrambledLower -> originalLower
	LabelCounter *big.Int          // Use pointer for gob
	CurrentLen   int
}

// Scrambler manages the renaming map for a specific type of identifier.
type Scrambler struct {
	sType         ScrambleType
	cfg           *config.Config
	caseSensitive bool
	mode          string
	targetLength  int
	minLength     int
	maxLength     int
	currentLength int             // Current generation length
	ignoreMap     map[string]bool // nameLower -> true
	ignorePrefix  []string        // List of prefixes (lowercase)

	// State to be persisted (protected by mutex)
	scrambleMap  map[string]string // originalLower -> scrambledLower
	rScrambleMap map[string]string // scrambledLower -> originalLower
	labelCounter *big.Int

	mu sync.RWMutex // Protect maps and counter
}

// NewScrambler creates and initializes a scrambler for a specific type.
func NewScrambler(sType ScrambleType, cfg *config.Config) (*Scrambler, error) {
	s := &Scrambler{
		sType:        sType,
		cfg:          cfg,
		scrambleMap:  make(map[string]string),
		rScrambleMap: make(map[string]string),
		ignoreMap:    make(map[string]bool),
		labelCounter: big.NewInt(0), // Initialize the counter
	}

	// Determine Case Sensitivity (same as before)
	switch sType {
	case TypeVariable, TypeProperty, TypeConstant, TypeClassConstant, TypeLabel:
		s.caseSensitive = true
	case TypeFunction, TypeClass, TypeInterface, TypeTrait, TypeNamespace, TypeMethod:
		s.caseSensitive = false
	default:
		return nil, fmt.Errorf("unknown scramble type: %s", sType)
	}

	// Load Scramble Settings (same as before)
	s.mode = strings.ToLower(cfg.ScrambleMode)
	if s.mode == "" {
		s.mode = "identifier"
	}
	s.minLength = minScrambleLen
	s.maxLength = maxIdentifierLen
	switch s.mode {
	case "identifier":
		// default max length ok
	case "hexa", "numeric":
		s.maxLength = maxHexNumericLen
	default:
		fmt.Fprintf(os.Stderr, "Warning: Invalid scramble_mode '%s', using 'identifier'.\n", cfg.ScrambleMode)
		s.mode = "identifier"
	}
	s.targetLength = cfg.ScrambleLength
	if s.targetLength < s.minLength {
		s.targetLength = s.minLength
	}
	if s.targetLength > s.maxLength {
		s.targetLength = s.maxLength
	}
	s.currentLength = s.targetLength // Initialize current length

	// Load Ignore Lists from Config (same as before)
	var ignoreList []string
	var prefixList []string
	switch sType {
	case TypeVariable:
		ignoreList = cfg.IgnoreVariables
		prefixList = cfg.IgnoreVariablesPrefix
	case TypeFunction:
		ignoreList = append(ignoreList, cfg.IgnoreFunctions...)
		ignoreList = append(ignoreList, cfg.IgnoreClasses...)
		ignoreList = append(ignoreList, cfg.IgnoreInterfaces...)
		ignoreList = append(ignoreList, cfg.IgnoreTraits...)
		ignoreList = append(ignoreList, cfg.IgnoreNamespaces...)
		prefixList = append(prefixList, cfg.IgnoreFunctionsPrefix...)
		prefixList = append(prefixList, cfg.IgnoreClassesPrefix...)
		prefixList = append(prefixList, cfg.IgnoreInterfacesPrefix...)
		prefixList = append(prefixList, cfg.IgnoreTraitsPrefix...)
		prefixList = append(prefixList, cfg.IgnoreNamespacesPrefix...)
	case TypeProperty:
		ignoreList = cfg.IgnoreProperties
		prefixList = cfg.IgnorePropertiesPrefix
	case TypeMethod:
		ignoreList = cfg.IgnoreMethods
		prefixList = cfg.IgnoreMethodsPrefix
	case TypeClassConstant:
		ignoreList = cfg.IgnoreClassConstants
		prefixList = cfg.IgnoreClassConstantsPrefix
	case TypeConstant:
		ignoreList = cfg.IgnoreConstants
		prefixList = cfg.IgnoreConstantsPrefix
	case TypeLabel:
		ignoreList = cfg.IgnoreLabels
		prefixList = cfg.IgnoreLabelsPrefix
	}
	for _, item := range ignoreList {
		s.ignoreMap[strings.ToLower(item)] = true
	}
	for _, prefix := range prefixList {
		s.ignorePrefix = append(s.ignorePrefix, strings.ToLower(prefix))
	}

	// TODO: Load pre-defined PHP things to ignore

	return s, nil
}

// ShouldIgnore checks if a name should be ignored based on reserved words,
// specific ignore lists, and prefix lists.
func (s *Scrambler) ShouldIgnore(name string) bool {
	if isReserved(name, s.sType) {
		return true
	}
	if s.ignoreMap[strings.ToLower(name)] {
		return true
	}
	lowerName := strings.ToLower(name)
	for _, prefix := range s.ignorePrefix {
		if strings.HasPrefix(lowerName, prefix) {
			return true
		}
	}
	return false
}

// Scramble (same as before)
func (s *Scrambler) Scramble(originalName string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ShouldIgnore(originalName) {
		return originalName
	}
	lookupKey := originalName
	if !s.caseSensitive {
		lookupKey = strings.ToLower(originalName)
	}
	if scrambled, exists := s.scrambleMap[lookupKey]; exists {
		return scrambled
	}

	var newScrambled string
	var newScrambledLower string
	for attempt := 0; attempt < maxRegenAttempts; attempt++ {
		newScrambled = s.generateScrambledName() // Uses s.currentLength internally
		newScrambledLower = strings.ToLower(newScrambled)

		reservedCheckName := newScrambled
		if isReserved(reservedCheckName, s.sType) || s.ignoreMap[newScrambledLower] {
			continue
		}
		if _, exists := s.rScrambleMap[newScrambledLower]; exists {
			if attempt > 5 && s.currentLength < s.maxLength {
				s.currentLength++ // Increase generation length
			}
			continue
		}
		// Apply deterministic shuffling if needed
		finalScrambled := s.applyCaseShuffling(newScrambled)
		finalScrambledLower := strings.ToLower(finalScrambled)

		// Re-check for collisions with the final shuffled lowercase version
		if _, exists := s.rScrambleMap[finalScrambledLower]; exists {
			if attempt > 5 && s.currentLength < s.maxLength {
				s.currentLength++ // Increase generation length
			}
			continue // Collision still exists after shuffling, try generating again
		}

		// Store the final deterministically shuffled version
		s.scrambleMap[lookupKey] = finalScrambled
		s.rScrambleMap[finalScrambledLower] = lookupKey // Use lowercase for reverse lookup key
		return finalScrambled
	}
	fmt.Fprintf(os.Stderr, "Error: Failed to generate unique scrambled name for '%s' (type: %s) after %d attempts.\n", originalName, s.sType, maxRegenAttempts)
	s.scrambleMap[lookupKey] = lookupKey // Store original as fallback
	s.rScrambleMap[lookupKey] = lookupKey
	return originalName
}

// Unscramble looks up the original name given a scrambled name.
// It handles the '$' prefix for variables.
func (s *Scrambler) Unscramble(scrambledName string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lookupKey := scrambledName
	// If the scrambler type is for variables, strip the leading '$' if present.
	if s.sType == TypeVariable && strings.HasPrefix(lookupKey, "$") {
		lookupKey = lookupKey[1:] // Remove the first character ($)
	}

	// Always use lowercase for lookup in the reverse map, as keys are stored lowercase.
	lookupKeyLower := strings.ToLower(lookupKey)

	original, found := s.rScrambleMap[lookupKeyLower]
	return original, found
}

// generateScrambledName (same as before)
func (s *Scrambler) generateScrambledName() string {
	var firstChars, allChars string
	length := s.currentLength // Use the potentially increased length
	switch s.mode {
	case "numeric":
		firstChars = firstCharsNumeric
		allChars = allCharsNumeric
	case "hexa":
		firstChars = firstCharsHex
		allChars = allCharsHex
	case "identifier":
		fallthrough
	default:
		firstChars = firstCharsIdentifier
		allChars = allCharsIdentifier
	}
	if length < s.minLength {
		length = s.minLength
	}
	if length > s.maxLength {
		length = s.maxLength
	}
	sb := strings.Builder{}
	sb.Grow(length)
	idx := randInt(len(firstChars))
	sb.WriteByte(firstChars[idx])
	for i := 1; i < length; i++ {
		idx = randInt(len(allChars))
		sb.WriteByte(allChars[idx])
	}
	return sb.String()
}

// applyCaseShuffling applies deterministic case changes based on index if type is case-sensitive
func (s *Scrambler) applyCaseShuffling(name string) string {
	// Only apply shuffling if the type requires case sensitivity.
	if !s.caseSensitive {
		// For non-case-sensitive types (functions etc.), always use lowercase.
		return strings.ToLower(name)
	}

	// For case-sensitive types, apply deterministic shuffling based on the input name.
	shuffled := make([]byte, len(name))
	nameBytes := []byte(name) // Operate on input bytes

	// Use properties of the name itself for deterministic shuffling
	// Example: Uppercase letters at even positions, lowercase at odd
	for i, charByte := range nameBytes {
		if charByte >= 'a' && charByte <= 'z' {
			if i%2 == 0 {
				shuffled[i] = charByte - ('a' - 'A') // To uppercase
			} else {
				shuffled[i] = charByte // Keep lowercase
			}
		} else if charByte >= 'A' && charByte <= 'Z' {
			if i%2 == 0 {
				shuffled[i] = charByte // Keep uppercase
			} else {
				shuffled[i] = charByte + ('a' - 'A') // To lowercase
			}
		} else {
			// Keep non-letters as they are
			shuffled[i] = charByte
		}
	}
	return string(shuffled)
}

// GenerateLabelName (same as before)
func (s *Scrambler) GenerateLabelName(prefix string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	counterVal := s.labelCounter.String()
	s.labelCounter.Add(s.labelCounter, big.NewInt(1))
	generated := fmt.Sprintf("%s_%s", prefix, counterVal)
	// Rerun through Scramble to check ignores/reserved and store mapping for the generated label
	// Note: This internal call to Scramble needs to handle the lock correctly.
	// Since GenerateLabelName already holds the lock, Scramble needs to be careful
	// or we need a non-locking internal version.
	// For simplicity now, assume Scramble handles recursive lock attempt (RWMutex allows this).
	return s.scrambleNoLock(generated) // Use a non-locking version internally
}

// scrambleNoLock is the internal implementation without mutex locking.
// NOTE: This needs the same logic changes as Scramble for consistency.
func (s *Scrambler) scrambleNoLock(originalName string) string {
	if s.ShouldIgnore(originalName) {
		return originalName
	}
	lookupKey := originalName
	if !s.caseSensitive {
		lookupKey = strings.ToLower(originalName)
	}
	if scrambled, exists := s.scrambleMap[lookupKey]; exists {
		// Return the already deterministically shuffled version stored in the map
		return scrambled
	}

	var newScrambled string
	for attempt := 0; attempt < maxRegenAttempts; attempt++ {
		newScrambled = s.generateScrambledName() // Uses s.currentLength internally

		reservedCheckName := newScrambled
		if isReserved(reservedCheckName, s.sType) || s.ignoreMap[strings.ToLower(newScrambled)] {
			continue
		}

		// Apply deterministic shuffling if needed
		finalScrambled := s.applyCaseShuffling(newScrambled)
		finalScrambledLower := strings.ToLower(finalScrambled)

		// Re-check for collisions with the final shuffled lowercase version
		if _, exists := s.rScrambleMap[finalScrambledLower]; exists {
			if attempt > 5 && s.currentLength < s.maxLength {
				s.currentLength++ // Increase generation length
			}
			continue // Collision still exists after shuffling, try generating again
		}

		// Store the final deterministically shuffled version
		s.scrambleMap[lookupKey] = finalScrambled
		s.rScrambleMap[finalScrambledLower] = lookupKey // Use lowercase for reverse lookup key
		return finalScrambled
	}
	fmt.Fprintf(os.Stderr, "Error: Failed to generate unique scrambled name for '%s' (type: %s) after %d attempts (no lock).\n", originalName, s.sType, maxRegenAttempts)
	// Fallback: store original name
	finalOriginal := s.applyCaseShuffling(originalName) // Apply shuffling even to fallback
	s.scrambleMap[lookupKey] = finalOriginal
	s.rScrambleMap[strings.ToLower(finalOriginal)] = lookupKey
	return finalOriginal
}

// --- Context Persistence ---

// SaveState saves the scrambler's current mapping state to a file.
func (s *Scrambler) SaveState(filePath string) error {
	s.mu.RLock() // Read lock to access maps and counter
	state := scramblerState{
		Version:      contextVersion,
		ScrambleMap:  s.scrambleMap,
		RScrambleMap: s.rScrambleMap,
		LabelCounter: s.labelCounter,
		CurrentLen:   s.currentLength,
	}
	s.mu.RUnlock()

	var buffer bytes.Buffer
	encoder := gob.NewEncoder(&buffer)
	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("failed to encode scrambler state: %w", err)
	}

	if err := os.WriteFile(filePath, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write scrambler state to file %s: %w", filePath, err)
	}
	return nil
}

// LoadState loads the scrambler's state from a file, replacing the current state.
func (s *Scrambler) LoadState(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If file doesn't exist, it's not an error, just means no previous state.
		if os.IsNotExist(err) {
			return nil // No state to load
		}
		return fmt.Errorf("failed to read scrambler state file %s: %w", filePath, err)
	}

	buffer := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buffer)
	var state scramblerState

	if err := decoder.Decode(&state); err != nil {
		return fmt.Errorf("failed to decode scrambler state from file %s: %w", filePath, err)
	}

	// Check version compatibility
	if state.Version != contextVersion {
		return fmt.Errorf("incompatible context version: file has '%s', expected '%s'", state.Version, contextVersion)
	}

	// --- Update internal state (with lock) ---
	s.mu.Lock()
	defer s.mu.Unlock()

	// Important: Replace maps, don't merge, to reflect the loaded state accurately.
	s.scrambleMap = state.ScrambleMap
	s.rScrambleMap = state.RScrambleMap
	s.labelCounter = state.LabelCounter
	s.currentLength = state.CurrentLen

	// Ensure maps are not nil if the file contained empty ones
	if s.scrambleMap == nil {
		s.scrambleMap = make(map[string]string)
	}
	if s.rScrambleMap == nil {
		s.rScrambleMap = make(map[string]string)
	}
	if s.labelCounter == nil {
		s.labelCounter = big.NewInt(0)
	}

	return nil
}

// --- Utility Functions ---

// randInt (same as before)
func randInt(max int) int {
	if max <= 0 {
		return 0
	}
	nBig, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return int(nBig.Int64())
}

// randHex (same as before)
func randHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		panic(fmt.Sprintf("crypto/rand failed for hex: %v", err))
	}
	return hex.EncodeToString(bytes)
}

// All known scramble types
var AllScrambleTypes = []ScrambleType{
	TypeVariable,
	TypeFunction,
	TypeClass,
	TypeInterface,
	TypeTrait,
	TypeProperty,
	TypeMethod,
	TypeConstant,
	TypeClassConstant,
	TypeLabel,
	TypeNamespace,
}

// ParseScrambleType converts a string identifier to its corresponding ScrambleType constant.
// Returns an error if the type string is invalid.
func ParseScrambleType(typeStr string) (ScrambleType, error) {
	lowerType := strings.ToLower(strings.TrimSpace(typeStr))
	for _, sType := range AllScrambleTypes {
		if string(sType) == lowerType {
			return sType, nil
		}
	}
	return "", fmt.Errorf("invalid scramble type specified: '%s'", typeStr)
}

// --- Reserved Keywords/Names ---

// LookupObfuscated attempts to find the obfuscated name for the given original name.
// Returns the obfuscated name and a boolean indicating if the name was found.
func (s *Scrambler) LookupObfuscated(original string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try to find the name in the mapping
	obfuscated, found := s.scrambleMap[original]
	if !found {
		return "", false
	}

	return obfuscated, true
}

