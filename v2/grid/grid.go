package grid

import (
	"fmt"
	"gopop/v2/utils"
	"math"
	"math/rand"
)

const (
	EMPTY int = iota
	WALL
	// FOOD
	RESERVED_CELL_TYPES
)

type MapType int

const (
	MIDDLE_WALL MapType = iota
)

type Grid struct {
	Data          [][]int
	WallLocations []Coord
	Type          MapType
}

func NewGrid(xSize, ySize int, gridMap int) *Grid {
	data := make([][]int, xSize)
	for i := int(0); i < xSize; i++ {
		data[i] = make([]int, ySize)
	}

	g := &Grid{
		Data: data,
		Type: MapType(gridMap),
	}
	g.CreateWall()
	return g
}

func (grid Grid) SizeX() int {
	return len(grid.Data)
}

func (grid Grid) SizeY() int {
	return len(grid.Data[0])
}

func (grid *Grid) ZeroFill() {
	for x := range grid.Data {
		for y := range grid.Data[x] {
			grid.Data[x][y] = EMPTY
		}
	}
	grid.WallLocations = []Coord{}
}

func (g *Grid) CreateWall() {
	switch g.Type {
	case MIDDLE_WALL:
		// Width 1 wall
		width := 5
		center := g.SizeX() / 2
		minX := center - width/2
		maxX := center + width/2
		minY := g.SizeY() / 4
		maxY := minY + g.SizeY()/2
		g.DrawBox(minX, minY, maxX, maxY)
	}
}

func (g *Grid) DrawBox(minX, minY, maxX, maxY int) {
	for x := minX; x < maxX; x++ {
		for y := minY; y < maxY; y++ {
			coord := Coord{X: x, Y: y}
			g.Set(coord, WALL)
			g.WallLocations = append(g.WallLocations, coord)
		}
	}
}

func (grid Grid) Print() {
	// We print from top left to bottom right
	str := ""
	fmt.Printf("%d\n", grid.SizeX())
	fmt.Printf("%d\n", grid.SizeY())
	for y := grid.SizeY() - 1; y > int(0); y-- {
		for x := int(0); x < grid.SizeX(); x++ {
			str += fmt.Sprintf("%d", grid.Data[x][y])
		}
		str += "\n"
	}
	fmt.Print(str)
}

func (grid Grid) IsInBounds(loc Coord) bool {
	return 0 <= loc.X && loc.X < grid.SizeX() && 0 <= loc.Y && loc.Y < grid.SizeY()
}

func (grid Grid) IsEmptyAt(loc Coord) bool {
	return grid.Data[loc.X][loc.Y] == EMPTY
}

func (grid Grid) IsOccupiedAt(loc Coord) bool {
	return grid.Data[loc.X][loc.Y] != EMPTY && grid.Data[loc.X][loc.Y] != WALL
}

// IsBarrierAt()
func (grid Grid) IsBorder(loc Coord) bool {
	return loc.X == 0 || loc.X == grid.SizeX()-1 || loc.Y == 0 || loc.Y == grid.SizeY()-1
}

func (grid Grid) At(loc Coord) int {
	return grid.Data[loc.X][loc.Y]
}

func (grid *Grid) Set(loc Coord, id int) {
	grid.Data[loc.X][loc.Y] = id
}

func (g Grid) FindEmptyLocation() Coord {
	loc := Coord{}
	for {
		loc.X = rand.Intn(g.SizeX() - 1)
		loc.Y = rand.Intn(g.SizeY() - 1)
		if g.IsEmptyAt(loc) {
			return loc
		}
	}
}

func (g Grid) FindEmptyLocationRightHalf() Coord {
	loc := Coord{}
	for {
		loc.X = int(g.SizeX()/2 + rand.Intn(g.SizeX()/2) - 1)
		loc.Y = rand.Intn(g.SizeY() - 1)
		if g.IsEmptyAt(loc) {
			return loc
		}
	}
}

func (g Grid) GetNeighbours(loc Coord, radius float32) []Coord {
	coords := []Coord{}
	for dx := -utils.Min(int(radius), loc.X); dx <= utils.Min(int(radius), g.SizeX()-loc.X-1); dx++ {
		x := loc.X + dx
		extentY := int(math.Sqrt(float64(radius)*float64(radius) - float64(dx*dx)))
		for dy := -utils.Min(extentY, loc.Y); dy <= utils.Min(int(radius), g.SizeY()-loc.Y-1); dy++ {
			y := loc.Y + dy
			coords = append(coords, Coord{x, y})
		}
	}
	return coords
}

func (g Grid) CountNeighbours(loc Coord, radius float32, fn func(g Grid, x, y int) int) int {
	sum := 0
	for dx := -utils.Min(int(radius), loc.X); dx <= utils.Min(int(radius), g.SizeX()-loc.X-1); dx++ {
		x := loc.X + dx
		extentY := int(math.Sqrt(float64(radius)*float64(radius) - float64(dx*dx)))
		for dy := -utils.Min(extentY, loc.Y); dy <= utils.Min(int(radius), g.SizeY()-loc.Y-1); dy++ {
			y := loc.Y + dy
			sum += fn(g, x, y)
		}
	}
	return sum
}

func (g Grid) DensityNeighbours(loc Coord, radius float32, fn func(g Grid, x, y int) int) float32 {
	area := 0
	sum := 0
	for dx := -utils.Min(int(radius), loc.X); dx <= utils.Min(int(radius), g.SizeX()-loc.X-1); dx++ {
		x := loc.X + dx
		extentY := int(math.Sqrt(float64(radius)*float64(radius) - float64(dx*dx)))
		for dy := -utils.Min(extentY, loc.Y); dy <= utils.Min(int(radius), g.SizeY()-loc.Y-1); dy++ {
			y := loc.Y + dy
			area++
			sum += fn(g, x, y)
		}
	}
	return float32(sum) / float32(area)
}

func (g Grid) DensityAxis(loc Coord, radius float32, lastMoveDir Dir, fn func(g Grid, x, y int, dir Dir) float32) float32 {
	sum := float32(0)
	for dx := -utils.Min(int(radius), loc.X); dx <= utils.Min(int(radius), g.SizeX()-loc.X-1); dx++ {
		x := loc.X + dx
		extentY := int(math.Sqrt(float64(radius)*float64(radius) - float64(dx*dx)))
		for dy := -utils.Min(extentY, loc.Y); dy <= utils.Min(int(radius), g.SizeY()-loc.Y-1); dy++ {
			y := loc.Y + dy
			sum += fn(g, x, y, lastMoveDir)
		}
	}
	maxSumMag := float32(6 * radius)
	if sum > maxSumMag {
		fmt.Printf("Population density is impossibly large: %f", sum)
		sum = maxSumMag
	} else if sum < -maxSumMag {
		fmt.Printf("Population density is impossibly small: %f", sum)
		sum = -maxSumMag
	}
	return sum / maxSumMag
}
