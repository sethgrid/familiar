package conditions

import (
	"time"

	"github.com/qwert/promptfamiliar/internal/pet"
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

	// Priority 4: lonely
	if isLonely(p, now) {
		conds[CondLonely] = true
		if !contains(allOrdered, CondLonely) {
			allOrdered = append(allOrdered, CondLonely)
		}
	}

	// Priority 5: hungry
	if p.State.Hunger < 50 {
		conds[CondHungry] = true
		if !contains(allOrdered, CondHungry) {
			allOrdered = append(allOrdered, CondHungry)
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
