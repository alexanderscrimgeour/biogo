package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/world"
	"fmt"
	"image/color"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type creatureAnim struct {
	prevX, prevY float64
	curX, curY   float64
	r, g, b, a   uint8
	heading      float64
	mass         float64
	radius       float32
	sexual       bool
	tier         byte
}

// WorldRenderer handles all world-space rendering: temperature gradient,
// food, meat, creatures, FOV cones, selection highlight, and camera transform.
type WorldRenderer struct {
	sim              SimulationState
	renderWorld      *RenderWorld
	camera           Camera
	worldLayer       *ebiten.Image
	screenW          int
	screenH          int
	animByID         map[int]*creatureAnim
	lookup           map[int]int
	unitCircle       []struct{ x, y float32 }
	circleIsTemplate []uint16 // pre-baked; shared by food and circle creatures
	vertsPerCircle   int
	indicesPerCircle int
	circleCreatureVs []ebiten.Vertex
	sexualCreatureVs []ebiten.Vertex
	sexualCreatureIs []uint16
	nCreatureCircles int
	foodVs           []ebiten.Vertex // pooled; reset each frame
	nFoodCircles     int
	whiteImage       *ebiten.Image
	isDark           bool
	lastTickTime     time.Time
	tickDuration     time.Duration

	tierFilter int // -1 = show all; 0-3 = dim creatures not in this tier

	camDragging   bool
	camDragMoved  bool
	camDragStartX int
	camDragStartY int
	camDragLastX  int
	camDragLastY  int
}

// NewWorldRenderer creates a WorldRenderer for the given simulation.
func NewWorldRenderer(sim SimulationState) *WorldRenderer {
	wImg := ebiten.NewImage(1, 1)
	wImg.Fill(color.White)

	worldW := int(sim.WorldWidth()) * UnitSize
	worldH := int(sim.WorldHeight()) * UnitSize

	const segments = 12
	var unitCircle []struct{ x, y float32 }
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / segments
		unitCircle = append(unitCircle, struct{ x, y float32 }{
			x: float32(math.Cos(angle)),
			y: float32(math.Sin(angle)),
		})
	}

	vertsPerCircle := 1 + len(unitCircle)         // 14
	indicesPerCircle := (len(unitCircle) - 1) * 3 // 36
	maxCircles := 60_000 / vertsPerCircle         // 4285

	circleIsTemplate := make([]uint16, maxCircles*indicesPerCircle)
	for k := 0; k < maxCircles; k++ {
		base := uint16(k * vertsPerCircle)
		off := k * indicesPerCircle
		for i := uint16(1); i < uint16(len(unitCircle)); i++ {
			circleIsTemplate[off] = base
			circleIsTemplate[off+1] = base + i
			circleIsTemplate[off+2] = base + i + 1
			off += 3
		}
	}

	params := sim.GetParams()
	maxPop := params.Population.Max
	return &WorldRenderer{
		sim:              sim,
		renderWorld:      NewRenderWorld(0, 0, UnitSize),
		camera:           Camera{X: float64(worldW) / 2, Y: float64(worldH) / 2, Zoom: 1.0},
		animByID:         make(map[int]*creatureAnim, maxPop),
		lookup:           make(map[int]int, maxPop),
		whiteImage:       wImg,
		unitCircle:       unitCircle,
		circleIsTemplate: circleIsTemplate,
		vertsPerCircle:   vertsPerCircle,
		indicesPerCircle: indicesPerCircle,
		isDark:           true,
		tierFilter:       -1,
		circleCreatureVs: make([]ebiten.Vertex, 0, maxPop*vertsPerCircle),
		sexualCreatureVs: make([]ebiten.Vertex, 0, maxPop*3),
		sexualCreatureIs: make([]uint16, 0, maxPop*3),
		foodVs:           make([]ebiten.Vertex, 0, maxCircles*vertsPerCircle),
	}
}

// ToggleDark flips the world background theme.
func (wr *WorldRenderer) ToggleDark() { wr.isDark = !wr.isDark }

