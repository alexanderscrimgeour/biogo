package simulation

import (
	"fmt"
	"gopop/v2/grid"
	"gopop/v2/utils"
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

	SENSOR_COUNT
)

func (c Creature) GetSensor(sensorID byte, g *grid.Grid, p *Population, simStep int) float32 {
	var output float32
	switch sensorID {
	case AGE:
		output = float32(c.Age / Params.MaxAge)

	case ENERGY:
		output = float32(c.Energy / float32(c.Genome.MaxEnergy))

	case BOUNDARY_DIST:
		distX := utils.Min(c.Loc.X, Params.GridWidth-c.Loc.X-1)
		distY := utils.Min(c.Loc.Y, Params.GridHeight-c.Loc.Y-1)
		closest := utils.Min(distX, distY)
		maxPossible := utils.Max(Params.GridWidth/2-1, Params.GridHeight/2-1)
		output = float32(closest / maxPossible)

	case BOUNDARY_DIST_X:
		distX := utils.Min(c.Loc.X, Params.GridWidth-c.Loc.X-1)
		output = float32(distX) / float32(Params.GridWidth/2)

	case BOUNDARY_DIST_Y:
		distY := utils.Min(c.Loc.Y, Params.GridHeight-c.Loc.Y-1)
		output = float32(distY) / float32(Params.GridHeight/2)

	case LAST_MOVE_DIR_X:
		if c.LastMoveDir.X == 0 {
			output = 0.5
		} else {
			if c.LastMoveDir.X == -1 {
				output = 0
			} else {
				output = 1
			}
		}

	case LAST_MOVE_DIR_Y:
		if c.LastMoveDir.Y == 0 {
			output = 0.5
		} else {
			if c.LastMoveDir.Y == -1 {
				output = 0
			} else {
				output = 1
			}
		}

	case LOC_X:
		output = float32(c.Loc.X) / float32(Params.GridWidth-1)

	case LOC_Y:
		output = float32(c.Loc.Y) / float32(Params.GridHeight-1)

	case OSC1:
		val := int(c.Genome.OscPeriod)
		if val == 0 {
			val += 1
		}
		phase := float64(simStep % val / val)
		factor := math.Cos(phase * 2 * math.Pi)
		factor += 1
		factor /= 2
		// Clip round off error
		output = utils.RestrictFloat32(0, 1, float32(factor))
	case POPULATION_LOCAL_DENSITY:
		output = getLocalPopulationDensity(c.Loc, g)

	case POPULATION_FORWARD:
		output = getPopulationDensityAlongAxis(c.Loc, g, c.LastMoveDir)

	case POPULATION_LR:
		output = getPopulationDensityAlongAxis(c.Loc, g, c.LastMoveDir.Rotate90CW())

	case SIGHT_POPULATION_FORWARD:
		output = calculateSightPopFwd(c, g)

	case GENETIC_SIM_FORWARD:
		newLoc := grid.Coord{
			X: c.Loc.X + c.LastMoveDir.X,
			Y: c.Loc.Y + c.LastMoveDir.Y,
		}
		if g.IsInBounds(newLoc) && g.IsOccupiedAt(newLoc) {
			otherCreatureId := g.Data[newLoc.X][newLoc.Y]
			if otherCreatureId-grid.RESERVED_CELL_TYPES < 0 || otherCreatureId-grid.RESERVED_CELL_TYPES > len(p.Creatures) {
				fmt.Println("\nError: %d with %d reserved\n", otherCreatureId, grid.RESERVED_CELL_TYPES)
			} else {
				otherCreature := p.Creatures[otherCreatureId-grid.RESERVED_CELL_TYPES]
				if otherCreature.Alive {
					//TODO: This function performs very poorly, replace
					output = GenomeSimilarity(*c.Genome, *otherCreature.Genome)
				}
			}
		}
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
	if toTest > 0 && !g.IsInBounds(newLoc) {
		return float32(c.Genome.SightDistance) / float32(c.Genome.SightDistance)
	} else {
		return float32(count) / float32(c.Genome.SightDistance)
	}
}

func getLocalPopulationDensity(loc grid.Coord, g *grid.Grid) float32 {

	delta := func(g grid.Grid, x, y int) int {
		if g.IsOccupiedAt(grid.Coord{X: x, Y: y}) {
			return 1
		}
		return 0
	}
	return g.DensityNeighbours(loc, float32(Params.PopulationSensorRadius), delta)
}

func getPopulationDensityAlongAxis(loc grid.Coord, g *grid.Grid, lastMoveDir grid.Dir) float32 {
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

	return g.DensityAxis(loc, float32(Params.PopulationSensorRadius), lastMoveDir, delta)
}
