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
	
	// Save cursor position at the start - this is our anchor point
	fmt.Fprint(os.Stdout, "\033[s")

	for i := 0; i < totalFrames; i++ {
		frameIdx := i % len(anim.Frames)
		trimmedArt := trimmedFrames[frameIdx]
		frame := anim.Frames[frameIdx]
		
		// Use frame-specific duration if available, otherwise use calculated duration
		duration := frameDuration
		if frame.MS > 0 {
			duration = time.Duration(frame.MS) * time.Millisecond
		}

		// Always restore to the saved position (our anchor point)
		fmt.Fprint(os.Stdout, "\033[u")
		
		// Clear maxLines worth of space from the anchor position
		for j := 0; j < maxLines; j++ {
			fmt.Fprint(os.Stdout, "\033[K") // Clear from cursor to end of line
			if j < maxLines-1 {
				fmt.Fprint(os.Stdout, "\033[B") // Move down one line
			}
		}
		
		// Restore to anchor position again to draw
		fmt.Fprint(os.Stdout, "\033[u")

		// Print trimmed frame (no trailing newline)
		// After this, cursor will be at the end of the last line of the frame
		fmt.Fprint(os.Stdout, trimmedArt)
		os.Stdout.Sync() // Flush output
		
		// Add delay between frames (except for last frame)
		if i < totalFrames-1 {
			time.Sleep(duration)
		}
	}
	
	// After animation, move cursor to below the art
	// We're currently at the end of the last line of the last frame
	// Just move to the next line
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
