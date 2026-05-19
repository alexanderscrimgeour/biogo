package world

import (
	"math"
	"math/rand"
)

const fountainWaveFreq = 0.002

// FountainPoint is a single drifting spawn-centre with an associated drift angle.
type FountainPoint struct {
	Pos          Position
	angle        float64
	IsStationary bool
}

func recomputeStationary(fountains []FountainPoint, count int) {
	n := len(fountains)
	if count > n {
		count = n
	}
	if count < 0 {
		count = 0
	}
	for i := range fountains {
		fountains[i].IsStationary = i < count
	}
}

func initFountainSlice(w *World, n int, stationaryCount int) []FountainPoint {
	out := make([]FountainPoint, n)
	for i := range out {
		pos, ok := w.FindEmptyLocation()
		if !ok {
			pos = Position{X: float32(w.Width / 2), Y: float32(w.Height / 2)}
		}
		out[i] = FountainPoint{Pos: pos, angle: rand.Float64() * 2 * math.Pi}
	}
	recomputeStationary(out, stationaryCount)
	return out
}

func setFountainCount(w *World, fountains *[]FountainPoint, n int, stationaryCount int) {
	current := len(*fountains)
	if n > current {
		for i := current; i < n; i++ {
			var pos Position
			if current > 0 {
				pos = (*fountains)[rand.Intn(current)].Pos
			} else {
				var ok bool
				pos, ok = w.FindEmptyLocation()
				if !ok {
					pos = Position{X: float32(w.Width / 2), Y: float32(w.Height / 2)}
				}
			}
			*fountains = append(*fountains, FountainPoint{Pos: pos, angle: rand.Float64() * 2 * math.Pi})
		}
	} else if n < current {
		*fountains = (*fountains)[:n]
	}
	recomputeStationary(*fountains, stationaryCount)
}

func stepFountains(w *World, fountains []FountainPoint, driftSpeed float64) {
	for i := range fountains {
		if fountains[i].IsStationary {
			continue
		}
		fountains[i].angle += (rand.Float64() - 0.5) * 0.1
		newPos := Position{
			X: fountains[i].Pos.X + float32(math.Cos(fountains[i].angle)*driftSpeed),
			Y: fountains[i].Pos.Y + float32(math.Sin(fountains[i].angle)*driftSpeed),
		}
		if w.IsInBounds(newPos) && !w.IsWall(newPos) {
			fountains[i].Pos = newPos
		} else {
			fountains[i].angle += math.Pi + (rand.Float64()-0.5)*math.Pi*0.25
		}
	}
}

// InitFountains places fountain points for foliage, fungi, and meat at random valid locations.
func (w *World) InitFountains(foliageCount, fungiCount, meatCount, foliageStationary, fungiStationary, meatStationary int) {
	w.FoliageFountains = initFountainSlice(w, foliageCount, foliageStationary)
	w.FungiFountains = initFountainSlice(w, fungiCount, fungiStationary)
	w.MeatFountains = initFountainSlice(w, meatCount, meatStationary)
}

// SetFoliageFountainCount grows or shrinks the foliage fountain pool.
func (w *World) SetFoliageFountainCount(n, stationaryCount int) {
	setFountainCount(w, &w.FoliageFountains, n, stationaryCount)
}

// SetFungiFountainCount grows or shrinks the fungi fountain pool.
func (w *World) SetFungiFountainCount(n, stationaryCount int) {
	setFountainCount(w, &w.FungiFountains, n, stationaryCount)
}

// SetMeatFountainCount grows or shrinks the meat fountain pool.
func (w *World) SetMeatFountainCount(n, stationaryCount int) {
	setFountainCount(w, &w.MeatFountains, n, stationaryCount)
}

// RecomputeFoliageStationary re-evaluates IsStationary for all foliage fountains.
func (w *World) RecomputeFoliageStationary(count int) {
	recomputeStationary(w.FoliageFountains, count)
}

// RecomputeFungiStationary re-evaluates IsStationary for all fungi fountains.
func (w *World) RecomputeFungiStationary(count int) {
	recomputeStationary(w.FungiFountains, count)
}

// RecomputeMeatStationary re-evaluates IsStationary for all meat fountains.
func (w *World) RecomputeMeatStationary(count int) {
	recomputeStationary(w.MeatFountains, count)
}

// StepFountains advances all fountain groups by their respective drift speeds.
func (w *World) StepFountains(foliageDriftSpeed, fungiDriftSpeed, meatDriftSpeed float64) {
	stepFountains(w, w.FoliageFountains, foliageDriftSpeed)
	stepFountains(w, w.FungiFountains, fungiDriftSpeed)
	stepFountains(w, w.MeatFountains, meatDriftSpeed)
}