// CycleTierFilter steps through the tiers present in the current population:
// All → Tier 0 → Tier 1 → ... → All. Returns the new button label.
func (wr *WorldRenderer) CycleTierFilter() string {
	tierSet := make(map[int]struct{})
	for _, anim := range wr.animByID {
		tierSet[int(anim.tier)] = struct{}{}
	}
	tiers := make([]int, 0, len(tierSet))
	for t := range tierSet {
		tiers = append(tiers, t)
	}
	sort.Ints(tiers)

	if wr.tierFilter == -1 {
		if len(tiers) > 0 {
			wr.tierFilter = tiers[0]
		}
	} else {
		next := -1
		for _, t := range tiers {
			if t > wr.tierFilter {
				next = t
				break
			}
		}
		wr.tierFilter = next
	}

	if wr.tierFilter == -1 {
		return "Tier: All"
	}
	return fmt.Sprintf("Tier: %d", wr.tierFilter)
}

// Camera returns a pointer to the camera for external queries (e.g. ScreenToWorld).
func (wr *WorldRenderer) Camera() *Camera { return &wr.camera }

// CamDragMoved reports whether the last camera drag exceeded the movement threshold.
func (wr *WorldRenderer) CamDragMoved() bool { return wr.camDragMoved }

// CamDragStartPos returns the pixel where the camera drag started.
func (wr *WorldRenderer) CamDragStartPos() (int, int) {
	return wr.camDragStartX, wr.camDragStartY
}

// GetAnim returns the creature animation state for id if it exists.
func (wr *WorldRenderer) GetAnim(id int) (*creatureAnim, bool) {
	a, ok := wr.animByID[id]
	return a, ok
}

// GetLookup returns the snapshot index mapping creature ID → snapshot slice index.
func (wr *WorldRenderer) GetLookup() map[int]int { return wr.lookup }

// HandleContinuousInput processes camera pan (WASD + mouse drag) and zoom.
// Pass sliderDragging=true to suppress camera drag while a slider is active.
func (wr *WorldRenderer) HandleContinuousInput(sliderDragging bool) {
	mx, my := ebiten.CursorPosition()
	if !sliderDragging && ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && wr.camDragging {
		dx := mx - wr.camDragLastX
		dy := my - wr.camDragLastY
		if !wr.camDragMoved && (dx*dx+dy*dy > 9) {
			wr.camDragMoved = true
		}
		if wr.camDragMoved {
			wr.camera.X -= float64(dx) / wr.camera.Zoom
			wr.camera.Y -= float64(dy) / wr.camera.Zoom
		}
		wr.camDragLastX = mx
		wr.camDragLastY = my
	}

	const panSpeed = 5.0
	pan := panSpeed / wr.camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		wr.camera.Y -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		wr.camera.Y += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		wr.camera.X -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		wr.camera.X += pan
	}
	worldW := float64(wr.sim.WorldWidth()) * float64(UnitSize)
	worldH := float64(wr.sim.WorldHeight()) * float64(UnitSize)
	wr.camera.Clamp(worldW, worldH)
}

// StartCameraDrag begins a camera drag at (mx, my).
func (wr *WorldRenderer) StartCameraDrag(mx, my int) {
	wr.camDragging = true
	wr.camDragStartX, wr.camDragStartY = mx, my
	wr.camDragLastX, wr.camDragLastY = mx, my
	wr.camDragMoved = false
}

// StopCameraDrag ends the drag and returns whether the cursor moved.
func (wr *WorldRenderer) StopCameraDrag() bool {
	moved := wr.camDragMoved
	wr.camDragging = false
	return moved
}

// UpdateAnimations syncs animation state from the latest snapshot.
// Returns the tick duration for interpolation.
func (wr *WorldRenderer) UpdateAnimations(snapshot *simulation.StateSnapshot) time.Duration {
	now := time.Now()
	bs := float64(UnitSize)

	prevByID := make(map[int][2]float64, len(wr.animByID))
	for id, anim := range wr.animByID {
		prevByID[id] = [2]float64{anim.curX, anim.curY}
	}

	wr.lookup = make(map[int]int, len(snapshot.Creatures))
	currentIDs := make(map[int]bool, len(snapshot.Creatures))

	for i, cv := range snapshot.Creatures {
		wr.lookup[cv.ID] = i
		currentIDs[cv.ID] = true
		screenX := float64(cv.X * bs)
		screenY := float64(cv.Y * bs)

		if anim, ok := wr.animByID[cv.ID]; ok {
			if prev, ok := prevByID[cv.ID]; ok {
				anim.prevX, anim.prevY = prev[0], prev[1]
			} else {
				anim.prevX, anim.prevY = screenX, screenY
			}
			anim.curX, anim.curY = screenX, screenY
			anim.r, anim.g, anim.b, anim.a = cv.R, cv.G, cv.B, cv.A
			anim.heading = cv.Heading
			anim.mass = cv.CurrentMass
			anim.radius = float32(cv.Radius * bs)
			anim.sexual = cv.ReproductionType == 1
			anim.tier = cv.Tier
		} else {
			wr.animByID[cv.ID] = &creatureAnim{
				prevX: screenX, prevY: screenY,
				curX: screenX, curY: screenY,
				r: cv.R, g: cv.G, b: cv.B, a: cv.A,
				heading: cv.Heading,
				mass:    cv.CurrentMass,
				radius:  float32(cv.Radius * bs),
				sexual:  cv.ReproductionType == 1,
				tier:    cv.Tier,
			}
		}
	}

	for id := range wr.animByID {
		if !currentIDs[id] {
			delete(wr.animByID, id)
		}
	}

	if !wr.lastTickTime.IsZero() {
		wr.tickDuration = now.Sub(wr.lastTickTime)
	}
	wr.lastTickTime = now
	return wr.tickDuration
}

