package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"math"
	"math/rand"
)

const (
	AGE byte = iota
	ENERGY
	BOUNDARY_DIST
	BOUNDARY_DIST_X
	BOUNDARY_DIST_Y
	LAST_MOVE_DIR_X
	LAST_MOVE_DIR_Y
	LOC_X
	LOC_Y
	OSC1
	POPULATION_LOCAL_DENSITY
	POPULATION_FORWARD
	POPULATION_LR
	SIGHT_POPULATION_FORWARD
	GENETIC_SIM_FORWARD
	RANDOM
	SIGHT_FOOD_FORWARD
	POPULATION_FOV
	SATIATION

	SENSOR_COUNT
)

func (c Creature) GetSensor(sensorID byte, w *grid.World, p *Population, simStep int, params *Parameters) float32 {
	var output float32
	switch sensorID {
	case AGE:
		output = float32(c.Age) / float32(params.MaxExpectedAge)

	case ENERGY:
		output = c.Energy / float32(c.Genome.MaxEnergy)

	case BOUNDARY_DIST:
		distX := math.Min(c.Loc.X, float64(params.GridWidth)-c.Loc.X)
		distY := math.Min(c.Loc.Y, float64(params.GridHeight)-c.Loc.Y)
		closest := math.Min(distX, distY)
		maxPossible := math.Max(float64(params.GridWidth)/2, float64(params.GridHeight)/2)
		output = float32(closest / maxPossible)

	case BOUNDARY_DIST_X:
		distX := math.Min(c.Loc.X, float64(params.GridWidth)-c.Loc.X)
		output = float32(distX / (float64(params.GridWidth) / 2))

	case BOUNDARY_DIST_Y:
		distY := math.Min(c.Loc.Y, float64(params.GridHeight)-c.Loc.Y)
		output = float32(distY / (float64(params.GridHeight) / 2))

	case LAST_MOVE_DIR_X:
		output = float32(math.Cos(c.Heading))/2 + 0.5

	case LAST_MOVE_DIR_Y:
		output = float32(math.Sin(c.Heading))/2 + 0.5

	case LOC_X:
		output = float32(c.Loc.X / float64(params.GridWidth))

	case LOC_Y:
		output = float32(c.Loc.Y / float64(params.GridHeight))

	case OSC1:
		val := int(c.Genome.OscPeriod)
		if val == 0 {
			val = 1
		}
		phase := float64(simStep%val) / float64(val)
		factor := (math.Cos(phase*2*math.Pi) + 1) / 2
		output = utils.RestrictFloat32(0, 1, float32(factor))

	case POPULATION_LOCAL_DENSITY:
		output = getLocalPopulationDensity(c.Loc, w, params)

	case POPULATION_FORWARD:
		output = getPopulationDensityAlongAxis(c.Loc, c.Id, w, c.Heading, params)

	case POPULATION_LR:
		output = getPopulationDensityAlongAxis(c.Loc, c.Id, w, c.Heading+math.Pi/2, params)

	case SIGHT_POPULATION_FORWARD:
		output = calculateSightPopFwd(c, w)

	case GENETIC_SIM_FORWARD:
		output = calculateGeneticSimFwd(c, w, p)

	case SIGHT_FOOD_FORWARD:
		output = calculateSightFoodFwd(c, w)

	case POPULATION_FOV:
		output = calculatePopulationFOV(c, w)

	case SATIATION:
		minE := float32(params.MinEnergy)
		maxE := float32(c.Genome.MaxEnergy)
		if maxE > minE {
			output = (c.Energy - minE) / (maxE - minE)
		}

	case RANDOM:
		fallthrough
	default:
		output = rand.Float32()
	}
	return utils.RestrictFloat32(0, 1, output)
}

