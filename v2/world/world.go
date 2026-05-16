package world

import (
	"math"
	"math/rand"
)

// StartingCreatureID is the first valid creature ID; 0 is reserved for "empty".
const StartingCreatureID = 1

// FoodTypePlant and FoodTypeMeat distinguish the two food sources stored in the
// unified food spatial hash.
const (
	FoodTypePlant uint8 = 0
	FoodTypeMeat  uint8 = 1
)

// FoodMask* constants are bitmasks for type-filtered food queries.
const (
	FoodMaskPlant uint8 = 1 << FoodTypePlant // = 1
	FoodMaskMeat  uint8 = 1 << FoodTypeMeat  // = 2
	FoodMaskAll   uint8 = FoodMaskPlant | FoodMaskMeat
)

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

	// Unified food hash: plants (FoodTypePlant=0) and meat (FoodTypeMeat=1)
	// share one ID space and one SpatialHash. foodType[id] distinguishes them.
	foodPos     []Position
	foodMass    []float32
	foodActive  []bool
	foodType    []uint8
	plantCount  int
	meatCount   int
	freeFoodIDs []int
	foodHash    *SpatialHash

	// Gaussian fountain system: 3-5 drifting points that food spawns around.
	Fountains      []Position
	fountainAngles []float64
}

func NewWorld(width, height float64, _ int) *World {
	const initialCapacity = 25000
	const creatureCapacity = 20000
	const cellSize = 150.0
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
		foodType:        make([]uint8, 0, initialCapacity),
		freeFoodIDs:     make([]int, 0, 100),
		foodHash:        newSpatialHash(width, height, cellSize),
	}
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

func (w *World) GetCreaturesInRadius(center Position, radius float32, buffer []int) []int {
	return w.cHash.InRadius(center, radius, w.creaturePos, w.creatureActive, buffer)
}

// GetCreaturesInCone returns IDs of creatures within maxDist that lie inside
// the cone defined by heading ± halfFOVCos (cosine of the half-angle).
func (w *World) GetCreaturesInCone(center Position, heading float32, halfFOVCos float32, maxDist float32, buffer []int) []int {
	return w.cHash.InCone(center, heading, halfFOVCos, maxDist, w.creaturePos, w.creatureActive, buffer)
}

// --- Unified food/plant/meat spatial operations ---

// addFoodItem is the internal helper that inserts any food type into the unified hash.
func (w *World) addFoodItem(pos Position, mass float32, typ uint8) int {
	var id int
	if len(w.freeFoodIDs) > 0 {
		lastIdx := len(w.freeFoodIDs) - 1
		id = w.freeFoodIDs[lastIdx]
		w.freeFoodIDs = w.freeFoodIDs[:lastIdx]
		w.foodPos[id] = pos
		w.foodMass[id] = mass
		w.foodActive[id] = true
		w.foodType[id] = typ
	} else {
		id = len(w.foodPos)
		w.foodPos = append(w.foodPos, pos)
		w.foodMass = append(w.foodMass, mass)
		w.foodActive = append(w.foodActive, true)
		w.foodType = append(w.foodType, typ)
	}
	w.foodHash.Add(id, pos)
	return id
}

// removeFoodItem removes any food item from the unified hash and decrements the
// appropriate type counter.
func (w *World) removeFoodItem(id int) {
	w.foodHash.Remove(id, w.foodPos[id])
	if w.foodType[id] == FoodTypePlant {
		w.plantCount--
	} else {
		w.meatCount--
	}
	w.foodActive[id] = false
	w.freeFoodIDs = append(w.freeFoodIDs, id)
}

// AddPlant places a plant item with the given mass at pos and returns its ID.
func (w *World) AddPlant(pos Position, mass float32) int {
	id := w.addFoodItem(pos, mass, FoodTypePlant)
	w.plantCount++
	return id
}

// RemovePlant removes the plant item with the given ID.
func (w *World) RemovePlant(id int) {
	if id < 0 || id >= len(w.foodActive) || !w.foodActive[id] {
		return
	}
	w.removeFoodItem(id)
}

// AddMeat places a meat item with the given mass at pos and returns its ID.
func (w *World) AddMeat(pos Position, mass float32) int {
	id := w.addFoodItem(pos, mass, FoodTypeMeat)
	w.meatCount++
	return id
}

// RemoveMeat removes the meat item with the given ID.
func (w *World) RemoveMeat(id int) {
	if id < 0 || id >= len(w.foodActive) || !w.foodActive[id] {
		return
	}
	w.removeFoodItem(id)
}

// GetFoodPos returns the position of any food item (plant or meat) by ID.
func (w *World) GetFoodPos(id int) Position {
	return w.foodPos[id]
}

// GetFoodMass returns the mass of any food item (plant or meat) by ID.
func (w *World) GetFoodMass(id int) float32 {
	return w.foodMass[id]
}

// ReduceFoodMass subtracts amount from any food item's mass.
// Automatically removes the item when mass drops to zero or below.
// Returns remaining mass (0 if removed).
func (w *World) ReduceFoodMass(id int, amount float32) float32 {
	remaining := w.foodMass[id] - amount
	if remaining <= 0 {
		w.removeFoodItem(id)
		return 0
	}
	w.foodMass[id] = remaining
	return remaining
}

