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
	WorldWidth() float64
	WorldHeight() float64
	PopulationCount() int
	FoodCount() int
	AverageAge() float64
	CreatureMinMass() byte
	CreatureMaxMass() float64
	SaveCreature(id int) error
	Reset()
	TotalEnergy() float64
	TargetEnergy() float64
	CreatureDetail(id int) (simulation.CreatureDetailView, bool)
	SetSpawnMutationRate(rate float32)
	SpawnAt(x, y float64) bool
	SpawnClusterAt(x, y float64, count int) bool
	SpawnGenome(g *simulation.Genome) bool
	CreatureGenomeCopy(id int) (*simulation.Genome, bool)
	GetParams() *simulation.Parameters
	GetSnapshot() simulation.StateSnapshot
}

const historyLen = 5000

type histSample struct {
	pop         int
	food        int
	totalEnergy float64
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
	detailPanelH   = 468 // extra height for Edit Genome button
	detailTpad     = 8
	detailSaveBtnH = 22

	// History graph panel (top-left, below button bar)
	graphPanelX = 10
	graphPanelY = 44
	graphPanelW = 220
	graphPanelH = 120
	graphTextH  = 54
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
	radius       float32
	sexual       bool
}

type Game struct {
	sim                SimulationState
	RenderWorld        *RenderWorld
	statFont           font.Face
	whiteImage         *ebiten.Image
	animByID           map[int]*creatureAnim
	lastTickTime       time.Time
	tickDuration       time.Duration
	minCreatureMass    byte
	maxCreatureMass    float64
	MaxTotalEnergy     float64
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
	editGenomeBtn      *components.Button
	createGenomeBtn    *components.Button
	spawnSavedBtn      *components.Button
	spawnRandomBtn     *components.Button
	spawnPlacing       bool
	spawnMutSlider     *components.Slider
	detailsPanel       *components.Panel
	genomeEditor       *GenomeEditor
	savedGenomesPanel  *SavedGenomesPanel
	currentSnapshot    *simulation.StateSnapshot
	lookup             map[int]int
	unitCircle         []struct{ x, y float32 }
	creatureVs         []ebiten.Vertex
	creatureIs         []uint16
	camera             Camera
	worldLayer         *ebiten.Image
	camDragging        bool
	camDragMoved       bool
	camDragStartX      int
	camDragStartY      int
	camDragLastX       int
	camDragLastY       int
	simStepsPerTick    int
}

var UnitSize int = 2

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
		RenderWorld:        NewRenderWorld(0, 0, UnitSize),
		statFont:           statFont,
		whiteImage:         wImg,
		animByID:           make(map[int]*creatureAnim),
		minCreatureMass:    sim.CreatureMinMass(),
		maxCreatureMass:    sim.CreatureMaxMass(),
		selectedCreatureID: -1,
		spawnMutRate:       0.01,
		isDarkBackground:   true,
		simStepsPerTick:    1,
	}

	const segments = 12
	for i := 0; i <= segments; i++ {
		angle := float64(i) * 2 * math.Pi / segments
		g.unitCircle = append(g.unitCircle, struct{ x, y float32 }{
			x: float32(math.Cos(angle)),
			y: float32(math.Sin(angle)),
		})
	}
	// vertsPerCreature = 1 center + len(unitCircle) edge = 14; trisPerCreature = segments = 12
	const initCapCreatures = 500
	g.creatureVs = make([]ebiten.Vertex, 0, initCapCreatures*(1+segments+1))
	g.creatureIs = make([]uint16, 0, initCapCreatures*segments*3)

	worldW := int(sim.WorldWidth()) * UnitSize
	worldH := int(sim.WorldHeight()) * UnitSize
	g.worldLayer = ebiten.NewImage(worldW, worldH)
	g.camera = Camera{X: float64(worldW) / 2, Y: float64(worldH) / 2, Zoom: 1.0}

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
			g.selectedCreatureID = -1
			g.lastTickTime = time.Time{}
			g.histHead = 0
			g.histCount = 0
			g.currentSnapshot = nil
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
	g.createGenomeBtn = &components.Button{
		X: 782, Y: 10, W: 130, H: 24,
		Label:      "Create Genome",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		OnClick: func() {
			g.genomeEditor.Open(nil, g.sim.GetParams())
		},
	}
	g.spawnSavedBtn = &components.Button{
		X: 922, Y: 10, W: 120, H: 24,
		Label:      "Spawn Saved",
		Color:      components.ColorDefault,
		LabelColor: color.White,
		OnClick: func() {
			g.savedGenomesPanel.Open()
		},
	}
	g.genomeEditor = newGenomeEditor(func(genome *simulation.Genome, name string) {
		g.sim.SpawnGenome(genome)
		simulation.SaveCreatureToFileNamed(genome, name) //nolint:errcheck
	})
	g.savedGenomesPanel = newSavedGenomesPanel(func(genome *simulation.Genome) {
		g.sim.SpawnGenome(genome)
	})
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
	// const wallThickness = 10.0
	// bs := float64(UnitSize)
	// cx := sim.WorldWidth() / 2
	// cy := sim.WorldHeight() / 2

	// Vertical bar of the cross
	// g.RenderWorld.AddLine(
	// 	(cx-wallThickness/2)*bs, sim.WorldHeight()/4*bs,
	// 	(cx+wallThickness/2)*bs, sim.WorldHeight()*3/4*bs,
	// )
	// // Horizontal bar of the cross
	// g.RenderWorld.AddLine(
	// 	sim.WorldWidth()/4*bs, (cy-wallThickness/2)*bs,
	// 	sim.WorldWidth()*3/4*bs, (cy+wallThickness/2)*bs,
	// )
	return g
}

