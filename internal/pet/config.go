package pet

import (
	"time"
)

type EvolutionMode string

const (
	EvolutionModeHardCoded EvolutionMode = "hard-coded"
	EvolutionModeByAge     EvolutionMode = "by-age"
)

type HealthComputationMode string

const (
	HealthComputationAverage  HealthComputationMode = "average"
	HealthComputationWeighted HealthComputationMode = "weighted"
)

type PetConfig struct {
	Version               string                `toml:"version"`
	Name                  string                `toml:"name"`
	PetType               string                `toml:"petType,omitempty"` // Type of pet (cat, dancer, pixel, etc.)
	EvolutionMode         EvolutionMode         `toml:"evolutionMode"`
	Evolution             int                   `toml:"evolution"`
	MaxEvolution          int                   `toml:"maxEvolution"`
	CreatedAt             time.Time             `toml:"createdAt"`
	DecayEnabled          bool                  `toml:"decayEnabled"`
	DecayRate             float64               `toml:"decayRate"`
	HungerDecayPerHour    float64               `toml:"hungerDecayPerHour"`
	HappinessDecayPerHour float64               `toml:"happinessDecayPerHour"`
	EnergyDecayPerHour    float64               `toml:"energyDecayPerHour"`
	StoneThreshold        int                   `toml:"stoneThreshold"`
	InfirmEnabled         bool                  `toml:"infirmEnabled"`
	InfirmDecayMultiplier float64               `toml:"infirmDecayMultiplier"`
	StoneDecayMultiplier  float64               `toml:"stoneDecayMultiplier"`
	SleepDuration         time.Duration         `toml:"sleepDuration"`
	EventChance           float64               `toml:"eventChance"`
	HealthComputation     HealthComputationMode `toml:"healthComputation"`
	InteractionThreshold  int                   `toml:"interactionThreshold"`

	CacheTTL            time.Duration `toml:"cacheTTL"`
	AllowAnsiAnimations bool          `toml:"allowAnsiAnimations"`

	Animations map[string]AnimationConfig `toml:"animations"`
}

type AnimationConfig struct {
	Source string  `toml:"source"` // "inline" | "pixel" | "url" | "file"
	URL    string  `toml:"url,omitempty"`
	Path   string  `toml:"path,omitempty"`
	FPS    int     `toml:"fps"`
	Loops  int     `toml:"loops"`  // 0 or -1 = infinite
	Frames []Frame `toml:"frames"` // for source == "inline" or "pixel"
}

type Frame struct {
	Art    string     `toml:"art,omitempty"`    // For inline ASCII art
	Pixels [][]string `toml:"pixels,omitempty"` // For pixel art: 2D array of color hex codes
	MS     int        `toml:"ms,omitempty"`
}
