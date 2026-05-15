package world

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
// positions via SpatialHash grids for O(1) bucket lookup (flat array index vs
// map hashing).
type World struct {
	Width, Height float64
	Walls         []Wall

	creaturePos     []Position
	creatureActive  []bool
	creatureCount   int
	freeCreatureIDs []int
	cHash           *SpatialHash

	foodPos     []Position
	foodMass    []float32
	foodActive  []bool
	foodCount   int
	freeFoodIDs []int
	fHash       *SpatialHash

	meatPos     []Position
	meatMass    []float32
	meatActive  []bool
	meatCount   int
	freeMeatIDs []int
	mHash       *SpatialHash

	// Gaussian fountain system: 3-5 drifting points that food spawns around.
	Fountains      []Position
	fountainAngles []float64
}

func NewWorld(width, height float64, _ int) *World {
	const initialCapacity = 25000
	const creatureCapacity = 20000
	const cellSize = 100.0
	// Pre-allocate slot 0 so that IDs start at StartingCreatureID (1).
	w := &World{
		Width:           width,
		Height:          height,
		creaturePos:     make([]Position, 1, creatureCapacity),
		creatureActive:  make([]bool, 1, creatureCapacity),
		freeCreatureIDs: make([]int, 0, 100),
		cHash:           newSpatialHash(width, height, cellSize),
		foodPos:         make([]Position, 0, initialCapacity),
		foodMass:        make([]float32, 0, initialCapacity),
		foodActive:      make([]bool, 0, initialCapacity),
		freeFoodIDs:     make([]int, 0, 100),
		fHash:           newSpatialHash(width, height, cellSize),
		meatPos:         make([]Position, 0, 1024),
		meatMass:        make([]float32, 0, 1024),
		meatActive:      make([]bool, 0, 1024),
		freeMeatIDs:     make([]int, 0, 100),
		mHash:           newSpatialHash(width, height, cellSize),
	}
	// w.createWalls(wallType)
	return w
}

// --- Creature spatial operations ---

// AddCreature registers a creature at pos and returns its assigned ID.
func (w *World) AddCreature(pos Position) int {
	var id int
	if len(w.freeCreatureIDs) > 0 {
		lastIdx := len(w.freeCreatureIDs) - 1
		id = w.freeCreatureIDs[lastIdx]
		w.freeCreatureIDs = w.freeCreatureIDs[:lastIdx]
		w.creaturePos[id] = pos
		w.creatureActive[id] = true
	} else {
		id = len(w.creaturePos)
		w.creaturePos = append(w.creaturePos, pos)
		w.creatureActive = append(w.creatureActive, true)
	}
	w.creatureCount++
	w.cHash.Add(id, pos)
	return id
}

func (w *World) MoveCreature(id int, newPos Position) {
	if id < 0 || id >= len(w.creatureActive) || !w.creatureActive[id] {
		return
	}
	w.cHash.Move(id, w.creaturePos[id], newPos)
	w.creaturePos[id] = newPos
}

func (w *World) RemoveCreature(id int) {
	if id < 0 || id >= len(w.creatureActive) || !w.creatureActive[id] {
		return
	}
	w.cHash.Remove(id, w.creaturePos[id])
	w.creatureActive[id] = false
	w.creatureCount--
	w.freeCreatureIDs = append(w.freeCreatureIDs, id)
}

func (w *World) GetCreaturePos(id int) (Position, bool) {
	if id < 0 || id >= len(w.creatureActive) || !w.creatureActive[id] {
		return Position{}, false
	}
	return w.creaturePos[id], true
}

func (w *World) CreatureCount() int { return w.creatureCount }

func (w *World) GetCreaturesInRadius(center Position, radius float64, buffer []int) []int {
	return w.cHash.InRadius(center, radius, w.creaturePos, w.creatureActive, buffer)
}

// GetCreaturesInCone returns IDs of creatures within maxDist that lie inside
// the cone defined by heading ± halfFOVCos (cosine of the half-angle).
func (w *World) GetCreaturesInCone(center Position, heading float64, halfFOVCos float64, maxDist float64, buffer []int) []int {
	return w.cHash.InCone(center, heading, halfFOVCos, maxDist, w.creaturePos, w.creatureActive, buffer)
}

// --- Food spatial operations ---

// AddFood places a food item with the given mass at pos and returns its ID.
func (w *World) AddFood(pos Position, mass float32) int {
	var id int
	if len(w.freeFoodIDs) > 0 {
		lastIdx := len(w.freeFoodIDs) - 1
		id = w.freeFoodIDs[lastIdx]
		w.freeFoodIDs = w.freeFoodIDs[:lastIdx]
		w.foodPos[id] = pos
		w.foodMass[id] = mass
		w.foodActive[id] = true
	} else {
		id = len(w.foodPos)
		w.foodPos = append(w.foodPos, pos)
		w.foodMass = append(w.foodMass, mass)
		w.foodActive = append(w.foodActive, true)
	}

	w.foodCount++
	w.fHash.Add(id, pos)
	return id
}

