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

	// Run animation
	PlayAnimation(anim)

	// Close write end and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()
	
	// Simulate what a terminal would show by processing escape sequences
	// This is a simplified simulation - real terminals are more complex
	simulatedOutput := simulateTerminalOutput(output)

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

	// The key test: verify frames overlay correctly
	// Count how many times each frame appears - should be equal to loops
	expectedPerFrame := anim.Loops
	if frame1Count != expectedPerFrame || frame2Count != expectedPerFrame {
		t.Errorf("Frame count mismatch - Frame1: %d, Frame2: %d, Expected each: %d", 
			frame1Count, frame2Count, expectedPerFrame)
	}
	
	// Check the simulated output - if animation stays in place, we should see
	// frames overlaying, not stacking vertically
	simulatedLines := strings.Split(simulatedOutput, "\n")
	// Count how many lines contain frame content
	frameLines := 0
	for _, line := range simulatedLines {
		if strings.Contains(line, "o") || strings.Contains(line, "|") || strings.Contains(line, "/") || strings.Contains(line, "\\") {
			frameLines++
		}
	}
	// If animation stays in place, we should see approximately maxLines (3) lines with frame content
	// If it's jumping, we'll see many more lines
	t.Logf("Simulated output has %d lines with frame content (expected ~%d if staying in place)", frameLines, 3)
	if frameLines > 10 {
		t.Errorf("Too many lines with frame content (%d) - suggests animation is jumping/not overlaying", frameLines)
		t.Logf("First 500 chars of simulated output:\n%s", simulatedOutput[:min(500, len(simulatedOutput))])
	}

	// Check that we're using cursor movements (not just printing sequentially)
	upMovesCount := strings.Count(output, "\033[")
	if upMovesCount < 10 {
		t.Errorf("Expected many cursor movement sequences, got %d", upMovesCount)
	}

	// Verify the pattern: after each frame print, we should move up
	// Count the pattern: frame content followed by move-up sequence
	frame1Pattern := "    o"
	frame2Pattern := "  \\o/"
	
	// Find all occurrences and check that move-up sequences follow
	frame1Positions := findAllSubstringPositions(output, frame1Pattern)
	frame2Positions := findAllSubstringPositions(output, frame2Pattern)
	
	allFrames := len(frame1Positions) + len(frame2Positions)
	
	t.Logf("Frame1 appears at positions: %v", frame1Positions[:min(3, len(frame1Positions))])
	t.Logf("Frame2 appears at positions: %v", frame2Positions[:min(3, len(frame2Positions))])
	
	// After each frame (except the last), there should be cursor movement
	// This could be move-up sequences (A) or cursor restore (u or 8)
	if allFrames > 0 {
		// Check that we have cursor movements (suggesting we're repositioning)
		moveUpCount := strings.Count(output, "A") // Count of "move up" sequences
		restoreCount := strings.Count(output, "\033[u") + strings.Count(output, "\0338") // Cursor restore
		totalRepositions := moveUpCount + restoreCount
		if totalRepositions < allFrames {
			t.Errorf("Not enough cursor repositioning sequences (%d move-up + %d restore = %d total) for %d frames - suggests frames aren't being repositioned", 
				moveUpCount, restoreCount, totalRepositions, allFrames)
		}
	}
}

func findAllSubstringPositions(s, substr string) []int {
	var positions []int
	start := 0
	for {
		pos := strings.Index(s[start:], substr)
		if pos == -1 {
			break
		}
		positions = append(positions, start+pos)
		start += pos + len(substr)
	}
	return positions
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// simulateTerminalOutput crudely simulates terminal behavior by processing escape sequences
// This is a simplified version - real terminals are much more complex
func simulateTerminalOutput(output string) string {
	lines := []string{""} // Start with one empty line
	currentLine := 0
	currentCol := 0
	
	i := 0
	for i < len(output) {
		if output[i] == '\033' && i+1 < len(output) && output[i+1] == '[' {
			// Parse escape sequence
			i += 2 // Skip \033[
			seq := ""
			for i < len(output) && output[i] >= '0' && output[i] <= '9' {
				seq += string(output[i])
				i++
			}
			cmd := byte(0)
			if i < len(output) {
				cmd = output[i]
				i++
			}
			
			count := 1
			if seq != "" {
				// Parse count (simplified - doesn't handle all cases)
				if len(seq) > 0 {
					count = 0
					for _, c := range seq {
						count = count*10 + int(c-'0')
					}
				}
			}
			
			switch cmd {
			case 'A': // Move up
				currentLine = max(0, currentLine-count)
			case 'B': // Move down
				currentLine += count
				// Extend lines array if needed
				for len(lines) <= currentLine {
					lines = append(lines, "")
				}
			case 'K': // Clear line
				if currentLine < len(lines) {
					lines[currentLine] = strings.Repeat(" ", currentCol) + lines[currentLine][min(currentCol, len(lines[currentLine])):]
				}
			case '2': // Clear entire line (part of \033[2K)
				if i < len(output) && output[i] == 'K' {
					i++
					if currentLine < len(lines) {
						lines[currentLine] = ""
					}
				}
			}
		} else if output[i] == '\n' {
			currentLine++
			currentCol = 0
			// Extend lines array if needed
			for len(lines) <= currentLine {
				lines = append(lines, "")
			}
			i++
		} else {
			// Regular character
			// Extend lines array if needed
			for len(lines) <= currentLine {
				lines = append(lines, "")
			}
			// Extend current line if needed
			for len(lines[currentLine]) <= currentCol {
				lines[currentLine] += " "
			}
			// Replace character at position
			lineBytes := []byte(lines[currentLine])
			if currentCol < len(lineBytes) {
				lineBytes[currentCol] = output[i]
			} else {
				lineBytes = append(lineBytes, output[i])
			}
			lines[currentLine] = string(lineBytes)
			currentCol++
			i++
		}
	}
	
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
