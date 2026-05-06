package simulation

import (
	"biogo/v2/grid"
	"biogo/v2/utils"
	"fmt"
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

	SENSOR_COUNT
)

func (c Creature) GetSensor(sensorID byte, g *grid.Grid, p *Population, simStep int, params *Parameters) float32 {
	var output float32
	switch sensorID {
	case AGE:
		output = float32(c.Age) / float32(params.MaxExpectedAge)

	case ENERGY:
		output = float32(c.Energy / float32(c.Genome.MaxEnergy))

	case BOUNDARY_DIST:
		distX := utils.Min(c.Loc.X, params.GridWidth-c.Loc.X-1)
		distY := utils.Min(c.Loc.Y, params.GridHeight-c.Loc.Y-1)
		closest := utils.Min(distX, distY)
		maxPossible := utils.Max(params.GridWidth/2-1, params.GridHeight/2-1)
		output = float32(closest / maxPossible)

	case BOUNDARY_DIST_X:
		distX := utils.Min(c.Loc.X, params.GridWidth-c.Loc.X-1)
		output = float32(distX) / float32(params.GridWidth/2)

	case BOUNDARY_DIST_Y:
		distY := utils.Min(c.Loc.Y, params.GridHeight-c.Loc.Y-1)
		output = float32(distY) / float32(params.GridHeight/2)

	case LAST_MOVE_DIR_X:
		if c.LastMoveDir.X == 0 {
			output = 0.5
		} else if c.LastMoveDir.X == -1 {
			output = 0
		} else {
			output = 1
		}

	case LAST_MOVE_DIR_Y:
		if c.LastMoveDir.Y == 0 {
			output = 0.5
		} else if c.LastMoveDir.Y == -1 {
			output = 0
		} else {
			output = 1
		}

	case LOC_X:
		output = float32(c.Loc.X) / float32(params.GridWidth-1)

	case LOC_Y:
		output = float32(c.Loc.Y) / float32(params.GridHeight-1)

	case OSC1:
		val := int(c.Genome.OscPeriod)
		if val == 0 {
			val += 1
		}
		phase := float64(simStep % val / val)
		factor := math.Cos(phase * 2 * math.Pi)
		factor += 1
		factor /= 2
		output = utils.RestrictFloat32(0, 1, float32(factor))

	case POPULATION_LOCAL_DENSITY:
		output = getLocalPopulationDensity(c.Loc, g, params)

	case POPULATION_FORWARD:
		output = getPopulationDensityAlongAxis(c.Loc, g, c.LastMoveDir, params)

	case POPULATION_LR:
		output = getPopulationDensityAlongAxis(c.Loc, g, c.LastMoveDir.Rotate90CW(), params)

	case SIGHT_POPULATION_FORWARD:
		output = calculateSightPopFwd(c, g)

	case GENETIC_SIM_FORWARD:
		newLoc := grid.Coord{
			X: c.Loc.X + c.LastMoveDir.X,
			Y: c.Loc.Y + c.LastMoveDir.Y,
		}
		if g.IsInBounds(newLoc) && g.IsOccupiedAt(newLoc) {
			otherCreatureId := g.Data[newLoc.X][newLoc.Y]
			if otherCreature, ok := p.Creatures[otherCreatureId]; ok {
				if otherCreature.Alive {
					output = GenomeSimilarity(*c.Genome, *otherCreature.Genome)
				}
			} else {
				fmt.Printf("\nError: creature id %d not found in population\n", otherCreatureId)
			}
		}

	case SIGHT_FOOD_FORWARD:
		output = calculateSightFoodFwd(c, g)

	case RANDOM:
		fallthrough
	default:
		output = rand.Float32()
	}
	if output < 0 || output > 1 {
		output = utils.RestrictFloat32(0, 1, output)
	}
	return output
}

// calculateSightFoodFwd returns 1.0 if food is immediately ahead, scaling down to ~0
// for food at max sight distance, and 0 if no food is visible in the sight line.
// The scan stops at walls or creatures.
func calculateSightFoodFwd(c Creature, g *grid.Grid) float32 {
	loc := grid.Coord{X: c.Loc.X + c.LastMoveDir.X, Y: c.Loc.Y + c.LastMoveDir.Y}
	dist := c.Genome.SightDistance
	for d := byte(1); d <= dist; d++ {
		if !g.IsInBounds(loc) {
			break
		}
		if g.IsFood(loc) {
			return 1.0 - float32(d-1)/float32(dist)
		}
		if !g.IsEmptyAt(loc) {
			break
		}
		loc = grid.Coord{X: loc.X + c.LastMoveDir.X, Y: loc.Y + c.LastMoveDir.Y}
	}
	return 0
}

func calculateSightPopFwd(c Creature, g *grid.Grid) float32 {
	count := 0
	newLoc := grid.Coord{
		X: c.Loc.X + c.LastMoveDir.X,
		Y: c.Loc.Y + c.LastMoveDir.Y,
	}
	toTest := c.Genome.SightDistance
	for toTest > 0 && g.IsInBounds(newLoc) && g.IsEmptyAt(newLoc) {
		count++
		newLoc = grid.Coord{
			X: newLoc.X + c.LastMoveDir.X,
			Y: newLoc.Y + c.LastMoveDir.Y,
		}
		toTest--
	}
	return float32(count) / float32(c.Genome.SightDistance)
}

func getLocalPopulationDensity(loc grid.Coord, g *grid.Grid, params *Parameters) float32 {
	delta := func(g grid.Grid, x, y int) int {
		if g.IsOccupiedAt(grid.Coord{X: x, Y: y}) {
			return 1
		}
		return 0
	}
	return g.DensityNeighbours(loc, float32(params.PopulationSensorRadius), delta)
}

func getPopulationDensityAlongAxis(loc grid.Coord, g *grid.Grid, lastMoveDir grid.Dir, params *Parameters) float32 {
	delta := func(g grid.Grid, x, y int, dir grid.Dir) float32 {
		tLoc := grid.Coord{X: x, Y: y}
		if tLoc != loc && g.IsOccupiedAt(tLoc) {
			offset := grid.GetDirection(loc, tLoc)
			posCos := grid.RaySameness(offset, dir)
			dist := float32(math.Sqrt(float64(offset.X*offset.X + offset.Y*offset.Y)))
			contrib := (1 / dist) * posCos
			return contrib
		}
		return 0
	}
	return g.DensityAxis(loc, float32(params.PopulationSensorRadius), lastMoveDir, delta)
}