// TotalPlantMass returns the sum of all active plant item masses.
func (w *World) TotalPlantMass() float64 {
	total := float64(0)
	for id, active := range w.foodActive {
		if active && w.foodType[id] == FoodTypePlant {
			total += float64(w.foodMass[id])
		}
	}
	return total
}

// TotalMeatMass returns the sum of all active meat item masses.
func (w *World) TotalMeatMass() float64 {
	total := float64(0)
	for id, active := range w.foodActive {
		if active && w.foodType[id] == FoodTypeMeat {
			total += float64(w.foodMass[id])
		}
	}
	return total
}

// PlantCount returns the number of active plant items.
func (w *World) PlantCount() int { return w.plantCount }

// MeatCount returns the number of active meat items.
func (w *World) MeatCount() int { return w.meatCount }

// GetFoodInRadius returns IDs of plant items within radius of center.
func (w *World) GetFoodInRadius(center Position, radius float32, buffer []int) []int {
	return w.GetFoodInRadiusByMask(center, radius, FoodMaskPlant, buffer)
}

// GetFoodInRadiusByMask returns IDs of food items within radius matching the type bitmask.
func (w *World) GetFoodInRadiusByMask(center Position, radius float32, mask uint8, buffer []int) []int {
	buffer = buffer[:0]
	rSq := radius * radius
	minBx, maxBx, minBy, maxBy := w.foodHash.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * w.foodHash.numY
		for by := minBy; by <= maxBy; by++ {
			for _, id := range w.foodHash.cells[base+by] {
				if id >= len(w.foodActive) || !w.foodActive[id] {
					continue
				}
				if (1<<w.foodType[id])&mask == 0 {
					continue
				}
				pos := w.foodPos[id]
				dx, dy := pos.X-center.X, pos.Y-center.Y
				if dx*dx+dy*dy <= rSq {
					buffer = append(buffer, id)
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
		x, y := float64(pos.X), float64(pos.Y)
		if x >= wall.X && x < wall.X+wall.W &&
			y >= wall.Y && y < wall.Y+wall.H {
			return true
		}
	}
	return false
}

func (w *World) IsInBounds(pos Position) bool {
	return pos.X >= 0 && float64(pos.X) < w.Width && pos.Y >= 0 && float64(pos.Y) < w.Height
}

func (w *World) ClampToBounds(pos Position) Position {
	if pos.X < 0 {
		pos.X = 0
	} else if float64(pos.X) >= w.Width {
		pos.X = float32(w.Width) - 0.001
	}
	if pos.Y < 0 {
		pos.Y = 0
	} else if float64(pos.Y) >= w.Height {
		pos.Y = float32(w.Height) - 0.001
	}
	return pos
}

func (w *World) SizeX() int { return int(w.Width) }
func (w *World) SizeY() int { return int(w.Height) }

func (w *World) FindEmptyLocation() (Position, bool) {
	for i := 0; i < 200; i++ {
		pos := Position{
			X: float32(rand.Float64() * w.Width),
			Y: float32(rand.Float64() * w.Height),
		}
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
		pos := Position{
			X: center.X + float32(math.Cos(angle)*dist),
			Y: center.Y + float32(math.Sin(angle)*dist),
		}
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
			pos = Position{X: float32(w.Width / 2), Y: float32(w.Height / 2)}
		}
		w.Fountains[i] = pos
		w.fountainAngles[i] = rand.Float64() * 2 * math.Pi
	}
}

// SetFountainCount grows or shrinks the fountain pool without re-seeding existing ones.
func (w *World) SetFountainCount(n int) {
	current := len(w.Fountains)
	if n > current {
		for i := current; i < n; i++ {
			pos, ok := w.FindEmptyLocation()
			if !ok {
				pos = Position{X: float32(w.Width / 2), Y: float32(w.Height / 2)}
			}
			w.Fountains = append(w.Fountains, pos)
			w.fountainAngles = append(w.fountainAngles, rand.Float64()*2*math.Pi)
		}
	} else if n < current {
		w.Fountains = w.Fountains[:n]
		w.fountainAngles = w.fountainAngles[:n]
	}
}

// StepFountains advances each fountain by driftSpeed units along its current angle,
// applying a small random angular perturbation each step. Fountains bounce off world
// edges and walls.
func (w *World) StepFountains(driftSpeed float64) {
	for i := range w.Fountains {
		w.fountainAngles[i] += (rand.Float64() - 0.5) * 0.1

		newPos := Position{
			X: w.Fountains[i].X + float32(math.Cos(w.fountainAngles[i])*driftSpeed),
			Y: w.Fountains[i].Y + float32(math.Sin(w.fountainAngles[i])*driftSpeed),
		}

		if w.IsInBounds(newPos) && !w.IsWall(newPos) {
			w.Fountains[i] = newPos
		} else {
			w.fountainAngles[i] += math.Pi + (rand.Float64()-0.5)*math.Pi*0.25
		}
	}
}

// SpawnPlant places n plant items (each with the given mass) sampled from Gaussian
// distributions centred on each fountain. Each item is assigned to a fountain
// uniformly at random, then offset by a 2-D normal with standard deviation sigma.
// randomFraction controls what proportion [0,1] is scattered uniformly at random.
// Items that fall outside the world bounds or inside a wall are retried; if the retry
// budget is exhausted the remainder are placed uniformly at random so the requested
// count is always satisfied.
func (w *World) SpawnPlant(n int, sigma float64, mass float32, randomFraction float64) {
	if n <= 0 {
		return
	}

	if randomFraction < 0 {
		randomFraction = 0
	} else if randomFraction > 1 {
		randomFraction = 1
	}
	randomCount := int(float64(n) * randomFraction)
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
			X: center.X + float32(rand.NormFloat64()*sigma),
			Y: center.Y + float32(rand.NormFloat64()*sigma),
		}
		if w.IsInBounds(pos) && !w.IsWall(pos) {
			w.AddPlant(pos, mass)
			spawned++
		}
	}

	if spawned < clusterCount {
		w.SpawnRandom(clusterCount-spawned, mass)
	}
}

