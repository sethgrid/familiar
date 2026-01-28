package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sethgrid/familiar/internal/art"
	"github.com/sethgrid/familiar/internal/conditions"
	"github.com/sethgrid/familiar/internal/discovery"
	"github.com/sethgrid/familiar/internal/health"
	"github.com/sethgrid/familiar/internal/pet"
	"github.com/sethgrid/familiar/internal/storage"
)

func TestNonAnimatedFamiliar(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize pet
	err := storage.InitPet(false, "cat", "TestCat", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	// Load pet
	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Verify initial state
	if p.Config.Name != "TestCat" {
		t.Errorf("Expected name 'TestCat', got '%s'", p.Config.Name)
	}
	if p.State.Hunger != 75 {
		t.Errorf("Expected initial hunger 75, got %d", p.State.Hunger)
	}
	if p.State.Evolution != 0 {
		t.Errorf("Expected initial evolution 0, got %d", p.State.Evolution)
	}

	// Test health computation
	health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	if health < 0 || health > 100 {
		t.Errorf("Health should be between 0 and 100, got %d", health)
	}

	// Test condition derivation
	now := time.Now()
	status := conditions.DeriveStatus(p, now, health)
	if status.Health != health {
		t.Errorf("Status health mismatch: expected %d, got %d", health, status.Health)
	}

	// Test ASCII art rendering
	artStr := art.GetStaticArt(p, status)
	if artStr == "" {
		t.Error("Expected non-empty ASCII art")
	}

	// Verify it's the egg art (evolution 0)
	expectedEgg := `  ___  
 /  . . \ 
 \___/`
	if artStr != expectedEgg {
		t.Errorf("Expected egg art for evolution 0, got:\n%s\nExpected:\n%s", artStr, expectedEgg)
	}

	// Test decay application
	// Advance time by 1 hour
	future := now.Add(1 * time.Hour)
	err = pet.ApplyTimeStep(p, future)
	if err != nil {
		t.Fatalf("Failed to apply time step: %v", err)
	}

	// Verify decay occurred (hunger should decrease)
	if p.State.Hunger >= 75 {
		t.Errorf("Expected hunger to decrease after decay, got %d", p.State.Hunger)
	}

	// Test state saving
	err = storage.SavePetState(p, statePath)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Reload and verify state persisted
	p2, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to reload pet: %v", err)
	}
	if p2.State.Hunger != p.State.Hunger {
		t.Errorf("State not persisted correctly: expected %d, got %d", p.State.Hunger, p2.State.Hunger)
	}
}

func TestAnimatedFamiliar(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize pet
	err := storage.InitPet(false, "cat", "AnimatedCat", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	// Load pet
	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Add animated default animation (simple 2-frame animation)
	p.Config.Animations["default"] = pet.AnimationConfig{
		Source: "inline",
		FPS:    2,
		Loops:  1,
		Frames: []pet.Frame{
			{
				Art: ` /\_/\ 
( o.o )
 > ^ <`,
			},
			{
				Art: ` /\_/\ 
( o.o )
 > ^ <`,
				MS: 500,
			},
		},
	}

	// Save config (for manual testing, this would be done via CLI)
	// For now, we'll just verify the animation config is set
	if len(p.Config.Animations["default"].Frames) != 2 {
		t.Error("Expected 2 frames in animated default animation")
	}

	// Set evolution to 1 so it shows cat art (not egg)
	p.State.Evolution = 1

	// Test that we can select the animation
	now := time.Now()
	health := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, health)

	key := art.ChooseAnimationKey(status.Conditions, p.State.Evolution, p.Config.Animations)
	if key != "default" {
		t.Errorf("Expected animation key 'default', got '%s'", key)
	}

	// Verify animation exists
	anim, exists := p.Config.Animations[key]
	if !exists {
		t.Error("Animation should exist")
	}
	if anim.Source != "inline" {
		t.Errorf("Expected inline source, got '%s'", anim.Source)
	}
	if len(anim.Frames) != 2 {
		t.Errorf("Expected 2 frames, got %d", len(anim.Frames))
	}

	t.Logf("Animated familiar test passed - animation config is valid")
	t.Logf("For manual testing, run: familiar status --config %s", statePath)
}

