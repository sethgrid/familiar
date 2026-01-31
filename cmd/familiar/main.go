package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

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

const Version = "v0.2.0"

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
	rootCmd.AddCommand(healthCmd)
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
			fmt.Printf("%s\n\n", name)
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

var healthCmd = &cobra.Command{
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
