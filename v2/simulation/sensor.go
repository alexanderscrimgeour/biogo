package simulation

import (
	"biogo/v2/world"
	"math"
	"math/rand"
)

const (

	// -- TIER 1 --

	// Always outputs 1; gives the network a learnable bias via connection weights.
	BIAS byte = iota
	// Current energy relative to maximum: -1 = empty, 0 = half full, 1 = full.
	ENERGY
	// Age mapped to [-1, 1]: -1 at birth, 0 at maturity (end of juvenile period), 1 at max age.
	AGE
	// Angle to the nearest food item relative to heading [-1, 1]; 0 = directly ahead.
	NEAREST_FOOD_ANGLE
	// Distance to the nearest food item normalised by sight range [-1, 1]; 1 = nothing visible.
	NEAREST_FOOD_DIST

	// -- TIER 2 --
	
	// Horizontal world position: -1 = left border, 0 = centre, 1 = right border.
	LOC_X
	// Vertical world position: -1 = top border, 0 = centre, 1 = bottom border.
	LOC_Y
	// Current heading angle normalised to [0, 1].
	HEADING
	// Current speed as a fraction of maximum speed [-1, 1].
	VELOCITY
	// Sinusoidal oscillator whose period is set by the creature's clock gene [-1, 1].
	OSC1
	// Proximity of the nearest obstacle (wall or creature) along heading; 0 = clear, 1 = adjacent.
	BLOCKED_FORWARD
	// Proximity to the nearest wall or world boundary [0, 1].
	WALL_PROXIMITY

	// -- TIER 3 --

	// Proximity-weighted density of food in the forward FOV cone [-1, 1]: -1 = empty, 0 = moderate density, 1 = fully dense.
	SIGHT_FOOD_FORWARD
	// Proximity-weighted density of meat in the forward FOV cone [-1, 1]: -1 = empty, 0 = moderate density, 1 = fully dense.
	SIGHT_MEAT_FORWARD
	// Proximity of the nearest lighter creature in the forward FOV; 1 = none visible [-1, 1].
	PREY_FORWARD
	// Angle to the nearest lighter creature relative to heading [-1, 1].
	NEAREST_PREY_ANGLE
	// Angle to the nearest heavier creature relative to heading [-1, 1].
	NEAREST_THREAT_ANGLE
	// Proximity of the nearest heavier creature in the forward FOV; 1 = none visible [-1, 1].
	THREAT_FORWARD
	// Current mass as a fraction of genome target mass [0, 1].
	MASS_FRACTION
	// 1 if still growing to adult mass, -1 if fully grown.
	JUVENILE
	// Stomach fullness: 1 = empty/starving, -1 = full.
	SATIATION
	// Rate of stomach content change per digestion tick; positive = absorbing faster than baseline.
	STOMACH_RATE
	
	// -- TIER 4 --
	
	// Bearing to the centre of mass of nearby creatures relative to own heading [-1, 1].
	POPULATION_LOCAL_CENTRE_OF_MASS
	// Proximity-weighted density of creatures within sight radius [-1, 1].
	POPULATION_LOCAL_DENSITY
	// Average heading of nearby creatures relative to own heading [-1, 1].
	POPULATION_LOCAL_HEADING
	// Density of living creatures in the forward FOV cone [-1, 1].
	SIGHT_POPULATION_FORWARD
	// Lateral offset of the creature centroid within the forward FOV cone [-1, 1].
	SIGHT_POPULATION_DENSITY_FORWARD
	// 1 if the nearest forward creature is physically touching this one, -1 otherwise.
	TOUCHING
	// Local temperature relative to world range; -1 = cold pole, 1 = warm equator.
	TEMPERATURE
	// Change in temperature since last position; negative = moving toward cooler zone.
	TEMPERATURE_DELTA
	// Change in energy since last tick, scaled and clamped [-1, 1].
	ENERGY_DELTA
	// Random noise sampled each tick; useful for stochastic behaviour.
	RANDOM
	// Average genetic similarity of all creatures within sight radius [-1, 1].
	KINSHIP_LOCAL
	// Distance to the nearest genetically similar creature (kin) in the forward FOV [-1, 1].
	KINSHIP_NEAREST_DISTANCE
	// Genetic similarity to the single nearest visible creature [-1, 1].
	KINSHIP_NEAREST
	// Food-to-creature ratio within sight range; positive = surplus, negative = scarcity.
	LOCAL_FOOD_PER_CAPITA

	SENSOR_COUNT
)

