package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"biogo/v2/ui/views"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// SimulationState is the interface the UI requires from the simulation.
type SimulationState interface {
	Update()
	WorldWidth() float64
	WorldHeight() float64
	PopulationCount() int
	FoliageCount() int
	FoliageEnergy() float64
	FungiCount() int
	FungiEnergy() float64
	MeatEnergy() float64
	AverageAge() float64
	AverageGeneration() float64
	SaveCreature(id int, name string) error
	Reset()
	TotalEnergy() float64
	TargetEnergy() float64
	SetTargetEnergy(v float64)
	CreatureDetail(id int) (simulation.CreatureDetailView, bool)
	SetFoliageProportion(v float64)
	SetFungiProportion(v float64)
	SetMeatProportion(v float64)
	SetFoliageRandomFraction(v float64)
	SetFungiRandomFraction(v float64)
	SetMeatRandomFraction(v float64)
	SetFoliageFountainCount(n int)
	SetFungiFountainCount(n int)
	SetMeatFountainCount(n int)
	SetFoliageDriftSpeed(v float64)
	SetFungiDriftSpeed(v float64)
	SetMeatDriftSpeed(v float64)
	SetFoliageRadius(v float64)
	SetFungiRadius(v float64)
	SetMeatRadius(v float64)
	SetTempMin(v float32)
	SetTempMax(v float32)
	SetWarmMetabolicMultiplier(v float32)
	SetColdSpeedMultiplier(v float32)
	SetClusterEnabled(v bool)
	SetClusterInterval(v int)
	SetClusterSize(v int)
	SetBaseMutationRate(v float32)
	SpawnAt(x, y float64) bool
	SpawnClusterAt(x, y float64, count int) bool
	SpawnGenome(g *simulation.Genome, generation float32) bool
	CreatureGenomeCopy(id int) (*simulation.Genome, bool)
	GetParams() *simulation.Parameters
	GetSnapshot() simulation.StateSnapshot
	FillSnapshot(dst *simulation.StateSnapshot)
	SaveGame(name string) error
	SaveGameTo(path string) error
	LoadGame(path string) error
	ListSavedGames() []simulation.SavedGame
}

// Game is the root ebiten.Game implementation. It coordinates WorldRenderer
// (world-space drawing) and UserInterface (HUD) while owning shared state.
type Game struct {
	sim                SimulationState
	world              *WorldRenderer
	ui                 *UserInterface
	selectedCreatureID int
	paused             bool
	spawnPlacing       bool
	simStepsPerTick    int
	snapshot           simulation.StateSnapshot
	currentSnapshot    *simulation.StateSnapshot
	tickDuration       time.Duration
	uiConsumedClick    bool

	// Sub-panel tracking allocations
	history           *views.PerformanceHistory
	genomeEditor      *views.GenomeEditor
	savedGenomesPanel *views.SavedGenomesPanel
	spawnRandomBtn    *components.Button // TODO(): Move
}

// NewGame constructs the full game, wiring all subsystems.
func NewGame(sim SimulationState) *Game {
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	rawFace, _ := opentype.NewFace(tt, &opentype.FaceOptions{Size: 16, DPI: 72, Hinting: font.HintingFull})
	smallRawFace, _ := opentype.NewFace(tt, &opentype.FaceOptions{Size: 12, DPI: 72, Hinting: font.HintingFull})

	g := &Game{
		sim:                sim,
		history:            views.NewPerformanceHistory(),
		selectedCreatureID: -1,
		spawnPlacing:       false,
		simStepsPerTick:    1,
	}

	g.world = NewWorldRenderer(sim)
	g.ui = NewUserInterface(textv2.NewGoXFace(rawFace), textv2.NewGoXFace(smallRawFace), sim, g)

	return g
}
func (g *Game) Update() error {
	g.handleModalPriorityInputs()
	g.handleScrollZoomInput()
	g.handleContinuousDragging()
	g.handleMousePressClicks()
	g.handleMouseReleaseClicks()
	g.handleKeyboardTriggers()

	if !g.paused {
		g.stepSimulationTick()
	}
	return nil
}
func (g *Game) handleModalPriorityInputs() {
	if g.genomeEditor != nil && g.genomeEditor.Visible {
		g.genomeEditor.HandleKeyInput()
	} else {
		g.ui.HandleSaveNameKeyInput()
	}
}

