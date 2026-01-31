package art

import (
	"fmt"
	"os"
	"strconv"
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

// ChooseAnimationKey selects the appropriate animation key based on conditions and evolution
func ChooseAnimationKey(conds map[conditions.Condition]bool, evolution int, animations map[string]pet.AnimationConfig) string {
	// If evolution is 0 and no special conditions, return "egg"
	if evolution == 0 {
		hasSpecialCondition := conds[conditions.CondHasMessage] ||
			conds[conditions.CondStone] ||
			conds[conditions.CondAsleep] ||
			conds[conditions.CondInfirm]
		if !hasSpecialCondition {
			// Check if egg animation exists
			if _, exists := animations["egg"]; exists {
				return "egg"
			}
		}
	}

	// Special handling: if asleep, prioritize it strongly
	// Asleep should override other lower-priority conditions like tired, happy, etc.
	if conds[conditions.CondAsleep] {
		// Try asleep animation first (with evolution prefix if applicable)
		if evolution > 0 {
			evolKey := fmt.Sprintf("e%d:asleep", evolution)
			if _, exists := animations[evolKey]; exists {
				return evolKey
			}
		}
		if _, exists := animations["asleep"]; exists {
			return "asleep"
		}
	}

	// Build key from conditions (check in priority order)
	var parts []string

	if conds[conditions.CondHasMessage] {
		parts = append(parts, "has-message")
	}
	if conds[conditions.CondStone] {
		parts = append(parts, "stone")
	}
	if conds[conditions.CondAsleep] {
		parts = append(parts, "asleep")
	}
	if conds[conditions.CondInfirm] {
		parts = append(parts, "infirm")
	}
	if conds[conditions.CondLonely] {
		parts = append(parts, "lonely")
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
		// Check if this is pixel art
		if anim.Source == "pixel" {
			// For pixel art, render the first frame
			if len(anim.Frames) > 0 {
				rendered := RenderPixelArt(anim.Frames[0])
				// If animation has multiple frames and animations are enabled, play animation
				if len(anim.Frames) > 1 && p.Config.AllowAnsiAnimations && isTerminal() {
					PlayPixelAnimation(anim)
					// Return empty string - animation already displayed the final frame
					return ""
				}
				return rendered
			}
		} else {
			// If animation has multiple frames and animations are enabled, play animation
			if len(anim.Frames) > 1 && p.Config.AllowAnsiAnimations && isTerminal() {
				PlayAnimation(anim)
				// Return empty string - animation already displayed the final frame
				return ""
			}
			// Otherwise return first frame
			return anim.Frames[0].Art
		}
	}

	// Fallback to hardcoded art based on state (check in priority order)
	if status.Conditions[conditions.CondHasMessage] {
		return getHasMessageCat()
	}
	if status.Conditions[conditions.CondStone] {
		return getStoneCat()
	}
	if status.Conditions[conditions.CondAsleep] {
		// For asleep, try to get from animation config first, otherwise fallback
		if anim, exists := p.Config.Animations["asleep"]; exists && len(anim.Frames) > 0 {
			return anim.Frames[0].Art
		}
		// No hardcoded asleep art, use default
		return getDefaultCat()
	}
	if status.Conditions[conditions.CondInfirm] {
		return getInfirmCat()
	}
	if p.State.Evolution == 0 {
		return getEggCat()
	}
	return getDefaultCat()
}

// PlayAnimation plays an animation by cycling through frames directly to stdout
func PlayAnimation(anim pet.AnimationConfig) {
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

	// Print buffer lines BELOW to ensure animation has room without scrolling
	// This prevents the animation from overwriting status text when terminal is near bottom
	bufferLines := maxLines + 10 // Extra buffer below animation
	for j := 0; j < bufferLines; j++ {
		fmt.Fprint(os.Stdout, "\n")
	}
	// Move back up to animation start position (we're now at the start of the buffer area)
	if bufferLines > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", bufferLines)
	}

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

// isTransparentPixel checks if a pixel value represents a transparent pixel
// Accepts: "", "transparent", all spaces, or all hash characters (for alignment)
func isTransparentPixel(pixel string) bool {
	if pixel == "" || pixel == "transparent" {
		return true
	}

	// Check if all characters are spaces
	allSpaces := true
	for _, r := range pixel {
		if r != ' ' {
			allSpaces = false
			break
		}
	}
	if allSpaces {
		return true
	}

	// Check if all characters are hashes (for alignment)
	allHashes := true
	for _, r := range pixel {
		if r != '#' {
			allHashes = false
			break
		}
	}
	if allHashes {
		return true
	}

	return false
}

// RenderPixelArt renders a pixel art frame to a string using ANSI color codes
// Uses half-block characters (▀ ▄) for 2 pixels per character cell
func RenderPixelArt(frame pet.Frame) string {
	if len(frame.Pixels) == 0 {
		return ""
	}

	var result strings.Builder
	height := len(frame.Pixels)

	// Process pairs of rows to use half-block characters
	for y := 0; y < height; y += 2 {
		row1 := frame.Pixels[y]
		var row2 []string
		if y+1 < height {
			row2 = frame.Pixels[y+1]
		}

		width := len(row1)
		if row2 != nil && len(row2) > width {
			width = len(row2)
		}

		for x := 0; x < width; x++ {
			topColor := ""
			if x < len(row1) {
				topColor = row1[x]
			}

			bottomColor := ""
			if row2 != nil && x < len(row2) {
				bottomColor = row2[x]
			}

			// Normalize transparent pixels
			topTransparent := isTransparentPixel(topColor)
			bottomTransparent := isTransparentPixel(bottomColor)

			// Render using half-block character
			if topTransparent && bottomTransparent {
				// Both transparent - use space
				result.WriteString(" ")
			} else if topTransparent {
				// Only bottom - use lower half block
				r, g, b := hexToRGB(bottomColor)
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▄\033[0m", r, g, b))
			} else if bottomTransparent {
				// Only top - use upper half block
				r, g, b := hexToRGB(topColor)
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▀\033[0m", r, g, b))
			} else if topColor == bottomColor {
				// Same color - use full block
				r, g, b := hexToRGB(topColor)
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm█\033[0m", r, g, b))
			} else {
				// Different colors - use upper half block with foreground and background
				topR, topG, topB := hexToRGB(topColor)
				bottomR, bottomG, bottomB := hexToRGB(bottomColor)
				// Set both foreground (top) and background (bottom) colors
				result.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm\033[48;2;%d;%d;%dm▀\033[0m",
					topR, topG, topB, bottomR, bottomG, bottomB))
			}
		}

		if y+2 < height {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// PlayPixelAnimation plays a pixel art animation
func PlayPixelAnimation(anim pet.AnimationConfig) {
	if len(anim.Frames) == 0 || len(anim.Frames) == 1 {
		return
	}

	// Calculate frame duration
	fps := anim.FPS
	if fps <= 0 {
		fps = 1
	}
	frameDuration := time.Second / time.Duration(fps)

	// Determine number of loops
	loops := anim.Loops
	if loops <= 0 {
		loops = 3
	}
	if loops > 10 {
		loops = 10
	}

	totalFrames := len(anim.Frames) * loops

	// Render all frames to strings
	renderedFrames := make([]string, len(anim.Frames))
	frameHeights := make([]int, len(anim.Frames))
	maxLines := 0
	for i, frame := range anim.Frames {
		rendered := RenderPixelArt(frame)
		renderedFrames[i] = strings.TrimRight(rendered, "\n\r")
		lines := strings.Count(renderedFrames[i], "\n") + 1
		frameHeights[i] = lines
		if lines > maxLines {
			maxLines = lines
		}
	}

	// Hide cursor
	fmt.Fprint(os.Stdout, "\033[?25l")

	// Print buffer lines BELOW to ensure animation has room without scrolling
	// This prevents the animation from overwriting status text when terminal is near bottom
	bufferLines := maxLines + 10 // Extra buffer below animation
	for j := 0; j < bufferLines; j++ {
		fmt.Fprint(os.Stdout, "\n")
	}
	// Move back up to animation start position (we're now at the start of the buffer area)
	if bufferLines > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", bufferLines)
	}

	for i := 0; i < totalFrames; i++ {
		frameIdx := i % len(anim.Frames)
		renderedArt := renderedFrames[frameIdx]
		frame := anim.Frames[frameIdx]

		// Use frame-specific duration if available
		duration := frameDuration
		if frame.MS > 0 {
			duration = time.Duration(frame.MS) * time.Millisecond
		}

		// Move back to start for frames after the first
		if i > 0 {
			prevFrameIdx := (i - 1) % len(anim.Frames)
			prevFrameHeight := frameHeights[prevFrameIdx]
			fmt.Fprint(os.Stdout, "\r")
			if prevFrameHeight > 1 {
				fmt.Fprintf(os.Stdout, "\033[%dA", prevFrameHeight-1)
			}
		}

		// Clear space
		for j := 0; j < maxLines; j++ {
			fmt.Fprint(os.Stdout, "\033[2K")
			if j < maxLines-1 {
				fmt.Fprint(os.Stdout, "\033[1B")
			}
		}

		if maxLines > 1 {
			fmt.Fprintf(os.Stdout, "\033[%dA", maxLines-1)
		}

		// Print the frame
		fmt.Fprint(os.Stdout, renderedArt)
		os.Stdout.Sync()

		// Add delay between frames
		if i < totalFrames-1 {
			time.Sleep(duration)
		}
	}

	// Move cursor below and show it
	fmt.Fprint(os.Stdout, "\n")
	fmt.Fprint(os.Stdout, "\033[?25h")
	os.Stdout.Sync()
}

// colorCode converts a hex color to ANSI 24-bit color code
func colorCode(hex string) string {
	if hex == "" || hex == "transparent" {
		return ""
	}

	// Remove # if present
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}

	// Parse hex
	r, g, b := hexToRGB(hex)
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hex string) (int, int, int) {
	// Remove # if present
	if strings.HasPrefix(hex, "#") {
		hex = hex[1:]
	}

	// Handle 3-digit hex
	if len(hex) == 3 {
		hex = string(hex[0]) + string(hex[0]) + string(hex[1]) + string(hex[1]) + string(hex[2]) + string(hex[2])
	}

	// Parse as 6-digit hex
	if len(hex) == 6 {
		r, _ := strconv.ParseInt(hex[0:2], 16, 64)
		g, _ := strconv.ParseInt(hex[2:4], 16, 64)
		b, _ := strconv.ParseInt(hex[4:6], 16, 64)
		return int(r), int(g), int(b)
	}

	// Default to black if invalid
	return 0, 0, 0
}