// Expected creature density
const kDensity = 0.00008

type SensorContext struct {
	SightFoodIDs      []int
	SightMeatIDs      []int
	SightCreatureIDs  []int
	SightCreatureSims []float32 // genome similarity to self; parallel to SightCreatureIDs
	LocalCreatureIDs  []int
	LocalCreatureSims []float32 // genome similarity to self; parallel to LocalCreatureIDs
}

func (c *Creature) UpdateSensorContext(world *world.World, p *Population, params *Parameters) {
	c.SightFoodBuffer, c.SightMeatBuffer, c.SightCreatureBuffer = world.GetAllInRadius(
		c.Loc, c.GetSightDistance(), c.SightFoodBuffer, c.SightMeatBuffer, c.SightCreatureBuffer,
	)
	// LocalCreatureBuffer contains the same set — copy instead of a second query.
	c.LocalCreatureBuffer = append(c.LocalCreatureBuffer[:0], c.SightCreatureBuffer...)

	c.Sensors.SightFoodIDs = c.SightFoodBuffer
	c.Sensors.SightMeatIDs = c.SightMeatBuffer
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
				c.SightCreatureSimBuffer[i] = c.cachedSimilarity(id, other)
			}
		}
	}
	c.Sensors.SightCreatureSims = c.SightCreatureSimBuffer

	// Same IDs → same similarities; copy to avoid a second GenomeSimilarity loop.
	c.LocalCreatureSimBuffer = append(c.LocalCreatureSimBuffer[:0], c.SightCreatureSimBuffer...)
	c.Sensors.LocalCreatureSims = c.LocalCreatureSimBuffer
}

