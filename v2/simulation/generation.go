package simulation

// GetTierFromGeneration returns a tier (0–3) based on per-tier generation thresholds.
func GetTierFromGeneration(generation float32, p *Parameters) byte {
	currentGenProgress := int(generation)
	switch {
	case currentGenProgress >= p.Evolution.Tier4Generation:
		return 3
	case currentGenProgress >= p.Evolution.Tier3Generation:
		return 2
	case currentGenProgress >= p.Evolution.Tier2Generation:
		return 1
	default:
		return 0
	}
}

func getTierBoundaries(generation float32, p *Parameters) (byte, byte) {
	tier := GetTierFromGeneration(generation, p)
	minBound := tier * 64
	maxBound := minBound + 63
	return minBound, maxBound
}
