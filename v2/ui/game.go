package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// SimulationState is the interface the UI requires
type SimulationState interface {
	Update()
	CreatureViews() map[int]simulation.CreatureView
	FoodViews() []simulation.FoodView
	CorpseViews() []simulation.CorpseView
	GridWidth() int
	GridHeight() int
	PopulationCount() int
	FoodCount() int
	AverageAge() float64
	// Prioritized Mass over Size
	CreatureMinMass() byte
	CreatureMaxMass() byte
	SaveCreature(id int) error
	Reset()
	CreatureDetail(id int) (simulation.CreatureDetailView, bool)
	SetSpawnMutationRate(rate float32)
}

var foodColor = color.RGBA{R: 50, G: 200, B: 60, A: 255}

const historyLen = 10000

type histSample struct {
	pop  int
	food int
}

const (

	// Spawn mutation rate slider
	spawnMutSliderX = 395
	spawnMutSliderY = 10
	spawnMutSliderW = 240
	spawnMutSliderH = 24
	spawnMutTrackX  = 530
	spawnMutTrackY  = 19
	spawnMutTrackW  = 100
	spawnMutTrackH  = 6

	spawnMutRateMin = 0.0001
	spawnMutRateMax = 0.2

	// Creature detail panel — sits below the history graph (graphPanelY+graphPanelH+4)
	detailPanelX   = 10
	detailPanelY   = 168
	detailPanelW   = 210
	detailPanelH   = 435
	detailTpad     = 8
	detailSaveBtnH = 22

	// History graph panel (top-left, below button bar)
	graphPanelX = 10
	graphPanelY = 44
	graphPanelW = 220
	graphPanelH = 120
	graphTextH  = 36
	graphPad    = 4
)

type Component interface {
	Draw(screen *ebiten.Image, x, y float32) (float32, float32)
}

type creatureAnim struct {
	prevX, prevY float64
	curX, curY   float64
	r, g, b, a   uint8
	heading      float64
	mass         byte // Prioritized mass
}

type Game struct {
	sim                SimulationState
	renderGrid         *RenderGrid
	foodBlobsByKey     map[string]*Blob
	corpseBlobsByID    map[int]*Blob
	statFont           font.Face
	whiteImage         *ebiten.Image
	animByID           map[int]*creatureAnim
	lastTickTime       time.Time
	tickDuration       time.Duration
	minCreatureMass    byte
	maxCreatureMass    byte
	saveFeedback       string
	saveFeedbackAt     time.Time
	isDarkBackground   bool
	selectedCreatureID int
	paused             bool
	spawnMutRate       float32
	spawnMutDragging   bool
	history            [historyLen]histSample
	histHead           int
	histCount          int
	pauseBtn           *components.Button
	newGameBtn         *components.Button
	themeBtn           *components.Button
	saveBtn            *components.Button
	spawnMutSlider     *components.Slider
	detailsPanel       *components.Panel
}

var BlockSize int = 2