func (c Creature) GetSensor(sensorID byte, w *world.World, p *Population, ctx *SensorContext, simStep int, params *Parameters) float32 {
	var output float32
	switch sensorID {
	case BIAS:
		output = 1
	case AGE:
		jp := c.cachedJuvenilePeriod
		maxAge := c.MaxAge(params)
		if jp > 0 && c.Age < jp {
			output = -1.0 + float32(c.Age)/float32(jp)
		} else if maxAge > jp {
			output = float32(c.Age-jp) / float32(maxAge-jp)
		}

	case ENERGY:
		maxE := c.MaxEnergy(params)
		if maxE > 0 {
			output = (c.Energy/maxE)*2 - 1
		}

	case LOC_X:
		output = float32(c.Loc.X/float32(params.WorldWidth))*2 - 1

	case LOC_Y:
		output = float32(c.Loc.Y/float32(params.WorldHeight))*2 - 1
	case OSC1:
		period := c.Clock
		if period < 2 {
			period = 2
		}

		// Keep it entirely in float32 arithmetic
		phase := float32(simStep%period) / float32(period)

		// Use math32 or direct cast without double conversion overhead
		output = float32(math.Sin(float64(phase * 2.0 * 3.149)))

	case POPULATION_LOCAL_DENSITY:
		output = calculateLocalPopulationDensity(c, ctx, p, params)
	case POPULATION_LOCAL_HEADING:
		output = calculateLocalPopulationHeading(c, ctx, p, params)
	case POPULATION_LOCAL_CENTRE_OF_MASS:
		output = getLocalPopulationCentreOfMass(c, ctx, p, params)
	case SIGHT_POPULATION_FORWARD:
		output = calcaulatePopulationDensityFov(c, w, p, ctx)
	case SIGHT_POPULATION_DENSITY_FORWARD:
		output = calculateSightPopCentroid(c, w, p, ctx)
	case SIGHT_FOOD_FORWARD:
		output = calculateFoodDensityFwd(c, w, ctx)

	case SIGHT_MEAT_FORWARD:
		output = calculateMeatDensityFwd(c, w, ctx)

	case SATIATION:
		cap := c.StomachCapacity(params)
		if cap > 0 {
			// Linear map: 0/cap (0.0) becomes 1.0, cap/cap (1.0) becomes -1.0
			output = 1.0 - (2.0 * (c.Stomach / cap))
		} else {
			output = 1.0 // If no capacity, treat as empty/starving
		}
	case HEADING:
		output = float32(c.Heading / math.Pi)
	case VELOCITY:
		output = calculateVelocity(c, params)

	case NEAREST_FOOD_ANGLE:
		output = calculateNearestFoodAngle(c, w, ctx)

	case NEAREST_FOOD_DIST:
		output = calculateNearestFoodDistFov(c, w, ctx, params)

	case THREAT_FORWARD:
		output = calculateNearestThreatDistFov(c, p, ctx, params)

	case PREY_FORWARD:
		output = calculateNearestPreyDistFov(c, p, ctx, params)
	case KINSHIP_NEAREST_DISTANCE:
		output = calculateDistanceToClosestKin(c, p, ctx, params)
	case KINSHIP_LOCAL:
		output = calculateLocalKinship(c, p, ctx)
	case KINSHIP_NEAREST:
		output = calculateNearestKinship(c, p, ctx)
	case MASS_FRACTION:
		if c.Genome.Mass > 0 {
			output = c.Mass / float32(c.Genome.Mass)
		}

	case BLOCKED_FORWARD:
		output = calculateBlockedFwd(c, w, p, ctx, params)

	case NEAREST_THREAT_ANGLE:
		output = calculateNearestThreatAngle(c, p, ctx)

	case NEAREST_PREY_ANGLE:
		output = calculateNearestPreyAngle(c, p, ctx)

	case WALL_PROXIMITY:
		output = calculateWallProximity(c, w, params)

	case STOMACH_RATE:
		if params.DigestionRate > 0 {
			output = float32((float64(c.LastStomach) - float64(c.Stomach)) / params.DigestionRate)
		}

	case LOCAL_FOOD_PER_CAPITA:
		output = calculateLocalFoodPerCapita(ctx, params)

	case JUVENILE:
		if c.IsJuvenile() {
			output = 1
		} else {
			output = -1
		}
	case ENERGY_DELTA:
		maxE := c.MaxEnergy(params)
		if maxE > 0 {
			output = tanhf((c.Energy - c.LastTickEnergy) * c.Responsiveness * 5)
		}

	case TEMPERATURE:
		temp := w.TemperatureAt(c.Loc.Y)
		// (temp - average) / half_range creates a linear -1 to 1 scale.
		midPoint := (world.TempWarm + world.TempCold) / 2
		halfRange := (world.TempWarm - world.TempCold) / 2
		tempCentered := (temp - midPoint) / halfRange
		return tanhf(tempCentered * 2.0)

	case TEMPERATURE_DELTA:
		currentTemp := w.TemperatureAt(c.Loc.Y)
		prevTemp := w.TemperatureAt(c.LastLoc.Y)
		// -1 = Cooling down fast, 0 = No change, 1 = Warming up fast
		output = tanhf((currentTemp - prevTemp) * c.Responsiveness * 5)
	case TOUCHING:
		output = calculateTouching(c, p, ctx)
	case RANDOM:
		fallthrough
	default:
		output = rand.Float32()
	}
	return output
}

func calculateVelocity(c Creature, p *Parameters) float32 {
	if p.MaxSpeedPerStep > 0 {
		output := float64(c.Velocity) / p.MaxSpeedPerStep
		if output > 1 {
			return 1
		}
		if output < -1 {
			return -1
		}
		return float32(output)
	}
	return 0
}

// calculateFoodDensityFwd returns a proximity-weighted density of food in the
// creature's forward FOV cone, mapped to [-1, 1]. -1 = empty, 1 = fully dense.
// Items closer to the creature contribute more; saturates at maxFoodDensity total weight.
func calculateFoodDensityFwd(c Creature, w *world.World, ctx *SensorContext) float32 {
	dist := c.GetSightDistance()
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	const maxFoodDensity = 8.0
	var sum float32
	for _, id := range ctx.SightFoodIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		if world.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}

		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		sum += 1.0 - d/dist
	}
	if sum > maxFoodDensity {
		sum = maxFoodDensity
	}
	return (sum/maxFoodDensity)*2 - 1
}

