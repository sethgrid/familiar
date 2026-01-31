package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/sethgrid/familiar/internal/art"
	"github.com/sethgrid/familiar/internal/conditions"
	"github.com/sethgrid/familiar/internal/discovery"
	"github.com/sethgrid/familiar/internal/health"
	"github.com/sethgrid/familiar/internal/pet"
	"github.com/sethgrid/familiar/internal/storage"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

const Version = "v0.4.0"

var familiarNames = []string{
	"Pip",
	"Shadow",
	"Whisper",
	"Ember",
	"Spark",
	"Echo",
}

func randomFamiliarName() string {
	return familiarNames[rand.Intn(len(familiarNames))]
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "familiar",
		Short: "PromptFamiliar - A terminal pet that lives in your prompt",
		Run: func(cmd *cobra.Command, args []string) {
			// If version flag is set, print version and exit
			if version, _ := cmd.Flags().GetBool("version"); version {
				fmt.Println(Version)
				return
			}
			// Otherwise show help
			cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to pet config file")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	rootCmd.AddCommand(summonCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(playCmd)
	rootCmd.AddCommand(restCmd)
	rootCmd.AddCommand(healCmd)
	rootCmd.AddCommand(adminCmd)
	rootCmd.AddCommand(messageCmd)
	rootCmd.AddCommand(acknowledgeCmd)
	rootCmd.AddCommand(awakenCmd)
	rootCmd.AddCommand(ossifyCmd)
	rootCmd.AddCommand(dismissCmd)
	rootCmd.AddCommand(banishCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var summonCmd = &cobra.Command{
	Use:   "summon [type] [name]",
	Short: "Summon a familiar (create new or restore dismissed)",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		global, _ := cmd.Flags().GetBool("global")

		var baseDir string
		if global {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}
			baseDir = home
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			baseDir = cwd
		}

		petDir := filepath.Join(baseDir, ".familiar")

		// Check if pet already exists
		statePath := filepath.Join(petDir, "pet.state.toml")
		if _, err := os.Stat(statePath); err == nil {
			return fmt.Errorf("a familiar already exists. Use 'dismiss' to soft-delete it first")
		}

		var petType string
		var name string

		if len(args) == 0 {
			// No args - try to restore most recent released pet
			releasedPath, err := storage.FindMostRecentReleased(petDir)
			if err != nil {
				// No dismissed pets found - create a new one with a random name
				rand.Seed(time.Now().UnixNano())
				petType = "cat"
				name = randomFamiliarName()
			} else {
				// Found a dismissed pet, restore it
				if err := storage.RestoreReleased(petDir, releasedPath); err != nil {
					return fmt.Errorf("failed to restore familiar: %w", err)
				}
				// Load to get the name
				configPath := filepath.Join(petDir, "pet.toml")
				p, err := storage.LoadPet(configPath, statePath)
				if err != nil {
					return fmt.Errorf("failed to load restored familiar: %w", err)
				}
				petName := p.Config.Name
				if p.State.NameOverride != "" {
					petName = p.State.NameOverride
				}
				fmt.Printf("Familiar '%s' restored!\n", petName)
				return nil
			}
		} else if len(args) == 1 {
			// One arg - could be name to restore or name for new pet
			// Try to find released pet with this name first
			releasedPath, err := storage.FindReleasedByName(petDir, args[0])
			if err == nil {
				// Found released pet with this name, restore it
				if err := storage.RestoreReleased(petDir, releasedPath); err != nil {
					return fmt.Errorf("failed to restore familiar: %w", err)
				}
				fmt.Printf("Familiar '%s' restored!\n", args[0])
				return nil
			}
			// Not found, treat as new pet name
			petType = "cat"
			name = args[0]
		} else {
			// Two args - type and name
			petType = args[0]
			name = args[1]
		}

		if err := storage.InitPet(global, petType, name, baseDir); err != nil {
			return fmt.Errorf("failed to summon familiar: %w", err)
		}

		fmt.Printf("Familiar '%s' summoned!\n", name)
		return nil
	},
}

func init() {
	summonCmd.Flags().Bool("global", false, "Create global familiar")
}

func loadPet() (*pet.Pet, string, string, error) {
	var statePath string
	var found bool
	var err error

	if configPath != "" {
		// Use provided config path (treating it as state path for now)
		statePath = configPath
		if _, err := os.Stat(statePath); err != nil {
			return nil, "", "", fmt.Errorf("state file not found: %s", statePath)
		}
		found = true
	} else {
		// Discover pet
		cwd, _ := os.Getwd()
		statePath, found, err = discovery.FindStateFile(cwd)
		if err != nil {
			return nil, "", "", err
		}
	}

	if !found {
		// Try global
		statePath = discovery.GlobalPetStatePath()
		if _, err := os.Stat(statePath); err != nil {
			return nil, "", "", fmt.Errorf("no familiar found. Run 'familiar init' to create one")
		}
	}

	petConfigPath := discovery.GetConfigPathFromState(statePath)
	p, err := storage.LoadPet(petConfigPath, statePath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to load familiar: %w", err)
	}

	return p, petConfigPath, statePath, nil
}

func executeStatefulCommand(fn func(*pet.Pet) error) error {
	p, _, statePath, err := loadPet()
	if err != nil {
		return err
	}

	// Apply decay
	now := time.Now()
	if err := pet.ApplyTimeStep(p, now); err != nil {
		return fmt.Errorf("failed to apply time step: %w", err)
	}

	// Execute command
	if err := fn(p); err != nil {
		return err
	}

	// Save state
	if err := storage.SavePetState(p, statePath); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show familiar status",
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		// Status command applies boost BEFORE decay, so the boost isn't reduced by decay
		p, _, statePath, err := loadPet()
		if err != nil {
			return err
		}

		// Apply boost first (before decay)
		p.State.Hunger = max(0, p.State.Hunger-5) // Decrease hunger (lower is better)
		p.State.Happiness = min(100, p.State.Happiness+5)
		p.State.Energy = min(100, p.State.Energy+5)

		// Now apply decay
		now := time.Now()
		if err := pet.ApplyTimeStep(p, now); err != nil {
			return fmt.Errorf("failed to apply time step: %w", err)
		}

		health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
		status := conditions.DeriveStatus(p, now, health)

		name := p.Config.Name
		if p.State.NameOverride != "" {
			name = p.State.NameOverride
		}

		if verbose {
			// Verbose mode: stats card
			primaryCondition := status.Primary
			fmt.Printf("%s is %s\n\n", name, primaryCondition)
			fmt.Printf("state: %s\n", conditions.FormatConditions(status.AllOrdered))
			fmt.Printf("health: %d\n", status.Health)
			fmt.Printf("hunger: %d\n", p.State.Hunger)
			fmt.Printf("happiness: %d\n", p.State.Happiness)
			fmt.Printf("energy: %d\n", p.State.Energy)
			fmt.Printf("evolution: %d\n\n", p.State.Evolution)
		} else {
			// Default concise mode
			primaryCondition := status.Primary
			if len(status.AllOrdered) > 0 {
				fmt.Printf("%s is %s\n\n", name, primaryCondition)
			} else {
				fmt.Printf("%s is %s\n\n", name, primaryCondition)
			}
		}

		fmt.Println(art.GetStaticArt(p, status))

		if p.State.Message != "" {
			fmt.Printf("\nMessage: %s\n", p.State.Message)
		}

		// Save state
		if err := storage.SavePetState(p, statePath); err != nil {
			return fmt.Errorf("failed to save state: %w", err)
		}

		return nil
	},
}

