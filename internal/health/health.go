package health

type ComputationMode string

const (
	ComputationAverage ComputationMode = "average"
	ComputationWeighted ComputationMode = "weighted"
)

func ComputeHealth(hunger, happiness, energy int, mode ComputationMode) int {
	var health int

	// Hunger is inverted: lower hunger = better health
	// Convert hunger to a "satisfaction" score: 100 - hunger
	hungerScore := 100 - hunger

	switch mode {
	case ComputationWeighted:
		health = int(float64(hungerScore)*0.3 + float64(happiness)*0.4 + float64(energy)*0.3)
	default: // average
		health = (hungerScore + happiness + energy) / 3
	}

	// Clamp to [0, 100]
	if health < 0 {
		health = 0
	}
	if health > 100 {
		health = 100
	}

	return health
}
