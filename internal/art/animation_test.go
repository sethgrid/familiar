package art

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/sethgrid/familiar/internal/pet"
)

func TestAnimationStaysInPlace(t *testing.T) {
	// Create a simple 2-frame animation (same as dancer)
	anim := pet.AnimationConfig{
		Source: "inline",
		FPS:    4,
		Loops:  3,
		Frames: []pet.Frame{
			{Art: "    o\n   /|\\\n   / \\"},
			{Art: "  \\o/\n   |\n  / \\"},
		},
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run animation in a goroutine so we can read output
	done := make(chan bool)
	go func() {
		playAnimation(anim)
		done <- true
	}()

	// Close write end and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	// Wait for animation to complete
	<-done

	t.Logf("Output length: %d bytes", len(output))
	t.Logf("First 300 chars of output:\n%q", output[:min(300, len(output))])

	// Count frame appearances
	frame1Count := strings.Count(output, "    o")
	frame2Count := strings.Count(output, "  \\o/")

	expectedCount := anim.Loops * len(anim.Frames)
	t.Logf("Frame counts - Frame1: %d, Frame2: %d, Expected total frames: %d", frame1Count, frame2Count, expectedCount)

	// Check for cursor movement sequences
	upMoves := strings.Count(output, "\033[")
	t.Logf("Cursor movement sequences: %d", upMoves)
	
	if upMoves < 10 {
		t.Errorf("Expected many cursor movement sequences, got %d", upMoves)
	}

	// The key test: count how many times we see the pattern that suggests jumping
	// If frames are overlaying correctly, we should see them in sequence
	// If jumping, we'll see extra blank lines or frames appearing in wrong positions
	
	// Split by newlines and analyze
	allLines := strings.Split(output, "\n")
	consecutiveEmpty := 0
	maxConsecutiveEmpty := 0
	for _, line := range allLines {
		trimmed := strings.TrimSpace(line)
		// Ignore lines with escape sequences (they're cursor movements)
		if trimmed == "" && !strings.Contains(line, "\033") {
			consecutiveEmpty++
			if consecutiveEmpty > maxConsecutiveEmpty {
				maxConsecutiveEmpty = consecutiveEmpty
			}
		} else {
			consecutiveEmpty = 0
		}
	}
	
	t.Logf("Max consecutive empty lines: %d", maxConsecutiveEmpty)
	
	// If we have more than 2 consecutive empty lines (beyond what's in the frames), something is wrong
	if maxConsecutiveEmpty > 2 {
		t.Errorf("Too many consecutive empty lines (%d), suggests animation is jumping/not overlaying correctly", maxConsecutiveEmpty)
		t.Logf("Sample of problematic area in output (chars 400-600):\n%q", output[min(400, len(output)):min(600, len(output))])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