func init() {
	statusCmd.Flags().BoolP("verbose", "v", false, "Show verbose stats card")
}

var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Feed your familiar",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			now := time.Now()

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			// Handle sleep state
			if p.State.IsAsleep {
				p.State.SleepAttempts++
				if p.State.SleepAttempts == 1 {
					fmt.Printf("%s is asleep\n", petName)
					return nil
				} else if p.State.SleepAttempts == 2 {
					fmt.Printf("%s is still asleep\n", petName)
					return nil
				} else {
					// Third attempt - wake up and take action
					p.State.IsAsleep = false
					p.State.SleepUntil = time.Time{}
					p.State.SleepAttempts = 0
					fmt.Printf("%s wakes up!\n", petName)
					// Continue to feed action below
				}
			}

			// Cannot feed if stone
			if p.State.IsStone {
				return fmt.Errorf("your familiar is stone. Use 'awaken' first")
			}

			// Evolve from egg (0) to first evolution (1) on first interaction
			if p.State.Evolution == 0 {
				p.State.Evolution = 1
			}

			// Decrease hunger (lower is better) and increase happiness
			p.State.Hunger = max(0, p.State.Hunger-20)
			p.State.Happiness = min(100, p.State.Happiness+10)

			// Record interaction
			p.State.LastFed = now
			p.State.LastFeeds = appendInteraction(p.State.LastFeeds, pet.Interaction{
				Time:   now,
				Action: pet.InteractionFeed,
			})

			fmt.Println("Fed your familiar!")
			return nil
		})
	},
}