func NewGame(sim SimulationState) *Game {
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	statFont, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	wImg := ebiten.NewImage(1, 1)
	wImg.Fill(color.White)

	g := &Game{
		sim:                sim,
		renderGrid:         NewRenderGrid(0, 0, BlockSize),
		foodBlobsByKey:     make(map[string]*Blob),
		corpseBlobsByID:    make(map[int]*Blob),
		statFont:           statFont,
		whiteImage:         wImg,
		animByID:           make(map[int]*creatureAnim),
		minCreatureMass:    sim.CreatureMinMass(),
		maxCreatureMass:    sim.CreatureMaxMass(),
		selectedCreatureID: -1,
		spawnMutRate:       0.01,
		isDarkBackground:   true,
	}
	g.pauseBtn = &components.Button{
		X: 10, Y: 10, W: 80, H: 24,
		Label:      "Pause",
		Color:      components.ColorButtonRed,
		LabelColor: color.White,
		OnClick: func() {
			g.paused = !g.paused
			// Update label dynamically based on state
			if g.paused {
				g.pauseBtn.Label = "Resume"
				g.pauseBtn.Color = components.ColorButtonGreen
			} else {
				g.pauseBtn.Label = "Pause"
				g.pauseBtn.Color = components.ColorButtonRed
			}
		},
	}
	g.newGameBtn = &components.Button{
		X: 100, Y: 10, W: 90, H: 24,
		Label:      "Restart",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		OnClick: func() {
			g.sim.Reset()
			g.animByID = make(map[int]*creatureAnim)
			for _, b := range g.foodBlobsByKey {
				g.renderGrid.RemoveFoodBlob(b)
			}
			g.foodBlobsByKey = make(map[string]*Blob)
			for _, b := range g.corpseBlobsByID {
				g.renderGrid.RemoveBlob(b)
			}
			g.corpseBlobsByID = make(map[int]*Blob)
			g.selectedCreatureID = -1
			g.lastTickTime = time.Time{}
			g.histHead = 0
			g.histCount = 0
		},
	}
	g.themeBtn = &components.Button{
		X: 200, Y: 10, W: 100, H: 24,
		Label:      "Theme",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		OnClick: func() {
			g.isDarkBackground = !g.isDarkBackground
		},
	}
	g.spawnMutSlider = &components.Slider{
		X: 315, Y: 10, W: 240, H: 24,
		TrackX: 530, TrackW: 100,
		Label:      "Mutation Rate",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		Min:        0.0001, Max: 0.2,
		Value: 0.01,
	}
	g.detailsPanel = &components.Panel{
		X:         float32(detailPanelX),
		Y:         float32(detailPanelY),
		W:         float32(detailPanelW),
		H:         float32(detailPanelH),
		BaseColor: color.RGBA{8, 10, 22, 215},   // Dark translucent navy
		Border:    color.RGBA{90, 90, 150, 255}, // Muted blue-gray
	}
	const wallThickness = 10.0
	bs := float64(BlockSize)
	cx := float64(sim.GridWidth()) / 2
	cy := float64(sim.GridHeight()) / 2

	// Vertical bar of the cross
	g.renderGrid.AddLine(
		(cx-wallThickness/2)*bs, float64(sim.GridHeight())/4*bs,
		(cx+wallThickness/2)*bs, float64(sim.GridHeight())*3/4*bs,
	)
	// Horizontal bar of the cross
	g.renderGrid.AddLine(
		float64(sim.GridWidth())/4*bs, (cy-wallThickness/2)*bs,
		float64(sim.GridWidth())*3/4*bs, (cy+wallThickness/2)*bs,
	)
	return g
}

func (g *Game) handleContinuousInput() {
	if g.spawnMutSlider.Dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			mx, _ := ebiten.CursorPosition()
			g.spawnMutSlider.UpdateValue(mx)
			g.sim.SetSpawnMutationRate(float32(g.spawnMutSlider.Value))
		} else {
			g.spawnMutSlider.Dragging = false
		}
	}
}

func (g *Game) handleInput() bool {
	isClick := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	mx, my := ebiten.CursorPosition()
	if isClick {

		if g.pauseBtn.IsClicked(mx, my) {
			g.pauseBtn.OnClick()
			return true
		}

		if g.newGameBtn.IsClicked(mx, my) {
			g.newGameBtn.OnClick()
			return true
		}

		if g.themeBtn.IsClicked(mx, my) {
			g.themeBtn.OnClick()
			return true
		}

		if g.saveBtn != nil {
			if g.saveBtn.IsClicked(mx, my) {
				g.saveBtn.OnClick()
				return true
			}
		}

		if g.spawnMutSlider.InBounds(mx, my) {
			g.spawnMutSlider.Dragging = true
			g.spawnMutSlider.UpdateValue(mx)
			g.sim.SetSpawnMutationRate(float32(g.spawnMutSlider.Value))
			return true
		}
	}
	return false

}