// calculateSightPopCentroid returns the average horizontal position of
// neighbors in FOV. [-1, 1]
func calculateSightPopCentroid(c Creature, w *world.World, p *Population, ctx *SensorContext) float32 {
	if len(ctx.SightCreatureIDs) == 0 {
		return 0.5
	}

	fwdX, fwdY := world.HeadingToVec(c.Heading)
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

		dot := world.CosSimilarity(fwdX, fwdY, dx, dy)
		if dot >= c.halfFOVCos {
			sideDot := (ndx * float64(sideX)) + (ndy * float64(sideY))
			totalSteer += sideDot
			count++
		}
	}

	if count == 0 {
		return 0
	}

	avgSteer := totalSteer / float64(count)
	halfFOVSin := math.Sqrt(1 - float64(c.halfFOVCos*c.halfFOVCos))

	if halfFOVSin > 0 {
		avgSteer /= halfFOVSin
	}

	if avgSteer > 1 {
		return 1
	}
	if avgSteer < -1 {
		return -1
	}
	return float32(avgSteer)
}

// calculateMeatDensityFwd returns a proximity-weighted density of meat in the
// creature's forward FOV cone, mapped to [-1, 1]. -1 = empty, 1 = fully dense.
func calculateMeatDensityFwd(c Creature, w *world.World, ctx *SensorContext) float32 {
	dist := c.GetSightDistance()
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	const maxMeatDensity = 8.0
	var sum float32
	for _, id := range ctx.SightMeatIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		if world.CosSimilarity(fwdX, fwdY, dx, dy) < halfFOVCos {
			continue
		}
		d := float32(math.Sqrt(float64(dx*dx + dy*dy)))
		sum += 1.0 - d/dist
	}
	if sum > maxMeatDensity {
		sum = maxMeatDensity
	}
	return (sum/maxMeatDensity)*2 - 1
}

// calcaulatePopulationDensityFov returns a density signal [-1,1] for living creatures in
// the forward FOV cone. 0 = none visible, 1 = cone is at capacity.
func calcaulatePopulationDensityFov(c Creature, w *world.World, p *Population, ctx *SensorContext) float32 {
	visionDist := c.GetSightDistance()
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	// Calculate the area of the vision cone: Area = (Dist^2 * Angle) / 2
	viewArea := 0.5 * float64(visionDist*visionDist) * math.Acos(float64(c.halfFOVCos)) * 2

	var sum float32
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		dist := fastDist(dx, dy)
		if dist == 0 || dist > visionDist {
			continue
		}

		dot := (fwdX*dx + fwdY*dy) / dist
		if dot < halfFOVCos {
			continue
		}

		sum += 1.0 - (dist / visionDist)
	}
	density := float64(sum) / viewArea

	unipolar := density / (kDensity + density)
	return float32((unipolar * 2.0) - 1.0)
}

func calculateLocalPopulationDensity(c Creature, ctx *SensorContext, p *Population, params *Parameters) float32 {
	visionDist := c.GetSightDistance()
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	// Calculate the nearby area, based on sign distance
	viewArea := math.Pi * float64(visionDist*visionDist)

	var sum float32
	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		dist := fastDist(dx, dy)
		if dist == 0 || dist > visionDist {
			continue
		}

		dot := (fwdX*dx + fwdY*dy) / dist
		if dot < halfFOVCos {
			continue
		}
		sum += 1.0 - (dist / visionDist)
	}

	density := float64(sum) / viewArea
	unipolar := density / (kDensity + density)
	return float32((unipolar * 2.0) - 1.0)
}

func calculateLocalPopulationHeading(c Creature, ctx *SensorContext, p *Population, params *Parameters) float32 {
	var sumX, sumY float32
	var count int

	rad := c.GetSightDistance()
	radiusSq := rad * rad

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
	diff := avgAngle - float64(c.Heading)

	// Standard wrap to [-PI, PI]
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}

	// Map to [-1, 1] range
	// 0 means "I am perfectly aligned with the group"
	return float32(diff * (1.0 / math.Pi))
}

