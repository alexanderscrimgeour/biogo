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
	LOC_X
	LOC_Y
	OSC1
	POPULATION_LOCAL_DENSITY
	POPULATION_LOCAL_HEADING
	POPULATION_LOCAL_CENTRE_OF_MASS
	SIGHT_POPULATION_FORWARD
	SIGHT_POPULATION_DENSITY_FORWARD
	SIGHT_FOOD_FORWARD
	SIGHT_CORPSE_FORWARD
	RANDOM
	SATIATION
	HEADING
	NEAREST_FOOD_ANGLE
	NEAREST_FOOD_DIST
	THREAT_FORWARD
	KINSHIP_LOCAL
	KINSHIP_FORWARD
	KINSHIP_NEAREST // genetic similarity to the single nearest living creature, any direction
	MASS_FRACTION
	BLOCKED_FORWARD
	PREY_FORWARD
	NEAREST_THREAT_ANGLE
	NEAREST_PREY_ANGLE
	WALL_PROXIMITY
	STOMACH_RATE
	LOCAL_FOOD_PER_CAPITA
	JUVENILE
	ENERGY_DELTA
	TEMPERATURE
	TEMPERATURE_DELTA

	SENSOR_COUNT
)

type SensorContext struct {
	SightFoodIDs      []int
	SightCreatureIDs  []int
	SightCreatureSims []float32 // genome similarity to self; parallel to SightCreatureIDs
	LocalCreatureIDs  []int
	LocalCreatureSims []float32 // genome similarity to self; parallel to LocalCreatureIDs
}

func (c *Creature) UpdateSensorContext(world *grid.World, p *Population, params *Parameters) {
	c.SightFoodBuffer = world.GetFoodInRadius(c.Loc, float64(c.Genome.SightDistance), c.SightFoodBuffer)
	c.SightCreatureBuffer = world.GetCreaturesInRadius(c.Loc, float64(c.Genome.SightDistance), c.SightCreatureBuffer)
	c.LocalCreatureBuffer = world.GetCreaturesInRadius(c.Loc, params.PopulationSensorRadius, c.LocalCreatureBuffer)

	c.Sensors.SightFoodIDs = c.SightFoodBuffer
	c.Sensors.SightCreatureIDs = c.SightCreatureBuffer
	c.Sensors.LocalCreatureIDs = c.LocalCreatureBuffer

	if p == nil {
		return
	}

	n := len(c.SightCreatureBuffer)
	if cap(c.SightCreatureSimBuffer) < n {
		c.SightCreatureSimBuffer = make([]float32, n)
	}
	c.SightCreatureSimBuffer = c.SightCreatureSimBuffer[:n]
	for i, id := range c.SightCreatureBuffer {
		if id != c.Id {
			if other, ok := p.Creatures[id]; ok {
				c.SightCreatureSimBuffer[i] = GenomeSimilarity(c.Genome, other.Genome)
			}
		}
	}
	c.Sensors.SightCreatureSims = c.SightCreatureSimBuffer

	m := len(c.LocalCreatureBuffer)
	if cap(c.LocalCreatureSimBuffer) < m {
		c.LocalCreatureSimBuffer = make([]float32, m)
	}
	c.LocalCreatureSimBuffer = c.LocalCreatureSimBuffer[:m]
	for i, id := range c.LocalCreatureBuffer {
		if id != c.Id {
			if other, ok := p.Creatures[id]; ok {
				c.LocalCreatureSimBuffer[i] = GenomeSimilarity(c.Genome, other.Genome)
			}
		}
	}
	c.Sensors.LocalCreatureSims = c.LocalCreatureSimBuffer
}

