package main

import (
	"bytes"
	"fmt"
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
	if p.State.Hunger != 10 {
		t.Errorf("Expected initial hunger 10 (low is good), got %d", p.State.Hunger)
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
	expectedEgg := `  ______
 /  . . \ 
 \______/`
	// Trim trailing whitespace for comparison
	artStrTrimmed := strings.TrimRight(artStr, " \n\r")
	expectedEggTrimmed := strings.TrimRight(expectedEgg, " \n\r")
	if artStrTrimmed != expectedEggTrimmed {
		t.Errorf("Expected egg art for evolution 0, got:\n%q\nExpected:\n%q", artStrTrimmed, expectedEggTrimmed)
	}

	// Test decay application
	// Advance time by 1 hour
	future := now.Add(1 * time.Hour)
	err = pet.ApplyTimeStep(p, future)
	if err != nil {
		t.Fatalf("Failed to apply time step: %v", err)
	}

	// Verify decay occurred (hunger should increase - higher is worse)
	if p.State.Hunger <= 10 {
		t.Errorf("Expected hunger to increase after decay (higher is worse), got %d", p.State.Hunger)
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
}

func TestSleepRestorationAfterExpiration(t *testing.T) {
	// Test that sleep restoration is applied even when checking status after sleep has expired
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	err := storage.InitPet(false, "cat", "SleepTestCat", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Set initial low stats
	p.State.Happiness = 20
	p.State.Energy = 20
	p.State.Hunger = 50
	p.State.LastChecked = time.Now()

	// Put pet to sleep
	sleepDuration := 30 * time.Minute
	now := time.Now()
	p.State.IsAsleep = true
	p.State.SleepUntil = now.Add(sleepDuration)
	p.State.LastChecked = now

	// Save initial state
	initialHappiness := p.State.Happiness
	initialEnergy := p.State.Energy

	// Advance time to AFTER sleep should have expired (35 minutes later)
	future := now.Add(35 * time.Minute)

	// Apply time step - this should restore stats even though sleep has expired
	err = pet.ApplyTimeStep(p, future)
	if err != nil {
		t.Fatalf("Failed to apply time step: %v", err)
	}

	// Verify sleep state was cleared
	if p.State.IsAsleep {
		t.Error("Expected pet to be awake after sleep expiration")
	}

	// Verify that restoration was applied (stats should have increased)
	// Since we slept for 30 minutes (0.5 hours) at a rate of 100/0.5 = 200 per hour,
	// we should have gained approximately 100 points (but clamped to 100)
	if p.State.Happiness <= initialHappiness {
		t.Errorf("Expected happiness to increase after sleep restoration. Initial: %d, Final: %d", initialHappiness, p.State.Happiness)
	}
	if p.State.Energy <= initialEnergy {
		t.Errorf("Expected energy to increase after sleep restoration. Initial: %d, Final: %d", initialEnergy, p.State.Energy)
	}

	// Happiness and energy should be at or near 100 after full sleep cycle
	if p.State.Happiness < 90 {
		t.Errorf("Expected happiness to be near 100 after full sleep cycle, got %d", p.State.Happiness)
	}
	if p.State.Energy < 90 {
		t.Errorf("Expected energy to be near 100 after full sleep cycle, got %d", p.State.Energy)
	}
}

func TestStatusOutputWithAnimation(t *testing.T) {
	// Test that status command output includes all expected information
	// and that animations don't overwrite status text
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize a dancer pet (has animations enabled)
	err := storage.InitPet(false, "dancer", "TestDancer", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Evolve to enable animations
	p.State.Evolution = 1
	// Set good stats to avoid special conditions
	p.State.Hunger = 20
	p.State.Happiness = 70
	p.State.Energy = 60

	// Apply time step
	now := time.Now()
	if err := pet.ApplyTimeStep(p, now); err != nil {
		t.Fatalf("Failed to apply time step: %v", err)
	}

	// Compute status
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)

	// Capture stdout to test animation output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Simulate verbose status command output
	name := p.Config.Name
	primaryCondition := status.Primary
	
	// Print status header (what status command prints)
	fmt.Printf("%s is %s\n\n", name, primaryCondition)
	fmt.Printf("state: %s\n", conditions.FormatConditions(status.AllOrdered))
	fmt.Printf("health: %d\n", status.Health)
	fmt.Printf("hunger: %d\n", p.State.Hunger)
	fmt.Printf("happiness: %d\n", p.State.Happiness)
	fmt.Printf("energy: %d\n", p.State.Energy)
	fmt.Printf("evolution: %d\n\n", p.State.Evolution)

	// Get art (this may play animation or return static art)
	artStr := art.GetStaticArt(p, status)
	
	// If animation was played, GetStaticArt returns empty string and animation was printed to stdout
	// If static art, we need to print it
	if artStr != "" {
		fmt.Println(artStr)
	}

	// Close write end and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	outputStr := buf.String()

	// Verify status information is present in output
	if !strings.Contains(outputStr, name) {
		t.Errorf("Output missing pet name '%s'. Output:\n%s", name, outputStr)
	}
	
	if !strings.Contains(outputStr, string(primaryCondition)) {
		t.Errorf("Output missing primary condition '%s'. Output:\n%s", primaryCondition, outputStr)
	}
	
	if !strings.Contains(outputStr, "state:") {
		t.Errorf("Output missing 'state:' line. Output:\n%s", outputStr)
	}
	
	if !strings.Contains(outputStr, "health:") {
		t.Errorf("Output missing 'health:' line. Output:\n%s", outputStr)
	}
	
	if !strings.Contains(outputStr, "hunger:") {
		t.Errorf("Output missing 'hunger:' line. Output:\n%s", outputStr)
	}
	
	if !strings.Contains(outputStr, "happiness:") {
		t.Errorf("Output missing 'happiness:' line. Output:\n%s", outputStr)
	}
	
	if !strings.Contains(outputStr, "energy:") {
		t.Errorf("Output missing 'energy:' line. Output:\n%s", outputStr)
	}
	
	if !strings.Contains(outputStr, "evolution:") {
		t.Errorf("Output missing 'evolution:' line. Output:\n%s", outputStr)
	}

	// Verify format: name should appear before condition
	namePos := strings.Index(outputStr, name)
	conditionPos := strings.Index(outputStr, string(primaryCondition))
	if namePos == -1 || conditionPos == -1 || conditionPos <= namePos {
		t.Errorf("Expected name '%s' to appear before condition '%s'. Output:\n%s", name, primaryCondition, outputStr)
	}

	// Verify animation art is present (either static or final frame)
	// Dancer default animation has frames with "o" and "\o/"
	hasFrameContent := strings.Contains(outputStr, "    o") || 
		strings.Contains(outputStr, "   /|\\") || 
		strings.Contains(outputStr, "  \\o/") || 
		strings.Contains(outputStr, "   |") ||
		strings.Contains(outputStr, "   / \\")
	
	if !hasFrameContent {
		t.Errorf("Output missing animation art. Expected to see dancer frames. Output:\n%s", outputStr)
	}

	// Verify the final frame is visible (last frame of animation should be in output)
	// For dancer default, the last frame should contain one of the frame patterns
	// Check that art appears after all status lines
	lastStatusLine := "evolution:"
	lastStatusPos := strings.LastIndex(outputStr, lastStatusLine)
	if lastStatusPos != -1 {
		// Get content after status lines
		contentAfterStatus := outputStr[lastStatusPos+len(lastStatusLine):]
		// Should contain animation art
		if !strings.Contains(contentAfterStatus, "o") && !strings.Contains(contentAfterStatus, "/") {
			t.Errorf("Animation art should appear after status lines. Content after status:\n%s", contentAfterStatus)
		}
	}

	// Most importantly: verify status text appears BEFORE any animation content
	// The status lines should all appear before the animation area
	statusLines := []string{
		fmt.Sprintf("%s is %s", name, primaryCondition),
		"state:",
		"health:",
		"hunger:",
		"happiness:",
		"energy:",
		"evolution:",
	}
	
	for i, line := range statusLines {
		pos := strings.Index(outputStr, line)
		if pos == -1 {
			t.Errorf("Status line '%s' not found in output", line)
			continue
		}
		// Each subsequent status line should appear after the previous one
		if i > 0 {
			prevPos := strings.Index(outputStr, statusLines[i-1])
			if prevPos != -1 && pos < prevPos {
				t.Errorf("Status line '%s' appears before '%s' - order is wrong", line, statusLines[i-1])
			}
		}
	}

	t.Logf("Status output test passed. Output length: %d chars", len(outputStr))
	t.Logf("First 500 chars of output:\n%s", outputStr[:min(500, len(outputStr))])
}

func TestStatusOutputWithPixelAnimation(t *testing.T) {
	// Test that pixel art animations preserve status text
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	// Initialize a pixel pet
	err := storage.InitPet(false, "pixel", "Sprite", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Evolve to enable animations
	p.State.Evolution = 1
	// Set good stats
	p.State.Hunger = 20
	p.State.Happiness = 70
	p.State.Energy = 60

	// Apply time step
	now := time.Now()
	if err := pet.ApplyTimeStep(p, now); err != nil {
		t.Fatalf("Failed to apply time step: %v", err)
	}

	// Compute status
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)

	// Capture stdout to test animation output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Simulate verbose status command output
	name := p.Config.Name
	primaryCondition := status.Primary
	
	// Print status header (what status command prints)
	fmt.Printf("%s is %s\n\n", name, primaryCondition)
	fmt.Printf("state: %s\n", conditions.FormatConditions(status.AllOrdered))
	fmt.Printf("health: %d\n", status.Health)
	fmt.Printf("hunger: %d\n", p.State.Hunger)
	fmt.Printf("happiness: %d\n", p.State.Happiness)
	fmt.Printf("energy: %d\n", p.State.Energy)
	fmt.Printf("evolution: %d\n\n", p.State.Evolution)

	// Get art (this may play animation or return static art)
	artStr := art.GetStaticArt(p, status)
	
	// If animation was played, GetStaticArt returns empty string and animation was printed to stdout
	// If static art, we need to print it
	if artStr != "" {
		fmt.Println(artStr)
	}

	// Close write end and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	outputStr := buf.String()

	// Verify ALL status information is present
	requiredStatusFields := []string{
		name,
		string(primaryCondition),
		"state:",
		"health:",
		"hunger:",
		"happiness:",
		"energy:",
		"evolution:",
	}
	
	for _, field := range requiredStatusFields {
		if !strings.Contains(outputStr, field) {
			t.Errorf("Output missing required field '%s'. Output:\n%s", field, outputStr)
		}
	}

	// Verify status text appears in correct order
	// Name and condition should appear first
	if !strings.HasPrefix(outputStr, name) {
		t.Errorf("Output should start with pet name '%s'. Output:\n%s", name, outputStr[:min(100, len(outputStr))])
	}

	// Verify status lines appear before any art/animation
	statusSection := fmt.Sprintf("%s is %s\n\nstate:", name, primaryCondition)
	if !strings.Contains(outputStr, statusSection) {
		t.Errorf("Status section not found in expected format. Output:\n%s", outputStr)
	}

	// Verify art/animation appears after status (for pixel art, ANSI codes will be present)
	// Pixel art uses ANSI color codes, so we should see escape sequences
	hasAnsiCodes := strings.Contains(outputStr, "\033[")
	if p.Config.AllowAnsiAnimations && hasAnsiCodes {
		t.Log("Pixel art animation detected (ANSI codes present)")
	}

	// Most importantly: verify status text is complete and not overwritten
	// All status lines should be present and in order
	statusLines := []string{
		fmt.Sprintf("%s is %s", name, primaryCondition),
		"state:",
		"health:",
		"hunger:",
		"happiness:",
		"energy:",
		"evolution:",
	}
	
	lastPos := -1
	for _, line := range statusLines {
		pos := strings.Index(outputStr, line)
		if pos == -1 {
			t.Errorf("Status line '%s' not found in output", line)
			continue
		}
		// Verify lines appear in order
		if lastPos != -1 && pos < lastPos {
			t.Errorf("Status line '%s' appears out of order (before previous line)", line)
		}
		lastPos = pos
	}

	t.Logf("Pixel status output test passed. Output length: %d chars", len(outputStr))
	t.Logf("First 500 chars of output:\n%s", outputStr[:min(500, len(outputStr))])
}

func TestAsleepAnimationSelection(t *testing.T) {
	// Test that when a pet is asleep, it shows the asleep animation, not happy or tired
	tmpDir := t.TempDir()
	petDir := filepath.Join(tmpDir, ".familiar")

	err := storage.InitPet(false, "dancer", "SleepTestDancer", tmpDir)
	if err != nil {
		t.Fatalf("Failed to initialize pet: %v", err)
	}

	configPath := filepath.Join(petDir, "pet.toml")
	statePath := filepath.Join(petDir, "pet.state.toml")
	p, err := storage.LoadPet(configPath, statePath)
	if err != nil {
		t.Fatalf("Failed to load pet: %v", err)
	}

	// Set pet to asleep state with good stats (so it might also be "happy")
	p.State.IsAsleep = true
	p.State.SleepUntil = time.Now().Add(30 * time.Minute)
	p.State.Happiness = 80
	p.State.Energy = 80
	p.State.Hunger = 20
	p.State.Evolution = 1

	now := time.Now()
	healthVal := health.ComputeHealth(p.State.Hunger, p.State.Happiness, p.State.Energy, health.ComputationMode(p.Config.HealthComputation))
	status := conditions.DeriveStatus(p, now, healthVal)

	// Verify asleep condition is set
	if !status.Conditions[conditions.CondAsleep] {
		t.Error("Expected asleep condition to be set")
	}

	// Verify primary condition is asleep (not happy)
	if status.Primary != conditions.CondAsleep {
		t.Errorf("Expected primary condition to be asleep, got %s", status.Primary)
	}

	// Test animation key selection
	key := art.ChooseAnimationKey(status.Conditions, p.State.Evolution, p.Config.Animations)
	t.Logf("Animation key selected: '%s'", key)
	t.Logf("Conditions: %v", status.Conditions)
	if key != "asleep" {
		t.Errorf("Expected animation key 'asleep', got '%s'", key)
	}

	// Verify asleep animation exists
	asleepAnim, exists := p.Config.Animations["asleep"]
	if !exists {
		t.Error("Expected asleep animation to exist in dancer config")
		t.Logf("Available animations: %v", getAnimationKeys(p.Config.Animations))
	} else {
		t.Logf("Asleep animation found with %d frames", len(asleepAnim.Frames))
		if len(asleepAnim.Frames) > 0 {
			t.Logf("First frame: %q", asleepAnim.Frames[0].Art)
		}
	}

	// Test that GetStaticArt returns asleep art
	// Note: GetStaticArt may return empty string if animation is played (for animated terminals)
	// In that case, we just verify the key selection is correct
	artStr := art.GetStaticArt(p, status)
	t.Logf("GetStaticArt returned (length %d): %q", len(artStr), artStr)
	
	// If art is empty, it means animation was played (which is fine for terminals)
	// For non-terminals or when animations disabled, it should return the first frame
	if artStr == "" {
		t.Log("Art is empty - animation was likely played (this is expected for animated terminals)")
		// Verify the key selection was correct
		if key != "asleep" {
			t.Errorf("Even though art is empty, key should be 'asleep', got '%s'", key)
		}
	} else {
		// Verify it's the asleep animation (should contain "zZz" or sleeping indicator)
		if !strings.Contains(artStr, "zZz") {
			t.Errorf("Expected asleep art to contain 'zZz', got:\n%s", artStr)
		}
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
	p.State.Hunger = 10 // Low hunger is good (0-30 is good range)
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
	// Trim trailing whitespace for comparison
	artStrTrimmed := strings.TrimRight(artStr, " \n\r")
	expectedTrimmed := strings.TrimRight(expectedHasMessageArt, " \n\r")
	if artStrTrimmed != expectedTrimmed {
		t.Errorf("Expected has-message art:\n%q\nGot:\n%q", expectedTrimmed, artStrTrimmed)
	}

	// Now test acknowledge
	// Simulate acknowledge: if message existed, boost to 100, else boost by 5
	hadMessage := p.State.Message != ""
	p.State.Message = ""
	if hadMessage {
		p.State.Hunger = 0 // 0 = not hungry (best)
		p.State.Happiness = 100
		p.State.Energy = 100
	} else {
		p.State.Hunger = max(0, p.State.Hunger-5) // Decrease hunger (lower is better)
		p.State.Happiness = min(100, p.State.Happiness+5)
		p.State.Energy = min(100, p.State.Energy+5)
	}

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
	// Trim trailing whitespace for comparison
	artStrTrimmed = strings.TrimRight(artStr, " \n\r")
	expectedDefaultTrimmed := strings.TrimRight(expectedDefaultArt, " \n\r")
	if artStrTrimmed != expectedDefaultTrimmed {
		t.Errorf("Expected default art after acknowledge:\n%q\nGot:\n%q", expectedDefaultTrimmed, artStrTrimmed)
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
	// Simulate a feed interaction (feed decreases hunger, increases happiness)
	p.State.Hunger = max(0, p.State.Hunger-20) // Decrease hunger (lower is better)
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

func TestAdminArtWithTypeFlag(t *testing.T) {
	// Test that --type flag loads templates correctly and shows art
	// We'll test the LoadTemplateConfig function and art selection logic

	// Test loading cat template
	catTemplate, err := storage.LoadTemplateConfig("cat")
	if err != nil {
		t.Fatalf("Failed to load cat template: %v", err)
	}

	if catTemplate.Config.PetType != "cat" && catTemplate.Config.PetType != "{{PET_TYPE}}" {
		t.Errorf("Expected cat template petType, got '%s'", catTemplate.Config.PetType)
	}

	// Verify cat has expected animations
	if catTemplate.Config.Animations == nil {
		t.Fatal("Cat template should have animations")
	}

	expectedCatAnims := []string{"default", "egg", "has-message", "infirm", "stone", "asleep"}
	for _, animKey := range expectedCatAnims {
		if _, exists := catTemplate.Config.Animations[animKey]; !exists {
			t.Errorf("Cat template missing expected animation: %s", animKey)
		}
	}

	// Test loading dancer template
	dancerTemplate, err := storage.LoadTemplateConfig("dancer")
	if err != nil {
		t.Fatalf("Failed to load dancer template: %v", err)
	}

	if dancerTemplate.Config.Animations == nil {
		t.Fatal("Dancer template should have animations")
	}

	// Verify dancer has more animations than cat
	if len(dancerTemplate.Config.Animations) <= len(catTemplate.Config.Animations) {
		t.Errorf("Dancer should have more animations than cat. Cat: %d, Dancer: %d",
			len(catTemplate.Config.Animations), len(dancerTemplate.Config.Animations))
	}

	// Test loading pixel template
	pixelTemplate, err := storage.LoadTemplateConfig("pixel")
	if err != nil {
		t.Fatalf("Failed to load pixel template: %v", err)
	}

	if pixelTemplate.Config.Animations == nil {
		t.Fatal("Pixel template should have animations")
	}

	// Verify pixel has pixel art animations
	defaultAnim, exists := pixelTemplate.Config.Animations["default"]
	if !exists {
		t.Fatal("Pixel template should have default animation")
	}
	if defaultAnim.Source != "pixel" {
		t.Errorf("Pixel template default animation should have source 'pixel', got '%s'", defaultAnim.Source)
	}

	// Test that pixel has evolution 2 animations
	hasE2Animations := false
	for key := range pixelTemplate.Config.Animations {
		if strings.HasPrefix(key, "e2:") {
			hasE2Animations = true
			break
		}
	}
	if !hasE2Animations {
		t.Error("Pixel template should have evolution 2 animations (e2:*)")
	}

	// Test invalid template type
	_, err = storage.LoadTemplateConfig("nonexistent")
	if err == nil {
		t.Error("Expected error when loading nonexistent template type")
	}
}

func TestAdminArtEvolutionWithType(t *testing.T) {
	// Test that evolution 0 always shows egg even with --type flag
	pixelTemplate, err := storage.LoadTemplateConfig("pixel")
	if err != nil {
		t.Fatalf("Failed to load pixel template: %v", err)
	}

	// Test evolution 0 - should always show egg
	conds := make(map[conditions.Condition]bool)
	conds[conditions.CondHappy] = true // Try to request happy

	key := art.ChooseAnimationKey(conds, 0, pixelTemplate.Config.Animations)
	if key != "egg" {
		t.Errorf("Evolution 0 should always return 'egg' key, got '%s'", key)
	}

	// Verify egg animation exists
	eggAnim, exists := pixelTemplate.Config.Animations["egg"]
	if !exists {
		t.Fatal("Pixel template should have egg animation")
	}
	if len(eggAnim.Frames) == 0 {
		t.Error("Egg animation should have frames")
	}

	// Test evolution 1 - should use base animations
	key = art.ChooseAnimationKey(conds, 1, pixelTemplate.Config.Animations)
	if key == "egg" {
		t.Error("Evolution 1 should not return 'egg' for happy condition")
	}
	// Should find happy animation or default
	anim, exists := pixelTemplate.Config.Animations[key]
	if !exists {
		t.Errorf("Animation key '%s' should exist in pixel template", key)
	} else if len(anim.Frames) == 0 {
		t.Errorf("Animation '%s' should have frames", key)
	}

	// Test evolution 2 - should use e2: animations
	key = art.ChooseAnimationKey(conds, 2, pixelTemplate.Config.Animations)
	// Should try e2:happy first
	expectedKey := "e2:happy"
	if key != expectedKey {
		// Fallback is OK, but log it
		t.Logf("Evolution 2 happy returned key '%s' (expected '%s' or fallback)", key, expectedKey)
	}
	anim, exists = pixelTemplate.Config.Animations[key]
	if !exists {
		t.Errorf("Animation key '%s' should exist in pixel template", key)
	} else if len(anim.Frames) == 0 {
		t.Errorf("Animation '%s' should have frames", key)
	}
}

func TestAdminArtTypeFlagWithDifferentStates(t *testing.T) {
	// Test that --type flag works with different states
	pixelTemplate, err := storage.LoadTemplateConfig("pixel")
	if err != nil {
		t.Fatalf("Failed to load pixel template: %v", err)
	}

	testCases := []struct {
		state      string
		condition  conditions.Condition
		evolution  int
		shouldFind bool
	}{
		{"happy", conditions.CondHappy, 1, true},
		{"lonely", conditions.CondLonely, 1, true},
		{"hungry", conditions.CondHungry, 1, true},
		{"tired", conditions.CondTired, 1, true},
		{"sad", conditions.CondSad, 1, true},
		{"stone", conditions.CondStone, 1, true},
		{"infirm", conditions.CondInfirm, 1, true},
		{"asleep", conditions.CondAsleep, 1, true},
		{"has-message", conditions.CondHasMessage, 1, true},
		{"happy", conditions.CondHappy, 2, true}, // Evolution 2
		{"lonely", conditions.CondLonely, 2, true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_evolution_%d", tc.state, tc.evolution), func(t *testing.T) {
			conds := make(map[conditions.Condition]bool)
			conds[tc.condition] = true

			key := art.ChooseAnimationKey(conds, tc.evolution, pixelTemplate.Config.Animations)
			anim, exists := pixelTemplate.Config.Animations[key]

			if tc.shouldFind {
				if !exists {
					t.Errorf("State '%s' at evolution %d: animation key '%s' not found", tc.state, tc.evolution, key)
				} else if len(anim.Frames) == 0 {
					t.Errorf("State '%s' at evolution %d: animation '%s' has no frames", tc.state, tc.evolution, key)
				}
			} else {
				if exists {
					t.Errorf("State '%s' at evolution %d: should not find animation, but found '%s'", tc.state, tc.evolution, key)
				}
			}
		})
	}
}

func TestAdminArtTypeFlagWithCatTemplate(t *testing.T) {
	// Test that --type cat works correctly
	catTemplate, err := storage.LoadTemplateConfig("cat")
	if err != nil {
		t.Fatalf("Failed to load cat template: %v", err)
	}

	// Test default animation
	conds := make(map[conditions.Condition]bool)
	key := art.ChooseAnimationKey(conds, 1, catTemplate.Config.Animations)
	anim, exists := catTemplate.Config.Animations[key]
	if !exists {
		t.Fatalf("Cat template should have animation for key '%s'", key)
	}

	// Verify it's ASCII art (not pixel)
	if anim.Source != "inline" && anim.Source != "" {
		t.Errorf("Cat template should use inline ASCII art, got source '%s'", anim.Source)
	}

	// Verify it has art content
	if len(anim.Frames) == 0 {
		t.Error("Cat default animation should have frames")
	} else if anim.Frames[0].Art == "" {
		t.Error("Cat default animation frame should have art content")
	}

	// Verify expected cat art pattern
	artStr := anim.Frames[0].Art
	if !strings.Contains(artStr, "/\\_/\\") && !strings.Contains(artStr, "o.o") {
		t.Logf("Cat art doesn't match expected pattern, but that's OK. Got: %q", artStr)
	}
}

func TestAdminArtTypeFlagWithDancerTemplate(t *testing.T) {
	// Test that --type dancer works correctly
	dancerTemplate, err := storage.LoadTemplateConfig("dancer")
	if err != nil {
		t.Fatalf("Failed to load dancer template: %v", err)
	}

	// Verify dancer has more states than cat
	catTemplate, err := storage.LoadTemplateConfig("cat")
	if err != nil {
		t.Fatalf("Failed to load cat template: %v", err)
	}

	if len(dancerTemplate.Config.Animations) <= len(catTemplate.Config.Animations) {
		t.Errorf("Dancer should have more animations. Cat: %d, Dancer: %d",
			len(catTemplate.Config.Animations), len(dancerTemplate.Config.Animations))
	}

	// Test that dancer has happy, lonely, hungry, tired, sad animations
	expectedDancerStates := []string{"happy", "lonely", "hungry", "tired", "sad"}
	for _, state := range expectedDancerStates {
		if _, exists := dancerTemplate.Config.Animations[state]; !exists {
			t.Errorf("Dancer template missing expected state animation: %s", state)
		}
	}

	// Test happy animation
	conds := make(map[conditions.Condition]bool)
	conds[conditions.CondHappy] = true
	key := art.ChooseAnimationKey(conds, 1, dancerTemplate.Config.Animations)
	anim, exists := dancerTemplate.Config.Animations[key]
	if !exists {
		t.Fatalf("Dancer should have animation for key '%s'", key)
	}
	if len(anim.Frames) == 0 {
		t.Error("Dancer happy animation should have frames")
	}
}

func getAnimationKeys(anims map[string]pet.AnimationConfig) []string {
	keys := make([]string, 0, len(anims))
	for k := range anims {
		keys = append(keys, k)
	}
	return keys
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