func getLocalPopulationCentreOfMass(c Creature, ctx *SensorContext, p *Population, params *Parameters) float32 {
	var sumX, sumY float32
	var count int
	rad := c.GetSightDistance()
	radiusSq := rad * rad

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
	avgX := sumX / float32(count)
	avgY := sumY / float32(count)

	relX := avgX - c.Loc.X
	relY := avgY - c.Loc.Y

	angleToCenter := math.Atan2(float64(relY), float64(relX))
	diff := angleToCenter - float64(c.Heading)

	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}

	// Map to [-1, 1] range
	// 0 means "I am perfectly aligned with the group"
	return float32(diff / math.Pi)
}

// calculateNearestFoodAngle returns the angle to the nearest food item within
// sight range, relative to the creature's heading, mapped to [-1, 1] where 0
// means directly ahead. Returns 0 when no food is visible; pair with
// NEAREST_FOOD_DIST to distinguish this from food directly behind.
func calculateNearestFoodAngle(c Creature, w *world.World, ctx *SensorContext) float32 {
	var bestDistSq float32 = math.MaxFloat32
	var bestDx, bestDy float32
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

	angleToFood := math.Atan2(float64(bestDy), float64(bestDx))
	diff := angleToFood - float64(c.Heading)

	if diff > math.Pi {
		diff -= 2 * math.Pi
	} else if diff < -math.Pi {
		diff += 2 * math.Pi
	}

	const invPi = 1.0 / math.Pi
	return float32(diff * invPi)
}

func calculateNearestFoodDistFov(c Creature, w *world.World, ctx *SensorContext, params *Parameters) float32 {
	maxDist := c.GetSightDistance()
	var bestDistSq float32 = math.MaxFloat32

	for _, id := range ctx.SightFoodIDs {
		pos := w.GetFoodPos(id)
		dx, dy := pos.X-c.Loc.X, pos.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 < bestDistSq {
			bestDistSq = d2
		}
	}

	if bestDistSq == math.MaxFloat32 {
		return 1.0
	}

	actualDist := float32(math.Sqrt(float64(bestDistSq)))
	normDist := actualDist / maxDist

	return (normDist * 2.0) - 1.0
}

// calculateNearestThreatDistFov returns a proximity-weighted signal [1,-1] for the nearest
// creature heavier than self within the forward FOV cone. 0 = no threat.
func calculateNearestThreatDistFov(c Creature, p *Population, ctx *SensorContext, params *Parameters) float32 {
	if p == nil {
		return 1.0
	}

	maxDist := c.GetSightDistance()
	halfFOVCos := c.halfFOVCos
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	var bestDistSq float32 = math.MaxFloat32

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

		if d2 > maxDist*maxDist || d2 == 0 {
			continue
		}

		actualDist := float32(math.Sqrt(float64(d2)))
		dot := (fwdX*dx + fwdY*dy) / actualDist
		if dot < halfFOVCos {
			continue
		}

		if d2 < bestDistSq {
			bestDistSq = d2
		}
	}

	if bestDistSq == math.MaxFloat32 {
		return 1.0
	}

	normDist := float32(math.Sqrt(float64(bestDistSq))) / maxDist
	return (normDist * 2.0) - 1.0
}

// calculateNearestPreyDistFov returns a proximity-weighted signal [1,-1] for the nearest
// creature lighter than self within the forward FOV cone. 0 = no prey.
func calculateNearestPreyDistFov(c Creature, p *Population, ctx *SensorContext, params *Parameters) float32 {
	if p == nil {
		return 1.0
	}

	maxDist := c.GetSightDistance()
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	var bestDistSq float32 = math.MaxFloat32

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

		if d2 > maxDist*maxDist || d2 == 0 {
			continue
		}

		actualDist := float32(math.Sqrt(float64(d2)))
		dot := (fwdX*dx + fwdY*dy) / actualDist
		if dot < c.halfFOVCos {
			continue
		}

		if d2 < bestDistSq {
			bestDistSq = d2
		}
	}

	if bestDistSq == math.MaxFloat32 {
		return 1.0
	}

	normDist := float32(math.Sqrt(float64(bestDistSq))) / maxDist
	return (normDist * 2.0) - 1.0
}