var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play with your familiar",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			now := time.Now()

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			// Handle sleep state
			if p.State.IsAsleep {
				p.State.SleepAttempts++
				if p.State.SleepAttempts == 1 {
					fmt.Printf("%s is asleep\n", petName)
					return nil
				} else if p.State.SleepAttempts == 2 {
					fmt.Printf("%s is still asleep\n", petName)
					return nil
				} else {
					// Third attempt - wake up and take action
					p.State.IsAsleep = false
					p.State.SleepUntil = time.Time{}
					p.State.SleepAttempts = 0
					fmt.Printf("%s wakes up!\n", petName)
					// Continue to play action below
				}
			}

			// Cannot play if stone
			if p.State.IsStone {
				return fmt.Errorf("your familiar is stone. Use 'awaken' first")
			}

			// Evolve from egg (0) to first evolution (1) on first interaction
			if p.State.Evolution == 0 {
				p.State.Evolution = 1
			}

			// Increase happiness, decrease energy
			p.State.Happiness = min(100, p.State.Happiness+15)
			p.State.Energy = max(0, p.State.Energy-10)

			// Record interaction
			p.State.LastPlayed = now
			p.State.LastPlays = appendInteraction(p.State.LastPlays, pet.Interaction{
				Time:   now,
				Action: pet.InteractionPlay,
			})

			fmt.Println("Played with your familiar!")
			return nil
		})
	},
}

var restCmd = &cobra.Command{
	Use:   "rest",
	Short: "Put your familiar to sleep (restorative sleep)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			now := time.Now()

			// If already asleep, do nothing
			if p.State.IsAsleep {
				petName := p.Config.Name
				if p.State.NameOverride != "" {
					petName = p.State.NameOverride
				}
				fmt.Printf("%s is already asleep\n", petName)
				return nil
			}

			// Cannot sleep if stone
			if p.State.IsStone {
				return fmt.Errorf("your familiar is stone. Use 'awaken' first")
			}

			// Set sleep duration
			sleepDuration := p.Config.SleepDuration
			if sleepDuration == 0 {
				sleepDuration = 30 * time.Minute // Default 30 minutes
			}

			// Put to sleep
			p.State.IsAsleep = true
			p.State.SleepUntil = now.Add(sleepDuration)
			p.State.SleepAttempts = 0

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			fmt.Printf("%s has fallen asleep (will wake in %s)\n", petName, sleepDuration)
			return nil
		})
	},
}

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Administrative commands for familiar management",
}

var adminHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Get health status for prompt",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))

			const resetCode = "\033[0m"

			// Always show 🐾
			icon := "🐾"

			// Determine color based on stone state or health percentage
			// Check stone state same way conditions does: IsStone OR health < threshold
			isStone := p.State.IsStone || health < p.Config.StoneThreshold
			var colorCode string
			if isStone {
				colorCode = "\033[90m" // Gray - Stone
			} else {
				switch {
				case health >= 80:
					colorCode = "\033[32m" // Green - Excellent
				case health >= 60:
					colorCode = "\033[33m" // Yellow - Good
				case health >= 40:
					colorCode = "\033[93m" // Bright Yellow - Fair
				case health >= 20:
					colorCode = "\033[38;5;208m" // Orange - Poor
				default:
					colorCode = "\033[31m" // Red - Critical
				}
			}

			// Single character to represent health or message
			var healthChar string
			if p.State.Message != "" {
				healthChar = "▲" // Triangle for message
			} else {
				healthChar = "●" // Circle for health
			}

			fmt.Printf("%s %s%s%s", icon, colorCode, healthChar, resetCode)
			return nil
		})
	},
}

var adminCompletionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for familiar.

To load completions:

Bash:
  $ source <(familiar admin completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ familiar admin completion bash > /etc/bash_completion.d/familiar
  # macOS:
  $ familiar admin completion bash > /usr/local/etc/bash_completion.d/familiar

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ familiar admin completion zsh > "${fpath[1]}/_familiar"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ familiar admin completion fish | source

  # To load completions for each session, execute once:
  $ familiar admin completion fish > ~/.config/fish/completions/familiar.fish

PowerShell:
  PS> familiar admin completion powershell | Out-String | Invoke-Expression

  # To load completions for each session:
  PS> familiar admin completion powershell > familiar.ps1
  # and source this file from your PowerShell profile.
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.ExactValidArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := cmd.Root()
		switch args[0] {
		case "bash":
			return root.GenBashCompletion(os.Stdout)
		case "zsh":
			return root.GenZshCompletion(os.Stdout)
		case "fish":
			return root.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return root.GenPowerShellCompletion(os.Stdout)
		}
		return fmt.Errorf("unsupported shell: %s", args[0])
	},
}