func (c Creature) GetSensor(sensorID byte, w *grid.World, p *Population, ctx *SensorContext, simStep int, params *Parameters) float32 {
	var output float32
	switch sensorID {
	case AGE:
		output = float32(c.Age) / float32(c.MaxAge(params))

	case ENERGY:
		maxE := c.MaxEnergy(params)
		if maxE > 0 {
			output = c.Energy / maxE
		}

	case LOC_X:
		output = float32(c.Loc.X / params.GridWidth)

	case LOC_Y:
		output = float32(c.Loc.Y / params.GridHeight)

	case OSC1:
		val := int(c.Genome.OscPeriod)
		if val == 0 {
			val = 1
		}
		phase := float64(simStep%val) / float64(val)
		factor := (math.Cos(phase*2*math.Pi) + 1) / 2
		output = utils.RestrictFloat32(0, 1, float32(factor))

	case POPULATION_LOCAL_DENSITY:
		output = getLocalPopulationDensity(ctx, params)
	case POPULATION_LOCAL_HEADING:
		output = getLocalPopulationHeading(c, ctx, p, params)
	case POPULATION_LOCAL_CENTRE_OF_MASS:
		output = getLocalPopulationCentreOfMass(c, ctx, p, params)
	case SIGHT_POPULATION_FORWARD:
		output = calculateSightPopFwd(c, w, p, ctx)
	case SIGHT_POPULATION_DENSITY_FORWARD:
		output = calculateSightPopCentroid(c, w, p, ctx)
	case SIGHT_FOOD_FORWARD:
		output = calculateFoodDensityFwd(c, w, ctx)

	case SIGHT_CORPSE_FORWARD:
		output = calculateCorpseDensityFwd(c, w, p, ctx)

	case SATIATION:
		cap := c.StomachCapacity(params)
		if cap > 0 {
			output = c.Stomach / cap
		}

	case HEADING:
		output = float32((c.Heading + math.Pi) / (2 * math.Pi))

	case NEAREST_FOOD_ANGLE:
		output = calculateNearestFoodAngle(c, w, ctx)

	case NEAREST_FOOD_DIST:
		output = calculateNearestFoodDist(c, w, ctx)

	case THREAT_FORWARD:
		output = calculateThreatFwd(c, p, ctx)

	case PREY_FORWARD:
		output = calculatePreyFwd(c, p, ctx)

	case KINSHIP_LOCAL:
		output = calculateKinshipLocal(c, p, ctx)

	case KINSHIP_FORWARD:
		output = calculateKinshipFwd(c, p, ctx)

	case KINSHIP_NEAREST:
		output = calculateKinshipNearest(c, p, ctx)
	case MASS_FRACTION:
		if c.Genome.Mass > 0 {
			output = c.Mass / float32(c.Genome.Mass)
		}

	case BLOCKED_FORWARD:
		output = calculateBlockedFwd(c, w, params)

	case NEAREST_THREAT_ANGLE:
		output = calculateNearestThreatAngle(c, p, ctx)

	case NEAREST_PREY_ANGLE:
		output = calculateNearestPreyAngle(c, p, ctx)

	case WALL_PROXIMITY:
		output = calculateWallProximity(c, w, params)

	case STOMACH_RATE:
		if params.DigestionRate > 0 {
			output = (c.LastStomach - c.Stomach) / params.DigestionRate
		}

	case LOCAL_FOOD_PER_CAPITA:
		output = calculateLocalFoodPerCapita(ctx, params)

	case JUVENILE:
		if c.IsJuvenile(params) {
			output = 1
		}

	case ENERGY_DELTA:
		maxE := c.MaxEnergy(params)
		if maxE > 0 {
			delta := float32(c.Energy) - float32(c.LastTickEnergy)

			t := math.Tanh(float64(delta * c.Responsiveness * 5))

			output = float32(t)*0.5 + 0.5
		}

	case TEMPERATURE:
		temp := w.TemperatureAt(c.Loc.Y)
		tempNorm := (temp - grid.TempCold) / (grid.TempWarm - grid.TempCold)
		curvedOutput := math.Pow(float64(tempNorm), 2)
		return utils.RestrictFloat32(0, 1, float32(curvedOutput))

	case TEMPERATURE_DELTA:
		currentTemp := w.TemperatureAt(c.Loc.Y)
		prevTemp := w.TemperatureAt(c.LastLoc.Y)

		// 0.5 is "no change", > 0.5 is warming up, < 0.5 is cooling down
		delta := (currentTemp - prevTemp) * c.Responsiveness * 5
		return utils.RestrictFloat32(0, 1, 0.5+delta)
	case RANDOM:
		fallthrough
	default:
		output = rand.Float32()
	}
	return utils.RestrictFloat32(0, 1, output)
}

// calculateFoodDensityFwd returns a proximity-weighted density of food in the
// creature's forward FOV cone, normalised to [0, 1]. Items closer to the
// creature contribute more; the signal saturates at maxFoodDensity total weight.
func calculateFoodDensityFwd(c Creature, w *grid.World, ctx *SensorContext) float32 {
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	const maxFoodDensity = 8.0
	var sum float64
	for _, id := range ctx.SightFoodIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}

		d := math.Sqrt(dx*dx + dy*dy)
		sum += 1.0 - d/dist
	}
	return float32(math.Min(maxFoodDensity, sum) / maxFoodDensity)
}

