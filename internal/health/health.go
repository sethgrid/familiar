package health

type ComputationMode string

const (
	ComputationAverage ComputationMode = "average"
	ComputationWeighted ComputationMode = "weighted"
)

func ComputeHealth(hunger, happiness, energy int, mode ComputationMode) int {
	var health int

	switch mode {
	case ComputationWeighted:
		health = int(float64(hunger)*0.3 + float64(happiness)*0.4 + float64(energy)*0.3)
	default: // average
		health = (hunger + happiness + energy) / 3
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