var adminUpdateCmd = &cobra.Command{
	Use:   "update [pet-type]",
	Short: "Update pet config from template (preserves customizations)",
	Long: `Update your familiar's config file from the latest template.

This command:
  - Loads the latest template for your pet type (or the specified type)
  - Merges it with your existing config
  - Preserves your customizations (name, custom settings, etc.)
  - Updates animations and default settings from the template

If pet-type is not specified, it will be auto-detected from your current pet.
`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, configPath, _, err := loadPet()
		if err != nil {
			return err
		}

		var petType string
		if len(args) > 0 {
			petType = args[0]
		} else {
			// Try to detect pet type by checking which template matches
			petType, err = detectPetType(p)
			if err != nil {
				return fmt.Errorf("could not auto-detect pet type. Please specify: %w", err)
			}
		}

		// Find lib directory
		libDir, err := storage.FindLibDir()
		if err != nil {
			return fmt.Errorf("failed to find lib directory: %w", err)
		}

		// Load template
		templatePath := filepath.Join(libDir, petType+".toml")
		templateData, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", templatePath, err)
		}

		// Replace placeholders with valid dummy values for parsing
		// We need valid TOML syntax to parse, but the actual values don't matter
		// since we'll merge with existing config anyway
		templateContent := string(templateData)
		now := time.Now()
		dummyName := "TemplatePet"
		dummyCreatedAt := now.Format(time.RFC3339Nano)

		templateContent = strings.ReplaceAll(templateContent, "{{NAME}}", dummyName)
		templateContent = strings.ReplaceAll(templateContent, "{{CREATED_AT}}", dummyCreatedAt)

		// Parse template - handle duration strings like LoadPet does
		var templateConfig pet.PetConfig

		// Try to unmarshal directly first
		if err := toml.Unmarshal([]byte(templateContent), &templateConfig); err != nil {
			// If it fails, try parsing as map to handle duration strings
			var configMap map[string]interface{}
			if err2 := toml.Unmarshal([]byte(templateContent), &configMap); err2 != nil {
				return fmt.Errorf("failed to parse template: %w", err)
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
				return fmt.Errorf("failed to re-marshal template: %w", err)
			}

			if err := toml.Unmarshal(configBytes, &templateConfig); err != nil {
				return fmt.Errorf("failed to parse template: %w", err)
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

		// Merge: preserve user values, update from template
		mergedConfig := mergeConfig(p.Config, templateConfig)

		// Ensure petType is set (replace placeholder or use detected type)
		if mergedConfig.PetType == "" || mergedConfig.PetType == "{{PET_TYPE}}" {
			mergedConfig.PetType = petType
		}

		// Save merged config
		configData, err := toml.Marshal(mergedConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}

		if err := os.WriteFile(configPath, configData, 0644); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		petName := p.Config.Name
		if p.State.NameOverride != "" {
			petName = p.State.NameOverride
		}

		fmt.Printf("%s's config has been updated from %s template\n", petName, petType)
		return nil
	},
}

