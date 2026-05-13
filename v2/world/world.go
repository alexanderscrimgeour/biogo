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
// positions via spatial-hash buckets for efficient neighbourhood queries.
//
// Buckets use a packed int64 key (high 32 bits = bucket-x, low 32 bits = bucket-y)
// so Go's map uses the fast64 hash path instead of the slower memhash128 path
// required for [2]int keys. Each bucket stores a plain []int slice rather than a
// nested map[int]bool, eliminating inner-map iteration overhead entirely.
type World struct {
	Width, Height float64
	Walls         []Wall

	creaturePos     []Position
	creatureActive  []bool
	creatureCount   int
	freeCreatureIDs []int
	cBuckets        map[int64][]int // packed bucket key → creature IDs

	foodPos     []Position
	foodMass    []float32
	foodActive  []bool
	foodCount   int
	freeFoodIDs []int
	fBuckets    map[int64][]int // packed bucket key → food IDs
	bucketSize  float64

	// Gaussian fountain system: 3-5 drifting points that food spawns around.
	Fountains      []Position
	fountainAngles []float64
}

func NewWorld(width, height float64, _ int) *World {
	const initialCapacity = 25000
	const creatureCapacity = 20000
	// Pre-allocate slot 0 so that IDs start at StartingCreatureID (1).
	w := &World{
		Width:           width,
		Height:          height,
		creaturePos:     make([]Position, 1, creatureCapacity),
		creatureActive:  make([]bool, 1, creatureCapacity),
		freeCreatureIDs: make([]int, 0, 100),
		cBuckets:        make(map[int64][]int),
		foodPos:         make([]Position, 0, initialCapacity),
		foodMass:        make([]float32, 0, initialCapacity),
		foodActive:      make([]bool, 0, initialCapacity),
		freeFoodIDs:     make([]int, 0, 100),
		fBuckets:        make(map[int64][]int),
		bucketSize:      100.0,
	}
	// w.createWalls(wallType)
	return w
}

// bucketKey packs a world-space position into a single int64 map key.
// The high 32 bits hold the x bucket index, low 32 bits hold the y bucket index.
// This lets Go use its fast64 hash path rather than the slower memhash128 used for [2]int.
func (w *World) bucketKey(pos Position) int64 {
	bx := int32(math.Floor(pos.X / w.bucketSize))
	by := int32(math.Floor(pos.Y / w.bucketSize))
	return int64(bx)<<32 | int64(uint32(by))
}

// packedKey packs raw integer bucket coordinates, used in radius scan loops
// to avoid constructing a Position just to compute the key.
func packedKey(bx, by int) int64 {
	return int64(int32(bx))<<32 | int64(uint32(int32(by)))
}

// bucketRemove removes id from the bucket slice via swap-and-truncate (O(n) scan, O(1) remove).
func bucketRemove(slice []int, id int) []int {
	for i, v := range slice {
		if v == id {
			slice[i] = slice[len(slice)-1]
			return slice[:len(slice)-1]
		}
	}
	return slice
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
	key := w.bucketKey(pos)
	w.cBuckets[key] = append(w.cBuckets[key], id)
	return id
}

func (w *World) MoveCreature(id int, newPos Position) {
	if id < 0 || id >= len(w.creatureActive) || !w.creatureActive[id] {
		return
	}
	oldKey := w.bucketKey(w.creaturePos[id])
	w.cBuckets[oldKey] = bucketRemove(w.cBuckets[oldKey], id)
	w.creaturePos[id] = newPos
	key := w.bucketKey(newPos)
	w.cBuckets[key] = append(w.cBuckets[key], id)
}

func (w *World) RemoveCreature(id int) {
	if id < 0 || id >= len(w.creatureActive) || !w.creatureActive[id] {
		return
	}
	key := w.bucketKey(w.creaturePos[id])
	w.cBuckets[key] = bucketRemove(w.cBuckets[key], id)
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
	buffer = buffer[:0]

	invBucketSize := 1.0 / w.bucketSize
	rSq := radius * radius

	minBx := int((center.X - radius) * invBucketSize)
	maxBx := int((center.X + radius) * invBucketSize)
	minBy := int((center.Y - radius) * invBucketSize)
	maxBy := int((center.Y + radius) * invBucketSize)

	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			bucket := w.cBuckets[packedKey(bx, by)]

			for _, id := range bucket {
				if !w.creatureActive[id] {
					continue
				}
				pos := w.creaturePos[id]

				dx := pos.X - center.X
				dy := pos.Y - center.Y
				distSq := dx*dx + dy*dy

				if distSq <= rSq {
					buffer = append(buffer, id)
				}
			}
		}
	}
	return buffer
}

