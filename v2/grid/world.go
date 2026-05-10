package grid

import (
	"math"
	"math/rand"
)

// StartingCreatureID is the first valid creature ID; 0 is reserved for "empty".
const StartingCreatureID = 1

type FoodSpawnMode int

const (
	SpawnRandom FoodSpawnMode = iota
	SpawnClustered
	SpawnTrail
)

// Wall is an axis-aligned rectangular obstacle in world-space.
type Wall struct {
	X, Y, W, H float64
}

// World is the continuous-space simulation arena. It tracks creature and food
// positions via spatial-hash buckets for efficient neighbourhood queries.
type World struct {
	Width, Height float64
	Walls         []Wall

	creaturePos map[int]Position
	cBuckets    map[[2]int]map[int]bool

	foodPos      map[int]Position
	fBuckets     map[[2]int]map[int]bool
	nextFoodID   int
	spawnMode    FoodSpawnMode
	lastPatchPos Position
	lastAngle    float64
	bucketSize   float64
}

func NewWorld(width, height float64, wallType int) *World {
	w := &World{
		Width:       width,
		Height:      height,
		creaturePos: make(map[int]Position),
		cBuckets:    make(map[[2]int]map[int]bool),
		foodPos:     make(map[int]Position),
		fBuckets:    make(map[[2]int]map[int]bool),
		bucketSize:  20.0,
		spawnMode:   SpawnClustered,
	}
	w.createWalls(wallType)
	return w
}

func (w *World) bucketKey(pos Position) [2]int {
	return [2]int{
		int(math.Floor(pos.X / w.bucketSize)),
		int(math.Floor(pos.Y / w.bucketSize)),
	}
}

// --- Creature spatial operations ---

func (w *World) AddCreature(id int, pos Position) {
	w.creaturePos[id] = pos
	key := w.bucketKey(pos)
	if w.cBuckets[key] == nil {
		w.cBuckets[key] = make(map[int]bool)
	}
	w.cBuckets[key][id] = true
}

func (w *World) MoveCreature(id int, newPos Position) {
	if oldPos, ok := w.creaturePos[id]; ok {
		delete(w.cBuckets[w.bucketKey(oldPos)], id)
	}
	w.creaturePos[id] = newPos
	key := w.bucketKey(newPos)
	if w.cBuckets[key] == nil {
		w.cBuckets[key] = make(map[int]bool)
	}
	w.cBuckets[key][id] = true
}

func (w *World) RemoveCreature(id int) {
	if pos, ok := w.creaturePos[id]; ok {
		delete(w.cBuckets[w.bucketKey(pos)], id)
		delete(w.creaturePos, id)
	}
}

func (w *World) GetCreaturePos(id int) (Position, bool) {
	pos, ok := w.creaturePos[id]
	return pos, ok
}

// GetCreaturesInRadius returns IDs of all creatures within the given radius of center.
func (w *World) GetCreaturesInRadius(center Position, radius float64) []int {
	minBx := int(math.Floor((center.X - radius) / w.bucketSize))
	maxBx := int(math.Floor((center.X + radius) / w.bucketSize))
	minBy := int(math.Floor((center.Y - radius) / w.bucketSize))
	maxBy := int(math.Floor((center.Y + radius) / w.bucketSize))
	rSq := radius * radius
	var result []int
	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			for id := range w.cBuckets[[2]int{bx, by}] {
				pos := w.creaturePos[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y
				if dx*dx+dy*dy <= rSq {
					result = append(result, id)
				}
			}
		}
	}
	return result
}

// GetCreaturesInCone returns IDs of creatures within maxDist that lie inside
// the cone defined by heading ± halfFOVCos (cosine of the half-angle).
func (w *World) GetCreaturesInCone(center Position, heading float64, halfFOVCos float64, maxDist float64) []int {
	fwdX, fwdY := HeadingToVec(heading)
	var result []int
	for _, id := range w.GetCreaturesInRadius(center, maxDist) {
		pos := w.creaturePos[id]
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		if dx == 0 && dy == 0 {
			continue
		}
		if CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
			result = append(result, id)
		}
	}
	return result
}

// --- Food spatial operations ---

// AddFood places a food item at pos and returns its ID.
func (w *World) AddFood(pos Position) int {
	w.nextFoodID++
	id := w.nextFoodID
	w.foodPos[id] = pos
	key := w.bucketKey(pos)
	if w.fBuckets[key] == nil {
		w.fBuckets[key] = make(map[int]bool)
	}
	w.fBuckets[key][id] = true
	return id
}

func (w *World) RemoveFood(id int) {
	if pos, ok := w.foodPos[id]; ok {
		delete(w.fBuckets[w.bucketKey(pos)], id)
		delete(w.foodPos, id)
	}
}

func (w *World) GetFoodPos(id int) Position {
	return w.foodPos[id]
}

func (w *World) FoodCount() int {
	return len(w.foodPos)
}

// FoodPositions returns the live food map (read-only use; do not mutate).
func (w *World) FoodPositions() map[int]Position {
	return w.foodPos
}

// GetFoodInRadius returns IDs of food items within radius of center.
func (w *World) GetFoodInRadius(center Position, radius float64) []int {
	minBx := int(math.Floor((center.X - radius) / w.bucketSize))
	maxBx := int(math.Floor((center.X + radius) / w.bucketSize))
	minBy := int(math.Floor((center.Y - radius) / w.bucketSize))
	maxBy := int(math.Floor((center.Y + radius) / w.bucketSize))
	rSq := radius * radius
	var result []int
	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			for id := range w.fBuckets[[2]int{bx, by}] {
				pos := w.foodPos[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y
				if dx*dx+dy*dy <= rSq {
					result = append(result, id)
				}
			}
		}
	}
	return result
}

