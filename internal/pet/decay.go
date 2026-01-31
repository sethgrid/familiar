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

	// Track if pet was asleep during this time period
	// We need to check this BEFORE clearing the sleep state
	wasAsleep := p.State.IsAsleep
	var sleepElapsedHours float64
	var sleepExpired bool

	if wasAsleep && !p.State.SleepUntil.IsZero() {
		if now.After(p.State.SleepUntil) {
			// Sleep has expired - calculate how much time was spent asleep
			sleepElapsedHours = p.State.SleepUntil.Sub(p.State.LastChecked).Hours()
			if sleepElapsedHours < 0 {
				sleepElapsedHours = 0
			}
			if sleepElapsedHours > elapsedHours {
				sleepElapsedHours = elapsedHours
			}
			sleepExpired = true
		} else {
			// Still asleep - all elapsed time was sleep time
			sleepElapsedHours = elapsedHours
		}
	}

	// Clear sleep state if expired (but we've already captured the sleep time above)
	if sleepExpired {
		p.State.IsAsleep = false
		p.State.SleepUntil = time.Time{}
		p.State.SleepAttempts = 0
	}

	// Determine decay multiplier
	mult := p.Config.DecayRate
	if p.State.IsInfirm {
		mult *= p.Config.InfirmDecayMultiplier
	}
	if p.State.IsStone {
		mult *= p.Config.StoneDecayMultiplier
	}

	var hunger, happiness, energy float64

	if wasAsleep && sleepElapsedHours > 0 {
		// During sleep: restore stats, slow hunger growth
		// Hunger grows much slower (10% of normal rate) during sleep time
		hungerDuringSleep := sleepElapsedHours * p.Config.HungerDecayPerHour * mult * 0.1

		// Calculate remaining awake time (if any)
		awakeElapsedHours := elapsedHours - sleepElapsedHours
		hungerDuringAwake := awakeElapsedHours * p.Config.HungerDecayPerHour * mult

		hunger = float64(p.State.Hunger) + hungerDuringSleep + hungerDuringAwake

		// Happiness and energy increase during sleep
		// Calculate restoration rate based on sleep duration
		sleepDuration := p.Config.SleepDuration
		if sleepDuration == 0 {
			sleepDuration = 30 * time.Minute // Default 30 minutes
		}
		sleepHours := sleepDuration.Hours()

		// Restore at a rate that would get from 0 to 100 over the sleep duration
		// This ensures full restoration if slept the full cycle
		var restoreRatePerHour float64
		if sleepHours > 0 {
			restoreRatePerHour = 100.0 / sleepHours // Points per hour to reach 100
		} else {
			restoreRatePerHour = 200.0 // Default high rate if duration is invalid
		}

		// Apply sleep restoration for the time spent asleep
		happinessFromSleep := sleepElapsedHours * restoreRatePerHour

		// Apply normal decay for awake time (if any)
		happinessFromAwake := -awakeElapsedHours * p.Config.HappinessDecayPerHour * mult
		energyFromAwake := -awakeElapsedHours * p.Config.EnergyDecayPerHour * mult

		happiness = float64(p.State.Happiness) + happinessFromSleep + happinessFromAwake
		energy = float64(p.State.Energy) + sleepElapsedHours*restoreRatePerHour + energyFromAwake
	} else {
		// Normal decay
		// Hunger is inverted: decay increases hunger (higher = more hungry)
		hunger = float64(p.State.Hunger) + elapsedHours*p.Config.HungerDecayPerHour*mult
		happiness = float64(p.State.Happiness) - elapsedHours*p.Config.HappinessDecayPerHour*mult
		energy = float64(p.State.Energy) - elapsedHours*p.Config.EnergyDecayPerHour*mult
	}

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

	// Automatic infirm removal: if health is high enough remove infirm
	if p.State.IsInfirm && computedHealth >= 50 {
		p.State.IsInfirm = false
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
