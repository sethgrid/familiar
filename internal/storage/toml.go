package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/qwert/promptfamiliar/internal/pet"
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
	if err := toml.Unmarshal(configData, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
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

func InitPet(global bool, name string, baseDir string) error {
	petDir := filepath.Join(baseDir, ".familiar")
	if err := os.MkdirAll(petDir, 0755); err != nil {
		return fmt.Errorf("failed to create pet directory: %w", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")

	now := time.Now()
	config := pet.PetConfig{
		Version:                 "1.0",
		Name:                    name,
		EvolutionMode:           pet.EvolutionModeByAge,
		Evolution:               0,
		MaxEvolution:            DefaultMaxEvolution,
		CreatedAt:               now,
		DecayEnabled:            true,
		DecayRate:               DefaultDecayRate,
		HungerDecayPerHour:      DefaultHungerDecayPerHour,
		HappinessDecayPerHour:   DefaultHappinessDecayPerHour,
		EnergyDecayPerHour:      DefaultEnergyDecayPerHour,
		StoneThreshold:          DefaultStoneThreshold,
		InfirmEnabled:           true,
		InfirmDecayMultiplier:   DefaultInfirmDecayMultiplier,
		StoneDecayMultiplier:    DefaultStoneDecayMultiplier,
		EventChance:             DefaultEventChance,
		HealthComputation:       pet.HealthComputationAverage,
		InteractionThreshold:   DefaultInteractionThreshold,
		CacheTTL:                DefaultCacheTTL,
		AllowAnsiAnimations:     false,
		Animations:              make(map[string]pet.AnimationConfig),
	}

	// Add default ASCII cat animations
	config.Animations["default"] = pet.AnimationConfig{
		Source: "inline",
		FPS:    1,
		Loops:  1,
		Frames: []pet.Frame{
			{Art: ` /\_/\ 
( o.o )
 > ^ <`},
		},
	}

	config.Animations["infirm"] = pet.AnimationConfig{
		Source: "inline",
		FPS:    1,
		Loops:  1,
		Frames: []pet.Frame{
			{Art: ` /\_/\ 
( x.x )
 > ^ <`},
		},
	}

	config.Animations["stone"] = pet.AnimationConfig{
		Source: "inline",
		FPS:    1,
		Loops:  1,
		Frames: []pet.Frame{
			{Art: ` /\_/\ 
( +.+ )
 > ^ <`},
		},
	}

	config.Animations["egg"] = pet.AnimationConfig{
		Source: "inline",
		FPS:    1,
		Loops:  1,
		Frames: []pet.Frame{
			{Art: `  ___  
 /  . . \ 
 \___/`},
		},
	}

	state := pet.PetState{
		ConfigRef:    configPath,
		NameOverride: name,
		Hunger:       75,
		Happiness:    80,
		Energy:       60,
		Evolution:    0,
		IsInfirm:     false,
		IsStone:      false,
		Message:      "",
		LastFed:      now,
		LastPlayed:   now,
		LastVisited:  now,
		LastChecked:  now,
		LastVisits:   []pet.Interaction{},
		LastFeeds:    []pet.Interaction{},
		LastPlays:    []pet.Interaction{},
	}

	// Write config
	configData, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	// Write state
	stateData, err := toml.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	if err := os.WriteFile(statePath, stateData, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}