func (w *World) RemoveFood(id int) {
	if id < 0 || id >= len(w.foodActive) || !w.foodActive[id] {
		return
	}
	w.fHash.Remove(id, w.foodPos[id])
	w.foodActive[id] = false
	w.foodCount--
	w.freeFoodIDs = append(w.freeFoodIDs, id)
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

// TotalFoodMass returns the sum of all active food item masses in the world.
func (w *World) TotalFoodMass() float64 {
	total := float64(0)
	for id, active := range w.foodActive {
		if active {
			total += float64(w.foodMass[id])
		}
	}
	return total
}

func (w *World) FoodCount() int {
	return w.foodCount
}

// FoodPositions returns the live food map (read-only use; do not mutate).
func (w *World) FoodPositions() []Position {
	return w.foodPos
}

// GetFoodInRadius returns IDs of food items within radius of center.
func (w *World) GetFoodInRadius(center Position, radius float64, buffer []int) []int {
	return w.fHash.InRadius(center, radius, w.foodPos, w.foodActive, buffer)
}

func (w *World) GetFoodInCone(center Position, heading float64, halfFOVCos float64, maxDist float64, buffer []int) []int {
	return w.fHash.InCone(center, heading, halfFOVCos, maxDist, w.foodPos, w.foodActive, buffer)
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


func (w *World) FindEmptyLocation() (Position, bool) {
	for i := 0; i < 200; i++ {
		pos := Position{X: rand.Float64() * w.Width, Y: rand.Float64() * w.Height}
		if !w.IsWall(pos) {
			return pos, true
		}
	}
	return Position{}, false
}

// FindEmptyLocationNear returns a random non-wall position within radius of center.
// Falls back to FindEmptyLocation if no nearby position is found.
func (w *World) FindEmptyLocationNear(center Position, radius float64) Position {
	for i := 0; i < 30; i++ {
		angle := rand.Float64() * 2 * math.Pi
		dist := rand.Float64() * radius
		pos := Position{X: center.X + math.Cos(angle)*dist, Y: center.Y + math.Sin(angle)*dist}
		pos = w.ClampToBounds(pos)
		if !w.IsWall(pos) {
			return pos
		}
	}
	pos, _ := w.FindEmptyLocation()
	return pos
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
	if n <= 0 {
		return
	}

	randomScatterFactor := 0.15 // 15% of food spawns anywhere
	randomCount := int(float64(n) * randomScatterFactor)
	clusterCount := n - randomCount

	if randomCount > 0 {
		w.SpawnRandom(randomCount, mass)
	}

	if len(w.Fountains) == 0 {
		w.SpawnRandom(clusterCount, mass)
		return
	}

	maxAttempts := clusterCount * 20
	spawned := 0
	for attempts := 0; spawned < clusterCount && attempts < maxAttempts; attempts++ {
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

	// 4. Fallback for any fountain items that failed geometry checks
	if spawned < clusterCount {
		w.SpawnRandom(clusterCount-spawned, mass)
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

// TempCold and TempWarm define the ambient temperature range across the world's
// Y axis. The top 20% is TempCold, the bottom 20% is TempWarm, with a linear
// gradient in between.
const (
	TempCold = float32(10.0)
	TempWarm = float32(40.0)
)

// TemperatureAt returns the ambient temperature in Celsius at world y-coordinate y.
func (w *World) TemperatureAt(y float64) float32 {
	const coldBandEnd = 0.2
	const warmBandStart = 0.8
	norm := y / w.Height
	if norm <= coldBandEnd {
		return TempCold
	}
	if norm >= warmBandStart {
		return TempWarm
	}
	t := float32((norm - coldBandEnd) / (warmBandStart - coldBandEnd))
	return TempCold + t*(TempWarm-TempCold)
}

func (w *World) ForEachActiveFood(fn func(id int, x, y float64, r float64)) {
	for id, active := range w.foodActive {
		if active {
			pos := w.foodPos[id]
			r := math.Sqrt(float64(w.foodMass[id]) * math.Pi)
			fn(id, pos.X, pos.Y, r)
		}
	}
}

// --- Meat spatial operations ---

func (w *World) AddMeat(pos Position, mass float32) int {
	var id int
	if len(w.freeMeatIDs) > 0 {
		lastIdx := len(w.freeMeatIDs) - 1
		id = w.freeMeatIDs[lastIdx]
		w.freeMeatIDs = w.freeMeatIDs[:lastIdx]
		w.meatPos[id] = pos
		w.meatMass[id] = mass
		w.meatActive[id] = true
	} else {
		id = len(w.meatPos)
		w.meatPos = append(w.meatPos, pos)
		w.meatMass = append(w.meatMass, mass)
		w.meatActive = append(w.meatActive, true)
	}
	w.meatCount++
	w.mHash.Add(id, pos)
	return id
}

func (w *World) RemoveMeat(id int) {
	if id < 0 || id >= len(w.meatActive) || !w.meatActive[id] {
		return
	}
	w.mHash.Remove(id, w.meatPos[id])
	w.meatActive[id] = false
	w.meatCount--
	w.freeMeatIDs = append(w.freeMeatIDs, id)
}

func (w *World) GetMeatPos(id int) Position { return w.meatPos[id] }

func (w *World) GetMeatMass(id int) float32 { return w.meatMass[id] }

func (w *World) ReduceMeatMass(id int, amount float32) float32 {
	remaining := w.meatMass[id] - amount
	if remaining <= 0 {
		w.RemoveMeat(id)
		return 0
	}
	w.meatMass[id] = remaining
	return remaining
}

func (w *World) MeatCount() int { return w.meatCount }

func (w *World) TotalMeatMass() float64 {
	total := float64(0)
	for id, active := range w.meatActive {
		if active {
			total += float64(w.meatMass[id])
		}
	}
	return total
}

func (w *World) GetMeatInRadius(center Position, radius float64, buffer []int) []int {
	return w.mHash.InRadius(center, radius, w.meatPos, w.meatActive, buffer)
}

func (w *World) GetMeatInCone(center Position, heading float64, halfFOVCos float64, maxDist float64, buffer []int) []int {
	return w.mHash.InCone(center, heading, halfFOVCos, maxDist, w.meatPos, w.meatActive, buffer)
}

// GetAllInRadius fills all three buffers (food, meat, creature) in a single cell
// traversal. All three hashes share the same grid dimensions, so bounds are
// computed once and each cell is visited exactly once.
func (w *World) GetAllInRadius(center Position, radius float64, foodBuf, meatBuf, creatureBuf []int) ([]int, []int, []int) {
	foodBuf = foodBuf[:0]
	meatBuf = meatBuf[:0]
	creatureBuf = creatureBuf[:0]
	rSq := radius * radius
	minBx, maxBx, minBy, maxBy := w.cHash.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * w.cHash.numY
		for by := minBy; by <= maxBy; by++ {
			idx := base + by
			for _, id := range w.fHash.cells[idx] {
				if id < len(w.foodActive) && w.foodActive[id] {
					pos := w.foodPos[id]
					dx, dy := pos.X-center.X, pos.Y-center.Y
					if dx*dx+dy*dy <= rSq {
						foodBuf = append(foodBuf, id)
					}
				}
			}
			for _, id := range w.mHash.cells[idx] {
				if id < len(w.meatActive) && w.meatActive[id] {
					pos := w.meatPos[id]
					dx, dy := pos.X-center.X, pos.Y-center.Y
					if dx*dx+dy*dy <= rSq {
						meatBuf = append(meatBuf, id)
					}
				}
			}
			for _, id := range w.cHash.cells[idx] {
				if id < len(w.creatureActive) && w.creatureActive[id] {
					pos := w.creaturePos[id]
					dx, dy := pos.X-center.X, pos.Y-center.Y
					if dx*dx+dy*dy <= rSq {
						creatureBuf = append(creatureBuf, id)
					}
				}
			}
		}
	}
	return foodBuf, meatBuf, creatureBuf
}

// GetFoodAndMeatInRadius fills foodBuf and meatBuf for a radius query in one cell
// traversal, with no directional filter.
func (w *World) GetFoodAndMeatInRadius(center Position, radius float64, foodBuf, meatBuf []int) ([]int, []int) {
	foodBuf = foodBuf[:0]
	meatBuf = meatBuf[:0]
	rSq := radius * radius
	minBx, maxBx, minBy, maxBy := w.fHash.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * w.fHash.numY
		for by := minBy; by <= maxBy; by++ {
			idx := base + by
			for _, id := range w.fHash.cells[idx] {
				if id < len(w.foodActive) && w.foodActive[id] {
					pos := w.foodPos[id]
					dx, dy := pos.X-center.X, pos.Y-center.Y
					if dx*dx+dy*dy <= rSq {
						foodBuf = append(foodBuf, id)
					}
				}
			}
			for _, id := range w.mHash.cells[idx] {
				if id < len(w.meatActive) && w.meatActive[id] {
					pos := w.meatPos[id]
					dx, dy := pos.X-center.X, pos.Y-center.Y
					if dx*dx+dy*dy <= rSq {
						meatBuf = append(meatBuf, id)
					}
				}
			}
		}
	}
	return foodBuf, meatBuf
}

func (w *World) ForEachActiveMeat(fn func(id int, x, y float64, r float64)) {
	for id, active := range w.meatActive {
		if active {
			pos := w.meatPos[id]
			r := math.Sqrt(float64(w.meatMass[id]) * math.Pi)
			fn(id, pos.X, pos.Y, r)
		}
	}
}
