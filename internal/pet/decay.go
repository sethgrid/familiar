package pet

import (
	"time"
)

func ApplyTimeStep(p *Pet, now time.Time) error {
	// Initialize LastChecked if zero
	if p.State.LastChecked.IsZero() {
		p.State.LastChecked = now
		return nil
	}

	elapsedHours := now.Sub(p.State.LastChecked).Hours()

	if !p.Config.DecayEnabled || elapsedHours <= 0 {
		p.State.LastChecked = now
		return nil
	}

	// Determine decay multiplier
	mult := p.Config.DecayRate
	if p.State.IsInfirm {
		mult *= p.Config.InfirmDecayMultiplier
	}
	if p.State.IsStone {
		mult *= p.Config.StoneDecayMultiplier
	}

	// Apply decay
	// Hunger is inverted: decay increases hunger (higher = more hungry)
	hunger := float64(p.State.Hunger) + elapsedHours*p.Config.HungerDecayPerHour*mult
	happiness := float64(p.State.Happiness) - elapsedHours*p.Config.HappinessDecayPerHour*mult
	energy := float64(p.State.Energy) - elapsedHours*p.Config.EnergyDecayPerHour*mult

	// Clamp to [0, 100]
	p.State.Hunger = clamp(int(hunger), 0, 100)
	p.State.Happiness = clamp(int(happiness), 0, 100)
	p.State.Energy = clamp(int(energy), 0, 100)

	// Compute health for stone check
	// Hunger is inverted: lower hunger = better health
	// Convert hunger to a "satisfaction" score: 100 - hunger
	hungerScore := 100 - p.State.Hunger

	var computedHealth int
	switch p.Config.HealthComputation {
	case HealthComputationWeighted:
		computedHealth = int(float64(hungerScore)*0.3 + float64(p.State.Happiness)*0.4 + float64(p.State.Energy)*0.3)
	default: // average
		computedHealth = (hungerScore + p.State.Happiness + p.State.Energy) / 3
	}
	if computedHealth < 0 {
		computedHealth = 0
	}
	if computedHealth > 100 {
		computedHealth = 100
	}

	// Check for stone state
	if computedHealth < p.Config.StoneThreshold && !p.State.IsStone {
		p.State.IsStone = true
	}

	p.State.LastChecked = now
	return nil
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
