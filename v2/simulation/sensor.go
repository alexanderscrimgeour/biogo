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
	SIGHT_POPULATION_FORWARD
	SIGHT_FOOD_FORWARD
	SIGHT_CORPSE_FORWARD
	RANDOM
	SATIATION
	FACING_DIR
	NEAREST_FOOD_ANGLE
	NEAREST_FOOD_DIST
	THREAT_FORWARD
	KINSHIP_LOCAL
	ENERGY_RATE
	MASS_FRACTION
	BLOCKED_FORWARD
	PREY_FORWARD
	NEAREST_THREAT_ANGLE
	NEAREST_PREY_ANGLE
	WALL_PROXIMITY
	STOMACH_RATE
	LOCAL_FOOD_PER_CAPITA
	JUVENILE

	SENSOR_COUNT
)

type SensorContext struct {
	SightFoodIDs     []int
	SightCreatureIDs []int
	LocalCreatureIDs []int
}

func (c *Creature) BuildSensorContext(world *grid.World, params *Parameters) *SensorContext {
	return &SensorContext{
		SightFoodIDs:     world.GetFoodInRadius(c.Loc, float64(c.Genome.SightDistance)),
		SightCreatureIDs: world.GetCreaturesInRadius(c.Loc, float64(c.Genome.SightDistance)),
		LocalCreatureIDs: world.GetCreaturesInRadius(c.Loc, params.PopulationSensorRadius),
	}
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

	case SIGHT_POPULATION_FORWARD:
		output = calculateSightPopFwd(c, w, p, ctx)

	case SIGHT_FOOD_FORWARD:
		output = calculateFoodDensityFwd(c, w, ctx)

	case SIGHT_CORPSE_FORWARD:
		output = calculateCorpseDensityFwd(c, w, p, ctx)

	case SATIATION:
		cap := c.StomachCapacity(params)
		if cap > 0 {
			output = c.Stomach / cap
		}

	case FACING_DIR:
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

	case ENERGY_RATE:
		maxE := c.MaxEnergy(params)
		if maxE > 0 {
			delta := c.Energy - c.LastTickEnergy
			output = float32(delta/maxE*10) + 0.5
		}

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
		cap := c.StomachCapacity(params)
		if cap > 0 {
			delta := c.Stomach - c.LastStomach
			output = float32(delta/cap*10) + 0.5
		}

	case LOCAL_FOOD_PER_CAPITA:
		output = calculateLocalFoodPerCapita(ctx, params)

	case JUVENILE:
		if c.IsJuvenile(params) {
			output = 1
		}

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
	maxExpected := math.Pi * radius * radius / 4.0
	return float32(float64(len(ctx.LocalCreatureIDs)) / math.Max(1, maxExpected))
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

// calculateKinshipLocal returns the average genetic similarity [0,1] of all
// living creatures within the population sensor radius. 0 = none nearby.
func calculateKinshipLocal(c Creature, p *Population, ctx *SensorContext) float32 {
	var total float32
	count := 0
	for _, id := range ctx.LocalCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}
		total += GenomeSimilarity(*c.Genome, *other.Genome)
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float32(count)
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
