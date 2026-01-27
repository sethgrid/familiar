package art

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sethgrid/familiar/internal/conditions"
	"github.com/sethgrid/familiar/internal/pet"
)

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

func ChooseAnimationKey(conds map[conditions.Condition]bool, evolution int, animations map[string]pet.AnimationConfig) string {
	// If evolution is 0 and no special conditions, return "egg"
	if evolution == 0 {
		hasSpecialCondition := conds[conditions.CondHasMessage] || 
			conds[conditions.CondStone] || 
			conds[conditions.CondInfirm]
		if !hasSpecialCondition {
			// Check if egg animation exists
			if _, exists := animations["egg"]; exists {
				return "egg"
			}
		}
	}

	// Build key from conditions
	var parts []string
	
	if conds[conditions.CondHasMessage] {
		parts = append(parts, "has-message")
	}
	if conds[conditions.CondStone] {
		parts = append(parts, "stone")
	}
	if conds[conditions.CondInfirm] {
		parts = append(parts, "infirm")
	}
	if conds[conditions.CondHungry] {
		parts = append(parts, "hungry")
	}
	if conds[conditions.CondTired] {
		parts = append(parts, "tired")
	}
	if conds[conditions.CondSad] {
		parts = append(parts, "sad")
	}

	key := strings.Join(parts, "+")
	if key == "" {
		key = "default"
	}

	// Try evolution-specific key first
	if evolution > 0 {
		evolKey := fmt.Sprintf("e%d:%s", evolution, key)
		if _, exists := animations[evolKey]; exists {
			return evolKey
		}
	}

	// Fallback to base key
	if _, exists := animations[key]; exists {
		return key
	}

	// Progressive fallback
	if len(parts) > 1 {
		// Try with fewer conditions
		for i := len(parts) - 1; i > 0; i-- {
			fallbackKey := strings.Join(parts[:i], "+")
			if _, exists := animations[fallbackKey]; exists {
				return fallbackKey
			}
		}
	}

	return "default"
}

func GetStaticArt(p *pet.Pet, status conditions.DerivedStatus) string {
	key := ChooseAnimationKey(status.Conditions, p.State.Evolution, p.Config.Animations)
	
	// Try to get animation from config
	if anim, exists := p.Config.Animations[key]; exists && len(anim.Frames) > 0 {
		// If animation has multiple frames and animations are enabled, play animation
		if len(anim.Frames) > 1 && p.Config.AllowAnsiAnimations && isTerminal() {
			playAnimation(anim)
			// Return empty string - animation already displayed the final frame
			return ""
		}
		// Otherwise return first frame
		return anim.Frames[0].Art
	}
	
	// Fallback to hardcoded art based on state
	if status.Conditions[conditions.CondStone] {
		return getStoneCat()
	}
	if status.Conditions[conditions.CondInfirm] {
		return getInfirmCat()
	}
	if status.Conditions[conditions.CondHasMessage] {
		return getHasMessageCat()
	}
	if p.State.Evolution == 0 {
		return getEggCat()
	}
	return getDefaultCat()
}

// playAnimation plays an animation by cycling through frames directly to stdout
func playAnimation(anim pet.AnimationConfig) {
	if len(anim.Frames) == 0 || len(anim.Frames) == 1 {
		return
	}

	// Calculate frame duration
	fps := anim.FPS
	if fps <= 0 {
		fps = 1
	}
	frameDuration := time.Second / time.Duration(fps)

	// Determine number of loops (limit to reasonable number)
	loops := anim.Loops
	if loops <= 0 {
		loops = 3 // Default to 3 loops for demo
	}
	if loops > 10 {
		loops = 10 // Cap at 10 loops max
	}

	totalFrames := len(anim.Frames) * loops

	// Trim trailing newlines from frames and calculate their heights
	trimmedFrames := make([]string, len(anim.Frames))
	frameHeights := make([]int, len(anim.Frames))
	maxLines := 0
	for i, frame := range anim.Frames {
		trimmed := strings.TrimRight(frame.Art, "\n\r")
		trimmedFrames[i] = trimmed
		lines := strings.Count(trimmed, "\n") + 1
		frameHeights[i] = lines
		if lines > maxLines {
			maxLines = lines
		}
	}

	// Hide cursor
	fmt.Fprint(os.Stdout, "\033[?25l")

	for i := 0; i < totalFrames; i++ {
		frameIdx := i % len(anim.Frames)
		trimmedArt := trimmedFrames[frameIdx]
		frame := anim.Frames[frameIdx]
		
		// Use frame-specific duration if available, otherwise use calculated duration
		duration := frameDuration
		if frame.MS > 0 {
			duration = time.Duration(frame.MS) * time.Millisecond
		}

		// For frames after the first, move back to start
		// After printing a trimmed frame (no trailing newline), cursor is at the END of the last line
		// To get back to the start of the first line:
		// 1. Move to start of current line (\r) - now at start of last line
		// 2. Move up by (frameHeight - 1) to get to start of first line
		if i > 0 {
			prevFrameIdx := (i - 1) % len(anim.Frames)
			prevFrameHeight := frameHeights[prevFrameIdx]
			// Move to start of current line (we're at end of last line of previous frame)
			fmt.Fprint(os.Stdout, "\r")
			// Move up by (frameHeight - 1) to get to start of first line
			// If frame is 3 lines, we're at start of line 3, move up 2 to get to line 1
			if prevFrameHeight > 1 {
				fmt.Fprintf(os.Stdout, "\033[%dA", prevFrameHeight-1)
			}
		}

		// Clear exactly maxLines worth of space from current position
		// We're at the start position (line 1), so clear downward
		for j := 0; j < maxLines; j++ {
			fmt.Fprint(os.Stdout, "\033[2K") // Clear entire line
			if j < maxLines-1 {
				fmt.Fprint(os.Stdout, "\033[1B") // Move down exactly 1 line
			}
		}
		
		// Move back up to start - we moved down (maxLines-1) times, so move back up that much
		// After this, we're back at the start position (line 1)
		if maxLines > 1 {
			fmt.Fprintf(os.Stdout, "\033[%dA", maxLines-1)
		}

		// Print the frame
		// After printing (no trailing newline), cursor will be at the END of the last line
		fmt.Fprint(os.Stdout, trimmedArt)
		os.Stdout.Sync()
		
		// Add delay between frames (except for last frame)
		if i < totalFrames-1 {
			time.Sleep(duration)
		}
	}
	
	// After animation, move cursor to below the art
	// We're currently at the start of the line AFTER the last frame
	// The last frame was maxLines tall, so we're already positioned correctly
	// Just ensure cursor is visible and add a newline for spacing
	fmt.Fprint(os.Stdout, "\n")
	
	// Show cursor
	fmt.Fprint(os.Stdout, "\033[?25h")
	os.Stdout.Sync()
}

func getDefaultCat() string {
	return ` /\_/\ 
( o.o )
 > ^ <`
}

func getInfirmCat() string {
	return ` /\_/\ 
( x.x )
 > ^ <`
}

func getStoneCat() string {
	return ` /\_/\ 
( +.+ )
 > ^ <`
}

func getEggCat() string {
	return `  ___  
 /  . . \ 
 \___/`
}

func getHasMessageCat() string {
	return ` /\_/\ 
( o.o )
 > ^ <*`
}