func (g *Game) handleScrollZoomInput() {
	if _, scrollY := ebiten.Wheel(); scrollY != 0 {
		if g.savedGenomesPanel != nil && g.savedGenomesPanel.Visible {
			g.savedGenomesPanel.Scroll(int(scrollY))
		} else {
			cam := g.world.Camera()
			cam.Zoom = math.Max(0.05, math.Min(20.0, cam.Zoom*math.Pow(1.15, scrollY)))
		}
	}
}

func (g *Game) handleContinuousDragging() {
	if g.genomeEditor != nil && g.genomeEditor.Visible {
		mx, my := ebiten.CursorPosition()
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.genomeEditor.HandleDrag(mx, my)
		} else {
			g.genomeEditor.HandleRelease()
		}
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
			g.genomeEditor.HandleRightDrag(mx, my)
		} else {
			g.genomeEditor.HandleRightRelease()
		}
	} else {
		g.world.HandleContinuousInput(g.ui.AnySliderDragging())
		g.ui.HandleContinuousInput()
	}
}

func (g *Game) handleMousePressClicks() {
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		if g.genomeEditor != nil && g.genomeEditor.Visible {
			g.genomeEditor.HandleRightClick(mx, my)
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		switch {
		case g.savedGenomesPanel != nil && g.savedGenomesPanel.Visible:
			g.savedGenomesPanel.HandleInput(mx, my)
			g.uiConsumedClick = true
		case g.genomeEditor != nil && g.genomeEditor.Visible:
			g.genomeEditor.HandleInput(mx, my)
			g.uiConsumedClick = true
		case g.ui.HandleClick(mx, my):
			g.uiConsumedClick = true
		case g.spawnPlacing:
			g.uiConsumedClick = false
			sw, sh := ebiten.WindowSize()
			wx, wy := g.world.Camera().ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
			bs := float64(UnitSize)
			g.sim.SpawnClusterAt(wx/bs, wy/bs, 5)
		default:
			g.uiConsumedClick = false
			g.world.StartCameraDrag(mx, my)
		}
	}
}

func (g *Game) handleMouseReleaseClicks() {
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		moved := g.world.StopCameraDrag()
		if !g.uiConsumedClick && !moved {
			sx, sy := g.world.CamDragStartPos()
			sw, sh := ebiten.WindowSize()
			g.selectedCreatureID = g.world.TrySelectCreature(sx, sy, sw, sh, g.selectedCreatureID)
		}
		g.uiConsumedClick = false
	}
}

func (g *Game) handleKeyboardTriggers() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.spawnPlacing = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.simStepsPerTick = nextSimRate(g.simStepsPerTick, 1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		g.simStepsPerTick = nextSimRate(g.simStepsPerTick, -1)
	}
}

func (g *Game) stepSimulationTick() {
	for i := 1; i < g.simStepsPerTick; i++ {
		g.sim.Update()
	}
	g.sim.Update()

	g.history.Append(views.HistSample{
		Pop:           g.sim.PopulationCount(),
		FoliageEnergy: g.sim.FoliageEnergy(),
		FungiEnergy:   g.sim.FungiEnergy(),
		MeatEnergy:    g.sim.MeatEnergy(),
		TotalEnergy:   g.sim.TotalEnergy(),
	})

	g.sim.FillSnapshot(&g.snapshot)
	g.currentSnapshot = &g.snapshot
	g.tickDuration = g.world.UpdateAnimations(&g.snapshot)
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.world.Draw(screen, g.currentSnapshot, g.selectedCreatureID)
	g.ui.Draw(screen, UIDrawState{
		selectedID:      g.selectedCreatureID,
		paused:          g.paused,
		spawnPlacing:    g.spawnPlacing,
		simStepsPerTick: g.simStepsPerTick,
		tickDuration:    g.tickDuration,
	}, g)
}

func (g *Game) Layout(w, h int) (int, int) { return w, h }

func (g *Game) histHead() int {
	return g.history.CurrentHead()
}

func (g *Game) histCount() int {
	return g.history.TotalCount()
}

func (g *Game) resetHistory() {
	g.history.Reset()
}

func (g *Game) histSample(i int) views.HistSample {
	return g.history.SampleAt(i)
}