// waveMultipliers returns normalised per-fountain output weights using a sine wave
// so each fountain peaks and troughs at different times, creating moving abundance patches.
func waveMultipliers(fountains []FountainPoint, tick int) []float64 {
	mults := make([]float64, len(fountains))
	total := 0.0
	for i := range fountains {
		phase := float64(i) * 0.5
		mults[i] = 0.5 + 0.5*math.Sin(float64(tick)*fountainWaveFreq+phase)
		total += mults[i]
	}
	if total > 0 {
		for i := range mults {
			mults[i] /= total
		}
	}
	return mults
}

func spawnClustered(w *World, n int, sigma float64, mass float32, fountains []FountainPoint, tick int, addFn func(Position, float32) int) int {
	total := 0
	mults := waveMultipliers(fountains, tick)
	for i, f := range fountains {
		alloc := int(float64(n)*mults[i] + 0.5)
		if alloc == 0 {
			continue
		}
		maxAttempts := alloc * 20
		got := 0
		for attempts := 0; got < alloc && attempts < maxAttempts; attempts++ {
			pos := Position{
				X: f.Pos.X + float32(rand.NormFloat64()*sigma),
				Y: f.Pos.Y + float32(rand.NormFloat64()*sigma),
			}
			if w.IsInBounds(pos) && !w.IsWall(pos) {
				addFn(pos, mass)
				got++
			}
		}
		total += got
	}
	return total
}

func spawnFood(w *World, n int, sigma float64, mass float32, randomFraction float64,
	fountains []FountainPoint, tick int, clustered func(int) int, random func(int)) {
	if n <= 0 {
		return
	}
	if randomFraction < 0 {
		randomFraction = 0
	} else if randomFraction > 1 {
		randomFraction = 1
	}
	rnd := int(float64(n) * randomFraction)
	clust := n - rnd
	if rnd > 0 {
		random(rnd)
	}
	if len(fountains) == 0 {
		random(clust)
		return
	}
	if got := clustered(clust); got < clust {
		random(clust - got)
	}
}

// SpawnRandom places n foliage items at uniformly random valid positions.
func (w *World) SpawnRandom(n int, mass float32) {
	for i := 0; i < n; i++ {
		if pos, ok := w.FindEmptyLocation(); ok {
			w.AddFoliage(pos, mass)
		}
	}
}

// SpawnRandomFungi places n fungi items at uniformly random valid positions.
func (w *World) SpawnRandomFungi(n int, mass float32) {
	for i := 0; i < n; i++ {
		if pos, ok := w.FindEmptyLocation(); ok {
			w.AddFungi(pos, mass)
		}
	}
}

// SpawnRandomMeat places n meat items at uniformly random valid positions.
func (w *World) SpawnRandomMeat(n int, mass float32) {
	for i := 0; i < n; i++ {
		if pos, ok := w.FindEmptyLocation(); ok {
			w.AddMeat(pos, mass)
		}
	}
}

// SpawnFoliage places n foliage items clustered around foliage fountains.
func (w *World) SpawnFoliage(n int, sigma float64, mass float32, randomFraction float64, tick int) {
	spawnFood(w, n, sigma, mass, randomFraction, w.FoliageFountains, tick,
		func(k int) int { return spawnClustered(w, k, sigma, mass, w.FoliageFountains, tick, w.AddFoliage) },
		func(k int) { w.SpawnRandom(k, mass) })
}

// SpawnFungi places n fungi items clustered around fungi fountains.
func (w *World) SpawnFungi(n int, sigma float64, mass float32, randomFraction float64, tick int) {
	spawnFood(w, n, sigma, mass, randomFraction, w.FungiFountains, tick,
		func(k int) int { return spawnClustered(w, k, sigma, mass, w.FungiFountains, tick, w.AddFungi) },
		func(k int) { w.SpawnRandomFungi(k, mass) })
}

// SpawnMeat places n meat items clustered around meat fountains.
func (w *World) SpawnMeat(n int, sigma float64, mass float32, randomFraction float64, tick int) {
	spawnFood(w, n, sigma, mass, randomFraction, w.MeatFountains, tick,
		func(k int) int { return spawnClustered(w, k, sigma, mass, w.MeatFountains, tick, w.AddMeat) },
		func(k int) { w.SpawnRandomMeat(k, mass) })
}
