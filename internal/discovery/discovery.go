package discovery

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindStateFile(startDir string) (string, bool, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve start directory: %w", err)
	}

	for {
		statePath := filepath.Join(dir, ".familiar", "pet.state.toml")
		if _, err := os.Stat(statePath); err == nil {
			return statePath, true, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", false, nil
}

func GlobalPetStatePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".familiar", "pet.state.toml")
}

func GetConfigPathFromState(statePath string) string {
	// Config is in the same directory
	return filepath.Join(filepath.Dir(statePath), "pet.toml")
}
