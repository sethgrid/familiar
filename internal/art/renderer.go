package art

import (
	"fmt"
	"strings"

	"github.com/sethgrid/familiar/internal/conditions"
	"github.com/sethgrid/familiar/internal/pet"
)

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