// calculateSightPopCentroid returns the average horizontal position of
// neighbors in FOV. 0.0 = Far Left, 0.5 = Dead Ahead, 1.0 = Far Right.
func calculateSightPopCentroid(c Creature, w *grid.World, p *Population, ctx *SensorContext) float32 {
	if len(ctx.SightCreatureIDs) == 0 {
		return 0.5
	}

	fwdX, fwdY := grid.HeadingToVec(c.Heading)
	sideX, sideY := -fwdY, fwdX

	var totalSteer float64
	count := 0

	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		distSq := dx*dx + dy*dy
		if distSq == 0 {
			continue
		}

		dist := math.Sqrt(float64(distSq))
		ndx, ndy := float64(dx)/dist, float64(dy)/dist

		dot := grid.CosSimilarity(fwdX, fwdY, dx, dy)
		if dot >= c.halfFOVCos {
			sideDot := (ndx * float64(sideX)) + (ndy * float64(sideY))
			totalSteer += sideDot
			count++
		}
	}

	if count == 0 {
		return 0.5
	}

	avgSteer := totalSteer / float64(count)
	halfFOVSin := math.Sqrt(1 - float64(c.halfFOVCos*c.halfFOVCos))
	output := avgSteer / halfFOVSin
	return float32(math.Max(-1, math.Min(1, output)))
}

// calculateCorpseDensityFwd returns a proximity-weighted density of corpses in
// the creature's forward FOV cone, normalised to [0, 1].
func calculateCorpseDensityFwd(c Creature, w *grid.World, p *Population, ctx *SensorContext) float32 {
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	const maxCorpseDensity = 5.0
	var sum float64
	for _, id := range ctx.SightCreatureIDs {
		other, ok := p.Creatures[id]
		if !ok || other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 == 0 || grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}

		sum += 1.0 - math.Sqrt(d2)/dist
	}
	return float32(math.Min(maxCorpseDensity, sum) / maxCorpseDensity)
}

// calculateSightPopFwd returns a density signal [0,1] for living creatures in
// the forward FOV cone. 0 = none visible, 1 = cone is at capacity.
func calculateSightPopFwd(c Creature, w *grid.World, p *Population, ctx *SensorContext) float32 {
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	count := 0
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		if dx == 0 && dy == 0 {
			continue
		}
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
			count++
		}
	}
	const maxExpected = 10.0
	return float32(math.Min(1.0, float64(count)/maxExpected))
}

func getLocalPopulationDensity(ctx *SensorContext, params *Parameters) float32 {
	radius := params.PopulationSensorRadius

	expectedAreaPerCreature := 100.0
	maxExpected := (math.Pi * radius * radius) / expectedAreaPerCreature

	// Calculate raw density ratio
	rawDensity := float64(len(ctx.LocalCreatureIDs)) / math.Max(1, maxExpected)

	// Apply tanh to compress the range to [0, 1)
	return float32(math.Tanh(rawDensity))
}

func getLocalPopulationHeading(c Creature, ctx *SensorContext, p *Population, params *Parameters) float32 {
	var sumX, sumY float32
	var count int

	radiusSq := params.PopulationSensorRadius * params.PopulationSensorRadius

	for _, id := range ctx.LocalCreatureIDs {
		if id == c.Id {
			continue
		}
		neighbor, ok := p.Creatures[id]
		if !ok || !neighbor.Alive {
			continue
		}
		dx := neighbor.Loc.X - c.Loc.X
		dy := neighbor.Loc.Y - c.Loc.Y
		distSq := dx*dx + dy*dy

		if distSq <= radiusSq {
			sumX += float32(math.Cos(float64(neighbor.Heading)))
			sumY += float32(math.Sin(float64(neighbor.Heading)))
			count++
		}
	}

	if count == 0 {
		return 0.5 // Neutral: No one in range to follow
	}

	avgAngle := math.Atan2(float64(sumY), float64(sumX))

	diff := avgAngle - c.Heading

	// Standard wrap to [-PI, PI]
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}

	// Map to [0, 1] range
	// 0.5 means "I am perfectly aligned with the group"
	return float32((diff / (2 * math.Pi)) + 0.5)
}

