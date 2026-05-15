package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuBarH       = float32(34)
	menuBarPad     = float32(10)
	menuBarSpacing = float32(8)

	detailPanelW = float32(210)
	detailPad    = float32(8)
	detailSpacing = float32(4)

	leftStackX       = float32(10)
	leftStackStartY  = menuBarH + 4
	leftStackSpacing = float32(4)
)

// UIDrawState carries per-frame data from Game to UserInterface.Draw.
type UIDrawState struct {
	selectedID      int
	paused          bool
	spawnPlacing    bool
	simStepsPerTick int
	tickDuration    time.Duration
}

// UserInterface owns all HUD rendering: menu bar, left panel stack, stats,
// modals, and the spawn cursor overlay.
type UserInterface struct {
	font *textv2.GoXFace
	sim  SimulationState

	menuBar    *components.MenuBar
	leftStack  *LeftPanelStack
	histGraph  *HistoryGraph
	histIdx    int

	detailIdx int
	nnIdx     int

	currentSaveBtn *components.Button
	currentEditBtn *components.Button

	genomeEditor      *GenomeEditor
	savedGenomesPanel *SavedGenomesPanel

	saveFeedback   string
	saveFeedbackAt time.Time

	// references so buttons can trigger game-level actions
	onSaveCreature    func() error
	onEditCreature    func()
	onSpawnAtWorld    func(wx, wy float64)
	onPause           func()
	onRestart         func()
	onToggleTheme     func()
	onCreateGenome    func()
	onSpawnSaved      func()
	onToggleSpawn     func()
}