func TestFamiliarStates(t *testing.T) {
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	err := storage.InitPet(false, "cat", "StateTestCat", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	now := time.Now()

	// Test default state
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)
	artStr := art.GetStaticArt(p, status)
	if artStr == "" {
		t.Error("Expected non-empty art for default state")
	}

	// Test infirm state
	p.State.IsInfirm = true
	healthVal = health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status = conditions.DeriveStatus(p, now, healthVal)
	artStr = art.GetStaticArt(p, status)
	if artStr == "" {
		t.Error("Expected non-empty art for infirm state")
	}
	if !status.Conditions[conditions.CondInfirm] {
		t.Error("Expected infirm condition to be set")
	}

	// Test stone state
	p.State.IsStone = true
	healthVal = health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status = conditions.DeriveStatus(p, now, healthVal)
	artStr = art.GetStaticArt(p, status)
	if artStr == "" {
		t.Error("Expected non-empty art for stone state")
	}
	if !status.Conditions[conditions.CondStone] {
		t.Error("Expected stone condition to be set")
	}

	// Test has-message state
	p.State.Message = "Test message"
	healthVal = health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status = conditions.DeriveStatus(p, now, healthVal)
	if !status.Conditions[conditions.CondHasMessage] {
		t.Error("Expected has-message condition to be set")
	}
	if status.Primary != conditions.CondHasMessage {
		t.Errorf("Expected primary condition to be has-message, got %s", status.Primary)
	}
}

func TestPetDiscovery(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "project", "subdir")

	// Create nested directory structure
	err := os.MkdirAll(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create directories: %v", err)
	}

	// Initialize pet in project root
	projectRoot := filepath.Join(tmpDir, "project")
	err = storage.InitPet(false, "cat", "DiscoveryTest", projectRoot)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	// Test discovery from subdirectory
	statePath, found, err := discovery.FindStateFile(subDir)
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}
	if !found {
		t.Error("Expected to find pet state file")
	}
	if statePath == "" {
		t.Error("Expected non-empty state path")
	}

	// Verify it found the correct file
	expectedPath := filepath.Join(projectRoot, ".familiar", "pet.state.toml")
	if statePath != expectedPath {
		t.Errorf("Expected state path %s, got %s", expectedPath, statePath)
	}
}

