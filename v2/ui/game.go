package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"math"
	"sort"
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
	GridWidth() float64
	GridHeight() float64
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
	SpawnAt(x, y float64) bool
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

	// Neural network graph panel — sits below the creature detail panel
	nnPanelX    = detailPanelX
	nnPanelY    = detailPanelY + detailPanelH + 4
	nnPanelW    = 320
	nnColSensor = 120 // x-offset within panel for sensor nodes
	nnColAction = 200 // x-offset within panel for action nodes
	nnInnerPadX = 18  // gap between sensor/action nodes and the nearest neuron column
	nnNodeSpY   = 15  // vertical space allocated per node
	nnTitleH    = 20  // height of title row
	nnPadding   = 6   // internal padding
	nnNodeR     = 4   // node radius
	nnBarMaxW   = 18  // max width of sensor/action activity bar in pixels
	nnBarH      = 3   // height of activity bar in pixels
	nnFooterH   = 16  // height of footer row below the node graph
)

type Component interface {
	Draw(screen *ebiten.Image, x, y float32) (float32, float32)
}

type creatureAnim struct {
	prevX, prevY float64
	curX, curY   float64
	r, g, b, a   uint8
	heading      float64
	mass         float64 // Prioritized mass
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
	spawnRandomBtn     *components.Button
	spawnPlacing       bool
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
	g.spawnRandomBtn = &components.Button{
		X: 642, Y: 10, W: 130, H: 24,
		Label:      "Spawn Random",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		OnClick: func() {
			g.spawnPlacing = !g.spawnPlacing
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
	cx := sim.GridWidth() / 2
	cy := sim.GridHeight() / 2

	// Vertical bar of the cross
	g.renderGrid.AddLine(
		(cx-wallThickness/2)*bs, sim.GridHeight()/4*bs,
		(cx+wallThickness/2)*bs, sim.GridHeight()*3/4*bs,
	)
	// Horizontal bar of the cross
	g.renderGrid.AddLine(
		sim.GridWidth()/4*bs, (cy-wallThickness/2)*bs,
		sim.GridWidth()*3/4*bs, (cy+wallThickness/2)*bs,
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

		if g.spawnRandomBtn.IsClicked(mx, my) {
			g.spawnRandomBtn.OnClick()
			return true
		}

		if g.spawnPlacing {
			bs := float64(BlockSize)
			g.sim.SpawnAt(float64(mx)/bs, float64(my)/bs)
			return true
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.spawnPlacing = false
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
				anim.mass = cv.CurrentMass // Prioritise mass
			} else {
				g.animByID[cv.ID] = &creatureAnim{
					prevX: screenX, prevY: screenY,
					curX: screenX, curY: screenY,
					r: cv.R, g: cv.G, b: cv.B, a: cv.A,
					heading: cv.Heading,
					mass:    cv.CurrentMass,
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
		// 1. Calculate Interpolated Position
		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		cx, cy := float32(lerpX+half), float32(lerpY+half)

		// 2. Calculate Scaled Radius based on Mass
		var r float32
		minRadius := float32(BlockSize) * 0.8
		maxExtraRadius := float32(BlockSize) * 4.0

		if massRange > 0 {
			massT := (float64(anim.mass) - float64(g.minCreatureMass)) / massRange
			if massT < 0 {
				massT = 0
			}
			if massT > 1 {
				massT = 1
			}
			// Scale r: Starting at 80% BlockSize, growing up to +400% based on mass
			r = minRadius + (float32(massT) * maxExtraRadius)
		} else {
			r = minRadius
		}

		// 3. Prepare Colors
		cr, cg, cb, ca := float32(anim.r)/255, float32(anim.g)/255, float32(anim.b)/255, float32(anim.a)/255

		// 4. Generate Circle Vertices (12 segments)
		baseIdx := uint16(len(creatureVs))
		const segments = 12

		// Center vertex for the triangle fan
		creatureVs = append(creatureVs, ebiten.Vertex{
			DstX: cx, DstY: cy,
			ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
		})

		for i := 0; i <= segments; i++ {
			angle := float64(i) * 2 * math.Pi / segments
			creatureVs = append(creatureVs, ebiten.Vertex{
				DstX:   cx + r*float32(math.Cos(angle)),
				DstY:   cy + r*float32(math.Sin(angle)),
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
		}

		// 5. Build Indices for the Fan
		for i := uint16(1); i <= segments; i++ {
			creatureIs = append(creatureIs, baseIdx, baseIdx+i, baseIdx+i+1)
		}
	}

	if len(creatureVs) > 0 {
		screen.DrawTriangles(creatureVs, creatureIs, g.whiteImage, nil)
	}

	g.drawHistoryGraph(screen)
	if g.selectedCreatureID != -1 {
		g.drawSelectionHighlight(screen)
		if detail, ok := g.sim.CreatureDetail(g.selectedCreatureID); ok {
			g.drawCreatureDetail(screen, detail)
			g.drawNeuralNetGraph(screen, detail)
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
	if g.spawnPlacing {
		g.spawnRandomBtn.Color = components.ColorButtonGreen
		g.spawnRandomBtn.Label = "Cancel Spawn"
	} else {
		g.spawnRandomBtn.Color = components.ColorDefault
		g.spawnRandomBtn.Label = "Spawn Random"
	}
	g.spawnRandomBtn.Draw(screen, g.statFont)
	if g.spawnPlacing {
		mx, my := ebiten.CursorPosition()
		cx, cy := float32(mx), float32(my)
		const arm = float32(10)
		vector.StrokeLine(screen, cx-arm, cy, cx+arm, cy, 1.5, color.RGBA{255, 220, 80, 220}, false)
		vector.StrokeLine(screen, cx, cy-arm, cx, cy+arm, 1.5, color.RGBA{255, 220, 80, 220}, false)
		vector.StrokeCircle(screen, cx, cy, arm*0.6, 1, color.RGBA{255, 220, 80, 160}, false)
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
		clr := color.RGBA{
			R: uint8(cv.R),
			G: uint8(cv.G),
			B: uint8(cv.B),
			A: 40,
		}
		for i := range vs {
			vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = float32(clr.R)/255, float32(clr.G)/255, float32(clr.B)/255, 0.15
		}
		screen.DrawTriangles(vs, is, g.whiteImage, nil)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

func (g *Game) addStatLine(img *ebiten.Image, desc string, val string, row int) {
	x := int(g.sim.GridWidth())*BlockSize - 200
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
	energyTxt := &components.Label{Text: fmt.Sprintf("Energy: %.f/%.f", d.Energy, d.MaxEnergy), Font: g.statFont, Color: color.White}
	energyTxt.Draw(screen, currX, currY)
	currY += 5

	eBar := &components.EnergyBar{Value: d.Energy, Max: d.MaxEnergy, MaxColor: color.RGBA{55, 185, 55, 255}, MinColor: color.RGBA{190, 55, 55, 255}, Width: p.W - (detailTpad * 2)}
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
	stomach := &components.Label{Text: fmt.Sprintf("Stomach: %.0f/%.0f", d.Stomach, d.StomachCapacity), Font: g.statFont, Color: color.White}
	stomach.Draw(screen, currX, currY)
	currY += 5
	stomachBar := &components.EnergyBar{Value: d.Stomach, Max: d.StomachCapacity, MaxColor: color.RGBA{55, 185, 55, 255}, MinColor: color.RGBA{190, 55, 55, 255}, Width: p.W - (detailTpad * 2)}
	_, h = stomachBar.Draw(screen, currX, currY)
	currY += h + 15
	dopamine := &components.Label{Text: fmt.Sprintf("Dopamine:  %.02f", d.Dopamine), Font: g.statFont, Color: color.White}
	dopamine.Draw(screen, currX, currY)
	currY += 5
	dopBar := &components.EnergyBar{Value: d.Dopamine, Max: float32(1.2), MaxColor: color.RGBA{216, 27, 96, 1}, MinColor: color.RGBA{48, 63, 159, 1}, Width: p.W - (detailTpad * 2)}
	_, h = dopBar.Draw(screen, currX, currY)
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
		Y:     int(currY) + 100,
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
		// gVal := uint8((1.0-(gy/chartSize))*185) + 70
		gVal := uint8((gy/chartSize)*185) + 70
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

func (g *Game) drawNeuralNetGraph(screen *ebiten.Image, d simulation.CreatureDetailView) {
	nn := d.NeuralNet
	if len(nn.Edges) == 0 && len(nn.HiddenNeuronIDs) == 0 {
		return
	}

	// Collect unique sensor and action IDs referenced by edges.
	sensorSet := map[byte]bool{}
	actionSet := map[byte]bool{}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR {
			sensorSet[e.SourceID] = true
		}
		if e.SinkType == simulation.ACTION {
			actionSet[e.SinkID] = true
		}
	}
	sensors := make([]byte, 0, len(sensorSet))
	for s := range sensorSet {
		sensors = append(sensors, s)
	}
	sort.Slice(sensors, func(i, j int) bool { return sensors[i] < sensors[j] })
	actions := make([]byte, 0, len(actionSet))
	for a := range actionSet {
		actions = append(actions, a)
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i] < actions[j] })
	neurons := nn.HiddenNeuronIDs // already sorted

	// Assign a topological depth to each hidden neuron.
	// Neurons fed directly by sensors start at depth 1; each neuron→neuron hop adds 1.
	// Undriven neurons (no sensor or neuron input) sit at depth 0.
	// Iterative relaxation with a safety cap tolerates cycles.
	const maxLayerDepth = 8
	neuronDepth := map[byte]int{}
	for _, id := range neurons {
		neuronDepth[id] = 0
	}
	for _, e := range nn.Edges {
		if e.SourceType == simulation.SENSOR && e.SinkType == simulation.NEURON {
			if neuronDepth[e.SinkID] < 1 {
				neuronDepth[e.SinkID] = 1
			}
		}
	}
	for iter := 0; iter < maxLayerDepth; iter++ {
		changed := false
		for _, e := range nn.Edges {
			if e.SourceType == simulation.NEURON && e.SinkType == simulation.NEURON && e.SourceID != e.SinkID {
				newD := neuronDepth[e.SourceID] + 1
				if newD > maxLayerDepth {
					newD = maxLayerDepth
				}
				if newD > neuronDepth[e.SinkID] {
					neuronDepth[e.SinkID] = newD
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}

	// Group neurons by depth; track the maximum depth present.
	byDepth := map[int][]byte{}
	numDepths := 0
	for _, id := range neurons {
		dep := neuronDepth[id]
		byDepth[dep] = append(byDepth[dep], id)
		if dep+1 > numDepths {
			numDepths = dep + 1
		}
	}
	for dep := range byDepth {
		sort.Slice(byDepth[dep], func(i, j int) bool { return byDepth[dep][i] < byDepth[dep][j] })
	}

	// Map each neuron to a column X: depths are spread evenly within the inner
	// area, which is inset from the sensor and action nodes by nnInnerPadX on
	// each side so connection lines have clear breathing room.
	neuronColX := map[byte]float32{}
	innerLeft := float32(nnColSensor + nnInnerPadX)
	innerRight := float32(nnColAction - nnInnerPadX)
	innerSpan := innerRight - innerLeft
	for id, dep := range neuronDepth {
		frac := float32(dep+1) / float32(numDepths+1)
		neuronColX[id] = innerLeft + innerSpan*frac
	}

	// Panel height from the densest column across sensors, any neuron layer, actions.
	maxCount := len(sensors)
	if len(actions) > maxCount {
		maxCount = len(actions)
	}
	for _, ns := range byDepth {
		if len(ns) > maxCount {
			maxCount = len(ns)
		}
	}
	panelH := float32(nnTitleH+nnPadding) + float32(maxCount)*nnNodeSpY + float32(nnPadding) + float32(nnFooterH)
	if panelH < 40 {
		panelH = 40
	}

	px := float32(nnPanelX)
	py := float32(nnPanelY)

	// Panel background.
	vector.DrawFilledRect(screen, px, py, nnPanelW, panelH, color.RGBA{8, 10, 22, 215}, false)
	vector.StrokeRect(screen, px, py, nnPanelW, panelH, 1, color.RGBA{90, 90, 150, 255}, false)
	text.Draw(screen, "NEURAL NETWORK", g.statFont, int(px)+nnPadding, int(py)+14, color.RGBA{120, 120, 180, 255})

	contentTop := py + float32(nnTitleH+nnPadding)
	contentH := panelH - float32(nnTitleH+nnPadding*2) - float32(nnFooterH)

	nodeY := func(count, i int) float32 {
		if count == 0 {
			return contentTop + contentH/2
		}
		step := contentH / float32(count)
		return contentTop + step*float32(i) + step/2
	}

	sensorY := map[byte]float32{}
	for i, s := range sensors {
		sensorY[s] = nodeY(len(sensors), i)
	}
	// Each depth column distributes its own neurons independently.
	neuronY := map[byte]float32{}
	for dep, ns := range byDepth {
		for i, id := range ns {
			neuronY[id] = nodeY(len(ns), i)
		}
		_ = dep
	}
	actionY := map[byte]float32{}
	for i, a := range actions {
		actionY[a] = nodeY(len(actions), i)
	}

	// Draw edges, coloured by weight: green = positive, orange-red = negative.
	for _, e := range nn.Edges {
		var srcX, srcY, dstX, dstY float32

		switch e.SourceType {
		case simulation.SENSOR:
			srcX = px + float32(nnColSensor)
			srcY = sensorY[e.SourceID]
		case simulation.NEURON:
			srcX = px + neuronColX[e.SourceID]
			srcY = neuronY[e.SourceID]
		}
		switch e.SinkType {
		case simulation.NEURON:
			dstX = px + neuronColX[e.SinkID]
			dstY = neuronY[e.SinkID]
		case simulation.ACTION:
			dstX = px + float32(nnColAction)
			dstY = actionY[e.SinkID]
		}

		// Self-loop: small circle above the neuron node.
		if e.SourceType == simulation.NEURON && e.SinkType == simulation.NEURON && e.SourceID == e.SinkID {
			loopX := px + neuronColX[e.SourceID]
			loopY := neuronY[e.SourceID] - nnNodeR*2
			var path vector.Path
			path.Arc(loopX, loopY, nnNodeR*1.5, 0, 2*math.Pi, vector.Clockwise)
			vs, is := path.AppendVerticesAndIndicesForStroke(nil, nil, &vector.StrokeOptions{Width: 1})
			lc := nnEdgeColor(e.Weight)
			cr, cg, cb, ca := float32(lc.R)/255, float32(lc.G)/255, float32(lc.B)/255, float32(lc.A)/255
			for i := range vs {
				vs[i].ColorR, vs[i].ColorG, vs[i].ColorB, vs[i].ColorA = cr, cg, cb, ca
			}
			screen.DrawTriangles(vs, is, g.whiteImage, nil)
			continue
		}

		vector.StrokeLine(screen, srcX, srcY, dstX, dstY, 1, nnEdgeColor(e.Weight), false)
	}

	// Sensor nodes + right-aligned labels + left-anchored activity bars.
	for _, s := range sensors {
		nx := px + float32(nnColSensor)
		ny := sensorY[s]
		vector.DrawFilledCircle(screen, nx, ny, nnNodeR, color.RGBA{80, 150, 220, 255}, false)
		lbl := nnSensorName(s)
		lblW := float32(len(lbl)) * 8
		labelX := nx - lblW - 10
		if labelX < px+2 {
			labelX = px + 2
		}
		text.Draw(screen, lbl, g.statFont, int(labelX), int(ny)+5, color.RGBA{160, 180, 220, 255})
		if val, ok := nn.SensorValues[s]; ok {
			barX := px + float32(nnPadding)
			barY := ny - float32(nnBarH)/2
			fillW := float32(nnBarMaxW) * clamp(val)
			vector.DrawFilledRect(screen, barX, barY, float32(nnBarMaxW), float32(nnBarH), color.RGBA{20, 40, 70, 180}, false)
			if fillW > 0 {
				vector.DrawFilledRect(screen, barX, barY, fillW, float32(nnBarH), color.RGBA{80, 160, 240, 220}, false)
			}
		}
	}

	// Hidden neuron nodes — one column per depth layer.
	for _, id := range neurons {
		nx := px + neuronColX[id]
		ny := neuronY[id]
		vector.DrawFilledCircle(screen, nx, ny, nnNodeR, color.RGBA{200, 180, 80, 255}, false)
	}

	// Action nodes + left-aligned labels + right-anchored activity bars.
	for _, a := range actions {
		ax := px + float32(nnColAction)
		ay := actionY[a]
		vector.DrawFilledCircle(screen, ax, ay, nnNodeR, color.RGBA{220, 100, 80, 255}, false)
		lbl := nnActionName(a)
		text.Draw(screen, lbl, g.statFont, int(ax)+10, int(ay)+5, color.RGBA{220, 160, 150, 255})
		if val, ok := nn.ActionValues[a]; ok {
			barX := px + float32(nnPanelW) - float32(nnPadding) - float32(nnBarMaxW)
			barY := ay - float32(nnBarH)/2
			norm := float32(math.Tanh(float64(val)))
			absNorm := norm
			if absNorm < 0 {
				absNorm = -absNorm
			}
			fillW := float32(nnBarMaxW) * absNorm
			vector.DrawFilledRect(screen, barX, barY, float32(nnBarMaxW), float32(nnBarH), color.RGBA{60, 25, 15, 180}, false)
			if fillW > 0 {
				fc := color.RGBA{240, 130, 80, 220}
				if norm < 0 {
					fc = color.RGBA{80, 130, 240, 220}
				}
				vector.DrawFilledRect(screen, barX, barY, fillW, float32(nnBarH), fc, false)
			}
		}
	}

	const baseLearningRate = 0.01
	footerY := int(py+panelH) - nnPadding - 2
	text.Draw(screen, fmt.Sprintf("Learning Rate: %.4f", baseLearningRate*d.Dopamine), g.statFont, int(px)+nnPadding, footerY, color.RGBA{120, 120, 180, 220})
}

func nnEdgeColor(w float32) color.RGBA {
	absW := w
	if absW < 0 {
		absW = -absW
	}
	brightness := absW
	if brightness > 1 {
		brightness = 1
	}
	if w >= 0 {
		return color.RGBA{0, uint8(brightness*200 + 30), 0, 180}
	}
	return color.RGBA{uint8(brightness*180 + 30), uint8(brightness * 40), 0, 160}
}

func nnSensorName(id byte) string {
	names := [...]string{
		"Age", "Energy", "Loc X", "Loc Y", "Osc 1",
		"Density", "See Pop", "See Food", "See Corpse",
		"Random", "Satiety", "Facing", "Food Ang",
		"Food Dist", "Threat", "Kinship", "Burn Rate",
		"Mass %", "Blocked", "Prey", "Threat Ang",
		"Prey Ang", "Wall Prox", "Digest", "Food/Cap",
		"Juvenile",
	}
	if int(id) < len(names) {
		return names[id]
	}
	return fmt.Sprintf("S%d", id)
}

func nnActionName(id byte) string {
	names := [...]string{
		"Move", "Rotate",
		"SetOsc", "SetResp", "SetLearn", "Rest",
	}
	if int(id) < len(names) {
		return names[id]
	}
	return fmt.Sprintf("A%d", id)
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

	panelFill := color.RGBA{8, 10, 22, 160}
	panelStroke := color.RGBA{50, 60, 90, 180}
	vector.DrawFilledRect(screen, graphPanelX, graphPanelY, graphPanelW, graphPanelH, panelFill, false)
	vector.StrokeRect(screen, graphPanelX, graphPanelY, graphPanelW, graphPanelH, 1, panelStroke, false)

	popColor := color.RGBA{100, 180, 255, 255}
	foodColor := color.RGBA{80, 210, 100, 255}

	text.Draw(screen, fmt.Sprintf("Pop: %d", g.sim.PopulationCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+15, popColor)
	text.Draw(screen, fmt.Sprintf("Food: %d", g.sim.FoodCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+31, foodColor)

	gx := float32(graphPanelX + graphPad)
	gy := float32(graphPanelY + graphTextH)
	gw := float32(graphPanelW - graphPad*2)
	gh := float32(graphPanelH - graphTextH - graphPad)

	globalMax := 1
	for i := 0; i < g.histCount; i++ {
		idx := ((g.histHead-1-i)%historyLen + historyLen) % historyLen
		s := g.history[idx]

		if s.pop > globalMax {
			globalMax = s.pop
		}
		if s.food > globalMax {
			globalMax = s.food
		}
	}

	steps := int(gw)
	if steps > g.histCount {
		steps = g.histCount
	}

	g.drawGraphLine(screen, gx, gy, gw, gh, steps, globalMax,
		color.RGBA{80, 210, 100, 200}, func(s histSample) int { return s.food })

	g.drawGraphLine(screen, gx, gy, gw, gh, steps, globalMax,
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
