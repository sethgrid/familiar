package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/sethgrid/familiar/internal/pet"
)

const (
	DefaultMaxEvolution       = 5
	DefaultDecayRate          = 1.0
	DefaultHungerDecayPerHour = 2.0
	DefaultHappinessDecayPerHour = 1.5
	DefaultEnergyDecayPerHour = 1.0
	DefaultStoneThreshold    = 10
	DefaultInfirmDecayMultiplier = 1.5
	DefaultStoneDecayMultiplier  = 0.1
	DefaultSleepDuration     = 30 * time.Minute
	DefaultEventChance        = 0.01
	DefaultInteractionThreshold = 3
	DefaultCacheTTL           = 24 * time.Hour
)

func LoadPet(configPath, statePath string) (*pet.Pet, error) {
	// Load state first
	stateData, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state pet.PetState
	if err := toml.Unmarshal(stateData, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Load config
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config pet.PetConfig
	
	// Try to unmarshal directly first
	if err := toml.Unmarshal(configData, &config); err != nil {
		// If it fails, try parsing as map to handle duration strings
		var configMap map[string]interface{}
		if err2 := toml.Unmarshal(configData, &configMap); err2 != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}

		// Parse sleepDuration if it's a string
		if sleepDur, ok := configMap["sleepDuration"]; ok {
			if sleepDurStr, ok := sleepDur.(string); ok {
				parsed, err := time.ParseDuration(sleepDurStr)
				if err == nil {
					configMap["sleepDuration"] = int64(parsed)
				}
			}
		}

		// Re-marshal and unmarshal to get proper struct
		configBytes, err := toml.Marshal(configMap)
		if err != nil {
			return nil, fmt.Errorf("failed to re-marshal config: %w", err)
		}

		if err := toml.Unmarshal(configBytes, &config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	} else {
		// Successfully unmarshaled, but check if sleepDuration needs parsing
		// (This handles the case where it was stored as a string in an existing file)
		var configMap map[string]interface{}
		if err := toml.Unmarshal(configData, &configMap); err == nil {
			if sleepDur, ok := configMap["sleepDuration"]; ok {
				if sleepDurStr, ok := sleepDur.(string); ok {
					parsed, err := time.ParseDuration(sleepDurStr)
					if err == nil {
						config.SleepDuration = parsed
					}
				}
			}
		}
	}

	return &pet.Pet{
		Config: config,
		State:  state,
	}, nil
}

func SavePetState(p *pet.Pet, statePath string) error {
	data, err := toml.Marshal(p.State)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(statePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// FindLibDir attempts to find the lib directory relative to the executable or source
func FindLibDir() (string, error) {
	return findLibDir()
}

// findLibDir attempts to find the lib directory relative to the executable or source
func findLibDir() (string, error) {
	// Try to find lib relative to executable
	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		// Check if lib exists relative to executable
		libPath := filepath.Join(execDir, "lib", "v1")
		if _, err := os.Stat(libPath); err == nil {
			return libPath, nil
		}
		// Try going up one level (if executable is in bin/)
		parentLibPath := filepath.Join(filepath.Dir(execDir), "lib", "v1")
		if _, err := os.Stat(parentLibPath); err == nil {
			return parentLibPath, nil
		}
	}

	// Try relative to current working directory
	cwd, err := os.Getwd()
	if err == nil {
		libPath := filepath.Join(cwd, "lib", "v1")
		if _, err := os.Stat(libPath); err == nil {
			return libPath, nil
		}
		// Try going up directories to find lib
		dir := cwd
		for i := 0; i < 10; i++ {
			libPath := filepath.Join(dir, "lib", "v1")
			if _, err := os.Stat(libPath); err == nil {
				return libPath, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", fmt.Errorf("could not find lib/v1 directory")
}

func InitPet(global bool, petType string, name string, baseDir string) error {
	petDir := filepath.Join(baseDir, ".familiar")
	if err := os.MkdirAll(petDir, 0755); err != nil {
		return fmt.Errorf("failed to create pet directory: %w", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")

	// Find lib directory
	libDir, err := findLibDir()
	if err != nil {
		return fmt.Errorf("failed to find lib directory: %w", err)
	}

	// Read template files
	templateConfigPath := filepath.Join(libDir, petType+".toml")
	templateStatePath := filepath.Join(libDir, petType+".state.toml")

	configTemplate, err := os.ReadFile(templateConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read template config file %s: %w", templateConfigPath, err)
	}

	stateTemplate, err := os.ReadFile(templateStatePath)
	if err != nil {
		return fmt.Errorf("failed to read template state file %s: %w", templateStatePath, err)
	}

	now := time.Now()
	createdAtStr := now.Format(time.RFC3339Nano)

	// Replace placeholders in config template
	configContent := string(configTemplate)
	configContent = strings.ReplaceAll(configContent, "{{NAME}}", name)
	configContent = strings.ReplaceAll(configContent, "{{CREATED_AT}}", createdAtStr)
	configContent = strings.ReplaceAll(configContent, "{{PET_TYPE}}", petType)

	// Replace placeholders in state template
	stateContent := string(stateTemplate)
	stateContent = strings.ReplaceAll(stateContent, "{{NAME}}", name)
	stateContent = strings.ReplaceAll(stateContent, "{{CONFIG_REF}}", configPath)
	stateContent = strings.ReplaceAll(stateContent, "{{CREATED_AT}}", createdAtStr)

	// Write config
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Write state
	if err := os.WriteFile(statePath, []byte(stateContent), 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadTemplateConfig loads a pet config from a template file
// This is used for previewing animations without needing an installed pet
func LoadTemplateConfig(petType string) (*pet.Pet, error) {
	// Find lib directory
	libDir, err := findLibDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find lib directory: %w", err)
	}

	// Load template
	templatePath := filepath.Join(libDir, petType+".toml")
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", templatePath, err)
	}

	// Replace placeholders with valid dummy values for parsing
	templateContent := string(templateData)
	now := time.Now()
	dummyName := "TemplatePet"
	dummyCreatedAt := now.Format(time.RFC3339Nano)

	templateContent = strings.ReplaceAll(templateContent, "{{NAME}}", dummyName)
	templateContent = strings.ReplaceAll(templateContent, "{{CREATED_AT}}", dummyCreatedAt)
	templateContent = strings.ReplaceAll(templateContent, "{{PET_TYPE}}", petType)

	// Parse template - handle duration strings like LoadPet does
	var templateConfig pet.PetConfig

	// Try to unmarshal directly first
	if err := toml.Unmarshal([]byte(templateContent), &templateConfig); err != nil {
		// If it fails, try parsing as map to handle duration strings
		var configMap map[string]interface{}
		if err2 := toml.Unmarshal([]byte(templateContent), &configMap); err2 != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}

		// Parse sleepDuration if it's a string
		if sleepDur, ok := configMap["sleepDuration"]; ok {
			if sleepDurStr, ok := sleepDur.(string); ok {
				parsed, err := time.ParseDuration(sleepDurStr)
				if err == nil {
					configMap["sleepDuration"] = int64(parsed)
				}
			}
		}

		// Re-marshal and unmarshal to get proper struct
		configBytes, err := toml.Marshal(configMap)
		if err != nil {
			return nil, fmt.Errorf("failed to re-marshal template: %w", err)
		}

		if err := toml.Unmarshal(configBytes, &templateConfig); err != nil {
			return nil, fmt.Errorf("failed to parse template: %w", err)
		}
	} else {
		// Successfully unmarshaled, but check if sleepDuration needs parsing
		var configMap map[string]interface{}
		if err := toml.Unmarshal([]byte(templateContent), &configMap); err == nil {
			if sleepDur, ok := configMap["sleepDuration"]; ok {
				if sleepDurStr, ok := sleepDur.(string); ok {
					parsed, err := time.ParseDuration(sleepDurStr)
					if err == nil {
						templateConfig.SleepDuration = parsed
					}
				}
			}
		}
	}

	// Create a minimal pet with just the config (no state needed for art preview)
	return &pet.Pet{
		Config: templateConfig,
		State:  pet.PetState{}, // Empty state - we only need config for animations
	}, nil
}

// ReleasePet soft-deletes a pet by renaming files with .released.{timestamp}
func ReleasePet(petDir, configPath, statePath, petName string) error {
	now := time.Now()
	timestamp := strconv.FormatInt(now.Unix(), 10)

	// Sanitize pet name for filename (remove special chars)
	safeName := strings.ReplaceAll(petName, " ", "_")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")

	// Rename config file
	newConfigPath := filepath.Join(petDir, fmt.Sprintf("pet.%s.released.%s.toml", safeName, timestamp))
	if err := os.Rename(configPath, newConfigPath); err != nil {
		return fmt.Errorf("failed to rename config file: %w", err)
	}

	// Rename state file
	newStatePath := filepath.Join(petDir, fmt.Sprintf("pet.state.%s.released.%s.toml", safeName, timestamp))
	if err := os.Rename(statePath, newStatePath); err != nil {
		// Try to restore config if state rename fails
		os.Rename(newConfigPath, configPath)
		return fmt.Errorf("failed to rename state file: %w", err)
	}

	return nil
}

// BanishPet permanently deletes a pet
func BanishPet(petDir, configPath, statePath string) error {
	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete config file: %w", err)
	}
	if err := os.Remove(statePath); err != nil {
		return fmt.Errorf("failed to delete state file: %w", err)
	}
	return nil
}

// ReleasedPetInfo holds information about a released pet
type ReleasedPetInfo struct {
	Name      string
	Timestamp int64
	ConfigPath string
	StatePath  string
}

// FindMostRecentReleased finds the most recently released pet
func FindMostRecentReleased(petDir string) (string, error) {
	released, err := findAllReleased(petDir)
	if err != nil {
		return "", err
	}
	if len(released) == 0 {
		return "", fmt.Errorf("no released pets found")
	}
	// Return the path to the most recent one (they're sorted by timestamp desc)
	return released[0].StatePath, nil
}

// FindReleasedByName finds a released pet by name
func FindReleasedByName(petDir, name string) (string, error) {
	released, err := findAllReleased(petDir)
	if err != nil {
		return "", err
	}
	
	// Try exact match first
	for _, r := range released {
		if r.Name == name {
			return r.StatePath, nil
		}
	}
	
	// Try case-insensitive match
	lowerName := strings.ToLower(name)
	for _, r := range released {
		if strings.ToLower(r.Name) == lowerName {
			return r.StatePath, nil
		}
	}
	
	return "", fmt.Errorf("no released pet found with name '%s'", name)
}

// findAllReleased finds all released pets
func findAllReleased(petDir string) ([]ReleasedPetInfo, error) {
	entries, err := os.ReadDir(petDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read pet directory: %w", err)
	}

	var released []ReleasedPetInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		name := entry.Name()
		// Look for pattern: pet.state.{name}.released.{timestamp}.toml
		if strings.HasPrefix(name, "pet.state.") && strings.Contains(name, ".released.") && strings.HasSuffix(name, ".toml") {
			// Extract name and timestamp
			// pet.state.{name}.released.{timestamp}.toml
			parts := strings.Split(name, ".")
			if len(parts) >= 5 {
				// parts: ["pet", "state", "{name}", "released", "{timestamp}", "toml"]
				// Find where "released" is
				releasedIdx := -1
				for i, part := range parts {
					if part == "released" {
						releasedIdx = i
						break
					}
				}
				if releasedIdx > 2 && releasedIdx < len(parts)-2 {
					// Name is everything between "state" and "released"
					petName := strings.Join(parts[2:releasedIdx], ".")
					timestampStr := parts[releasedIdx+1]
					timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
					if err == nil {
						statePath := filepath.Join(petDir, name)
						
						// Find corresponding config file
						configName := strings.Replace(name, "pet.state.", "pet.", 1)
						configPath := filepath.Join(petDir, configName)
						
						released = append(released, ReleasedPetInfo{
							Name:       petName,
							Timestamp:  timestamp,
							ConfigPath: configPath,
							StatePath:  statePath,
						})
					}
				}
			}
		}
	}

	// Sort by timestamp descending (most recent first)
	sort.Slice(released, func(i, j int) bool {
		return released[i].Timestamp > released[j].Timestamp
	})

	return released, nil
}

// RestoreReleased restores a released pet by renaming files back
func RestoreReleased(petDir, releasedStatePath string) error {
	// Extract the released filename
	releasedStateFile := filepath.Base(releasedStatePath)
	
	// Find the corresponding config file
	// pet.state.{name}.released.{timestamp}.toml -> pet.{name}.released.{timestamp}.toml
	releasedConfigFile := strings.Replace(releasedStateFile, "pet.state.", "pet.", 1)
	releasedConfigPath := filepath.Join(petDir, releasedConfigFile)
	
	// Check if config file exists
	if _, err := os.Stat(releasedConfigPath); err != nil {
		return fmt.Errorf("released config file not found: %s", releasedConfigPath)
	}
	
	// Restore to pet.toml and pet.state.toml
	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	
	// Check if pet already exists
	if _, err := os.Stat(statePath); err == nil {
		return fmt.Errorf("a familiar already exists. Use 'release' first")
	}
	
	// Rename files back
	if err := os.Rename(releasedConfigPath, configPath); err != nil {
		return fmt.Errorf("failed to restore config file: %w", err)
	}
	if err := os.Rename(releasedStatePath, statePath); err != nil {
		// Try to restore config if state restore fails
		os.Rename(configPath, releasedConfigPath)
		return fmt.Errorf("failed to restore state file: %w", err)
	}
	
	return nil
}
