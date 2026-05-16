package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"image/color"
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
	PlantCount() int
	PlantEnergy() float64
	MeatEnergy() float64
	AverageAge() float64
	AverageGeneration() float64
	CreatureMinMass() byte
	CreatureMaxMass() float64
	SaveCreature(id int, name string) error
	Reset()
	TotalEnergy() float64
	TargetEnergy() float64
	CreatureDetail(id int) (simulation.CreatureDetailView, bool)
	SetSpawnMutationRate(rate float32)
	SpawnAt(x, y float64) bool
	SpawnClusterAt(x, y float64, count int) bool
	SpawnGenome(g *simulation.Genome, generation float32) bool
	CreatureGenomeCopy(id int) (*simulation.Genome, bool)
	GetParams() *simulation.Parameters
	GetSnapshot() simulation.StateSnapshot
}

const historyLen = 5000

type histSample struct {
	pop         int
	plantEnergy float64
	meatEnergy  float64
	totalEnergy float64
}

// UnitSize is the pixel size of one simulation unit.
var UnitSize int = 2

// Game is the root ebiten.Game implementation. It coordinates WorldRenderer
// (world-space drawing) and UserInterface (HUD) while owning shared state.
type Game struct {
	sim             SimulationState
	world           *WorldRenderer
	ui              *UserInterface
	selectedCreatureID int
	paused          bool
	spawnPlacing    bool
	history         [historyLen]histSample
	histHead        int
	histCount       int
	simStepsPerTick int
	currentSnapshot *simulation.StateSnapshot
	tickDuration    time.Duration

	// Held by UI for input routing (updated each frame by UserInterface)
	spawnMutSlider  *components.Slider
	spawnRandomBtn  *components.Button
	genomeEditor    *GenomeEditor
	savedGenomesPanel *SavedGenomesPanel
}

// NewGame constructs the full game, wiring all subsystems.
func NewGame(sim SimulationState) *Game {
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	rawFace, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	statFont := textv2.NewGoXFace(rawFace)

	g := &Game{
		sim:                sim,
		selectedCreatureID: -1,
		spawnPlacing:       false,
		simStepsPerTick:    1,
	}

	g.world = NewWorldRenderer(sim)
	g.ui = NewUserInterface(statFont, sim, g)

	return g
}

func (g *Game) Update() error {
	// Modal input takes priority
	if g.genomeEditor != nil && g.genomeEditor.visible {
		g.genomeEditor.HandleKeyInput()
	} else {
		g.ui.HandleSaveNameKeyInput()
	}

	// Scroll wheel: zoom or saved-genome list scroll
	_, scrollY := ebiten.Wheel()
	if scrollY != 0 {
		if g.savedGenomesPanel != nil && g.savedGenomesPanel.visible {
			g.savedGenomesPanel.Scroll(int(scrollY))
		} else {
			cam := g.world.Camera()
			cam.Zoom *= math.Pow(1.15, scrollY)
			if cam.Zoom < 0.1 {
				cam.Zoom = 0.1
			} else if cam.Zoom > 10.0 {
				cam.Zoom = 10.0
			}
		}
	}

	// Continuous input (camera drag, slider drag)
	sliderDragging := g.spawnMutSlider != nil && g.spawnMutSlider.Dragging
	if g.genomeEditor != nil && g.genomeEditor.visible {
		mx, my := ebiten.CursorPosition()
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.genomeEditor.HandleDrag(mx, my)
		} else {
			g.genomeEditor.HandleRelease()
		}
	} else {
		g.world.HandleContinuousInput(sliderDragging)
		g.ui.HandleContinuousInput()
	}

	// Mouse-down
	mx, my := ebiten.CursorPosition()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if g.genomeEditor != nil && g.genomeEditor.visible {
			g.genomeEditor.HandleInput(mx, my)
		} else if g.savedGenomesPanel != nil && g.savedGenomesPanel.visible {
			g.savedGenomesPanel.HandleInput(mx, my)
		} else if g.ui.HandleClick(mx, my) {
			// UI consumed the click
		} else if g.spawnPlacing {
			sw, sh := ebiten.WindowSize()
			cam := g.world.Camera()
			wx, wy := cam.ScreenToWorld(float64(mx), float64(my), float64(sw), float64(sh))
			bs := float64(UnitSize)
			g.sim.SpawnClusterAt(wx/bs, wy/bs, 5)
		} else {
			g.world.StartCameraDrag(mx, my)
		}
	}

	// Mouse-up
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		moved := g.world.StopCameraDrag()
		if !moved {
			sx, sy := g.world.CamDragStartPos()
			sw, sh := ebiten.WindowSize()
			newID := g.world.TrySelectCreature(sx, sy, sw, sh, g.selectedCreatureID)
			g.selectedCreatureID = newID
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.spawnPlacing = false
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.simStepsPerTick++
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) && g.simStepsPerTick > 1 {
		g.simStepsPerTick--
	}

	if !g.paused {
		// Run all steps except last without capturing positions
		for i := 1; i < g.simStepsPerTick; i++ {
			g.sim.Update()
		}
		g.sim.Update()

		g.history[g.histHead] = histSample{
			pop:         g.sim.PopulationCount(),
			plantEnergy: g.sim.PlantEnergy(),
			meatEnergy:  g.sim.MeatEnergy(),
			totalEnergy: g.sim.TotalEnergy(),
		}
		g.histHead = (g.histHead + 1) % historyLen
		if g.histCount < historyLen {
			g.histCount++
		}

		snapshot := g.sim.GetSnapshot()
		g.currentSnapshot = &snapshot

		g.tickDuration = g.world.UpdateAnimations(&snapshot)
	}

	return nil
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

// drawText is a package-level helper used by multiple files.
func drawText(dst *ebiten.Image, str string, face *textv2.GoXFace, x, y int, clr color.Color) {
	op := &textv2.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(clr)
	textv2.Draw(dst, str, face, op)
}

// clamp restricts v to [0, 1].
func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
