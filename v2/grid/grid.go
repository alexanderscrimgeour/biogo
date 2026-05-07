package grid

import (
	"biogo/v2/utils"
	"fmt"
	"math"
	"math/rand"
)

const (
	EMPTY int = iota
	WALL
	FOOD
	RESERVED_CELL_TYPES
)

type MapType int

const (
	MIDDLE_WALL MapType = iota
	CROSS_WALL
)

type Grid struct {
	Data          [][]int
	WallLocations []Coord
	FoodLocations []Coord
	Torodial      bool
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
	grid.FoodLocations = []Coord{}
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
	case CROSS_WALL:
		width := 5
		centerX := g.SizeX() / 2
		centerY := g.SizeY() / 2

		// Vertical Bar: narrow X range, full Y range (middle 50%)
		g.DrawBox(centerX-width/2, g.SizeY()/4, centerX+width/2, 3*g.SizeY()/4)

		// Horizontal Bar: full X range (middle 50%), narrow Y range
		// NOTE: Use centerX logic for X and centerY logic for Y
		g.DrawBox(g.SizeX()/4, centerY-width/2, 3*g.SizeX()/4, centerY+width/2)
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
	return grid.Data[loc.X][loc.Y] >= RESERVED_CELL_TYPES
}

func (grid Grid) IsFood(loc Coord) bool {
	return grid.Data[loc.X][loc.Y] == FOOD
}
func (g *Grid) SpawnFood(n int) {
	totalSpawned := 0
	patchSize := 25
	radius := 10

	for totalSpawned < n {
		// 1. Pick a new seed for this patch
		seed, ok := g.FindEmptyLocation()
		if !ok {
			break
		}

		for i := 0; i < patchSize && totalSpawned < n; i++ {
			placed := false
			for attempts := 0; attempts < 20; attempts++ {
				offsetX := rand.Intn(radius*2+1) - radius
				offsetY := rand.Intn(radius*2+1) - radius
				loc := Coord{X: seed.X + offsetX, Y: seed.Y + offsetY}

				if g.IsInBounds(loc) && g.At(loc) == EMPTY {
					g.Data[loc.X][loc.Y] = FOOD
					g.FoodLocations = append(g.FoodLocations, loc)
					totalSpawned++
					placed = true
					break
				}
			}

			if !placed {
				break
			}
		}
	}
}

func (g *Grid) RemoveFood(loc Coord) {
	g.Data[loc.X][loc.Y] = EMPTY
	for i, fl := range g.FoodLocations {
		if fl == loc {
			g.FoodLocations = append(g.FoodLocations[:i], g.FoodLocations[i+1:]...)
			return
		}
	}
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

func (g Grid) FindEmptyLocation() (Coord, bool) {
	for i := 0; i < 100; i++ {
		loc := Coord{X: rand.Intn(g.SizeX()), Y: rand.Intn(g.SizeY())}
		if g.IsEmptyAt(loc) {
			return loc, true
		}
	}
	for x := 0; x < g.SizeX(); x++ {
		for y := 0; y < g.SizeY(); y++ {
			loc := Coord{X: x, Y: y}
			if g.IsEmptyAt(loc) {
				return loc, true
			}
		}
	}
	return Coord{}, false
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
		// fmt.Printf("Population density is impossibly large: %f", sum)
		sum = maxSumMag
	} else if sum < -maxSumMag {
		// fmt.Printf("Population density is impossibly small: %f", sum)
		sum = -maxSumMag
	}
	return sum / maxSumMag
}

func (g *Grid) WrapCoords(c Coord) Coord {
	w := len(g.Data)
	h := len(g.Data[0])

	return Coord{
		X: (c.X%w + w) % w,
		Y: (c.Y%h + h) % h,
	}
}
