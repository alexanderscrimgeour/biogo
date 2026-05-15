package ui

import (
	"biogo/v2/simulation"
	"image/color"
	"math"
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
}

// WorldRenderer handles all world-space rendering: temperature gradient,
// food, meat, creatures, FOV cones, selection highlight, and camera transform.
type WorldRenderer struct {
	sim          SimulationState
	renderWorld  *RenderWorld
	camera       Camera
	worldLayer   *ebiten.Image
	animByID     map[int]*creatureAnim
	lookup       map[int]int
	unitCircle   []struct{ x, y float32 }
	creatureVs   []ebiten.Vertex
	creatureIs   []uint16
	whiteImage   *ebiten.Image
	isDark       bool
	lastTickTime time.Time
	tickDuration time.Duration

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

	const initCap = 500
	return &WorldRenderer{
		sim:         sim,
		renderWorld: NewRenderWorld(0, 0, UnitSize),
		camera:      Camera{X: float64(worldW) / 2, Y: float64(worldH) / 2, Zoom: 1.0},
		worldLayer:  ebiten.NewImage(worldW, worldH),
		animByID:    make(map[int]*creatureAnim),
		lookup:      make(map[int]int),
		whiteImage:  wImg,
		unitCircle:  unitCircle,
		isDark:      true,
		creatureVs:  make([]ebiten.Vertex, 0, initCap*(1+segments+1)),
		creatureIs:  make([]uint16, 0, initCap*segments*3),
	}
}