// GetCreaturesInCone returns IDs of creatures within maxDist that lie inside
// the cone defined by heading ± halfFOVCos (cosine of the half-angle).
func (w *World) GetCreaturesInCone(center Position, heading float64, halfFOVCos float64, maxDist float64, buffer []int) []int {
	buffer = buffer[:0]

	// 1. Setup math constants
	fwdX, fwdY := HeadingToVec(heading)
	invBucketSize := 1.0 / w.bucketSize
	rSq := maxDist * maxDist

	// 2. Define search bounds in the grid
	minBx := int((center.X - maxDist) * invBucketSize)
	maxBx := int((center.X + maxDist) * invBucketSize)
	minBy := int((center.Y - maxDist) * invBucketSize)
	maxBy := int((center.Y + maxDist) * invBucketSize)

	// 3. Single-pass spatial search and filtering
	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			bucket := w.cBuckets[packedKey(bx, by)]

			for _, id := range bucket {
				if !w.creatureActive[id] {
					continue
				}
				pos := w.creaturePos[id]

				dx := pos.X - center.X
				dy := pos.Y - center.Y
				distSq := dx*dx + dy*dy

				// Check radius first (cheapest math)
				if distSq <= rSq {
					// Avoid checking similarity against self (center)
					if dx == 0 && dy == 0 {
						continue
					}

					// Check cone angle (more expensive math)
					if CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
						buffer = append(buffer, id)
					}
				}
			}
		}
	}
	return buffer
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

	key := w.bucketKey(pos)
	w.fBuckets[key] = append(w.fBuckets[key], id)
	return id
}

func (w *World) RemoveFood(id int) {
	if id < 0 || id >= len(w.foodActive) || !w.foodActive[id] {
		return
	}

	pos := w.foodPos[id]
	key := w.bucketKey(pos)

	w.fBuckets[key] = bucketRemove(w.fBuckets[key], id)

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
	buffer = buffer[:0]

	invBucketSize := 1.0 / w.bucketSize

	minBx := int((center.X - radius) * invBucketSize)
	maxBx := int((center.X + radius) * invBucketSize)
	minBy := int((center.Y - radius) * invBucketSize)
	maxBy := int((center.Y + radius) * invBucketSize)

	rSq := radius * radius

	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			bucket := w.fBuckets[packedKey(bx, by)]

			for _, id := range bucket {
				if !w.foodActive[id] {
					continue
				}

				pos := w.foodPos[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y

				if dx*dx+dy*dy <= rSq {
					buffer = append(buffer, id)
				}
			}
		}
	}
	return buffer
}

func (w *World) GetFoodInCone(center Position, heading float64, halfFOVCos float64, maxDist float64, buffer []int) []int {
	buffer = buffer[:0]

	fwdX, fwdY := HeadingToVec(heading)
	invBucketSize := 1.0 / w.bucketSize
	rSq := maxDist * maxDist

	minBx := int((center.X - maxDist) * invBucketSize)
	maxBx := int((center.X + maxDist) * invBucketSize)
	minBy := int((center.Y - maxDist) * invBucketSize)
	maxBy := int((center.Y + maxDist) * invBucketSize)

	for bx := minBx; bx <= maxBx; bx++ {
		for by := minBy; by <= maxBy; by++ {
			for _, id := range w.fBuckets[packedKey(bx, by)] {
				if !w.foodActive[id] {
					continue
				}

				pos := w.foodPos[id]
				dx := pos.X - center.X
				dy := pos.Y - center.Y
				distSq := dx*dx + dy*dy

				// Check Radius first (cheaper math)
				if distSq <= rSq {
					// Check Cone second (more expensive math)
					if dx == 0 && dy == 0 || CosSimilarity(fwdX, fwdY, dx, dy) >= halfFOVCos {
						buffer = append(buffer, id)
					}
				}
			}
		}
	}
	return buffer
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

	randomScatterFactor := 0.05 // 5% of food spawns anywhere
	randomCount := int(float64(n) * randomScatterFactor)
	clusterCount := n - randomCount

	// multiplier := rand.NormFloat64()*0.083 + 0.75

	// // Clamp to strictly enforce the 50%-100% boundary
	// if multiplier < 0.5 {
	// 	multiplier = 0.5
	// }
	// if multiplier > 1.0 {
	// 	multiplier = 1.0
	// }

	// mass = mass * float32(multiplier)
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