func calculateDistanceToClosestKin(c Creature, p *Population, ctx *SensorContext, params *Parameters) float32 {
	if p == nil {
		return 1.0
	}

	maxDist := c.GetSightDistance()
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	var bestDistSq float32 = math.MaxFloat32

	for i, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}

		if ctx.SightCreatureSims[i] < params.MinMatingSimilarity {
			continue
		}

		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy

		if d2 > maxDist*maxDist || d2 == 0 {
			continue
		}

		actualDist := float32(math.Sqrt(float64(d2)))
		dot := (fwdX*dx + fwdY*dy) / actualDist
		if dot < c.halfFOVCos {
			continue
		}

		if d2 < bestDistSq {
			bestDistSq = d2
		}
	}

	if bestDistSq == math.MaxFloat32 {
		return 1.0
	}

	normDist := float32(math.Sqrt(float64(bestDistSq))) / maxDist
	return (normDist * 2.0) - 1.0
}

// calculateLocalKinship returns the average genetic similarity [0,1] of all
// living creatures within the population sensor radius. 0 = none nearby.
func calculateLocalKinship(c Creature, p *Population, ctx *SensorContext) float32 {
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
		return 1.0
	}

	avgSimilarity := total / float32(count)

	// Maps high similarity (1.0) to -1.0 and low similarity (0.0) to 1.0
	return (avgSimilarity * -2.0) + 1.0
}

// calculateNearestKinship returns the genetic similarity [0,1] to the single
// nearest living creature within sight range, regardless of heading direction.
// Returns 0 when no neighbours are visible. Useful as a mate-selection signal:
// creatures that are close and genetically similar score near 1.0.
func calculateNearestKinship(c Creature, p *Population, ctx *SensorContext) float32 {
	var bestDistSq float32 = math.MaxFloat32
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

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy

		if d2 < bestDistSq {
			bestDistSq = d2
			bestSim = ctx.SightCreatureSims[i]
			found = true
		}
	}

	if !found {
		return 0.0
	}
	centeredSim := bestSim - 0.5
	// Maps 1.0 (Clone) -> -1.0
	// Maps -1.0 (Stranger) -> -1.0
	return tanhf(centeredSim * -4.0)
}

// calculateBlockedFwd returns a proximity signal [0,1] for the nearest obstacle
// (wall, boundary, or creature) along the heading within sight distance.
// 0 = clear path, approaching 1 as the obstacle nears.
func calculateBlockedFwd(c Creature, w *world.World, p *Population, ctx *SensorContext, params *Parameters) float32 {
	sightDist := c.GetSightDistance()
	if sightDist == 0 {
		return 0
	}
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	// Step along the heading ray to find the nearest wall or boundary.
	blockDist := sightDist
	const steps = 20
	for i := 1; i <= steps; i++ {
		d := float32(i) / float32(steps) * sightDist
		probe := world.Position{X: c.Loc.X + fwdX*d, Y: c.Loc.Y + fwdY*d}
		if !w.IsInBounds(probe) || w.IsWall(probe) {
			blockDist = d
			break
		}
	}

	// Check if any visible creature lies along the forward ray.
	if p != nil && ctx != nil {
		for _, id := range ctx.SightCreatureIDs {
			if id == c.Id {
				continue
			}
			other, ok := p.Creatures[id]
			if !ok || !other.Alive {
				continue
			}
			dx := other.Loc.X - c.Loc.X
			dy := other.Loc.Y - c.Loc.Y
			// Projection of the vector onto the forward direction.
			proj := dx*fwdX + dy*fwdY
			if proj <= 0 || proj >= blockDist {
				continue
			}
			// Perpendicular distance from the ray to the other creature's centre.
			perpSq := (dx-fwdX*proj)*(dx-fwdX*proj) + (dy-fwdY*proj)*(dy-fwdY*proj)
			threshold := c.Radius + other.Radius
			if perpSq <= threshold*threshold {
				blockDist = proj
			}
		}
	}

	if blockDist >= sightDist {
		return 0
	}
	return 1.0 - blockDist/sightDist
}