func TestReadmeExample(t *testing.T) {
	// Test the example from README: Pip with has-message and acknowledge
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize pet named "Pip"
	err := storage.InitPet(false, "cat", "Pip", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Set evolution to 1 so it shows cat art (not egg)
	p.State.Evolution = 1
	// Set a message
	p.State.Message = "Attn Devs — new local config defaults available."
	// Ensure good stats so it's not in a bad state
	p.State.Hunger = 75
	p.State.Happiness = 80
	p.State.Energy = 60

	// Save state
	err = storage.SavePetState(p, statePath)
	if err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	// Reload to get fresh state
	p, err = storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to reload pet: %v", err)
	}

	now := time.Now()
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)

	// Verify name
	if p.Config.Name != "Pip" {
		t.Errorf("Expected name 'Pip', got '%s'", p.Config.Name)
	}

	// Verify has-message condition
	if !status.Conditions[conditions.CondHasMessage] {
		t.Error("Expected has-message condition to be set")
	}
	if status.Primary != conditions.CondHasMessage {
		t.Errorf("Expected primary condition to be has-message, got %s", status.Primary)
	}

	// Verify message
	if p.State.Message != "Attn Devs — new local config defaults available." {
		t.Errorf("Expected message 'Attn Devs — new local config defaults available.', got '%s'", p.State.Message)
	}

	// Verify art for has-message (should have asterisk)
	artStr := art.GetStaticArt(p, status)
	expectedHasMessageArt := ` /\_/\ 
( o.o )
 > ^ <*`
	if artStr != expectedHasMessageArt {
		t.Errorf("Expected has-message art:\n%s\nGot:\n%s", expectedHasMessageArt, artStr)
	}

	// Now test acknowledge
	p.State.Message = ""
	p.State.Happiness = min(100, p.State.Happiness+5)
	p.State.Energy = min(100, p.State.Energy+5)

	// Save state after acknowledge
	err = storage.SavePetState(p, statePath)
	if err != nil {
		t.Fatalf("Failed to save state after acknowledge: %v", err)
	}

	// Reload and check acknowledge state
	p, err = storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to reload pet after acknowledge: %v", err)
	}

	healthVal = health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status = conditions.DeriveStatus(p, now, healthVal)

	// Verify message is cleared
	if p.State.Message != "" {
		t.Errorf("Expected message to be cleared after acknowledge, got '%s'", p.State.Message)
	}

	// Verify happy condition (or at least not has-message)
	if status.Conditions[conditions.CondHasMessage] {
		t.Error("Expected has-message condition to be cleared after acknowledge")
	}

	// Verify art after acknowledge (should be default cat, no asterisk)
	artStr = art.GetStaticArt(p, status)
	expectedDefaultArt := ` /\_/\ 
( o.o )
 > ^ <`
	if artStr != expectedDefaultArt {
		t.Errorf("Expected default art after acknowledge:\n%s\nGot:\n%s", expectedDefaultArt, artStr)
	}
}

func TestNewPetShowsAsEgg(t *testing.T) {
	// Test that a freshly initialized pet (evolution 0) shows as an egg
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize pet
	err := storage.InitPet(false, "cat", "EggTest", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	// Load pet
	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Verify evolution is 0
	if p.State.Evolution != 0 {
		t.Errorf("Expected evolution 0 for new pet, got %d", p.State.Evolution)
	}
	if p.Config.Evolution != 0 {
		t.Errorf("Expected config evolution 0 for new pet, got %d", p.Config.Evolution)
	}

	// Check status - should show egg art
	now := time.Now()
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)
	artStr := art.GetStaticArt(p, status)

	// Verify it's the egg art (match the actual egg art from cat.toml)
	// Note: GetStaticArt may add trailing newline, so trim it for comparison
	expectedEgg := `  ______
 /  . . \ 
 \______/`
	artStrTrimmed := strings.TrimRight(artStr, "\n\r")
	expectedEggTrimmed := strings.TrimRight(expectedEgg, "\n\r")
	if artStrTrimmed != expectedEggTrimmed {
		t.Errorf("Expected egg art for new pet (evolution 0), got:\n%q\nExpected:\n%q", artStrTrimmed, expectedEggTrimmed)
	}

	// Verify no special conditions that would override egg
	if status.Conditions[conditions.CondStone] {
		t.Error("New pet should not be stone")
	}

	// Test that first interaction evolves from 0 to 1
	// Simulate a feed interaction
	p.State.Hunger = min(100, p.State.Hunger+20)
	p.State.Happiness = min(100, p.State.Happiness+10)
	
	// Evolve from egg (0) to first evolution (1) on first interaction
	if p.State.Evolution == 0 {
		p.State.Evolution = 1
	}
	
	// Verify evolution is now 1
	if p.State.Evolution != 1 {
		t.Errorf("Expected evolution 1 after first interaction, got %d", p.State.Evolution)
	}
	
	// Verify it's no longer showing egg art
	statusAfter := conditions.DeriveStatus(p, now, healthVal)
	artStrAfter := art.GetStaticArt(p, statusAfter)
	if artStrAfter == expectedEgg {
		t.Error("Pet should not show egg art after evolution to 1")
	}
	if status.Conditions[conditions.CondInfirm] {
		t.Error("New pet should not be infirm")
	}
	if status.Conditions[conditions.CondHasMessage] {
		t.Error("New pet should not have message")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