var adminArtCmd = &cobra.Command{
	Use:   "art [state|list]",
	Short: "Show art for a specific state or list available states",
	Long: `Show art for a specific state.

Examples:
  familiar admin art lonely                    # Show the lonely animation (current evolution)
  familiar admin art happy                     # Show the happy animation (current evolution)
  familiar admin art --evolution 2 happy       # Show evolution 2 happy animation
  familiar admin art --type cat happy          # Show cat template's happy animation
  familiar admin art --type pixel --evolution 2 happy  # Show pixel evolution 2 happy
  familiar admin art list                      # List all available animation states

Available states depend on your pet's configuration. Common states include:
  default, egg, lonely, hungry, tired, sad, happy, stone, infirm, asleep, has-message

Evolution 0 always shows egg art. Evolution 1 is the default for most pets.
Use --evolution flag to preview art at different evolution levels.
Use --type flag to view animations from a template type (cat, dancer, pixel) without needing that type installed.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		state := args[0]
		evolutionOverride, _ := cmd.Flags().GetInt("evolution")
		typeOverride, _ := cmd.Flags().GetString("type")

		var p *pet.Pet
		var err error

		// If --type is specified, load from template instead of installed pet
		if typeOverride != "" {
			p, err = storage.LoadTemplateConfig(typeOverride)
			if err != nil {
				return fmt.Errorf("failed to load template '%s': %w", typeOverride, err)
			}
			// For templates, default to evolution 1 if not specified
			if evolutionOverride < 0 {
				evolutionOverride = 1
			}
		} else {
			// Load installed pet
			p, _, _, err = loadPet()
			if err != nil {
				return err
			}
		}

		// Handle "list" command
		if state == "list" {
			if p.Config.Animations == nil || len(p.Config.Animations) == 0 {
				fmt.Println("No animations available")
				return nil
			}

			// Collect and sort animation keys
			keys := make([]string, 0, len(p.Config.Animations))
			for k := range p.Config.Animations {
				keys = append(keys, k)
			}

			// Sort keys for consistent output
			sort.Strings(keys)

			fmt.Println("Available animation states:")
			for _, key := range keys {
				anim := p.Config.Animations[key]
				frameCount := len(anim.Frames)
				source := anim.Source
				if source == "" {
					source = "inline"
				}
				fmt.Printf("  %s (%s, %d frame(s))\n", key, source, frameCount)
			}
			return nil
		}

		// Determine which evolution to use
		evolution := 1 // Default for templates
		if typeOverride == "" {
			// For installed pets, use their current evolution
			evolution = p.State.Evolution
		}
		if evolutionOverride >= 0 {
			evolution = evolutionOverride
		}

		// Evolution 0 always shows egg (unless it's a special state like stone egg)
		if evolution == 0 {
			// For egg, check if there's a state-specific egg (like stone+egg)
			// But for simplicity, just show egg art
			if state == "egg" || state == "default" {
				anim, exists := p.Config.Animations["egg"]
				if exists && len(anim.Frames) > 0 {
					return displayArt(anim)
				}
			}
			// For other states at evolution 0, still show egg
			anim, exists := p.Config.Animations["egg"]
			if exists && len(anim.Frames) > 0 {
				return displayArt(anim)
			}
			return fmt.Errorf("egg animation not found")
		}

		// For evolution > 0, use ChooseAnimationKey to find the right animation
		// Create a fake status with the requested condition
		conds := make(map[conditions.Condition]bool)

		// Map state string to condition
		switch state {
		case "has-message":
			conds[conditions.CondHasMessage] = true
		case "stone":
			conds[conditions.CondStone] = true
		case "asleep":
			conds[conditions.CondAsleep] = true
		case "infirm":
			conds[conditions.CondInfirm] = true
		case "lonely":
			conds[conditions.CondLonely] = true
		case "hungry":
			conds[conditions.CondHungry] = true
		case "tired":
			conds[conditions.CondTired] = true
		case "sad":
			conds[conditions.CondSad] = true
		case "happy":
			conds[conditions.CondHappy] = true
		case "default":
			// No conditions - will use default
		default:
			// Try to find animation directly by state name first
			anim, exists := p.Config.Animations[state]
			if exists && len(anim.Frames) > 0 {
				return displayArt(anim)
			}
			return fmt.Errorf("unknown state '%s'. Use 'familiar admin art list' to see available states", state)
		}

		// Use ChooseAnimationKey to find the right animation key for this evolution
		key := art.ChooseAnimationKey(conds, evolution, p.Config.Animations)

		// Try to get the animation
		anim, exists := p.Config.Animations[key]
		if !exists {
			// Fallback: try the state name directly
			anim, exists = p.Config.Animations[state]
			if !exists {
				return fmt.Errorf("animation for state '%s' at evolution %d not found. Use 'familiar admin art list' to see available states", state, evolution)
			}
		}

		if len(anim.Frames) == 0 {
			return fmt.Errorf("animation state '%s' has no frames", state)
		}

		return displayArt(anim)
	},
}

func displayArt(anim pet.AnimationConfig) error {
	// If animation has multiple frames, play the animation
	// Otherwise just show the first frame
	if len(anim.Frames) > 1 {
		// Play animation
		if anim.Source == "pixel" {
			art.PlayPixelAnimation(anim)
		} else {
			art.PlayAnimation(anim)
		}
		return nil
	}

	// Single frame - just display it
	if len(anim.Frames) > 0 {
		if anim.Source == "pixel" {
			// Render pixel art
			rendered := art.RenderPixelArt(anim.Frames[0])
			fmt.Print(rendered)
			if !strings.HasSuffix(rendered, "\n") {
				fmt.Println()
			}
		} else {
			// Display inline ASCII art
			artStr := anim.Frames[0].Art
			fmt.Print(artStr)
			if !strings.HasSuffix(artStr, "\n") {
				fmt.Println()
			}
		}
	}
	return nil
}

func init() {
	adminCmd.AddCommand(adminHealthCmd)
	adminCmd.AddCommand(adminCompletionCmd)
	adminCmd.AddCommand(adminUpdateCmd)
	adminArtCmd.Flags().IntP("evolution", "e", -1, "Evolution level to preview (default: current evolution for installed pet, 1 for templates)")
	adminArtCmd.Flags().StringP("type", "t", "", "Pet type template to use (cat, dancer, pixel) - ignores installed familiar")
	adminCmd.AddCommand(adminArtCmd)
}

var messageCmd = &cobra.Command{
	Use:   "message [text]",
	Short: "Set a message for your familiar",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			message := args[0]
			p.State.Message = message
			fmt.Printf("Message set: %s\n", message)
			return nil
		})
	},
}

var acknowledgeCmd = &cobra.Command{
	Use:   "acknowledge",
	Short: "Acknowledge your familiar (clears message, improves mood)",
	RunE: func(cmd *cobra.Command, args []string) error {
		silent, _ := cmd.Flags().GetBool("silent")
		return executeStatefulCommand(func(p *pet.Pet) error {
			hadMessage := p.State.Message != ""
			p.State.Message = ""

			if hadMessage {
				// If there was a message, boost everything to 100
				p.State.Hunger = 0 // 0 = not hungry (best)
				p.State.Happiness = 100
				p.State.Energy = 100
			} else {
				// If no message, boost by 5
				p.State.Hunger = max(0, p.State.Hunger-5) // Decrease hunger (lower is better)
				p.State.Happiness = min(100, p.State.Happiness+5)
				p.State.Energy = min(100, p.State.Energy+5)
			}

			if silent {
				// Silent mode: no output
				return nil
			}

			// Normal mode: show name, condition, art, and confirmation
			now := time.Now()
			health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
			status := conditions.DeriveStatus(p, now, health)

			name := p.Config.Name
			if p.State.NameOverride != "" {
				name = p.State.NameOverride
			}

			// After acknowledge, show "happy" if stats are good (as per README)
			displayCondition := status.Primary
			if p.State.Hunger < 30 && p.State.Happiness > 70 && p.State.Energy > 50 {
				displayCondition = conditions.CondHappy
			}

			fmt.Printf("%s\n", name)
			fmt.Printf("%s\n\n", displayCondition)
			fmt.Println(art.GetStaticArt(p, status))
			fmt.Printf("%s feels acknowledged\n", name)

			return nil
		})
	},
}

func init() {
	acknowledgeCmd.Flags().BoolP("silent", "s", false, "Silent mode: no output")
}

var dismissCmd = &cobra.Command{
	Use:   "dismiss",
	Short: "Dismiss your familiar (soft delete - can be restored)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, configPath, statePath, err := loadPet()
		if err != nil {
			return err
		}

		petName := p.Config.Name
		if p.State.NameOverride != "" {
			petName = p.State.NameOverride
		}

		petDir := filepath.Dir(statePath)
		if err := storage.ReleasePet(petDir, configPath, statePath, petName); err != nil {
			return fmt.Errorf("failed to dismiss familiar: %w", err)
		}

		fmt.Printf("Familiar '%s' has been dismissed (can be restored with 'summon')\n", petName)
		return nil
	},
}

var awakenCmd = &cobra.Command{
	Use:   "awaken",
	Short: "Awaken your familiar from stone or sleep state",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			now := time.Now()
			health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
			status := conditions.DeriveStatus(p, now, health)

			// Check stone state same way conditions does: IsStone OR health < threshold
			isStone := p.State.IsStone || health < p.Config.StoneThreshold
			isAsleep := p.State.IsAsleep

			if !isStone && !isAsleep {
				currentState := conditions.FormatConditions(status.AllOrdered)
				return fmt.Errorf("your familiar is not stone or asleep. It is %s", currentState)
			}

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			if isStone {
				// Remove stone state
				p.State.IsStone = false

				// Set to minimal life (safely above stone threshold)
				// Target health is stone threshold + 10 to ensure we don't immediately become stone again
				targetHealth := p.Config.StoneThreshold + 10
				if targetHealth > 100 {
					targetHealth = 100
				}

				// For average mode: health = (hungerScore + happiness + energy) / 3
				// For weighted mode: health = hungerScore*0.3 + happiness*0.4 + energy*0.3
				// Set all three to the same value to work for both modes
				// hungerScore = 100 - hunger, so hunger = 100 - targetHealth
				// Set hunger, happiness, and energy so health = targetHealth
				p.State.Hunger = 100 - targetHealth // Higher hunger = worse (inverted)
				p.State.Happiness = targetHealth
				p.State.Energy = targetHealth

				fmt.Printf("%s has awakened from stone!\n", petName)
			}

			if isAsleep {
				// Remove sleep state
				p.State.IsAsleep = false
				p.State.SleepUntil = time.Time{}
				p.State.SleepAttempts = 0

				fmt.Printf("%s has awakened from sleep!\n", petName)
			}

			return nil
		})
	},
}

var ossifyCmd = &cobra.Command{
	Use:   "ossify",
	Short: "Turn your familiar to stone",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			if p.State.IsStone {
				return fmt.Errorf("your familiar is already stone")
			}

			// Set to stone state (overrides sleep)
			p.State.IsStone = true
			p.State.IsAsleep = false
			p.State.SleepUntil = time.Time{}
			p.State.SleepAttempts = 0

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			fmt.Printf("%s has turned to stone\n", petName)
			return nil
		})
	},
}

var healCmd = &cobra.Command{
	Use:   "heal",
	Short: "Heal your familiar (boost energy and happiness, remove infirm)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return executeStatefulCommand(func(p *pet.Pet) error {
			// Boost energy and happiness by 3
			p.State.Energy = min(100, p.State.Energy+3)
			p.State.Happiness = min(100, p.State.Happiness+3)

			// Remove infirm if present
			if p.State.IsInfirm {
				p.State.IsInfirm = false
			}

			petName := p.Config.Name
			if p.State.NameOverride != "" {
				petName = p.State.NameOverride
			}

			fmt.Printf("%s has been healed\n", petName)
			return nil
		})
	},
}

var banishCmd = &cobra.Command{
	Use:   "banish",
	Short: "Banish your familiar (permanent delete)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p, configPath, statePath, err := loadPet()
		if err != nil {
			return err
		}

		petName := p.Config.Name
		if p.State.NameOverride != "" {
			petName = p.State.NameOverride
		}

		petDir := filepath.Dir(statePath)
		if err := storage.BanishPet(petDir, configPath, statePath); err != nil {
			return fmt.Errorf("failed to banish familiar: %w", err)
		}

		fmt.Printf("Familiar '%s' has been banished (permanently deleted)\n", petName)
		return nil
	},
}

func appendInteraction(interactions []pet.Interaction, newInteraction pet.Interaction) []pet.Interaction {
	// Keep max 5 interactions
	interactions = append(interactions, newInteraction)
	if len(interactions) > 5 {
		interactions = interactions[len(interactions)-5:]
	}
	return interactions
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// detectPetType tries to determine the pet type by comparing animations with known templates
func detectPetType(p *pet.Pet) (string, error) {
	// First, check if petType is explicitly set in config (ignore placeholder values)
	if p.Config.PetType != "" && p.Config.PetType != "{{PET_TYPE}}" {
		return p.Config.PetType, nil
	}

	// Quick heuristic: check animation source type
	if p.Config.Animations != nil {
		if defaultAnim, hasDefault := p.Config.Animations["default"]; hasDefault {
			if defaultAnim.Source == "pixel" {
				// Try pixel first if it's pixel art
				libDir, err := storage.FindLibDir()
				if err == nil {
					templatePath := filepath.Join(libDir, "pixel.toml")
					if _, err := os.Stat(templatePath); err == nil {
						return "pixel", nil
					}
				}
			}
		}
	}

	libDir, err := storage.FindLibDir()
	if err != nil {
		return "", err
	}

	// Check known pet types
	knownTypes := []string{"cat", "dancer", "pixel"}

	for _, petType := range knownTypes {
		templatePath := filepath.Join(libDir, petType+".toml")
		templateData, err := os.ReadFile(templatePath)
		if err != nil {
			continue
		}

		// Replace placeholders with valid dummy values for parsing
		templateContent := string(templateData)
		now := time.Now()
		dummyName := "TemplatePet"
		dummyCreatedAt := now.Format(time.RFC3339Nano)

		templateContent = strings.ReplaceAll(templateContent, "{{NAME}}", dummyName)
		templateContent = strings.ReplaceAll(templateContent, "{{CREATED_AT}}", dummyCreatedAt)

		var templateConfig pet.PetConfig
		if err := toml.Unmarshal([]byte(templateContent), &templateConfig); err != nil {
			continue
		}

		// Compare default animations to see if they match
		if matchesPetType(p, &templateConfig) {
			return petType, nil
		}
	}

	return "", fmt.Errorf("could not match pet to any known template")
}

// matchesPetType checks if the pet's config matches a template type
func matchesPetType(p *pet.Pet, template *pet.PetConfig) bool {
	// Compare key characteristics
	// Check if default animation matches (simplified check)
	if p.Config.Animations != nil && template.Animations != nil {
		petDefault, petHasDefault := p.Config.Animations["default"]
		templateDefault, templateHasDefault := template.Animations["default"]

		if petHasDefault && templateHasDefault {
			// Check animation source type first - this is a strong indicator
			if petDefault.Source == "pixel" && templateDefault.Source == "pixel" {
				// Both are pixel art - check if they have pixels
				if len(petDefault.Frames) > 0 && len(templateDefault.Frames) > 0 {
					petFrame := petDefault.Frames[0]
					templateFrame := templateDefault.Frames[0]
					if len(petFrame.Pixels) > 0 && len(templateFrame.Pixels) > 0 {
						// Both have pixel art - likely same type
						// Check dimensions are similar (within reasonable range)
						petHeight := len(petFrame.Pixels)
						templateHeight := len(templateFrame.Pixels)
						if petHeight > 0 && templateHeight > 0 {
							petWidth := len(petFrame.Pixels[0])
							templateWidth := len(templateFrame.Pixels[0])
							// Allow some variance in dimensions (within 2 pixels)
							heightDiff := petHeight - templateHeight
							widthDiff := petWidth - templateWidth
							if heightDiff >= -2 && heightDiff <= 2 && widthDiff >= -2 && widthDiff <= 2 {
								return true
							}
						}
					}
				}
			} else if petDefault.Source == "inline" && templateDefault.Source == "inline" {
				// For inline art, compare ASCII art
				if len(petDefault.Frames) > 0 && len(templateDefault.Frames) > 0 {
					petArt := petDefault.Frames[0].Art
					templateArt := templateDefault.Frames[0].Art
					// Simple comparison - if art matches, likely same type
					if petArt == templateArt {
						return true
					}
				}
			}
		}
	}

	// Fallback: check allowAnsiAnimations and other characteristics
	// Cat typically has allowAnsiAnimations=false, dancer has true, pixel has true
	if p.Config.AllowAnsiAnimations == template.AllowAnsiAnimations {
		// Additional check: compare some other settings
		if p.Config.StoneThreshold == template.StoneThreshold &&
			p.Config.InfirmEnabled == template.InfirmEnabled {
			return true
		}
	}

	return false
}

// mergeConfig merges template config with existing config, preserving user customizations
func mergeConfig(existing, template pet.PetConfig) pet.PetConfig {
	merged := template

	// Preserve user-specific values that should never change
	merged.Name = existing.Name
	merged.PetType = existing.PetType // Preserve existing petType, or use template's if not set
	if merged.PetType == "" {
		merged.PetType = template.PetType
	}
	merged.CreatedAt = existing.CreatedAt
	merged.Evolution = existing.Evolution
	merged.MaxEvolution = existing.MaxEvolution
	merged.EvolutionMode = existing.EvolutionMode

	// Update animations from template (this is the main thing we want to update)
	// This gets new animations like "asleep" that were added to templates
	merged.Animations = template.Animations

	// Preserve user's animation preferences
	merged.AllowAnsiAnimations = existing.AllowAnsiAnimations

	// Preserve user customizations for decay/behavior settings
	// These are things users might have tuned for their pet
	merged.DecayEnabled = existing.DecayEnabled
	merged.DecayRate = existing.DecayRate
	merged.HungerDecayPerHour = existing.HungerDecayPerHour
	merged.HappinessDecayPerHour = existing.HappinessDecayPerHour
	merged.EnergyDecayPerHour = existing.EnergyDecayPerHour

	// Preserve threshold and multiplier settings
	merged.StoneThreshold = existing.StoneThreshold
	merged.InfirmEnabled = existing.InfirmEnabled
	merged.InfirmDecayMultiplier = existing.InfirmDecayMultiplier
	merged.StoneDecayMultiplier = existing.StoneDecayMultiplier

	// Preserve sleep duration if user customized it
	if existing.SleepDuration != 0 {
		merged.SleepDuration = existing.SleepDuration
	}

	// Preserve other user settings
	merged.EventChance = existing.EventChance
	merged.HealthComputation = existing.HealthComputation
	merged.InteractionThreshold = existing.InteractionThreshold
	merged.CacheTTL = existing.CacheTTL

	return merged
}