// GetFoodInCone returns food IDs within maxDist inside the heading cone.
func (w *World) GetFoodInCone(center Position, heading float64, halfFOVCos float64, maxDist float64) []int {
	fwdX, fwdY := HeadingToVec(heading)
	var result []int
	for _, id := range w.GetFoodInRadius(center, maxDist) {
		pos := w.foodPos[id]
		dx := pos.X - center.X
		dy := pos.Y - center.Y
		if dx == 0 && dy == 0 {
			continue
		}
		if CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
			result = append(result, id)
		}
	}
	return result
}

// --- World geometry ---

func (w *World) IsWall(pos Position) bool {
	for i := range w.Walls {
		wall := &w.Walls[i]
		if pos.X >= wall.X && pos.X < wall.X+wall.W &&
			pos.Y >= wall.Y && pos.Y < wall.Y+wall.H {
			return true
		}
	}
	return false
}

func (w *World) IsInBounds(pos Position) bool {
	return pos.X >= 0 && pos.X < w.Width && pos.Y >= 0 && pos.Y < w.Height
}

func (w *World) ClampToBounds(pos Position) Position {
	if pos.X < 0 {
		pos.X = 0
	} else if pos.X >= w.Width {
		pos.X = w.Width - 0.001
	}
	if pos.Y < 0 {
		pos.Y = 0
	} else if pos.Y >= w.Height {
		pos.Y = w.Height - 0.001
	}
	return pos
}

func (w *World) SizeX() int { return int(w.Width) }
func (w *World) SizeY() int { return int(w.Height) }

func (w *World) createWalls(wallType int) {
	const wallThickness = 10.0
	switch wallType {
	case 1: // CROSS_WALL
		cx := w.Width / 2
		cy := w.Height / 2
		w.Walls = append(w.Walls, Wall{
			X: cx - wallThickness/2, Y: w.Height / 4,
			W: wallThickness, H: w.Height / 2,
		})
		w.Walls = append(w.Walls, Wall{
			X: w.Width / 4, Y: cy - wallThickness/2,
			W: w.Width / 2, H: wallThickness,
		})
	}
}

func (w *World) FindEmptyLocation() (Position, bool) {
	for i := 0; i < 200; i++ {
		pos := Position{X: rand.Float64() * w.Width, Y: rand.Float64() * w.Height}
		if !w.IsWall(pos) {
			return pos, true
		}
	}
	return Position{}, false
}

// SpawnFood is now a high-level dispatcher
func (w *World) SpawnFood(n int, patchRadius float64, patchSize int) {
	switch w.spawnMode {
	case SpawnRandom:
		w.SpawnRandom(n)
	case SpawnClustered:
		w.spawnClustered(n, patchRadius, patchSize)
	case SpawnTrail:
		w.spawnTrail(n, patchRadius, patchSize)
	}
}
func (w *World) spawnTrail(n int, patchRadius float64, patchSize int) {
	totalSpawned := 0
	stepDist := patchRadius * 1.5

	for totalSpawned < n {
		var nextCenter Position

		if w.lastPatchPos.X == 0 && w.lastPatchPos.Y == 0 {
			nextCenter, _ = w.FindEmptyLocation()
			w.lastAngle = rand.Float64() * 2 * math.Pi
		} else {
			foundValid := false
			for attempts := 0; attempts < 15; attempts++ {
				angleOffset := (rand.Float64() * math.Pi) - (math.Pi / 2)
				currentAngle := w.lastAngle + angleOffset

				candidate := Position{
					X: w.lastPatchPos.X + math.Cos(currentAngle)*stepDist,
					Y: w.lastPatchPos.Y + math.Sin(currentAngle)*stepDist,
				}

				if w.IsInBounds(candidate) && !w.IsWall(candidate) {
					nextCenter = candidate
					w.lastAngle = currentAngle
					foundValid = true
					break
				}

				w.lastAngle += math.Pi / 4
			}

			if !foundValid {
				nextCenter, _ = w.FindEmptyLocation()
				w.lastAngle = rand.Float64() * 2 * math.Pi
			}
		}

		w.lastPatchPos = nextCenter
		totalSpawned += w.placeClusterAt(nextCenter, n-totalSpawned, patchRadius, patchSize)
	}
}

func (w *World) placeClusterAt(center Position, remainingN int, radius float64, size int) int {
	spawned := 0
	for i := 0; i < size && spawned < remainingN; i++ {
		angle := rand.Float64() * 2 * math.Pi
		dist := radius * math.Sqrt(rand.Float64())
		pos := Position{
			X: center.X + math.Cos(angle)*dist,
			Y: center.Y + math.Sin(angle)*dist,
		}
		if w.IsInBounds(pos) && !w.IsWall(pos) {
			w.AddFood(pos)
			spawned++
		}
	}
	return spawned
}

func (w *World) SpawnRandom(n int) {
	for i := 0; i < n; i++ {
		pos, ok := w.FindEmptyLocation()
		if ok {
			w.AddFood(pos)
		}
	}
}

func (w *World) spawnClustered(n int, patchRadius float64, patchSize int) {
	totalSpawned := 0
	for totalSpawned < n {
		seed, ok := w.FindEmptyLocation()
		if !ok {
			break
		}
		totalSpawned += w.placeClusterAt(seed, n-totalSpawned, patchRadius, patchSize)
	}
}