// calculateNearestThreatAngle returns the angle to the nearest creature heavier
// than self within sight range, relative to heading, mapped to [0,1] where 0.5
// is directly ahead. Returns 0 when no threat is visible.
func calculateNearestThreatAngle(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	var bestDistSq float32 = math.MaxFloat32
	var bestDx, bestDy float32
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
	relAngle := world.NormalizeAngle(math.Atan2(float64(bestDy), float64(bestDx)) - float64(c.Heading))
	return tanhf(float32(relAngle) * 2.0)
}

// calculateNearestPreyAngle returns the angle to the nearest creature lighter
// than self within sight range, relative to heading, mapped to [0,1] where 0.5
// is directly ahead. Returns 0 when no prey is visible.
func calculateNearestPreyAngle(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return 0
	}
	var bestDistSq float32 = math.MaxFloat32
	var bestDx, bestDy float32
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
	relAngle := world.NormalizeAngle(math.Atan2(float64(bestDy), float64(bestDx)) - float64(c.Heading))
	return tanhf(float32(relAngle) * 2.0)
}

// calculateWallProximity returns a proximity signal [0,1] for the nearest wall
// or world boundary within sight distance. 0 = none within range, 1 = adjacent.
func calculateWallProximity(c Creature, w *world.World, params *Parameters) float32 {
	sightDist := c.GetSightDistance()
	if sightDist <= 0 {
		return -1
	}

	// Work in float64 for wall geometry (walls have float64 coords)
	cx, cy := float64(c.Loc.X), float64(c.Loc.Y)

	// Work in squared space to avoid Sqrt inside the loop
	minDistSq := math.Min(cx, params.WorldWidth-cx)
	minDistSq = minDistSq * minDistSq

	minYDist := math.Min(cy, params.WorldHeight-cy)
	minDistSq = math.Min(minDistSq, minYDist*minYDist)

	for _, wall := range w.Walls {
		nearX := math.Max(wall.X, math.Min(cx, wall.X+wall.W))
		nearY := math.Max(wall.Y, math.Min(cy, wall.Y+wall.H))
		dx := cx - nearX
		dy := cy - nearY

		dSq := dx*dx + dy*dy
		if dSq < minDistSq {
			minDistSq = dSq
		}
	}

	// Only Sqrt once at the very end
	minDist := float32(math.Sqrt(minDistSq))
	normDist := 1.0 - (minDist / sightDist)
	if normDist < 0 {
		normDist = 0
	} // Clamp to prevent negative values

	return tanhf(normDist * 3.0)
}

func calculateLocalFoodPerCapita(ctx *SensorContext, params *Parameters) float32 {
	foodCount := float64(len(ctx.SightFoodIDs))
	creatureCount := float64(math.Max(1, float64(len(ctx.LocalCreatureIDs))))

	ratio := foodCount / creatureCount
	saturation := 4.0
	input := (ratio - 1.0) / saturation
	return tanhf(float32(input) * 2.0)
}

// calculateTouching returns 1 if the nearest creature in the forward FOV is
// physically touching this creature (centres within combined radii), -1 otherwise.
func calculateTouching(c Creature, p *Population, ctx *SensorContext) float32 {
	if p == nil {
		return -1
	}
	fwdX, fwdY := world.HeadingToVec(c.Heading)

	var bestDistSq float32 = math.MaxFloat32
	var bestDist, bestRadius float32
	found := false

	for _, id := range ctx.SightCreatureIDs {
		if id == c.Id {
			continue
		}
		other, ok := p.Creatures[id]
		if !ok || !other.Alive {
			continue
		}

		dx, dy := other.Loc.X-c.Loc.X, other.Loc.Y-c.Loc.Y
		d2 := dx*dx + dy*dy
		if d2 == 0 || d2 >= bestDistSq {
			continue
		}

		dist := float32(math.Sqrt(float64(d2)))
		dot := (fwdX*dx + fwdY*dy) / dist
		if dot < c.halfFOVCos {
			continue
		}

		bestDistSq = d2
		bestDist = dist
		bestRadius = other.Radius
		found = true
	}

	if !found {
		return -1
	}
	if bestDist <= c.Radius+bestRadius {
		return 1
	}
	return -1
}

func fastDist(dx, dy float32) float32 {
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return 0.9604*dx + 0.3978*dy
	}
	return 0.9604*dy + 0.3978*dx
}
