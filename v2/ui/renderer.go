package ui

import (
	"biogo/v2/simulation"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type WorldRenderer struct {
	sim         SimulationState
	renderWorld *RenderWorld
	geometry    *GeometryBuffer
	camera      Camera
	input       *InteractionController

	animByID         map[int]*CreatureAnim
	lookup           map[int]int
	worldLayer       *ebiten.Image
	whiteImage       *ebiten.Image
	screenW, screenH int
	tierFilter       int
	isDark           bool
	lastTickTime     time.Time
	tickDuration     time.Duration

	circleCreatureVs []ebiten.Vertex
	sexualCreatureVs []ebiten.Vertex
	sexualCreatureIs []uint16
	foodVs           []ebiten.Vertex
	nFoodCircles     int
	nCreatureCircles int
}

func NewWorldRenderer(sim SimulationState) *WorldRenderer {
	wImg := ebiten.NewImage(1, 1)
	wImg.Fill(color.White)

	geom := NewGeometryBuffer(12)
	maxPop := sim.GetParams().Population.Max

	wr := &WorldRenderer{
		sim:         sim,
		renderWorld: NewRenderWorld(0, 0, UnitSize),
		geometry:    geom,
		camera:      Camera{Zoom: 1.0},
		animByID:    make(map[int]*CreatureAnim, maxPop),
		lookup:      make(map[int]int, maxPop),
		whiteImage:  wImg,
		tierFilter:  -1,
		isDark:      true,
	}

	wr.input = NewInteractionController(&wr.camera, sim)

	return wr
}

func (wr *WorldRenderer) Camera() *Camera { return &wr.camera }

func (wr *WorldRenderer) HandleContinuousInput(sliderDragging bool) {
	wr.input.HandleContinuousInput(sliderDragging)
}

func (wr *WorldRenderer) StartCameraDrag(x, y int) {
	wr.input.StartCameraDrag(x, y)
}

func (wr *WorldRenderer) StopCameraDrag() bool {
	return wr.input.StopCameraDrag()
}

func (wr *WorldRenderer) CamDragStartPos() (int, int) {
	return wr.input.CamDragStartPos()
}

func (wr *WorldRenderer) ensureSurface(sw, sh int) {
	if wr.worldLayer == nil || wr.screenW != sw || wr.screenH != sh {
		wr.worldLayer = ebiten.NewImage(sw, sh)
		wr.screenW, wr.screenH = sw, sh
	}
	wr.worldLayer.Clear()
}

func (wr *WorldRenderer) UpdateAnimations(snapshot *simulation.StateSnapshot) time.Duration {
	now := time.Now()
	bs := float64(UnitSize)

	currentIDs := make(map[int]bool, len(snapshot.Creatures))
	wr.lookup = make(map[int]int, len(snapshot.Creatures))

	for i, cv := range snapshot.Creatures {
		wr.lookup[cv.ID] = i
		currentIDs[cv.ID] = true
		screenX := float64(cv.X * bs)
		screenY := float64(cv.Y * bs)

		if anim, ok := wr.animByID[cv.ID]; ok {
			anim.PrevX, anim.PrevY = anim.CurX, anim.CurY
			anim.CurX, anim.CurY = screenX, screenY
			anim.R, anim.G, anim.B, anim.A = cv.R, cv.G, cv.B, cv.A
			anim.Heading = cv.Heading
			anim.Mass = cv.CurrentMass
			anim.Radius = float32(cv.Radius * bs)
			anim.Sexual = cv.ReproductionType == 1
			anim.Tier = cv.Tier
		} else {
			wr.animByID[cv.ID] = &CreatureAnim{
				PrevX: screenX, PrevY: screenY,
				CurX: screenX, CurY: screenY,
				R: cv.R, G: cv.G, B: cv.B, A: cv.A,
				Heading: cv.Heading,
				Mass:    cv.CurrentMass,
				Radius:  float32(cv.Radius * bs),
				Sexual:  cv.ReproductionType == 1,
				Tier:    cv.Tier,
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

func (wr *WorldRenderer) TrySelectCreature(mx, my, sw, sh int, currentSelected int) int {
	clickX, clickY := wr.camera.ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
	half := float64(UnitSize) / 2
	bestID, bestDist := -1, math.Inf(1)
	for id, anim := range wr.animByID {
		dx, dy := clickX-(anim.CurX+half), clickY-(anim.CurY+half)
		dist := math.Sqrt(dx*dx + dy*dy)
		hitRadius := float64(anim.Radius) + 15.0/wr.camera.Zoom
		if dist < hitRadius && dist < bestDist {
			bestDist, bestID = dist, id
		}
	}
	if bestID == -1 || bestID == currentSelected {
		return -1
	}
	return bestID
}

func (wr *WorldRenderer) CycleTierFilter() string {
	tierSet := make(map[int]struct{})
	for _, anim := range wr.animByID {
		tierSet[int(anim.Tier)] = struct{}{}
	}
	tiers := make([]int, 0, len(tierSet))
	for t := range tierSet {
		tiers = append(tiers, t)
	}

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

func (wr *WorldRenderer) ToggleDark() { wr.isDark = !wr.isDark }

func (wr *WorldRenderer) Draw(screen *ebiten.Image, snapshot *simulation.StateSnapshot, selectedID int) {

	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	wr.ensureSurface(sw, sh)
	wr.input.Update(false)

	camGeoM := wr.camera.GeoM(float64(sw), float64(sh))
	zoom := float32(wr.camera.Zoom)

	t := 1.0
	if wr.tickDuration > 0 {
		if elapsed := time.Since(wr.lastTickTime); elapsed < wr.tickDuration {
			t = float64(elapsed) / float64(wr.tickDuration)
		}
	}

	wr.drawBackgroundPass(sw, sh, camGeoM)

	if snapshot != nil {
		wr.drawFoodPass(snapshot, camGeoM, zoom)
		wr.drawCreaturePass(snapshot, camGeoM, zoom, t, selectedID)
	}

	screen.DrawImage(wr.worldLayer, nil)
}

func (wr *WorldRenderer) drawBackgroundPass(sw, sh int, camGeoM ebiten.GeoM) {
	totalWorldH := float64(int(wr.sim.WorldHeight()) * UnitSize)

	var coldR, coldG, coldB, warmR, warmG, warmB float32
	if wr.isDark {
		coldR, coldG, coldB = 10, 22, 62
		warmR, warmG, warmB = 72, 18, 8
	} else {
		coldR, coldG, coldB = 130, 160, 210
		warmR, warmG, warmB = 210, 115, 55
	}

	_, worldTopY := wr.camera.ScreenToWorld(0, 0, float64(sw), float64(sh))
	_, worldBotY := wr.camera.ScreenToWorld(0, float64(sh), float64(sw), float64(sh))
	worldDeltaY := worldBotY - worldTopY

	const bandH = float32(4)
	for y := float32(0); y < float32(sh); y += bandH {
		screenRatio := float64(y+bandH/2) / float64(sh)
		worldY := worldTopY + screenRatio*worldDeltaY

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

		r := uint8(coldR*(1-blend) + warmR*blend)
		gv := uint8(coldG*(1-blend) + warmG*blend)
		b := uint8(coldB*(1-blend) + warmB*blend)
		vector.FillRect(wr.worldLayer, 0, y, float32(sw), bandH, color.RGBA{r, gv, b, 255}, false)
	}

	wr.renderWorld.DrawBackground(wr.worldLayer, camGeoM)

	params := wr.sim.GetParams()
	radZoneWorldX := params.Environment.Radiation.ZoneWidth * float64(int(wr.sim.WorldWidth())*UnitSize)
	screenRadX, _ := camGeoM.Apply(radZoneWorldX, 0)
	if screenRadX > 0 {
		rw := float32(math.Min(screenRadX, float64(sw)))
		const strips = 16
		var lastX float32 = 0
		for i := 0; i < strips; i++ {
			nextFrac := float32(i+1) / float32(strips)
			nextX := float32(math.Round(float64(nextFrac * rw)))
			x := lastX
			w := nextX - lastX

			frac := float32(i) / float32(strips)
			alpha := uint8(1 + frac*frac*8)

			if w > 0 {
				vector.FillRect(wr.worldLayer, x, 0, w, float32(sh), color.NRGBA{100, 200, 30, alpha}, false)
			}
			lastX = nextX
		}
	}
	if screenRadX > 0 && screenRadX < float64(sw) {
		rx := float32(screenRadX)
		vector.StrokeLine(wr.worldLayer, rx, 0, rx, float32(sh), 10, color.NRGBA{140, 240, 50, 8}, false)
		vector.StrokeLine(wr.worldLayer, rx, 0, rx, float32(sh), 1.5, color.NRGBA{150, 250, 60, 15}, false)
	}

	worldW := float64(int(wr.sim.WorldWidth()) * UnitSize)
	worldH := float64(int(wr.sim.WorldHeight()) * UnitSize)
	bx0, by0 := camGeoM.Apply(0, 0)
	bx1, by1 := camGeoM.Apply(worldW, worldH)
	vector.StrokeRect(wr.worldLayer, float32(bx0), float32(by0), float32(bx1-bx0), float32(by1-by0), 2, color.RGBA{180, 180, 180, 100}, false)
}

func (wr *WorldRenderer) drawFoodPass(snapshot *simulation.StateSnapshot, camGeoM ebiten.GeoM, zoom float32) {
	const pr, pg, pb, pa = float32(65) / 255, float32(140) / 255, float32(55) / 255, float32(125) / 255
	const mr, mg, mb, ma = float32(180) / 255, float32(30) / 255, float32(30) / 255, float32(180) / 255
	const fr, fg, fb, fa = float32(150) / 255, float32(50) / 255, float32(190) / 255, float32(140) / 255

	bs := float64(UnitSize)
	wr.foodVs = wr.foodVs[:0]

	var foodIs []uint16
	segments := len(wr.geometry.UnitCircle)

	flushFood := func() {
		if len(foodIs) > 0 {
			wr.worldLayer.DrawTriangles(wr.foodVs, foodIs, wr.whiteImage, nil)
			wr.foodVs = wr.foodVs[:0]
			foodIs = foodIs[:0]
		}
	}

	for _, fv := range snapshot.Food {

		if len(wr.foodVs)+wr.geometry.VertsPerCircle > 60_000 {
			flushFood()
		}

		scx, scy := camGeoM.Apply(fv.X*bs, fv.Y*bs)
		cx, cy := float32(scx), float32(scy)
		r := float32(fv.Radius*bs) * zoom

		var cr, cg, cb, ca float32
		switch fv.Type {
		case 0:
			cr, cg, cb, ca = pr, pg, pb, pa
		case 1:
			cr, cg, cb, ca = mr, mg, mb, ma
		default:
			cr, cg, cb, ca = fr, fg, fb, fa
		}

		baseVert := uint16(len(wr.foodVs))

		wr.foodVs = append(wr.foodVs, ebiten.Vertex{
			DstX: cx, DstY: cy,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})

		for _, u := range wr.geometry.UnitCircle {
			wr.foodVs = append(wr.foodVs, ebiten.Vertex{
				DstX: cx + r*u.x, DstY: cy + r*u.y,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
		}

		for j := 0; j < segments; j++ {
			nextEdge := uint16(j + 2)
			if j == segments-1 {
				nextEdge = 1
			}
			foodIs = append(foodIs, baseVert, baseVert+1+uint16(j), baseVert+nextEdge)
		}
	}
	flushFood()
}

func (wr *WorldRenderer) drawCreaturePass(snapshot *simulation.StateSnapshot, camGeoM ebiten.GeoM, zoom float32, t float64, selectedID int) {
	wr.circleCreatureVs = wr.circleCreatureVs[:0]
	wr.sexualCreatureVs = wr.sexualCreatureVs[:0]
	wr.sexualCreatureIs = wr.sexualCreatureIs[:0]

	var circleCreatureIs []uint16
	segments := len(wr.geometry.UnitCircle)

	var selectedView *simulation.CreatureView
	var selectedCx, selectedCy float32

	for _, anim := range wr.animByID {
		lerpX := anim.PrevX + (anim.CurX-anim.PrevX)*t
		lerpY := anim.PrevY + (anim.CurY-anim.PrevY)*t
		slx, sly := camGeoM.Apply(lerpX, lerpY)
		cx, cy := float32(slx), float32(sly)
		scaledRadius := anim.Radius * zoom
		cr, cg, cb, ca := float32(anim.R)/255, float32(anim.G)/255, float32(anim.B)/255, float32(anim.A)/255

		if wr.tierFilter != -1 && int(anim.Tier) != wr.tierFilter {
			ca *= 0.2
		}

		if anim.Sexual {
			baseIdx := uint16(len(wr.sexualCreatureVs))
			cosH := float32(math.Cos(anim.Heading))
			sinH := float32(math.Sin(anim.Heading))
			cosH1 := float32(math.Cos(anim.Heading + 2.4))
			sinH1 := float32(math.Sin(anim.Heading + 2.4))
			cosH2 := float32(math.Cos(anim.Heading - 2.4))
			sinH2 := float32(math.Sin(anim.Heading - 2.4))

			wr.sexualCreatureVs = append(wr.sexualCreatureVs,
				ebiten.Vertex{DstX: cx + scaledRadius*cosH, DstY: cy + scaledRadius*sinH, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + scaledRadius*cosH1, DstY: cy + scaledRadius*sinH1, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
				ebiten.Vertex{DstX: cx + scaledRadius*cosH2, DstY: cy + scaledRadius*sinH2, ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca},
			)
			wr.sexualCreatureIs = append(wr.sexualCreatureIs, baseIdx, baseIdx+1, baseIdx+2)
		} else {
			if len(wr.circleCreatureVs)+wr.geometry.VertsPerCircle > 60_000 {
				wr.worldLayer.DrawTriangles(wr.circleCreatureVs, circleCreatureIs, wr.whiteImage, nil)
				wr.circleCreatureVs = wr.circleCreatureVs[:0]
				circleCreatureIs = circleCreatureIs[:0]
			}

			baseVert := uint16(len(wr.circleCreatureVs))
			wr.circleCreatureVs = append(wr.circleCreatureVs, ebiten.Vertex{
				DstX: cx, DstY: cy,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})

			for _, unit := range wr.geometry.UnitCircle {
				wr.circleCreatureVs = append(wr.circleCreatureVs, ebiten.Vertex{
					DstX: cx + scaledRadius*unit.x, DstY: cy + scaledRadius*unit.y,
					ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
				})
			}

			for j := 0; j < segments; j++ {
				nextEdge := uint16(j + 2)
				if j == segments-1 {
					nextEdge = 1
				}
				circleCreatureIs = append(circleCreatureIs, baseVert, baseVert+1+uint16(j), baseVert+nextEdge)
			}
		}
	}

	if len(wr.circleCreatureVs) > 0 {
		wr.worldLayer.DrawTriangles(wr.circleCreatureVs, circleCreatureIs, wr.whiteImage, nil)
	}
	if len(wr.sexualCreatureVs) > 0 {
		wr.worldLayer.DrawTriangles(wr.sexualCreatureVs, wr.sexualCreatureIs, wr.whiteImage, nil)
	}

	if selectedID != -1 && snapshot != nil {
		if idx, found := wr.lookup[selectedID]; found {
			if idx >= 0 && idx < len(snapshot.Creatures) {
				v := snapshot.Creatures[idx]
				selectedView = &v
				if anim, ok := wr.animByID[selectedID]; ok {
					lx := anim.PrevX + (anim.CurX-anim.PrevX)*t
					ly := anim.PrevY + (anim.CurY-anim.PrevY)*t
					sx, sy := camGeoM.Apply(lx, ly)
					selectedCx, selectedCy = float32(sx), float32(sy)
				}
			}
		}
	}

	if selectedView != nil {
		wr.drawFOVCones(wr.worldLayer, selectedView, selectedCx, selectedCy, zoom)
		wr.drawSelectionHighlight(wr.worldLayer, selectedID, camGeoM, zoom)
	}
}

func (wr *WorldRenderer) drawFOVCones(img *ebiten.Image, cv *simulation.CreatureView, cx, cy float32, zoom float32) {
	if cv.VisionRadius == 0 {
		return
	}
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

func (wr *WorldRenderer) drawSelectionHighlight(img *ebiten.Image, selectedID int, camGeoM ebiten.GeoM, zoom float32) {
	if anim, ok := wr.animByID[selectedID]; ok {
		sx, sy := camGeoM.Apply(anim.CurX, anim.CurY)
		sr := float32(UnitSize) * (5 + anim.Radius) * zoom
		vector.StrokeCircle(img, float32(sx), float32(sy), sr, 1.5, color.RGBA{255, 240, 80, 210}, false)
	}
}

func (wr *WorldRenderer) ResetAnimations() {
	wr.animByID = make(map[int]*CreatureAnim)
	wr.lastTickTime = time.Time{}
	wr.tickDuration = 0
}