func calculateSightFoodFwd(c Creature, w *grid.World) float32 {
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	best := float32(0)
	for _, id := range w.GetFoodInRadius(c.Loc, dist) {
		pos := w.GetFoodPos(id)
		dx := pos.X - c.Loc.X
		dy := pos.Y - c.Loc.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d == 0 {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}
		val := float32(1.0 - (d-1)/dist)
		if val < 0 {
			val = 0
		}
		if val > best {
			best = val
		}
	}
	return best
}

func calculateSightPopFwd(c Creature, w *grid.World) float32 {
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	count := 0
	for _, id := range w.GetCreaturesInRadius(c.Loc, dist) {
		if id == c.Id {
			continue
		}
		pos, _ := w.GetCreaturePos(id)
		dx := pos.X - c.Loc.X
		dy := pos.Y - c.Loc.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d == 0 {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
			count++
		}
	}
	// Return 1 (open) when empty, approaching 0 as cone fills with creatures.
	const maxExpected = 10.0
	frac := float64(count) / maxExpected
	if frac > 1 {
		frac = 1
	}
	return float32(1.0 - frac)
}

func calculateGeneticSimFwd(c Creature, w *grid.World, p *Population) float32 {
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	for _, id := range w.GetCreaturesInRadius(c.Loc, dist) {
		if id == c.Id {
			continue
		}
		pos, _ := w.GetCreaturePos(id)
		dx := pos.X - c.Loc.X
		dy := pos.Y - c.Loc.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d == 0 {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
			if other, ok := p.Creatures[id]; ok && other.Alive {
				return GenomeSimilarity(*c.Genome, *other.Genome)
			}
		}
	}
	return 0
}

// calculatePopulationFOV returns a distance-weighted signal [0,1] for the
// nearest creature within the FOV cone and eat radius. Returns 0 when no
// creature is present.
func calculatePopulationFOV(c Creature, w *grid.World) float32 {
	const eatRadius = 2.0
	halfFOVCos := math.Cos(float64(c.Genome.FieldOfView) / 2.0 * math.Pi / 180.0)
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	best := float32(0)
	for _, id := range w.GetCreaturesInRadius(c.Loc, eatRadius) {
		if id == c.Id {
			continue
		}
		pos, _ := w.GetCreaturePos(id)
		dx := pos.X - c.Loc.X
		dy := pos.Y - c.Loc.Y
		d := math.Sqrt(dx*dx + dy*dy)
		if d == 0 {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}
		val := float32(1.0 - d/eatRadius)
		if val > best {
			best = val
		}
	}
	return best
}

func getLocalPopulationDensity(loc grid.Position, w *grid.World, params *Parameters) float32 {
	radius := float64(params.PopulationSensorRadius)
	ids := w.GetCreaturesInRadius(loc, radius)
	// Normalize: approximate max density as one creature per 4 sq units in the area.
	maxExpected := math.Pi * radius * radius / 4.0
	if maxExpected < 1 {
		maxExpected = 1
	}
	return float32(float64(len(ids)) / maxExpected)
}

func getPopulationDensityAlongAxis(loc grid.Position, selfID int, w *grid.World, heading float64, params *Parameters) float32 {
	fwdX, fwdY := grid.HeadingToVec(heading)
	radius := float64(params.PopulationSensorRadius)
	sum := float32(0)
	for _, id := range w.GetCreaturesInRadius(loc, radius) {
		if id == selfID {
			continue
		}
		pos, _ := w.GetCreaturePos(id)
		dx := pos.X - loc.X
		dy := pos.Y - loc.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist == 0 {
			continue
		}
		cos := float32(grid.CosSimilarity(fwdX, fwdY, dx, dy))
		sum += (1.0 / float32(dist)) * cos
	}
	maxSumMag := float32(6 * params.PopulationSensorRadius)
	if sum > maxSumMag {
		sum = maxSumMag
	} else if sum < -maxSumMag {
		sum = -maxSumMag
	}
	// Returns [-1, 1]; the caller's RestrictFloat32 clips to [0, 1].
	return sum / maxSumMag
}