func (g *Game) handleContinuousInput() {
	mx, my := ebiten.CursorPosition()
	if g.genomeEditor.visible {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.genomeEditor.HandleDrag(mx, my)
		} else {
			g.genomeEditor.HandleRelease()
		}
		return
	}
	if g.savedGenomesPanel.visible {
		return // no continuous input needed for the list panel
	}
	if g.spawnMutSlider.Dragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.spawnMutSlider.UpdateValue(mx)
			g.sim.SetSpawnMutationRate(float32(g.spawnMutSlider.Value))
		} else {
			g.spawnMutSlider.Dragging = false
		}
	}

	// Camera drag (left mouse, started when no UI element consumed the press)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) && g.camDragging {
		dx := mx - g.camDragLastX
		dy := my - g.camDragLastY
		if !g.camDragMoved && (dx*dx+dy*dy > 9) {
			g.camDragMoved = true
		}
		if g.camDragMoved {
			g.camera.X -= float64(dx) / g.camera.Zoom
			g.camera.Y -= float64(dy) / g.camera.Zoom
		}
		g.camDragLastX = mx
		g.camDragLastY = my
	}

	// WASD pan
	const panSpeed = 5.0
	pan := panSpeed / g.camera.Zoom
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		g.camera.Y -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		g.camera.Y += pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		g.camera.X -= pan
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		g.camera.X += pan
	}
	worldW := float64(g.sim.WorldWidth()) * float64(UnitSize)
	worldH := float64(g.sim.WorldHeight()) * float64(UnitSize)
	g.camera.Clamp(worldW, worldH)
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

		if g.editGenomeBtn != nil {
			if g.editGenomeBtn.IsClicked(mx, my) {
				g.editGenomeBtn.OnClick()
				return true
			}
		}

		if g.createGenomeBtn.IsClicked(mx, my) {
			g.createGenomeBtn.OnClick()
			return true
		}

		if g.spawnSavedBtn.IsClicked(mx, my) {
			g.spawnSavedBtn.OnClick()
			return true
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
			sw, sh := ebiten.WindowSize()
			wx, wy := g.camera.ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
			bs := float64(UnitSize)
			g.sim.SpawnClusterAt(wx/bs, wy/bs, 5)
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

	if g.genomeEditor.visible {
		g.genomeEditor.HandleKeyInput()
	}

	_, scrollY := ebiten.Wheel()
	if scrollY != 0 {
		if g.savedGenomesPanel.visible {
			g.savedGenomesPanel.Scroll(int(scrollY))
		} else {
			g.camera.Zoom *= math.Pow(1.15, scrollY)
			if g.camera.Zoom < 0.1 {
				g.camera.Zoom = 0.1
			} else if g.camera.Zoom > 10.0 {
				g.camera.Zoom = 10.0
			}
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.genomeEditor.visible {
			g.genomeEditor.HandleInput(mx, my)
		} else if g.savedGenomesPanel.visible {
			g.savedGenomesPanel.HandleInput(mx, my)
		} else if !g.handleInput() {
			g.camDragging = true
			g.camDragStartX = mx
			g.camDragStartY = my
			g.camDragLastX = mx
			g.camDragLastY = my
			g.camDragMoved = false
		}
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		if g.camDragging && !g.camDragMoved {
			g.trySelectCreature(g.camDragStartX, g.camDragStartY)
		}
		g.camDragging = false
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.simStepsPerTick++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) && g.simStepsPerTick > 1 {
		g.simStepsPerTick--
	}

	if !g.paused {
		// Run all steps except the last without tracking positions.
		for i := 1; i < g.simStepsPerTick; i++ {
			g.sim.Update()
		}

		// Capture prev positions immediately before the final step so lerp
		// interpolates between the two most recent simulation positions.
		prevByID := make(map[int][2]float64, len(g.animByID))
		for id, anim := range g.animByID {
			prevByID[id] = [2]float64{anim.curX, anim.curY}
		}

		g.sim.Update()

		g.history[g.histHead] = histSample{pop: g.sim.PopulationCount(), food: g.sim.FoodCount(), totalEnergy: g.sim.TotalEnergy()}
		g.histHead = (g.histHead + 1) % historyLen
		if g.histCount < historyLen {
			g.histCount++
		}

		snapshot := g.sim.GetSnapshot()
		g.currentSnapshot = &snapshot

		g.lookup = make(map[int]int, len(snapshot.Creatures))
		currentIDs := make(map[int]bool, len(snapshot.Creatures))
		bs := float64(UnitSize)

		for i, cv := range snapshot.Creatures {
			g.lookup[cv.ID] = i
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
				anim.mass = cv.CurrentMass
				anim.radius = float32(cv.Radius * bs)
				anim.sexual = cv.ReproductionType == 1
			} else {
				g.animByID[cv.ID] = &creatureAnim{
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

		for id := range g.animByID {
			if !currentIDs[id] {
				delete(g.animByID, id)
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
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// --- World layer ---
	g.drawTemperatureBackground()
	g.RenderWorld.DrawBackground(g.worldLayer)

	t := 1.0
	if g.tickDuration > 0 && !g.paused {
		elapsed := time.Since(g.lastTickTime)
		t = float64(elapsed) / float64(g.tickDuration)
		if t > 1 {
			t = 1
		}
	}

	bs := float64(UnitSize)
	if g.currentSnapshot != nil {
		circ := len(g.unitCircle) // 13 edge verts (12 segments + closing vert)
		vertsPerCircle := 1 + circ

		// --- Batch food circles ---
		fr, fg, fb, fa := float32(65)/255, float32(140)/255, float32(55)/255, float32(125)/255
		fVs := make([]ebiten.Vertex, 0, min(len(g.currentSnapshot.Food)*vertsPerCircle, 60_000))
		fIs := make([]uint16, 0, min(len(g.currentSnapshot.Food)*circ*3, 60_000))
		flushFood := func() {
			if len(fVs) > 0 {
				g.worldLayer.DrawTriangles(fVs, fIs, g.whiteImage, nil)
				fVs = fVs[:0]
				fIs = fIs[:0]
			}
		}
		for _, fv := range g.currentSnapshot.Food {
			if len(fVs)+vertsPerCircle > 60_000 {
				flushFood()
			}
			cx, cy, r := float32(fv.X*bs), float32(fv.Y*bs), float32(fv.Radius*bs)
			base := uint16(len(fVs))
			fVs = append(fVs, ebiten.Vertex{DstX: cx, DstY: cy, ColorR: fr, ColorG: fg, ColorB: fb, ColorA: fa})
			for _, u := range g.unitCircle {
				fVs = append(fVs, ebiten.Vertex{DstX: cx + r*u.x, DstY: cy + r*u.y, ColorR: fr, ColorG: fg, ColorB: fb, ColorA: fa})
			}
			for i := uint16(1); i <= uint16(circ-1); i++ {
				fIs = append(fIs, base, base+i, base+i+1)
			}
		}
		flushFood()

		// --- Batch meat circles ---
		mr, mg, mb, ma := float32(180)/255, float32(30)/255, float32(30)/255, float32(180)/255
		mVs := make([]ebiten.Vertex, 0, min(len(g.currentSnapshot.Meat)*vertsPerCircle, 60_000))
		mIs := make([]uint16, 0, min(len(g.currentSnapshot.Meat)*circ*3, 60_000))
		flushMeat := func() {
			if len(mVs) > 0 {
				g.worldLayer.DrawTriangles(mVs, mIs, g.whiteImage, nil)
				mVs = mVs[:0]
				mIs = mIs[:0]
			}
		}
		for _, mv := range g.currentSnapshot.Meat {
			if len(mVs)+vertsPerCircle > 60_000 {
				flushMeat()
			}
			cx, cy, r := float32(mv.X*bs), float32(mv.Y*bs), float32(mv.Radius*bs)
			base := uint16(len(mVs))
			mVs = append(mVs, ebiten.Vertex{DstX: cx, DstY: cy, ColorR: mr, ColorG: mg, ColorB: mb, ColorA: ma})
			for _, u := range g.unitCircle {
				mVs = append(mVs, ebiten.Vertex{DstX: cx + r*u.x, DstY: cy + r*u.y, ColorR: mr, ColorG: mg, ColorB: mb, ColorA: ma})
			}
			for i := uint16(1); i <= uint16(circ-1); i++ {
				mIs = append(mIs, base, base+i, base+i+1)
			}
		}
		flushMeat()
	}

	g.creatureVs = g.creatureVs[:0]
	g.creatureIs = g.creatureIs[:0]
	// massRange := float64(g.maxCreatureMass) - float64(g.minCreatureMass)

	vertsPerCreature := 1 + len(g.unitCircle)
	maxVerts := 1 + len(g.unitCircle)
	if maxVerts < 3 {
		maxVerts = 3
	}
	flushCreatures := func() {
		if len(g.creatureVs) > 0 {
			g.worldLayer.DrawTriangles(g.creatureVs, g.creatureIs, g.whiteImage, nil)
			g.creatureVs = g.creatureVs[:0]
			g.creatureIs = g.creatureIs[:0]
		}
	}

	for _, anim := range g.animByID {
		if len(g.creatureVs)+vertsPerCreature > 60_000 {
			flushCreatures()
		}

		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		cx, cy := float32(lerpX), float32(lerpY)

		// var r float32 = float32(UnitSize) / 2
		// if massRange > 0 {
		// 	massT := (float64(anim.mass) - float64(g.minCreatureMass)) / massRange
		// 	if massT < 0 {
		// 		massT = 0
		// 	}
		// 	if massT > 1 {
		// 		massT = 1
		// 	}
		// 	r += float32(massT) * float32(UnitSize) * 4.0
		// }

		cr, cg, cb, ca := float32(anim.r)/255, float32(anim.g)/255, float32(anim.b)/255, float32(anim.a)/255
		baseIdx := uint16(len(g.creatureVs))

		if anim.sexual {
			// Vertex 0: The Nose (Extended forward)
			g.creatureVs = append(g.creatureVs, ebiten.Vertex{
				DstX:   cx + (anim.radius)*float32(math.Cos(anim.heading)),
				DstY:   cy + (anim.radius)*float32(math.Sin(anim.heading)),
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
			// Vertex 1: Rear Left
			g.creatureVs = append(g.creatureVs, ebiten.Vertex{
				DstX:   cx + anim.radius*float32(math.Cos(anim.heading+2.4)),
				DstY:   cy + anim.radius*float32(math.Sin(anim.heading+2.4)),
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
			// Vertex 2: Rear Right
			g.creatureVs = append(g.creatureVs, ebiten.Vertex{
				DstX:   cx + anim.radius*float32(math.Cos(anim.heading-2.4)),
				DstY:   cy + anim.radius*float32(math.Sin(anim.heading-2.4)),
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})

			g.creatureIs = append(g.creatureIs, baseIdx, baseIdx+1, baseIdx+2)
		} else {
			g.creatureVs = append(g.creatureVs, ebiten.Vertex{
				DstX: cx, DstY: cy,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
			for _, unit := range g.unitCircle {
				g.creatureVs = append(g.creatureVs, ebiten.Vertex{
					DstX:   cx + anim.radius*unit.x,
					DstY:   cy + anim.radius*unit.y,
					ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
				})
			}
			for i := uint16(1); i <= uint16(len(g.unitCircle)-1); i++ {
				g.creatureIs = append(g.creatureIs, baseIdx, baseIdx+i, baseIdx+i+1)
			}
		}
	}

	flushCreatures()

	if g.selectedCreatureID != -1 {
		g.drawSelectionHighlight(g.worldLayer)
		if g.currentSnapshot != nil {
			if idx, found := g.lookup[g.selectedCreatureID]; found {
				view := g.currentSnapshot.Creatures[idx]
				g.drawFOVCones(g.worldLayer, map[int]simulation.CreatureView{
					g.selectedCreatureID: view,
				}, t)
			}
		}
	}

	// Apply camera transform: blit world to screen.
	op := &ebiten.DrawImageOptions{}
	op.GeoM = g.camera.GeoM(float64(sw), float64(sh))
	screen.DrawImage(g.worldLayer, op)

	// --- HUD (identity transform) ---
	g.drawHistoryGraph(screen)

	if g.selectedCreatureID != -1 {
		if detail, ok := g.sim.CreatureDetail(g.selectedCreatureID); ok {
			g.drawCreatureDetail(screen, detail)
			g.drawNeuralNetGraph(screen, detail)
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

	g.spawnRandomBtn.Label = "Spawn Random"
	if g.spawnPlacing {
		g.spawnRandomBtn.Label = "Cancel Spawn"
	}
	g.spawnRandomBtn.Draw(screen, g.statFont)
	g.createGenomeBtn.Draw(screen, g.statFont)
	g.spawnSavedBtn.Draw(screen, g.statFont)

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
		simRate := float64(g.simStepsPerTick) / g.tickDuration.Seconds()
		g.addStatLine(screen, "Sim Rate", fmt.Sprintf("%.0f/s (%dx)", simRate, g.simStepsPerTick), 4)
	}

	g.savedGenomesPanel.Draw(screen, g.statFont)
	g.genomeEditor.Draw(screen, g.statFont)
}

func (g *Game) drawFOVCones(img *ebiten.Image, views map[int]simulation.CreatureView, t float64) {
	bs := float64(UnitSize)
	half := float32(UnitSize) / 2
	for _, cv := range views {
		if cv.SightDistance == 0 {
			continue
		}
		cx, cy := float32(cv.X*bs)+half, float32(cv.Y*bs)+half
		if anim, ok := g.animByID[cv.ID]; ok {
			cx = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			cy = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		r := float32(cv.SightDistance) * float32(UnitSize)
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
		img.DrawTriangles(vs, is, g.whiteImage, nil)
	}
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

func (g *Game) addStatLine(img *ebiten.Image, desc string, val string, row int) {
	x := img.Bounds().Dx() - 200
	text.Draw(img, fmt.Sprintf("%s: %s", desc, val), g.statFont, x, 20*row+3, color.White)
}

func foodKey(x, y float64) string { return fmt.Sprintf("%f,%f", x, y) }

func (g *Game) trySelectCreature(mx, my int) {
	sw, sh := ebiten.WindowSize()
	clickX, clickY := g.camera.ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
	half := float64(UnitSize) / 2
	bestID, bestDist := -1, math.Inf(1)
	for id, anim := range g.animByID {
		dx, dy := clickX-(anim.curX+half), clickY-(anim.curY+half)
		dist := math.Sqrt(dx*dx + dy*dy)
		hitRadius := float64(anim.radius) + float64(UnitSize)*3
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

func (g *Game) drawSelectionHighlight(img *ebiten.Image) {
	if anim, ok := g.animByID[g.selectedCreatureID]; ok {
		vector.StrokeCircle(img, float32(anim.curX), float32(anim.curY), float32(UnitSize)*(5+anim.radius), 1.5, color.RGBA{255, 240, 80, 210}, false)
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
	mass := &components.Label{Text: fmt.Sprintf("Mass:  %.0f / %.0f", d.CurrentMass, d.AdultMass), Font: g.statFont, Color: color.White}
	mass.Draw(screen, currX, currY)
	currY += h + 15
	stomach := &components.Label{Text: fmt.Sprintf("Stomach: %.0f/%.0f", d.Stomach, d.StomachCapacity), Font: g.statFont, Color: color.White}
	stomach.Draw(screen, currX, currY)
	currY += 5
	stomachBar := &components.EnergyBar{Value: float32(d.Stomach), Max: float32(d.StomachCapacity), MaxColor: color.RGBA{55, 185, 55, 255}, MinColor: color.RGBA{190, 55, 55, 255}, Width: p.W - (detailTpad * 2)}
	_, h = stomachBar.Draw(screen, currX, currY)
	currY += h + 15
	efficiency := &components.Label{Text: fmt.Sprintf("Food Eff: %.1f%%  Meat Eff: %.1f%%", d.FoodEfficiency*100, d.MeatEfficiency*100), Font: g.statFont, Color: color.White}
	efficiency.Draw(screen, currX, currY)
	currY += h + 15
	dopamine := &components.Label{Text: fmt.Sprintf("Dopamine:  %.02f", d.Dopamine), Font: g.statFont, Color: color.White}
	dopamine.Draw(screen, currX, currY)
	currY += 5
	dopBar := &components.EnergyBar{Value: d.Dopamine, Max: float32(1.2), MaxColor: color.RGBA{216, 27, 96, 255}, MinColor: color.RGBA{48, 63, 159, 255}, Width: p.W - (detailTpad * 2), Centered: true}
	_, h = dopBar.Draw(screen, currX, currY)
	currY += h + 15
	sight := &components.Label{Text: fmt.Sprintf("Sight: %.f  FOV: %.f°", d.SightDistance, d.FieldOfView), Font: g.statFont, Color: color.White}
	sight.Draw(screen, currX, currY)
	currY += h + 15
	mutation := &components.Label{Text: fmt.Sprintf("Mutation:  %.2f%%", d.MutationPct), Font: g.statFont, Color: color.White}
	mutation.Draw(screen, currX, currY)
	currY += h + 15
	reproStr := "Asexual"
	reproColor := color.RGBA{100, 180, 255, 255}
	if d.ReproductionType != 0 {
		reproStr = "Sexual"
		reproColor = color.RGBA{255, 120, 180, 255}
	}
	repro := &components.Label{Text: fmt.Sprintf("Reproduction: %s", reproStr), Font: g.statFont, Color: reproColor}
	repro.Draw(screen, currX, currY)
	currY += h + 25

	g.drawPhenotypeChart(screen, d, currX, currY)

	btnY := int(currY) + 100
	sBtn := &components.Button{
		Label: "Save Genome",
		X:     int(currX),
		Y:     btnY,
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

	editBtn := &components.Button{
		Label: "Edit Genome",
		X:     int(currX),
		Y:     btnY + detailSaveBtnH + 4,
		W:     int(p.W - (detailTpad * 2)),
		H:     int(detailSaveBtnH),
		Color: color.RGBA{40, 60, 130, 220},
		OnClick: func() {
			id := g.selectedCreatureID
			if genome, ok := g.sim.CreatureGenomeCopy(id); ok {
				g.genomeEditor.Open(genome, g.sim.GetParams())
			}
		},
	}
	g.editGenomeBtn = editBtn
	g.editGenomeBtn.Draw(screen, g.statFont)
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
		labelX := nx - lblW - 20
		if labelX < px+4 {
			labelX = px + 4
		}
		text.Draw(screen, lbl, g.statFont, int(labelX), int(ny)+5, color.RGBA{160, 180, 220, 255})

		if val, ok := nn.SensorValues[s]; ok {
			barMaxW := float32(nnBarMaxW)
			barX := px + float32(nnPadding)
			barY := ny - float32(nnBarH)/2
			centerX := barX + (barMaxW / 2)

			// 1. Draw Background Track (Dark)
			vector.DrawFilledRect(screen, barX, barY, barMaxW, float32(nnBarH), color.RGBA{20, 40, 70, 180}, false)

			// 2. Draw Center "Zero" Line (Optional but helpful for visual reference)
			vector.StrokeLine(screen, centerX, barY, centerX, barY+float32(nnBarH), 1, color.RGBA{255, 255, 255, 50}, false)

			// 3. Calculate Fill Width and Position
			// Clamp value to [-1, 1] just in case
			cVal := val
			if cVal > 1 {
				cVal = 1
			} else if cVal < -1 {
				cVal = -1
			}

			halfWidth := (barMaxW / 2) * cVal

			var fillColor color.RGBA
			var fillX float32

			if halfWidth >= 0 {
				// Positive: Grow right from center
				fillX = centerX
				fillColor = color.RGBA{80, 160, 240, 220} // Blueish
			} else {
				// Negative: Grow left from center
				fillX = centerX + halfWidth              // halfWidth is negative, so this moves left
				halfWidth = -halfWidth                   // Rect width must be positive
				fillColor = color.RGBA{240, 80, 80, 220} // Reddish
			}

			// 4. Draw the actual value bar
			if halfWidth > 0 {
				vector.DrawFilledRect(screen, fillX, barY, halfWidth, float32(nnBarH), fillColor, false)
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

		// Draw the node dot
		vector.DrawFilledCircle(screen, ax, ay, nnNodeR, color.RGBA{220, 100, 80, 255}, false)

		// Draw the label
		lbl := nnActionName(a)
		text.Draw(screen, lbl, g.statFont, int(ax)+10, int(ay)+5, color.RGBA{220, 160, 150, 255})

		if val, ok := nn.ActionValues[a]; ok {
			barX := px + float32(nnPanelW) - float32(nnPadding) - float32(nnBarMaxW)
			barY := ay - float32(nnBarH)/2

			norm := float32(val / (1.0 + math.Abs(val)))

			vector.DrawFilledRect(screen, barX, barY, float32(nnBarMaxW), float32(nnBarH), color.RGBA{60, 25, 15, 180}, false)

			centerX := barX + (float32(nnBarMaxW) / 2)
			vector.StrokeLine(screen, centerX, barY, centerX, barY+float32(nnBarH), 1, color.RGBA{255, 255, 255, 50}, false)

			if norm != 0 {
				halfMaxW := float32(nnBarMaxW) / 2
				fillW := halfMaxW * norm

				fc := color.RGBA{240, 130, 80, 220} // Orange for Positive
				if norm < 0 {
					fc = color.RGBA{80, 130, 240, 220} // Blue for Negative
				}

				drawX := centerX
				if norm < 0 {
					drawX = centerX + fillW // fillW is negative, so this moves left
				}
				absFillW := fillW
				if absFillW < 0 {
					absFillW = -absFillW
				}

				vector.DrawFilledRect(screen, drawX, barY, absFillW, float32(nnBarH), fc, false)
			}
		}
	}

	const baseNeuroplasticity = 0.01
	footerY := int(py+panelH) - nnPadding - 2
	text.Draw(screen, fmt.Sprintf("Learning Rate: %.4f", baseNeuroplasticity*d.Dopamine), g.statFont, int(px)+nnPadding, footerY, color.RGBA{120, 120, 180, 220})
}
func nnEdgeColor(w float32) color.RGBA {
	absW := float32(math.Abs(float64(w)))

	intensity := absW / 4.0
	if intensity > 1.0 {
		intensity = 1.0
	}

	v := uint8(intensity * 255)

	alpha := uint8(30 + (intensity * 190))

	if w >= 0 {
		return color.RGBA{R: 10, G: v, B: 10, A: alpha}
	}

	return color.RGBA{R: v, G: 10, B: 10, A: alpha}
}

func nnSensorName(id byte) string {
	names := [...]string{
		"Bias", "Age", "Energy", "Loc X", "Loc Y", "Osc",
		"LocalDensity", "LocalHeading", "LocalCOM",
		"PopFwd", "PopCentroid", "FoodFwd", "MeatFwd",
		"Random", "Satiety", "Heading", "Velocity", "Food Ang",
		"Food Dist", "Threat", "KinshipLcl", "KinshipFwd", "KinshipNearest",
		"Mass %", "Blocked", "Prey", "Threat Ang",
		"Prey Ang", "Wall Prox", "Digest", "Food/Cap",
		"Juvenile", "EnergyDelta", "Temp", "TempDelta",
	}
	if int(id) < len(names) {
		return names[id]
	}
	return fmt.Sprintf("S%d", id)
}

func nnActionName(id byte) string {
	names := [...]string{
		"Accelerate", "Rotate", "SetOsc", "SetResp",
		"SetLearn", "Rest", "Attack", "Reward", "Punish", "Mate", "Feed",
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

// drawTemperatureBackground paints a vertical temperature gradient onto worldLayer.
// Top 20% of the world is cold (blue), bottom 20% is warm (red/orange), with a
// linear gradient in between. Rendered as 4-pixel horizontal strips.
func (g *Game) drawTemperatureBackground() {
	worldW := float32(int(g.sim.WorldWidth()) * UnitSize)
	worldH := float32(int(g.sim.WorldHeight()) * UnitSize)

	var coldR, coldG, coldB uint8
	var warmR, warmG, warmB uint8
	if g.isDarkBackground {
		coldR, coldG, coldB = 10, 22, 62
		warmR, warmG, warmB = 72, 18, 8
	} else {
		coldR, coldG, coldB = 130, 160, 210
		warmR, warmG, warmB = 210, 115, 55
	}

	params := g.sim.GetParams()
	radZoneW := float32(params.RadiationZoneWidth * float64(int(g.sim.WorldWidth())*UnitSize))

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
		vector.DrawFilledRect(g.worldLayer, 0, y, worldW, bandH, color.RGBA{r, gv, b, 255}, false)
	}
	vector.DrawFilledRect(g.worldLayer, 0, 0, radZoneW, worldH, color.RGBA{100, 130, 50, 40}, false)
	vector.StrokeLine(g.worldLayer, radZoneW, 0, radZoneW, worldH, 2, color.RGBA{100, 255, 70, 60}, false)
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
	energyColor := color.RGBA{255, 230, 50, 255}

	text.Draw(screen, fmt.Sprintf("Pop: %d", g.sim.PopulationCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+15, popColor)
	text.Draw(screen, fmt.Sprintf("Food: %d", g.sim.FoodCount()), g.statFont,
		graphPanelX+graphPad, graphPanelY+31, foodColor)
	text.Draw(screen, fmt.Sprintf("Energy: %.2f %", (g.sim.TotalEnergy()/g.sim.TargetEnergy()*100)), g.statFont,
		graphPanelX+graphPad, graphPanelY+47, energyColor)

	gx := float32(graphPanelX + graphPad)
	gy := float32(graphPanelY + graphTextH)
	gw := float32(graphPanelW - graphPad*2)
	gh := float32(graphPanelH - graphTextH - graphPad)

	foodMax := 1
	popMax := 1
	energyMax := g.MaxTotalEnergy
	for i := 0; i < g.histCount; i++ {
		idx := ((g.histHead-1-i)%historyLen + historyLen) % historyLen
		s := g.history[idx]

		if s.pop > popMax {
			popMax = s.pop
		}
		if s.food > foodMax {
			foodMax = s.food
		}
		if s.totalEnergy > energyMax {
			g.MaxTotalEnergy = s.totalEnergy
		}
	}

	steps := int(gw)
	if steps > g.histCount {
		steps = g.histCount
	}

	g.drawGraphLine(screen, gx, gy, gw, gh, steps, foodMax*2,
		foodColor, func(s histSample) int { return s.food })

	g.drawGraphLine(screen, gx, gy, gw, gh, steps, popMax*2,
		popColor, func(s histSample) int { return s.pop })
	g.drawGraphLine(screen, gx, gy, gw, gh, steps, int(g.MaxTotalEnergy),
		energyColor, func(s histSample) int { return int(s.totalEnergy) })
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
