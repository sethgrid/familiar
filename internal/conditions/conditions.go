package conditions

import (
	"time"

	"github.com/sethgrid/familiar/internal/pet"
)

type Condition string

const (
	CondHasMessage Condition = "has-message"
	CondStone      Condition = "stone"
	CondInfirm     Condition = "infirm"
	CondLonely     Condition = "lonely"
	CondHungry     Condition = "hungry"
	CondTired      Condition = "tired"
	CondSad        Condition = "sad"
	CondHappy      Condition = "happy"
)

type DerivedStatus struct {
	Health     int
	Conditions map[Condition]bool
	Primary    Condition
	AllOrdered []Condition
}

func DeriveStatus(p *pet.Pet, now time.Time, health int) DerivedStatus {
	conds := make(map[Condition]bool)
	var allOrdered []Condition

	// Priority 1: has-message
	if p.State.Message != "" {
		conds[CondHasMessage] = true
		allOrdered = append(allOrdered, CondHasMessage)
	}

	// Priority 2: stone
	if p.State.IsStone || health < p.Config.StoneThreshold {
		conds[CondStone] = true
		if !contains(allOrdered, CondStone) {
			allOrdered = append(allOrdered, CondStone)
		}
	}

	// Priority 3: infirm
	if p.State.IsInfirm || (health < 30 && p.Config.InfirmEnabled) {
		conds[CondInfirm] = true
		if !contains(allOrdered, CondInfirm) {
			allOrdered = append(allOrdered, CondInfirm)
		}
	}

	// Priority 4: hungry
	if p.State.Hunger < 50 {
		conds[CondHungry] = true
		if !contains(allOrdered, CondHungry) {
			allOrdered = append(allOrdered, CondHungry)
		}
	}

	// Priority 5: lonely
	if isLonely(p, now) {
		conds[CondLonely] = true
		if !contains(allOrdered, CondLonely) {
			allOrdered = append(allOrdered, CondLonely)
		}
	}

	// Priority 6: tired
	if p.State.Energy < 40 {
		conds[CondTired] = true
		if !contains(allOrdered, CondTired) {
			allOrdered = append(allOrdered, CondTired)
		}
	}

	// Priority 7: sad
	if p.State.Happiness < 50 {
		conds[CondSad] = true
		if !contains(allOrdered, CondSad) {
			allOrdered = append(allOrdered, CondSad)
		}
	}

	// Priority 8: happy (default if no other conditions and attributes high)
	if len(allOrdered) == 0 || (p.State.Hunger > 70 && p.State.Happiness > 70 && p.State.Energy > 70) {
		conds[CondHappy] = true
		if !contains(allOrdered, CondHappy) {
			allOrdered = append(allOrdered, CondHappy)
		}
	}

	// Determine primary condition (first in priority order)
	primary := CondHappy
	if len(allOrdered) > 0 {
		primary = allOrdered[0]
	}

	return DerivedStatus{
		Health:     health,
		Conditions: conds,
		Primary:    primary,
		AllOrdered: allOrdered,
	}
}

func isLonely(p *pet.Pet, now time.Time) bool {
	threshold := p.Config.InteractionThreshold
	if threshold == 0 {
		threshold = 3 // default
	}

	count := 0
	cutoff := now.Add(-24 * time.Hour)

	// Count visits
	for _, visit := range p.State.LastVisits {
		if visit.Time.After(cutoff) {
			count++
		}
	}

	// Count feeds
	for _, feed := range p.State.LastFeeds {
		if feed.Time.After(cutoff) {
			count++
		}
	}

	// Count plays
	for _, play := range p.State.LastPlays {
		if play.Time.After(cutoff) {
			count++
		}
	}

	return count < threshold
}

func contains(slice []Condition, item Condition) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// FormatConditions formats a slice of conditions into a comma-separated string.
// Returns "happy" if the slice is empty.
// Special handling: if "stone" is present, all other conditions are ignored
// except "has-message", which is appended as "and has a message".
func FormatConditions(conds []Condition) string {
	if len(conds) == 0 {
		return "happy"
	}

	// Check if stone is present
	hasStone := false
	hasMessage := false
	for _, c := range conds {
		if c == CondStone {
			hasStone = true
		}
		if c == CondHasMessage {
			hasMessage = true
		}
	}

	// If stone is present, only show stone and optionally has-message
	if hasStone {
		if hasMessage {
			return "stone and has a message"
		}
		return "stone"
	}

	// Normal formatting for non-stone conditions
	var parts []string
	for _, c := range conds {
		if c == CondHasMessage {
			continue
		}
		parts = append(parts, string(c))
	}

	// Handle case where only has-message was present and was filtered out of parts
	if len(parts) == 0 {
		if hasMessage {
			return "has a message"
		}
		return "happy"
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += ", " + parts[i]
	}
	if hasMessage {
		result += " and has a message"
	}
	return result
}
