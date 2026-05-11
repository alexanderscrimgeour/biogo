package grid

import (
	"math"
	"math/rand"
)

// StartingCreatureID is the first valid creature ID; 0 is reserved for "empty".
const StartingCreatureID = 1

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

	foodPos    map[int]Position
	foodMass   map[int]float32
	fBuckets   map[[2]int]map[int]bool
	nextFoodID int
	bucketSize float64

	// Gaussian fountain system: 3-5 drifting points that food spawns around.
	Fountains      []Position
	fountainAngles []float64
}

func NewWorld(width, height float64, wallType int) *World {
	w := &World{
		Width:       width,
		Height:      height,
		creaturePos: make(map[int]Position),
		cBuckets:    make(map[[2]int]map[int]bool),
		foodPos:     make(map[int]Position),
		foodMass:    make(map[int]float32),
		fBuckets:    make(map[[2]int]map[int]bool),
		bucketSize:  20.0,
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

// AddFood places a food item with the given mass at pos and returns its ID.
func (w *World) AddFood(pos Position, mass float32) int {
	w.nextFoodID++
	id := w.nextFoodID
	w.foodPos[id] = pos
	w.foodMass[id] = mass
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
		delete(w.foodMass, id)
	}
}

func (w *World) GetFoodPos(id int) Position {
	return w.foodPos[id]
}

// GetFoodMass returns the remaining mass of food item id.
func (w *World) GetFoodMass(id int) float32 {
	return w.foodMass[id]
}

// ReduceFoodMass subtracts amount from food item id's mass.
// If the remaining mass drops to zero or below the item is automatically removed.
// Returns the remaining mass (0 if removed).
func (w *World) ReduceFoodMass(id int, amount float32) float32 {
	remaining := w.foodMass[id] - amount
	if remaining <= 0 {
		w.RemoveFood(id)
		return 0
	}
	w.foodMass[id] = remaining
	return remaining
}

// TotalFoodMass returns the sum of all food item masses currently in the world.
func (w *World) TotalFoodMass() float64 {
	total := float64(0)
	for _, m := range w.foodMass {
		total += float64(m)
	}
	return total
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

// --- Fountain system ---

// InitFountains places count fountain points at random valid locations with random drift angles.
func (w *World) InitFountains(count int) {
	w.Fountains = make([]Position, count)
	w.fountainAngles = make([]float64, count)
	for i := range w.Fountains {
		pos, ok := w.FindEmptyLocation()
		if !ok {
			pos = Position{X: w.Width / 2, Y: w.Height / 2}
		}
		w.Fountains[i] = pos
		w.fountainAngles[i] = rand.Float64() * 2 * math.Pi
	}
}

// StepFountains advances each fountain by driftSpeed units along its current angle,
// applying a small random angular perturbation each step. Fountains bounce off world
// edges and walls.
func (w *World) StepFountains(driftSpeed float64) {
	for i := range w.Fountains {
		w.fountainAngles[i] += (rand.Float64() - 0.5) * 0.1

		newPos := Position{
			X: w.Fountains[i].X + math.Cos(w.fountainAngles[i])*driftSpeed,
			Y: w.Fountains[i].Y + math.Sin(w.fountainAngles[i])*driftSpeed,
		}

		if w.IsInBounds(newPos) && !w.IsWall(newPos) {
			w.Fountains[i] = newPos
		} else {
			// Bounce: reverse direction with a small random jitter so fountains
			// don't get trapped oscillating against a boundary.
			w.fountainAngles[i] += math.Pi + (rand.Float64()-0.5)*math.Pi*0.25
		}
	}
}

// SpawnFood places n food items (each with the given mass) sampled from Gaussian
// distributions centred on each fountain. Each item is assigned to a fountain
// uniformly at random, then offset by a 2-D normal with standard deviation sigma.
// Items that fall outside the world bounds or inside a wall are retried; if the retry
// budget is exhausted the remainder are placed uniformly at random so the requested
// count is always satisfied.
func (w *World) SpawnFood(n int, sigma float64, mass float32) {
	if len(w.Fountains) == 0 || n <= 0 {
		w.SpawnRandom(n, mass)
		return
	}

	maxAttempts := n * 20
	spawned := 0
	for attempts := 0; spawned < n && attempts < maxAttempts; attempts++ {
		fi := rand.Intn(len(w.Fountains))
		center := w.Fountains[fi]
		pos := Position{
			X: center.X + rand.NormFloat64()*sigma,
			Y: center.Y + rand.NormFloat64()*sigma,
		}
		if w.IsInBounds(pos) && !w.IsWall(pos) {
			w.AddFood(pos, mass)
			spawned++
		}
	}

	// Fallback for any items not placed via Gaussian (e.g. fountain near edge).
	if spawned < n {
		w.SpawnRandom(n-spawned, mass)
	}
}

// SpawnRandom places n food items (each with the given mass) at uniformly random valid positions.
func (w *World) SpawnRandom(n int, mass float32) {
	for i := 0; i < n; i++ {
		pos, ok := w.FindEmptyLocation()
		if ok {
			w.AddFood(pos, mass)
		}
	}
}