// ToggleDark flips the world background theme.
func (wr *WorldRenderer) ToggleDark() { wr.isDark = !wr.isDark }

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
		} else {
			wr.animByID[cv.ID] = &creatureAnim{
				prevX: screenX, prevY: screenY,
				curX: screenX, curY: screenY,
				r: cv.R, g: cv.G, b: cv.B, a: cv.A,
				heading: cv.Heading,
				mass:    cv.CurrentMass,
				radius:  float32(cv.Radius * bs),
				sexual:  cv.ReproductionType == 1,
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

	t := 1.0
	if wr.tickDuration > 0 {
		elapsed := time.Since(wr.lastTickTime)
		t = float64(elapsed) / float64(wr.tickDuration)
		if t > 1 {
			t = 1
		}
	}

	wr.drawTemperatureBackground()
	wr.renderWorld.DrawBackground(wr.worldLayer)

	bs := float64(UnitSize)
	if snapshot != nil {
		circ := len(wr.unitCircle)
		vertsPerCircle := 1 + circ

		fr, fg, fb, fa := float32(65)/255, float32(140)/255, float32(55)/255, float32(125)/255
		fVs := make([]ebiten.Vertex, 0, min(len(snapshot.Food)*vertsPerCircle, 60_000))
		fIs := make([]uint16, 0, min(len(snapshot.Food)*circ*3, 60_000))
		flushFood := func() {
			if len(fVs) > 0 {
				wr.worldLayer.DrawTriangles(fVs, fIs, wr.whiteImage, nil)
				fVs, fIs = fVs[:0], fIs[:0]
			}
		}
		for _, fv := range snapshot.Food {
			if len(fVs)+vertsPerCircle > 60_000 {
				flushFood()
			}
			cx, cy, r := float32(fv.X*bs), float32(fv.Y*bs), float32(fv.Radius*bs)
			base := uint16(len(fVs))
			fVs = append(fVs, ebiten.Vertex{DstX: cx, DstY: cy, ColorR: fr, ColorG: fg, ColorB: fb, ColorA: fa})
			for _, u := range wr.unitCircle {
				fVs = append(fVs, ebiten.Vertex{DstX: cx + r*u.x, DstY: cy + r*u.y, ColorR: fr, ColorG: fg, ColorB: fb, ColorA: fa})
			}
			for i := uint16(1); i <= uint16(circ-1); i++ {
				fIs = append(fIs, base, base+i, base+i+1)
			}
		}
		flushFood()

		mr, mg, mb2, ma := float32(180)/255, float32(30)/255, float32(30)/255, float32(180)/255
		mVs := make([]ebiten.Vertex, 0, min(len(snapshot.Meat)*vertsPerCircle, 60_000))
		mIs := make([]uint16, 0, min(len(snapshot.Meat)*circ*3, 60_000))
		flushMeat := func() {
			if len(mVs) > 0 {
				wr.worldLayer.DrawTriangles(mVs, mIs, wr.whiteImage, nil)
				mVs, mIs = mVs[:0], mIs[:0]
			}
		}
		for _, mv := range snapshot.Meat {
			if len(mVs)+vertsPerCircle > 60_000 {
				flushMeat()
			}
			cx, cy, r := float32(mv.X*bs), float32(mv.Y*bs), float32(mv.Radius*bs)
			base := uint16(len(mVs))
			mVs = append(mVs, ebiten.Vertex{DstX: cx, DstY: cy, ColorR: mr, ColorG: mg, ColorB: mb2, ColorA: ma})
			for _, u := range wr.unitCircle {
				mVs = append(mVs, ebiten.Vertex{DstX: cx + r*u.x, DstY: cy + r*u.y, ColorR: mr, ColorG: mg, ColorB: mb2, ColorA: ma})
			}
			for i := uint16(1); i <= uint16(circ-1); i++ {
				mIs = append(mIs, base, base+i, base+i+1)
			}
		}
		flushMeat()
	}

	wr.creatureVs = wr.creatureVs[:0]
	wr.creatureIs = wr.creatureIs[:0]
	vertsPerCreature := 1 + len(wr.unitCircle)
	flushCreatures := func() {
		if len(wr.creatureVs) > 0 {
			wr.worldLayer.DrawTriangles(wr.creatureVs, wr.creatureIs, wr.whiteImage, nil)
			wr.creatureVs = wr.creatureVs[:0]
			wr.creatureIs = wr.creatureIs[:0]
		}
	}

	for _, anim := range wr.animByID {
		if len(wr.creatureVs)+vertsPerCreature > 60_000 {
			flushCreatures()
		}
		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		cx, cy := float32(lerpX), float32(lerpY)
		cr, cg, cb, ca := float32(anim.r)/255, float32(anim.g)/255, float32(anim.b)/255, float32(anim.a)/255
		baseIdx := uint16(len(wr.creatureVs))

		if anim.sexual {
			wr.creatureVs = append(wr.creatureVs,
				ebiten.Vertex{DstX: cx + anim.radius*float32(math.Cos(anim.heading)), DstY: cy + anim.radius*float32(math.Sin(anim.heading)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + anim.radius*float32(math.Cos(anim.heading+2.4)), DstY: cy + anim.radius*float32(math.Sin(anim.heading+2.4)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + anim.radius*float32(math.Cos(anim.heading-2.4)), DstY: cy + anim.radius*float32(math.Sin(anim.heading-2.4)), ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
			)
			wr.creatureIs = append(wr.creatureIs, baseIdx, baseIdx+1, baseIdx+2)
		} else {
			wr.creatureVs = append(wr.creatureVs, ebiten.Vertex{DstX: cx, DstY: cy, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
			for _, unit := range wr.unitCircle {
				wr.creatureVs = append(wr.creatureVs, ebiten.Vertex{DstX: cx + anim.radius*unit.x, DstY: cy + anim.radius*unit.y, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca})
			}
			for i := uint16(1); i <= uint16(len(wr.unitCircle)-1); i++ {
				wr.creatureIs = append(wr.creatureIs, baseIdx, baseIdx+i, baseIdx+i+1)
			}
		}
	}

	if selectedID != -1 && snapshot != nil {
		if idx, found := wr.lookup[selectedID]; found {
			view := snapshot.Creatures[idx]
			wr.drawFOVCones(wr.worldLayer, map[int]simulation.CreatureView{selectedID: view}, t)
		}
	}
	flushCreatures()

	if selectedID != -1 {
		wr.drawSelectionHighlight(wr.worldLayer, selectedID)
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM = wr.camera.GeoM(float64(sw), float64(sh))
	screen.DrawImage(wr.worldLayer, op)
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
		hitRadius := float64(anim.radius) + float64(UnitSize)*3
		if dist < hitRadius && dist < bestDist {
			bestDist, bestID = dist, id
		}
	}
	if bestID == -1 || bestID == currentSelected {
		return -1
	}
	return bestID
}

func (wr *WorldRenderer) drawTemperatureBackground() {
	worldW := float32(int(wr.sim.WorldWidth()) * UnitSize)
	worldH := float32(int(wr.sim.WorldHeight()) * UnitSize)

	var coldR, coldG, coldB, warmR, warmG, warmB uint8
	if wr.isDark {
		coldR, coldG, coldB = 10, 22, 62
		warmR, warmG, warmB = 72, 18, 8
	} else {
		coldR, coldG, coldB = 130, 160, 210
		warmR, warmG, warmB = 210, 115, 55
	}

	params := wr.sim.GetParams()
	radZoneW := float32(params.RadiationZoneWidth * float64(int(wr.sim.WorldWidth())*UnitSize))

	const bandH = float32(4)
	for y := float32(0); y < worldH; y += bandH {
		normY := y / worldH
		var t float32
		switch {
		case normY <= 0.2:
			t = 0
		case normY >= 0.8:
			t = 1
		default:
			t = (normY - 0.2) / 0.6
		}
		r := uint8(float32(coldR)*(1-t) + float32(warmR)*t)
		gv := uint8(float32(coldG)*(1-t) + float32(warmG)*t)
		b := uint8(float32(coldB)*(1-t) + float32(warmB)*t)
		vector.FillRect(wr.worldLayer, 0, y, worldW, bandH, color.RGBA{r, gv, b, 255}, false)
	}
	vector.FillRect(wr.worldLayer, 0, 0, radZoneW, worldH, color.RGBA{100, 130, 50, 40}, false)
	vector.StrokeLine(wr.worldLayer, radZoneW, 0, radZoneW, worldH, 2, color.RGBA{100, 255, 70, 60}, false)
}

func (wr *WorldRenderer) drawFOVCones(img *ebiten.Image, views map[int]simulation.CreatureView, t float64) {
	bs := float64(UnitSize)
	half := float32(UnitSize) / 2
	for _, cv := range views {
		if cv.SightDistance == 0 {
			continue
		}
		cx, cy := float32(cv.X*bs)+half, float32(cv.Y*bs)+half
		if anim, ok := wr.animByID[cv.ID]; ok {
			cx = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			cy = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		r := float32(cv.SightDistance) * float32(UnitSize)
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

func (wr *WorldRenderer) drawSelectionHighlight(img *ebiten.Image, selectedID int) {
	if anim, ok := wr.animByID[selectedID]; ok {
		vector.StrokeCircle(img, float32(anim.curX), float32(anim.curY), float32(UnitSize)*(5+anim.radius), 1.5, color.RGBA{255, 240, 80, 210}, false)
	}
}