// NewUserInterface constructs the UI, wiring up all interactive elements.
func NewUserInterface(
	font *textv2.GoXFace,
	sim SimulationState,
	game *Game,
) *UserInterface {
	ui := &UserInterface{
		font: font,
		sim:  sim,
	}

	// ── Mutation-rate slider ──────────────────────────────────────────────────
	mutSlider := &components.Slider{
		W: 240, H: 24,
		TrackOffX: 135, TrackW: 100,
		Label:      "Mut",
		Font:       font,
		LabelColor: color.White,
		Min: 0.0001, Max: 0.2,
		Value: 0.01,
		OnChange: func(v float64) {
			sim.SetSpawnMutationRate(float32(v))
		},
	}
	game.spawnMutSlider = mutSlider

	// ── Menu bar buttons ─────────────────────────────────────────────────────
	pauseBtn := &components.Button{W: 80, H: 24, Label: "Pause", Color: components.ColorButtonRed, LabelColor: color.White, Font: font}
	pauseBtn.OnClick = func() {
		game.paused = !game.paused
		if game.paused {
			pauseBtn.Label = "Resume"
			pauseBtn.Color = components.ColorButtonGreen
		} else {
			pauseBtn.Label = "Pause"
			pauseBtn.Color = components.ColorButtonRed
		}
	}
	restartBtn := &components.Button{W: 90, H: 24, Label: "Restart", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	restartBtn.OnClick = func() {
		game.sim.Reset()
		game.world.ResetAnimations()
		game.selectedCreatureID = -1
		game.histHead = 0
		game.histCount = 0
		game.currentSnapshot = nil
	}
	themeBtn := &components.Button{W: 100, H: 24, Label: "Theme", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	themeBtn.OnClick = func() { game.world.ToggleDark() }

	spawnRandomBtn := &components.Button{W: 130, H: 24, Label: "Spawn Random", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	spawnRandomBtn.OnClick = func() {
		game.spawnPlacing = !game.spawnPlacing
	}
	game.spawnRandomBtn = spawnRandomBtn

	createGenomeBtn := &components.Button{W: 130, H: 24, Label: "Create Genome", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	createGenomeBtn.OnClick = func() {
		game.genomeEditor.Open(nil, game.sim.GetParams())
	}
	spawnSavedBtn := &components.Button{W: 120, H: 24, Label: "Spawn Saved", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	spawnSavedBtn.OnClick = func() {
		game.savedGenomesPanel.Open()
	}

	mb := &components.MenuBar{
		H:       menuBarH,
		Padding: menuBarPad,
		Spacing: menuBarSpacing,
		Color:   color.RGBA{12, 14, 28, 220},
	}
	mb.AddButton(pauseBtn)
	mb.AddButton(restartBtn)
	mb.AddButton(themeBtn)
	mb.AddSlider(mutSlider)
	mb.AddButton(spawnRandomBtn)
	mb.AddButton(createGenomeBtn)
	mb.AddButton(spawnSavedBtn)
	ui.menuBar = mb

	// ── Left panel stack ──────────────────────────────────────────────────────
	ui.leftStack = &LeftPanelStack{
		X:       leftStackX,
		StartY:  leftStackStartY,
		Spacing: leftStackSpacing,
	}

	histGraph := &HistoryGraph{
		Font:      font,
		getCount:  func() int { return game.histCount },
		getHead:   func() int { return game.histHead },
		getSample: func(i int) histSample { return game.history[i] },
		sim:       sim,
	}
	ui.histGraph = histGraph
	ui.histIdx = ui.leftStack.Register(histGraph)
	ui.detailIdx = ui.leftStack.Register(nil)
	ui.nnIdx = ui.leftStack.Register(nil)

	// ── Modals ────────────────────────────────────────────────────────────────
	ui.genomeEditor = newGenomeEditor(font, func(genome *simulation.Genome, name string) {
		game.sim.SpawnGenome(genome)
		simulation.SaveCreatureToFileNamed(genome, name) //nolint:errcheck
	})
	game.genomeEditor = ui.genomeEditor

	ui.savedGenomesPanel = newSavedGenomesPanel(func(genome *simulation.Genome) {
		game.sim.SpawnGenome(genome)
	})
	game.savedGenomesPanel = ui.savedGenomesPanel

	return ui
}

// HandleClick processes a mouse-down event; returns true if consumed.
func (ui *UserInterface) HandleClick(mx, my int) bool {
	if ui.menuBar.HandleClick(mx, my) {
		return true
	}
	if ui.currentSaveBtn != nil && ui.currentSaveBtn.IsClicked(mx, my) {
		if ui.currentSaveBtn.OnClick != nil {
			ui.currentSaveBtn.OnClick()
		}
		return true
	}
	if ui.currentEditBtn != nil && ui.currentEditBtn.IsClicked(mx, my) {
		if ui.currentEditBtn.OnClick != nil {
			ui.currentEditBtn.OnClick()
		}
		return true
	}
	return false
}

// HandleContinuousInput forwards drag events to the menu bar (slider drag).
func (ui *UserInterface) HandleContinuousInput() {
	mx, _ := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		ui.menuBar.HandleDrag(mx)
	} else {
		ui.menuBar.HandleRelease()
	}
}

// Draw renders the entire HUD layer.
func (ui *UserInterface) Draw(screen *ebiten.Image, state UIDrawState, game *Game) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Update creature detail and NN slots
	if state.selectedID != -1 {
		if detail, ok := ui.sim.CreatureDetail(state.selectedID); ok {
			detailPanel := ui.buildDetailPanel(detail, state.selectedID, game)
			ui.leftStack.Set(ui.detailIdx, detailPanel, true)

			nnGraph := &NeuralNetGraph{Font: ui.font, Data: detail}
			ui.leftStack.Set(ui.nnIdx, nnGraph, true)
		} else {
			ui.leftStack.Set(ui.detailIdx, nil, false)
			ui.leftStack.Set(ui.nnIdx, nil, false)
			ui.currentSaveBtn = nil
			ui.currentEditBtn = nil
		}
	} else {
		ui.leftStack.Set(ui.detailIdx, nil, false)
		ui.leftStack.Set(ui.nnIdx, nil, false)
		ui.currentSaveBtn = nil
		ui.currentEditBtn = nil
	}

	ui.menuBar.Draw(screen)
	ui.leftStack.Draw(screen)

	// Spawn label update on button
	if game.spawnRandomBtn != nil {
		if state.spawnPlacing {
			game.spawnRandomBtn.Label = "Cancel Spawn"
		} else {
			game.spawnRandomBtn.Label = "Spawn Random"
		}
	}

	// Spawn placing crosshair cursor
	if state.spawnPlacing {
		mx, my := ebiten.CursorPosition()
		cx, cy := float32(mx), float32(my)
		const arm = float32(10)
		vector.StrokeLine(screen, cx-arm, cy, cx+arm, cy, 1.5, color.RGBA{255, 220, 80, 220}, false)
		vector.StrokeLine(screen, cx, cy-arm, cx, cy+arm, 1.5, color.RGBA{255, 220, 80, 220}, false)
		vector.StrokeCircle(screen, cx, cy, arm*0.6, 1, color.RGBA{255, 220, 80, 160}, false)
	}

	// Top-right stats
	x := sw - 200
	if ui.font != nil {
		drawText(screen, fmt.Sprintf("Population: %d", ui.sim.PopulationCount()), ui.font, x, 23, color.White)
		drawText(screen, fmt.Sprintf("Food: %d", ui.sim.FoodCount()), ui.font, x, 43, color.White)
		drawText(screen, fmt.Sprintf("Avg Age: %.0f", ui.sim.AverageAge()), ui.font, x, 63, color.White)
		if state.tickDuration > 0 {
			simRate := float64(state.simStepsPerTick) / state.tickDuration.Seconds()
			drawText(screen, fmt.Sprintf("Sim Rate: %.0f/s (%dx)", simRate, state.simStepsPerTick), ui.font, x, 83, color.White)
		}
	}

	// Save feedback
	if ui.saveFeedback != "" && time.Since(ui.saveFeedbackAt) < 2*time.Second {
		if ui.font != nil {
			drawText(screen, ui.saveFeedback, ui.font, sw/2-30, sh-40, color.RGBA{100, 255, 120, 255})
		}
	}

	// Modals (drawn last, on top of everything)
	ui.savedGenomesPanel.Draw(screen, ui.font)
	ui.genomeEditor.Draw(screen, ui.font)
}

// buildDetailPanel constructs a Panel containing all creature stat rows.
// It is rebuilt each frame so data is always fresh.
func (ui *UserInterface) buildDetailPanel(d simulation.CreatureDetailView, creatureID int, game *Game) *components.Panel {
	innerW := detailPanelW - detailPad*2

	p := &components.Panel{
		W:         detailPanelW,
		Padding:   detailPad,
		Spacing:   detailSpacing,
		BaseColor: color.RGBA{8, 10, 22, 215},
		Border:    color.RGBA{90, 90, 150, 255},
	}

	// Title
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Creature #%d", d.ID),
		Font:  ui.font,
		Color: color.RGBA{255, 220, 80, 255},
	})

	// Energy
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Energy: %.f/%.f", d.Energy, d.MaxEnergy),
		Font:  ui.font,
		Color: color.White,
	})
	p.Add(&components.EnergyBar{
		Value:    d.Energy,
		Max:      d.MaxEnergy,
		MaxColor: color.RGBA{55, 185, 55, 255},
		MinColor: color.RGBA{190, 55, 55, 255},
		Width:    innerW,
	})

	// Age
	juvenileStr := "Adult"
	if d.IsJuvenile {
		juvenileStr = fmt.Sprintf("Juvenile(%d)", d.JuvenilePeriod-d.Age)
	}
	p.Add(&components.Label{Text: fmt.Sprintf("Age: %d, %s", d.Age, juvenileStr), Font: ui.font, Color: color.White})

	// Last action
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Actions: %s", d.LastAction),
		Font:  ui.font,
		Color: components.ColorButtonGreen,
	})

	// Mass
	p.Add(&components.Label{Text: fmt.Sprintf("Mass: %.0f / %.0f", d.CurrentMass, d.AdultMass), Font: ui.font, Color: color.White})

	// Stomach
	p.Add(&components.Label{Text: fmt.Sprintf("Stomach: %.0f/%.0f", d.Stomach, d.StomachCapacity), Font: ui.font, Color: color.White})
	p.Add(&components.EnergyBar{
		Value:    float32(d.Stomach),
		Max:      float32(d.StomachCapacity),
		MaxColor: color.RGBA{55, 185, 55, 255},
		MinColor: color.RGBA{190, 55, 55, 255},
		Width:    innerW,
	})

	// Efficiency
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Food Eff: %.1f%%  Meat Eff: %.1f%%", d.FoodEfficiency*100, d.MeatEfficiency*100),
		Font:  ui.font,
		Color: color.White,
	})

	// Dopamine
	p.Add(&components.Label{Text: fmt.Sprintf("Dopamine: %.02f", d.Dopamine), Font: ui.font, Color: color.White})
	p.Add(&components.EnergyBar{
		Value:    d.Dopamine,
		Max:      float32(1.2),
		MaxColor: color.RGBA{216, 27, 96, 255},
		MinColor: color.RGBA{48, 63, 159, 255},
		Width:    innerW,
		Centered: true,
	})

	// Sight
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Sight: %.f  FOV: %.f°", d.SightDistance, d.FieldOfView),
		Font:  ui.font,
		Color: color.White,
	})

	// Mutation
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Mutation: %.2f%%", d.MutationPct),
		Font:  ui.font,
		Color: color.White,
	})

	// Reproduction
	reproStr := "Asexual"
	reproColor := color.RGBA{100, 180, 255, 255}
	if d.ReproductionType != 0 {
		reproStr = "Sexual"
		reproColor = color.RGBA{255, 120, 180, 255}
	}
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Reproduction: %s", reproStr),
		Font:  ui.font,
		Color: reproColor,
	})

	// Phenotype chart
	p.Add(&PhenotypeChart{Font: ui.font, Data: d})

	// Save button
	saveBtn := &components.Button{
		W: innerW, H: 22,
		Label:      "Save Genome",
		Color:      color.RGBA{40, 100, 60, 220},
		LabelColor: color.White,
		Font:       ui.font,
	}
	saveBtn.OnClick = func() {
		if err := game.sim.SaveCreature(creatureID); err != nil {
			ui.saveFeedback = "Save failed"
		} else {
			ui.saveFeedback = "Saved!"
		}
		ui.saveFeedbackAt = time.Now()
	}
	p.Add(saveBtn)
	ui.currentSaveBtn = saveBtn

	// Edit button
	editBtn := &components.Button{
		W: innerW, H: 22,
		Label:      "Edit Genome",
		Color:      color.RGBA{40, 60, 130, 220},
		LabelColor: color.White,
		Font:       ui.font,
	}
	editBtn.OnClick = func() {
		if genome, ok := game.sim.CreatureGenomeCopy(creatureID); ok {
			game.genomeEditor.Open(genome, game.sim.GetParams())
		}
	}
	p.Add(editBtn)
	ui.currentEditBtn = editBtn

	return p
}