func (g *Game) Update() error {
	g.handleContinuousInput()
	mx, my := ebiten.CursorPosition()
	now := time.Now()

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if !g.handleInput() {
			g.trySelectCreature(mx, my)
		}
	}

	if !g.paused {
		prevByID := make(map[int][2]float64, len(g.animByID))
		for id, anim := range g.animByID {
			prevByID[id] = [2]float64{anim.curX, anim.curY}
		}

		g.sim.Update()

		g.history[g.histHead] = histSample{pop: g.sim.PopulationCount(), food: g.sim.FoodCount()}
		g.histHead = (g.histHead + 1) % historyLen
		if g.histCount < historyLen {
			g.histCount++
		}

		views := g.sim.CreatureViews()
		currentIDs := make(map[int]bool, len(views))
		bs := float64(BlockSize)

		for _, cv := range views {
			currentIDs[cv.ID] = true
			screenX := float64(cv.X * bs)
			screenY := float64(cv.Y * bs)

			if anim, ok := g.animByID[cv.ID]; ok {
				if prev, ok := prevByID[cv.ID]; ok {
					anim.prevX, anim.prevY = prev[0], prev[1]
				} else {
					anim.prevX, anim.prevY = screenX, screenY
				}
				anim.curX, anim.curY = screenX, screenY
				anim.r, anim.g, anim.b, anim.a = cv.R, cv.G, cv.B, cv.A
				anim.heading = cv.Heading
				anim.mass = cv.Mass // Prioritise mass
			} else {
				g.animByID[cv.ID] = &creatureAnim{
					prevX: screenX, prevY: screenY,
					curX: screenX, curY: screenY,
					r: cv.R, g: cv.G, b: cv.B, a: cv.A,
					heading: cv.Heading,
					mass:    cv.Mass,
				}
			}
		}
		for id := range g.animByID {
			if !currentIDs[id] {
				delete(g.animByID, id)
			}
		}

		// Reconcile food
		foodViews := g.sim.FoodViews()
		currentFood := make(map[string]bool, len(foodViews))
		for _, fv := range foodViews {
			key := foodKey(fv.X, fv.Y)
			currentFood[key] = true
			if _, ok := g.foodBlobsByKey[key]; !ok {
				blob := g.renderGrid.AddFoodBlob(BlockSize, foodColor)
				blob.Translate(float64(fv.X*bs), float64(fv.Y*bs))
				g.foodBlobsByKey[key] = blob
			}
		}
		for key, blob := range g.foodBlobsByKey {
			if !currentFood[key] {
				g.renderGrid.RemoveFoodBlob(blob)
				delete(g.foodBlobsByKey, key)
			}
		}

		// Reconcile corpses
		corpseViews := g.sim.CorpseViews()
		currentCorpses := make(map[int]bool, len(corpseViews))
		for _, cv := range corpseViews {
			currentCorpses[cv.ID] = true
			alpha := uint8(cv.EnergyFraction * 220)
			corpseColor := color.RGBA{R: 120, G: 60, B: 20, A: alpha}
			if blob, ok := g.corpseBlobsByID[cv.ID]; ok {
				blob.Move(float64(cv.X*bs), float64(cv.Y*bs))
				blob.SetColor(corpseColor)
			} else {
				blob := g.renderGrid.AddBlob(BlockSize, corpseColor)
				blob.Translate(float64(cv.X*bs), float64(cv.Y*bs))
				if g.corpseBlobsByID == nil {
					g.corpseBlobsByID = make(map[int]*Blob)
				}
				g.corpseBlobsByID[cv.ID] = blob
			}
		}
		for id, blob := range g.corpseBlobsByID {
			if !currentCorpses[id] {
				g.renderGrid.RemoveBlob(blob)
				delete(g.corpseBlobsByID, id)
			}
		}

		if !g.lastTickTime.IsZero() {
			g.tickDuration = now.Sub(g.lastTickTime)
		}
		g.lastTickTime = now
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.isDarkBackground {
		screen.Fill(color.RGBA{5, 5, 10, 255})
	} else {
		screen.Fill(color.RGBA{100, 100, 100, 255})
	}
	g.renderGrid.DrawBackground(screen)

	t := 1.0
	if g.tickDuration > 0 && !g.paused {
		elapsed := time.Since(g.lastTickTime)
		t = float64(elapsed) / float64(g.tickDuration)
		if t > 1 {
			t = 1
		}
	}

	for _, blob := range g.corpseBlobsByID {
		blob.Draw(screen)
	}

	var creatureVs []ebiten.Vertex
	var creatureIs []uint16
	massRange := float64(g.maxCreatureMass) - float64(g.minCreatureMass)
	half := float64(BlockSize) / 2

	for _, anim := range g.animByID {
		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		cx, cy := float32(lerpX+half), float32(lerpY+half)

		var r float64
		if massRange > 0 {
			massT := (float64(anim.mass) - float64(g.minCreatureMass)) / massRange
			r = float64(BlockSize) + math.Max(0, math.Min(1, massT))*float64(5*BlockSize)
		} else {
			r = float64(BlockSize)
		}

		cr, cg, cb, ca := float32(anim.r)/255, float32(anim.g)/255, float32(anim.b)/255, float32(anim.a)/255
		baseIdx := uint16(len(creatureVs))
		for _, offset := range [3]float64{0, 2 * math.Pi / 3, -2 * math.Pi / 3} {
			creatureVs = append(creatureVs, ebiten.Vertex{
				DstX:   cx + float32(r)*float32(math.Cos(anim.heading+offset)),
				DstY:   cy + float32(r)*float32(math.Sin(anim.heading+offset)),
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
		}
		creatureIs = append(creatureIs, baseIdx, baseIdx+1, baseIdx+2)
	}
	if len(creatureVs) > 0 {
		screen.DrawTriangles(creatureVs, creatureIs, g.whiteImage, nil)
	}

	g.drawHistoryGraph(screen)
	if g.selectedCreatureID != -1 {
		g.drawSelectionHighlight(screen)
		if detail, ok := g.sim.CreatureDetail(g.selectedCreatureID); ok {
			g.drawCreatureDetail(screen, detail)
			g.drawFOVCones(screen, map[int]simulation.CreatureView{g.selectedCreatureID: g.sim.CreatureViews()[g.selectedCreatureID]}, t)
		} else {
			g.selectedCreatureID = -1
		}
	}
	g.pauseBtn.Draw(screen, g.statFont)
	g.themeBtn.Draw(screen, g.statFont)
	g.newGameBtn.Draw(screen, g.statFont)
	if g.spawnMutSlider != nil {
		g.drawSpawnMutSlider(screen)
	}

	g.addStatLine(screen, "Population", fmt.Sprintf("%d", g.sim.PopulationCount()), 1)
	g.addStatLine(screen, "Food", fmt.Sprintf("%d", g.sim.FoodCount()), 2)
	g.addStatLine(screen, "Avg Age", fmt.Sprintf("%.0f", g.sim.AverageAge()), 3)
	if g.tickDuration > 0 {
		tickRate := 1.0 / g.tickDuration.Seconds()
		g.addStatLine(screen, "Tick Rate", fmt.Sprintf("%.0f/s", tickRate), 4)
	}
}

func (g *Game) drawFOVCones(screen *ebiten.Image, views map[int]simulation.CreatureView, t float64) {
	bs := float64(BlockSize)
	half := float32(BlockSize) / 2
	for _, cv := range views {
		if cv.SightDistance == 0 {
			continue
		}
		cx, cy := float32(cv.X*bs)+half, float32(cv.Y*bs)+half
		if anim, ok := g.animByID[cv.ID]; ok {
			cx = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			cy = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		r := float32(cv.SightDistance) * float32(BlockSize)
		halfFOV := float64(cv.FieldOfView) / 2.0 * math.Pi / 180.0
		var path vector.Path
		path.MoveTo(cx, cy)
		path.Arc(cx, cy, r, float32(cv.Heading-halfFOV), float32(cv.Heading+halfFOV), vector.Clockwise)
		path.Close()
		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		clr := color.RGBA{cv.R, cv.G, cv.B, 40}
		for i := range vs {
			vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, 0.15
		}
		screen.DrawTriangles(vs, is, g.whiteImage, nil)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

func (g *Game) addStatLine(img *ebiten.Image, desc string, val string, row int) {
	x := g.sim.GridWidth()*BlockSize - 200
	text.Draw(img, fmt.Sprintf("%s: %s", desc, val), g.statFont, x, 20*row+3, color.White)
}

func foodKey(x, y float64) string { return fmt.Sprintf("%f,%f", x, y) }

func (g *Game) trySelectCreature(mx, my int) {
	clickX, clickY := float64(mx), float64(my)
	half, hitRadius := float64(BlockSize)/2, float64(BlockSize)*6
	bestID, bestDist := -1, math.Inf(1)
	for id, anim := range g.animByID {
		dx, dy := clickX-(anim.curX+half), clickY-(anim.curY+half)
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < hitRadius && dist < bestDist {
			bestDist, bestID = dist, id
		}
	}
	if bestID == -1 || bestID == g.selectedCreatureID {
		g.selectedCreatureID = -1
	} else {
		g.selectedCreatureID = bestID
	}
}

func (g *Game) drawSelectionHighlight(screen *ebiten.Image) {
	if anim, ok := g.animByID[g.selectedCreatureID]; ok {
		vector.StrokeCircle(screen, float32(anim.curX)+float32(BlockSize)/2, float32(anim.curY)+float32(BlockSize)/2, float32(BlockSize)*5, 1.5, color.RGBA{255, 240, 80, 210}, false)
	}
}
func (g *Game) drawCreatureDetail(screen *ebiten.Image, d simulation.CreatureDetailView) {
	p := g.detailsPanel
	p.Draw(screen, false) // The main panel background

	currX, currY := p.X+detailTpad, p.Y+16

	// Title
	title := &components.Label{Text: fmt.Sprintf("Creature #%d", d.ID), Font: g.statFont, Color: color.RGBA{255, 220, 80, 255}}
	_, h := title.Draw(screen, currX, currY)
	currY += h + 4

	// Energy Section
	energyTxt := &components.Label{Text: fmt.Sprintf("Energy: %.f/%d", d.Energy, d.MaxEnergy), Font: g.statFont, Color: color.White}
	energyTxt.Draw(screen, currX, currY)
	currY += 5

	eBar := &components.EnergyBar{Value: d.Energy, Max: float32(d.MaxEnergy), Width: p.W - (detailTpad * 2)}
	_, h = eBar.Draw(screen, currX, currY)
	currY += h + 15
	juvenileStr := "Adult"
	if d.IsJuvenile {
		juvenileStr = fmt.Sprintf("Juvenile(%d)", d.JuvenilePeriod-d.Age)
	}
	age := &components.Label{Text: fmt.Sprintf("Age:  %d, %s", d.Age, juvenileStr), Font: g.statFont, Color: color.White}
	age.Draw(screen, currX, currY)
	currY += h + 15
	action := &components.Label{Text: fmt.Sprintf("Actions: %s", d.LastAction), Font: g.statFont, Color: components.ColorButtonGreen}
	action.Draw(screen, currX, currY)
	currY += h + 15
	mass := &components.Label{Text: fmt.Sprintf("Mass:  %.0f / %d", d.CurrentMass, d.AdultMass), Font: g.statFont, Color: color.White}
	mass.Draw(screen, currX, currY)
	currY += h + 15
	sight := &components.Label{Text: fmt.Sprintf("Sight: %d  FOV: %d°", d.SightDistance, d.FieldOfView), Font: g.statFont, Color: color.White}
	sight.Draw(screen, currX, currY)
	currY += h + 15
	layers := &components.Label{Text: fmt.Sprintf("Layers: %d  Genes: %d", d.NeuronCount, d.BrainLength), Font: g.statFont, Color: color.White}
	layers.Draw(screen, currX, currY)
	currY += h + 15
	mutation := &components.Label{Text: fmt.Sprintf("Mutation:  %.2f%%", d.MutationPct), Font: g.statFont, Color: color.White}
	mutation.Draw(screen, currX, currY)
	currY += h + 25

	g.drawPhenotypeChart(screen, d, currX, currY)

	sBtn := &components.Button{
		Label: "Save Genome",
		X:     int(currX),
		Y:     int(currY) + 150,
		W:     int(p.W - (detailTpad * 2)),
		H:     int(detailSaveBtnH),
		Color: color.RGBA{40, 100, 60, 220},
		OnClick: func() {
			if err := g.sim.SaveCreature(g.selectedCreatureID); err != nil {
				g.saveFeedback = "Save failed"
			} else {
				g.saveFeedback = "Saved!"
			}
			g.saveFeedbackAt = time.Now()
		},
	}
	g.saveBtn = sBtn
	g.saveBtn.Draw(screen, g.statFont)
}

func (g *Game) drawPhenotypeChart(screen *ebiten.Image, d simulation.CreatureDetailView, x, y float32) {
	const chartSize = 64.0
	const barWidth = 10.0
	const spacing = 14.0
	const barLabelOffset = 14.0

	// 1. Draw Title
	text.Draw(screen, "GENETIC PROFILE", g.statFont, int(x), int(y)-12, color.RGBA{120, 120, 180, 255})

	// 2. Draw Gradient Square
	step := float32(2)
	for gy := float32(0); gy < chartSize; gy += step {
		// Y-axis: Intelligence (Green). 1.0 (Top) to 0.0 (Bottom)
		gVal := uint8((1.0-(gy/chartSize))*185) + 70
		for rx := float32(0); rx < chartSize; rx += step {
			// X-axis: Physicality (Red). 0.0 (Left) to 1.0 (Right)
			rVal := uint8((rx/chartSize)*185) + 70
			vector.DrawFilledRect(screen, x+rx, y+gy, step, step, color.RGBA{rVal, gVal, d.B, 255}, false)
		}
	}

	// 3. Draw Crosshair
	rPerc := clamp((float32(d.R) - 70) / 185)
	gPerc := clamp((float32(d.G) - 70) / 185)

	cx := x + (rPerc * chartSize)
	cy := y + (chartSize - (gPerc * chartSize))

	vector.StrokeLine(screen, cx, y, cx, y+chartSize, 1, color.White, false)
	vector.StrokeLine(screen, x, cy, x+chartSize, cy, 1, color.White, false)

	// 4. Draw Quad-Bar Meter Group (I, P, S, M)
	mx := x + chartSize + 15
	meters := []struct {
		val float32
		clr color.RGBA
		lbl string
	}{
		{gPerc, color.RGBA{100, 255, 100, 255}, "I"},
		{rPerc, color.RGBA{255, 100, 100, 255}, "P"},
		{clamp((float32(d.B) - 70) / 185), color.RGBA{100, 150, 255, 255}, "S"},
		{clamp(d.MutationPct / 100.0), color.RGBA{255, 100, 255, 255}, "M"},
	}

	for i, m := range meters {
		currX := mx + float32(i)*spacing
		// Background track
		vector.DrawFilledRect(screen, currX, y, barWidth, chartSize, color.RGBA{20, 20, 25, 255}, false)
		// Foreground fill
		vector.DrawFilledRect(screen, currX, y+(chartSize-(m.val*chartSize)), barWidth, m.val*chartSize, m.clr, false)
		// Label
		text.Draw(screen, m.lbl, g.statFont, int(currX)+2, int(y+chartSize)+barLabelOffset, m.clr)
	}
}

// Helper to keep logic clean
func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (g *Game) applySpawnMutSlider(mx int) {
	t := float32(mx-spawnMutTrackX) / float32(spawnMutTrackW)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	g.spawnMutRate = float32(spawnMutRateMin + float64(t)*(spawnMutRateMax-spawnMutRateMin))
	g.sim.SetSpawnMutationRate(g.spawnMutRate)
}

func (g *Game) drawSpawnMutSlider(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, spawnMutSliderX, spawnMutSliderY, spawnMutSliderW, spawnMutSliderH, color.RGBA{30, 30, 50, 220}, false)
	vector.DrawFilledRect(screen, spawnMutTrackX, spawnMutTrackY, spawnMutTrackW, spawnMutTrackH, color.RGBA{60, 60, 80, 255}, false)
	t := (float64(g.spawnMutRate) - spawnMutRateMin) / (spawnMutRateMax - spawnMutRateMin)
	vector.DrawFilledRect(screen, float32(spawnMutTrackX), float32(spawnMutTrackY), float32(spawnMutTrackW)*float32(t), float32(spawnMutTrackH), color.RGBA{80, 140, 210, 255}, false)
	text.Draw(screen, fmt.Sprintf("Mut: %.4f", g.spawnMutRate), g.statFont, spawnMutSliderX+5, spawnMutSliderY+17, color.White)
}

// drawHistoryGraph renders a semi-transparent line graph of population and food
// history in the top-left corner. The rightmost point is the current tick.
func (g *Game) drawHistoryGraph(screen *ebiten.Image) {
	if g.histCount < 2 {
		return
	}

	vector.DrawFilledRect(screen, graphPanelX, graphPanelY, graphPanelW, graphPanelH,
		color.RGBA{8, 10, 22, 160}, false)
	vector.StrokeRect(screen, graphPanelX, graphPanelY, graphPanelW, graphPanelH, 1,
		color.RGBA{50, 60, 90, 180}, false)

	text.Draw(screen, fmt.Sprintf("Pop: %d", g.sim.PopulationCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+15, color.RGBA{100, 180, 255, 255})
	text.Draw(screen, fmt.Sprintf("Food: %d", g.sim.FoodCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+31, color.RGBA{80, 210, 100, 255})

	gx := float32(graphPanelX + graphPad)
	gy := float32(graphPanelY + graphTextH)
	gw := float32(graphPanelW - graphPad*2)
	gh := float32(graphPanelH - graphTextH - graphPad)

	maxPop, maxFood := 1, 1
	for i := 0; i < g.histCount; i++ {
		idx := ((g.histHead-1-i)%historyLen + historyLen) % historyLen
		s := g.history[idx]
		if s.pop > maxPop {
			maxPop = s.pop
		}
		if s.food > maxFood {
			maxFood = s.food
		}
	}

	steps := int(gw)
	if steps > g.histCount {
		steps = g.histCount
	}

	g.drawGraphLine(screen, gx, gy, gw, gh, steps, maxFood,
		color.RGBA{80, 210, 100, 200}, func(s histSample) int { return s.food })
	g.drawGraphLine(screen, gx, gy, gw, gh, steps, maxPop,
		color.RGBA{100, 180, 255, 200}, func(s histSample) int { return s.pop })
}

// drawGraphLine draws a single series onto the graph area using a batched
// DrawTriangles call for efficiency.
func (g *Game) drawGraphLine(screen *ebiten.Image, gx, gy, gw, gh float32, steps, maxVal int, clr color.RGBA, getValue func(histSample) int) {
	if steps < 2 || maxVal == 0 {
		return
	}

	var path vector.Path
	for i := 0; i < steps; i++ {
		frac := float64(i) / float64(steps-1)
		logicalPos := float64(g.histHead-g.histCount) + frac*float64(g.histCount-1)
		sampleIdx := ((int(math.Round(logicalPos)) % historyLen) + historyLen) % historyLen
		val := getValue(g.history[sampleIdx])

		x := gx + float32(i)/float32(steps-1)*gw
		y := gy + gh*(1-float32(val)/float32(maxVal))
		if y < gy {
			y = gy
		}
		if y > gy+gh {
			y = gy + gh
		}

		if i == 0 {
			path.MoveTo(x, y)
		} else {
			path.LineTo(x, y)
		}
	}

	vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: 1.5})
	cr, cg, cb, ca := float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, float32(clr.A)/255
	for i := range vs {
		vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = cr, cg, cb, ca
	}
	screen.DrawTriangles(vs, is, g.whiteImage, nil)
}