func getLocalPopulationCentreOfMass(c Creature, ctx *SensorContext, p *Population, params *Parameters) float32 {
	var sumX, sumY float64
	var count int

	radiusSq := params.PopulationSensorRadius * params.PopulationSensorRadius

	for _, id := range ctx.LocalCreatureIDs {
		if id == c.Id {
			continue
		}
		neighbor, ok := p.Creatures[id]
		if !ok || !neighbor.Alive {
			continue
		}

		dx := neighbor.Loc.X - c.Loc.X
		dy := neighbor.Loc.Y - c.Loc.Y
		distSq := dx*dx + dy*dy

		if distSq <= radiusSq {
			sumX += neighbor.Loc.X
			sumY += neighbor.Loc.Y
			count++
		}
	}

	if count == 0 {
		return 0.5 // Neutral: No neighbors to form a center
	}

	avgX := sumX / float64(count)
	avgY := sumY / float64(count)

	relX := avgX - c.Loc.X
	relY := avgY - c.Loc.Y

	angleToCenter := math.Atan2(float64(relY), float64(relX))

	diff := angleToCenter - c.Heading

	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}

	// Map to [0, 1] range
	// 0.5 means the "Heart" of the crowd is directly in front of you.
	return float32((diff / (2 * math.Pi)) + 0.5)
}

// calculateNearestFoodAngle returns the angle to the nearest food item within
// sight range, relative to the creature's heading, mapped to [0, 1] where 0.5
// means directly ahead. Returns 0 when no food is visible; pair with
// NEAREST_FOOD_DIST to distinguish this from food directly behind.
func calculateNearestFoodAngle(c Creature, w *grid.World, ctx *SensorContext) float32 {
	bestDistSq := math.MaxFloat64
	var bestDx, bestDy float64
	found := false
	for _, id := range ctx.SightFoodIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 < bestDistSq {
			bestDistSq, bestDx, bestDy, found = d2, dx, dy, true
		}
	}
	if !found {
		return 0
	}
	relAngle := grid.NormalizeAngle(math.Atan2(bestDy, bestDx) - c.Heading)
	return float32((relAngle + math.Pi) / (2 * math.Pi))
}

// calculateNearestFoodDist returns proximity to the nearest food item within
// sight range: 1 = adjacent, approaching 0 at max range, 0 = none visible.
func calculateNearestFoodDist(c Creature, w *grid.World, ctx *SensorContext) float32 {
	dist := float64(c.Genome.SightDistance)
	bestDistSq := math.MaxFloat64
	for _, id := range ctx.SightFoodIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 < bestDistSq {
			bestDistSq = d2
		}
	}
	if bestDistSq == math.MaxFloat64 {
		return 0
	}
	return float32(1.0 - math.Sqrt(bestDistSq)/dist)
}

// calculateThreatFwd returns a proximity-weighted signal [0,1] for the nearest
// creature heavier than self within the forward FOV cone. 0 = no threat.
func calculateThreatFwd(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)
	best := float32(0)
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive || other.Mass <= c.Mass {
			continue
		}
		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}
		d2 := dx*dx + dy*dy
		if d2 == 0 {
			continue
		}
		val := float32(1.0 - math.Sqrt(d2)/dist)
		if val > best {
			best = val
		}
	}
	return best
}

// calculatePreyFwd returns a proximity-weighted signal [0,1] for the nearest
// live creature lighter than self within the forward FOV cone. 0 = no prey visible.
func calculatePreyFwd(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	dist := float64(c.Genome.SightDistance)
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)
	best := float32(0)
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive || other.Mass >= c.Mass {
			continue
		}
		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}
		d2 := dx*dx + dy*dy
		if d2 == 0 {
			continue
		}
		val := float32(1.0 - math.Sqrt(d2)/dist)
		if val > best {
			best = val
		}
	}
	return best
}

func calculateKinshipFwd(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}

	dist := float64(c.Genome.SightDistance)
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := grid.HeadingToVec(c.Heading)
	best := float32(0)

	for i, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}

		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		// 1. Spatial Check: Is it in the vision cone?
		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		if grid.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}

		d2 := dx*dx + dy*dy
		if d2 == 0 || d2 > dist*dist {
			continue
		}

		// 3. Signal Strength: Scale pre-computed similarity by proximity.
		// A twin far away is a faint signal; a twin in your face is 1.0.
		proximity := float32(1.0 - math.Sqrt(d2)/dist)
		val := ctx.SightCreatureSims[i] * proximity

		if val > best {
			best = val
		}
	}
	return best
}

// calculateKinshipLocal returns the average genetic similarity [0,1] of all
// living creatures within the population sensor radius. 0 = none nearby.
func calculateKinshipLocal(c Creature, p *Population, ctx *SensorContext) float32 {
	var total float32
	count := 0
	for i, id := range ctx.LocalCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}
		total += ctx.LocalCreatureSims[i]
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float32(count)
}