// SpawnRandom places n plant items (each with the given mass) at uniformly random valid positions.
func (w *World) SpawnRandom(n int, mass float32) {
	for i := 0; i < n; i++ {
		pos, ok := w.FindEmptyLocation()
		if ok {
			w.AddPlant(pos, mass)
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
func (w *World) TemperatureAt(y float32) float32 {
	const coldBandEnd = 0.2
	const warmBandStart = 0.8
	norm := float64(y) / w.Height
	if norm <= coldBandEnd {
		return TempCold
	}
	if norm >= warmBandStart {
		return TempWarm
	}
	t := float32((norm - coldBandEnd) / (warmBandStart - coldBandEnd))
	return TempCold + t*(TempWarm-TempCold)
}

// ForEachActiveFood iterates all active food items (plants and meat) and calls fn
// with each item's ID, position, radius, and type.
func (w *World) ForEachActiveFood(fn func(id int, x, y float64, r float64, typ uint8)) {
	for id, active := range w.foodActive {
		if active {
			pos := w.foodPos[id]
			r := math.Sqrt(float64(w.foodMass[id]) * math.Pi)
			fn(id, float64(pos.X), float64(pos.Y), r, w.foodType[id])
		}
	}
}

// GetFoodAndMeatInRadius fills plantBuf and meatBuf for a radius query in a single
// cell traversal of the unified food hash.
func (w *World) GetFoodAndMeatInRadius(center Position, radius float32, plantBuf, meatBuf []int) ([]int, []int) {
	plantBuf = plantBuf[:0]
	meatBuf = meatBuf[:0]
	rSq := radius * radius
	minBx, maxBx, minBy, maxBy := w.foodHash.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * w.foodHash.numY
		for by := minBy; by <= maxBy; by++ {
			for _, id := range w.foodHash.cells[base+by] {
				if id >= len(w.foodActive) || !w.foodActive[id] {
					continue
				}
				pos := w.foodPos[id]
				dx, dy := pos.X-center.X, pos.Y-center.Y
				if dx*dx+dy*dy > rSq {
					continue
				}
				if w.foodType[id] == FoodTypePlant {
					plantBuf = append(plantBuf, id)
				} else {
					meatBuf = append(meatBuf, id)
				}
			}
		}
	}
	return plantBuf, meatBuf
}

// GetAllInRadius fills all three buffers (plant, meat, creature) in a single pass
// over the cell bounds. The food hash and creature hash share the same grid dimensions,
// so both are visited in one loop.
func (w *World) GetAllInRadius(center Position, radius float32, plantBuf, meatBuf, creatureBuf []int) ([]int, []int, []int) {
	plantBuf = plantBuf[:0]
	meatBuf = meatBuf[:0]
	creatureBuf = creatureBuf[:0]
	rSq := radius * radius
	// cHash and foodHash share the same grid layout (same width/height/cellSize).
	minBx, maxBx, minBy, maxBy := w.cHash.cellBounds(center, radius)
	for bx := minBx; bx <= maxBx; bx++ {
		base := bx * w.cHash.numY
		for by := minBy; by <= maxBy; by++ {
			idx := base + by
			for _, id := range w.foodHash.cells[idx] {
				if id >= len(w.foodActive) || !w.foodActive[id] {
					continue
				}
				pos := w.foodPos[id]
				dx, dy := pos.X-center.X, pos.Y-center.Y
				if dx*dx+dy*dy > rSq {
					continue
				}
				if w.foodType[id] == FoodTypePlant {
					plantBuf = append(plantBuf, id)
				} else {
					meatBuf = append(meatBuf, id)
				}
			}
			for _, id := range w.cHash.cells[idx] {
				if id >= len(w.creatureActive) || !w.creatureActive[id] {
					continue
				}
				pos := w.creaturePos[id]
				dx, dy := pos.X-center.X, pos.Y-center.Y
				if dx*dx+dy*dy <= rSq {
					creatureBuf = append(creatureBuf, id)
				}
			}
		}
	}
	return plantBuf, meatBuf, creatureBuf
}