// ResetAnimations clears all animation state (e.g. after a simulation reset).
func (wr *WorldRenderer) ResetAnimations() {
	wr.animByID = make(map[int]*creatureAnim)
	wr.lastTickTime = time.Time{}
}

// Draw renders the world to screen, applying the camera transform.
func (wr *WorldRenderer) Draw(screen *ebiten.Image, snapshot *simulation.StateSnapshot, selectedID int) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	if wr.worldLayer == nil || wr.screenW != sw || wr.screenH != sh {
		wr.worldLayer = ebiten.NewImage(sw, sh)
		wr.screenW = sw
		wr.screenH = sh
	}
	wr.worldLayer.Clear()

	camGeoM := wr.camera.GeoM(float64(sw), float64(sh))
	zoom := float32(wr.camera.Zoom)

	t := 1.0
	if wr.tickDuration > 0 {
		elapsed := time.Since(wr.lastTickTime)
		t = float64(elapsed) / float64(wr.tickDuration)
		if t > 1 {
			t = 1
		}
	}

	wr.drawTemperatureBackground(sw, sh, camGeoM)
	wr.renderWorld.DrawBackground(wr.worldLayer, camGeoM)

	bs := float64(UnitSize)
	if snapshot != nil {
		// Foliage: muted green. Meat: dark red. Fungi: purple.
		const pr, pg, pb, pa = float32(65) / 255, float32(140) / 255, float32(55) / 255, float32(125) / 255
		const mr, mg, mb, ma = float32(180) / 255, float32(30) / 255, float32(30) / 255, float32(180) / 255
		const fr, fg, fb, fa = float32(150) / 255, float32(50) / 255, float32(190) / 255, float32(140) / 255

		wr.foodVs = wr.foodVs[:0]
		wr.nFoodCircles = 0
		flushFood := func() {
			if wr.nFoodCircles > 0 {
				wr.worldLayer.DrawTriangles(wr.foodVs, wr.circleIsTemplate[:wr.nFoodCircles*wr.indicesPerCircle], wr.whiteImage, nil)
				wr.foodVs = wr.foodVs[:0]
				wr.nFoodCircles = 0
			}
		}
		for _, fv := range snapshot.Food {
			if len(wr.foodVs)+wr.vertsPerCircle > 60_000 {
				flushFood()
			}
			scx, scy := camGeoM.Apply(fv.X*bs, fv.Y*bs)
			cx, cy := float32(scx), float32(scy)
			r := float32(fv.Radius*bs) * zoom
			var cr, cg, cb, ca float32
			switch fv.Type {
			case world.FoodTypeFoliage:
				cr, cg, cb, ca = pr, pg, pb, pa
			case world.FoodTypeFungi:
				cr, cg, cb, ca = fr, fg, fb, fa
			default:
				cr, cg, cb, ca = mr, mg, mb, ma
			}
			base := len(wr.foodVs)
			wr.foodVs = wr.foodVs[:base+wr.vertsPerCircle]
			wr.foodVs[base] = ebiten.Vertex{DstX: cx, DstY: cy, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
			for j, u := range wr.unitCircle {
				wr.foodVs[base+1+j] = ebiten.Vertex{DstX: cx + r*u.x, DstY: cy + r*u.y, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
			}
			wr.nFoodCircles++
		}
		flushFood()
	}

	wr.circleCreatureVs = wr.circleCreatureVs[:0]
	wr.sexualCreatureVs = wr.sexualCreatureVs[:0]
	wr.sexualCreatureIs = wr.sexualCreatureIs[:0]
	wr.nCreatureCircles = 0

	for _, anim := range wr.animByID {
		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		slx, sly := camGeoM.Apply(lerpX, lerpY)
		cx, cy := float32(slx), float32(sly)
		scaledRadius := anim.radius * zoom
		cr, cg, cb, ca := float32(anim.r)/255, float32(anim.g)/255, float32(anim.b)/255, float32(anim.a)/255
		if wr.tierFilter != -1 && int(anim.tier) != wr.tierFilter {
			ca *= 0.2
		}

		if anim.sexual {
			baseIdx := uint16(len(wr.sexualCreatureVs))
			wr.sexualCreatureVs = append(wr.sexualCreatureVs,
				ebiten.Vertex{DstX: cx + scaledRadius*float32(math.Cos(anim.heading)), DstY: cy + scaledRadius*float32(math.Sin(anim.heading)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + scaledRadius*float32(math.Cos(anim.heading+2.4)), DstY: cy + scaledRadius*float32(math.Sin(anim.heading+2.4)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + scaledRadius*float32(math.Cos(anim.heading-2.4)), DstY: cy + scaledRadius*float32(math.Sin(anim.heading-2.4)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
			)
			wr.sexualCreatureIs = append(wr.sexualCreatureIs, baseIdx, baseIdx+1, baseIdx+2)
		} else {
			if len(wr.circleCreatureVs)+wr.vertsPerCircle > 60_000 {
				wr.worldLayer.DrawTriangles(wr.circleCreatureVs, wr.circleIsTemplate[:wr.nCreatureCircles*wr.indicesPerCircle], wr.whiteImage, nil)
				wr.circleCreatureVs = wr.circleCreatureVs[:0]
				wr.nCreatureCircles = 0
			}
			base := len(wr.circleCreatureVs)
			wr.circleCreatureVs = wr.circleCreatureVs[:base+wr.vertsPerCircle]
			wr.circleCreatureVs[base] = ebiten.Vertex{DstX: cx, DstY: cy, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
			for j, unit := range wr.unitCircle {
				wr.circleCreatureVs[base+1+j] = ebiten.Vertex{DstX: cx + scaledRadius*unit.x, DstY: cy + scaledRadius*unit.y, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca}
			}
			wr.nCreatureCircles++
		}
	}

	if selectedID != -1 && snapshot != nil {
		if idx, found := wr.lookup[selectedID]; found {
			view := snapshot.Creatures[idx]
			wr.drawFOVCones(wr.worldLayer, map[int]simulation.CreatureView{selectedID: view}, t, camGeoM, zoom)
		}
	}
	if wr.nCreatureCircles > 0 {
		wr.worldLayer.DrawTriangles(wr.circleCreatureVs, wr.circleIsTemplate[:wr.nCreatureCircles*wr.indicesPerCircle], wr.whiteImage, nil)
	}
	if len(wr.sexualCreatureVs) > 0 {
		wr.worldLayer.DrawTriangles(wr.sexualCreatureVs, wr.sexualCreatureIs, wr.whiteImage, nil)
	}

	if selectedID != -1 {
		wr.drawSelectionHighlight(wr.worldLayer, selectedID, camGeoM, zoom)
	}

	screen.DrawImage(wr.worldLayer, nil)
}

// TrySelectCreature finds the closest creature to the screen click.
// Returns the creature ID, or -1 if none found. Deselects if clicking the
// already-selected creature.
func (wr *WorldRenderer) TrySelectCreature(mx, my, sw, sh int, currentSelected int) int {
	clickX, clickY := wr.camera.ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
	half := float64(UnitSize) / 2
	bestID, bestDist := -1, math.Inf(1)
	for id, anim := range wr.animByID {
		dx, dy := clickX-(anim.curX+half), clickY-(anim.curY+half)
		dist := math.Sqrt(dx*dx + dy*dy)
		hitRadius := float64(anim.radius) + 15.0/wr.camera.Zoom
		if dist < hitRadius && dist < bestDist {
			bestDist, bestID = dist, id
		}
	}
	if bestID == -1 || bestID == currentSelected {
		return -1
	}
	return bestID
}

func (wr *WorldRenderer) drawTemperatureBackground(sw, sh int, camGeoM ebiten.GeoM) {
	totalWorldH := float64(int(wr.sim.WorldHeight()) * UnitSize)

	var coldR, coldG, coldB, warmR, warmG, warmB uint8
	if wr.isDark {
		coldR, coldG, coldB = 10, 22, 62
		warmR, warmG, warmB = 72, 18, 8
	} else {
		coldR, coldG, coldB = 130, 160, 210
		warmR, warmG, warmB = 210, 115, 55
	}

	const bandH = float32(4)
	for y := float32(0); y < float32(sh); y += bandH {
		_, worldY := wr.camera.ScreenToWorld(0, float64(y+bandH/2), float64(sw), float64(sh))
		normY := worldY / totalWorldH
		if normY < 0 {
			normY = 0
		} else if normY > 1 {
			normY = 1
		}
		var blend float32
		switch {
		case normY <= 0.2:
			blend = 0
		case normY >= 0.8:
			blend = 1
		default:
			blend = float32((normY - 0.2) / 0.6)
		}
		r := uint8(float32(coldR)*(1-blend) + float32(warmR)*blend)
		gv := uint8(float32(coldG)*(1-blend) + float32(warmG)*blend)
		b := uint8(float32(coldB)*(1-blend) + float32(warmB)*blend)
		vector.FillRect(wr.worldLayer, 0, y, float32(sw), bandH, color.RGBA{r, gv, b, 255}, false)
	}

	params := wr.sim.GetParams()
	radZoneWorldX := params.Environment.Radiation.ZoneWidth * float64(int(wr.sim.WorldWidth())*UnitSize)
	screenRadX, _ := camGeoM.Apply(radZoneWorldX, 0)
	if screenRadX > 0 {
		rw := float32(math.Min(screenRadX, float64(sw)))
		vector.FillRect(wr.worldLayer, 0, 0, rw, float32(sh), color.RGBA{100, 130, 50, 18}, false)
	}
	if screenRadX > 0 && screenRadX < float64(sw) {
		vector.StrokeLine(wr.worldLayer, float32(screenRadX), 0, float32(screenRadX), float32(sh), 2, color.RGBA{100, 200, 70, 40}, false)
	}

	worldW := float64(int(wr.sim.WorldWidth()) * UnitSize)
	worldH := float64(int(wr.sim.WorldHeight()) * UnitSize)
	bx0, by0 := camGeoM.Apply(0, 0)
	bx1, by1 := camGeoM.Apply(worldW, worldH)
	vector.StrokeRect(wr.worldLayer, float32(bx0), float32(by0), float32(bx1-bx0), float32(by1-by0), 2, color.RGBA{180, 180, 180, 100}, false)
}

func (wr *WorldRenderer) drawFOVCones(img *ebiten.Image, views map[int]simulation.CreatureView, t float64, camGeoM ebiten.GeoM, zoom float32) {
	bs := float64(UnitSize)
	half := float32(UnitSize) / 2
	for _, cv := range views {
		if cv.VisionRadius == 0 {
			continue
		}
		worldCX := float32(cv.X*bs) + half
		worldCY := float32(cv.Y*bs) + half
		if anim, ok := wr.animByID[cv.ID]; ok {
			worldCX = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			worldCY = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		scx, scy := camGeoM.Apply(float64(worldCX), float64(worldCY))
		cx, cy := float32(scx), float32(scy)
		r := float32(cv.VisionRadius+cv.Radius) * float32(UnitSize) * zoom
		halfFOV := float64(cv.FieldOfView) / 2.0 * math.Pi / 180.0
		var path vector.Path
		path.MoveTo(cx, cy)
		path.Arc(cx, cy, r, float32(cv.Heading-halfFOV), float32(cv.Heading+halfFOV), vector.Clockwise)
		path.Close()
		const alpha = float32(0.25)
		var cs ebiten.ColorScale
		cs.Scale(float32(cv.R)/255*alpha, float32(cv.G)/255*alpha, float32(cv.B)/255*alpha, alpha)
		vector.FillPath(img, &path, nil, &vector.DrawPathOptions{ColorScale: cs})
	}
}

func (wr *WorldRenderer) drawSelectionHighlight(img *ebiten.Image, selectedID int, camGeoM ebiten.GeoM, zoom float32) {
	if anim, ok := wr.animByID[selectedID]; ok {
		sx, sy := camGeoM.Apply(anim.curX, anim.curY)
		sr := float32(UnitSize) * (5 + anim.radius) * zoom
		vector.StrokeCircle(img, float32(sx), float32(sy), sr, 1.5, color.RGBA{255, 240, 80, 210}, false)
	}
}