// calculateKinshipNearest returns the genetic similarity [0,1] to the single
// nearest living creature within sight range, regardless of heading direction.
// Returns 0 when no neighbours are visible. Useful as a mate-selection signal:
// creatures that are close and genetically similar score near 1.0.
func calculateKinshipNearest(c Creature, p *Population, ctx *SensorContext) float32 {
	bestDistSq := math.MaxFloat64
	bestSim := float32(0)
	found := false
	for i, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}
		dx := other.Loc.X - c.Loc.X
		dy := other.Loc.Y - c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 < bestDistSq {
			bestDistSq = d2
			bestSim = ctx.SightCreatureSims[i]
			found = true
		}
	}
	if !found {
		return 0
	}
	return bestSim
}

// calculateBlockedFwd returns a proximity signal [0,1] for the nearest obstacle
// (wall or boundary) along the heading within sight distance.
// 0 = clear path, approaching 1 as the obstacle nears.
func calculateBlockedFwd(c Creature, w *grid.World, params *Parameters) float32 {
	sightDist := float64(c.Genome.SightDistance)
	if sightDist == 0 {
		return 0
	}
	fwdX, fwdY := grid.HeadingToVec(c.Heading)

	// Step along the heading ray to find the nearest wall or boundary.
	blockDist := sightDist
	const steps = 20
	for i := 1; i <= steps; i++ {
		d := float64(i) / float64(steps) * sightDist
		probe := grid.Position{X: c.Loc.X + fwdX*d, Y: c.Loc.Y + fwdY*d}
		if !w.IsInBounds(probe) || w.IsWall(probe) {
			blockDist = d
			break
		}
	}

	if blockDist >= sightDist {
		return 0
	}
	return float32(1.0 - blockDist/sightDist)
}

// calculateNearestThreatAngle returns the angle to the nearest creature heavier
// than self within sight range, relative to heading, mapped to [0,1] where 0.5
// is directly ahead. Returns 0 when no threat is visible.
func calculateNearestThreatAngle(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	bestDistSq := math.MaxFloat64
	var bestDx, bestDy float64
	found := false
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive || other.Mass <= c.Mass {
			continue
		}
		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 > 0 && d2 < bestDistSq {
			bestDistSq, bestDx, bestDy, found = d2, dx, dy, true
		}
	}
	if !found {
		return 0
	}
	relAngle := grid.NormalizeAngle(math.Atan2(bestDy, bestDx) - c.Heading)
	return float32((relAngle + math.Pi) / (2 * math.Pi))
}

// calculateNearestPreyAngle returns the angle to the nearest creature lighter
// than self within sight range, relative to heading, mapped to [0,1] where 0.5
// is directly ahead. Returns 0 when no prey is visible.
func calculateNearestPreyAngle(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	bestDistSq := math.MaxFloat64
	var bestDx, bestDy float64
	found := false
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive || other.Mass >= c.Mass {
			continue
		}
		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 > 0 && d2 < bestDistSq {
			bestDistSq, bestDx, bestDy, found = d2, dx, dy, true
		}
	}
	if !found {
		return 0
	}
	relAngle := grid.NormalizeAngle(math.Atan2(bestDy, bestDx) - c.Heading)
	return float32((relAngle + math.Pi) / (2 * math.Pi))
}

// calculateWallProximity returns a proximity signal [0,1] for the nearest wall
// or world boundary within sight distance. 0 = none within range, 1 = adjacent.
func calculateWallProximity(c Creature, w *grid.World, params *Parameters) float32 {
	sightDist := float64(c.Genome.SightDistance)
	if sightDist == 0 {
		return 0
	}

	// Distance to world boundaries.
	minDist := math.Min(c.Loc.X, params.GridWidth-c.Loc.X)
	minDist = math.Min(minDist, math.Min(c.Loc.Y, params.GridHeight-c.Loc.Y))

	// Distance to obstacle walls (nearest point on each rectangle).
	for _, wall := range w.Walls {
		nearX := math.Max(wall.X, math.Min(c.Loc.X, wall.X+wall.W))
		nearY := math.Max(wall.Y, math.Min(c.Loc.Y, wall.Y+wall.H))
		dx := c.Loc.X - nearX
		dy := c.Loc.Y - nearY
		d := math.Sqrt(dx*dx + dy*dy)
		if d < minDist {
			minDist = d
		}
	}

	if minDist >= sightDist {
		return 0
	}
	return float32(1.0 - minDist/sightDist)
}

// calculateLocalFoodPerCapita returns a signal [0,1] for local food availability
// relative to local competition. Saturates at 4 food items per creature in the area.
func calculateLocalFoodPerCapita(ctx *SensorContext, params *Parameters) float32 {
	foodCount := float64(len(ctx.SightFoodIDs))
	creatureCount := float64(len(ctx.LocalCreatureIDs))
	return float32(math.Min(1.0, (foodCount/math.Max(1, creatureCount))/4.0))
}
