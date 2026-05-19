package ui

import (
	"biogo/v2/simulation"
	"biogo/v2/ui/components"
	"fmt"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	textv2 "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuBarH       = float32(34)
	menuBarPad     = float32(10)
	menuBarSpacing = float32(8)

	detailPanelW  = float32(210)
	detailPad     = float32(8)
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
	font      *textv2.GoXFace
	smallFont *textv2.GoXFace
	sim       SimulationState

	menuBar      *components.MenuBar
	leftStack    *LeftPanelStack
	histGraph    *HistoryGraph
	histStatsIdx int
	histIdx      int

	detailIdx int
	nnIdx     int

	currentSaveBtn   *components.Button
	currentEditBtn   *components.Button
	currentSaveInput *components.TextInputField

	genomeEditor      *GenomeEditor
	savedGenomesPanel *SavedGenomesPanel

	saveFeedback     string
	saveFeedbackAt   time.Time
	saveCreatureName string
	saveNameFocused  bool
	saveCreatureID   int

	foodDropdown     *Dropdown
	climateDropdown  *Dropdown
	spawnDropdown    *Dropdown
	saveGameDropdown *Dropdown

	foodBtn     *components.Button
	climateBtn  *components.Button
	spawnBtn    *components.Button
	saveGameBtn *components.Button

	saveGameNameInput   *components.TextInputField
	saveGameName        string
	saveGameNameFocused bool
	saveGameFeedback    string
	saveGameFeedbackAt  time.Time

	speedLabel *components.Button

	// references so buttons can trigger game-level actions
	onSaveCreature func() error
	onEditCreature func()
	onSpawnAtWorld func(wx, wy float64)
	onPause        func()
	onRestart      func()
	onToggleTheme  func()
	onCreateGenome func()
	onSpawnSaved   func()
	onToggleSpawn  func()
}

// NewUserInterface constructs the UI, wiring up all interactive elements.
func NewUserInterface(
	font *textv2.GoXFace,
	smallFont *textv2.GoXFace,
	sim SimulationState,
	game *Game,
) *UserInterface {
	ui := &UserInterface{
		font:           font,
		smallFont:      smallFont,
		sim:            sim,
		saveCreatureID: -1,
	}

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
	// ── Save Game button + dropdown ───────────────────────────────────────────
	saveGameBtn := &components.Button{W: 100, H: 24, Label: "Worlds", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	ui.saveGameBtn = saveGameBtn
	ui.saveGameNameInput = &components.TextInputField{
		W: saveGamePanelW - ddPad*2, H: 24,
		Placeholder: "(save name...)",
		Font:        font,
	}
	ui.saveGameDropdown = newSaveGameDropdown(font, saveGameBtn)

	saveGameBtn.OnClick = func() {
		ui.foodDropdown.Close()
		ui.climateDropdown.Close()
		ui.spawnDropdown.Close()
		if !ui.saveGameDropdown.IsOpen() {
			// Refresh save list on every open.
			saves := game.sim.ListSavedGames()
			rebuildSaveGameItems(
				ui.saveGameDropdown,
				font,
				ui.saveGameNameInput,
				func() string { return ui.saveGameName },
				func() bool { return ui.saveGameNameFocused },
				func() { ui.saveGameNameFocused = true },
				func() {
					// Save as New
					if err := game.sim.SaveGame(ui.saveGameName); err != nil {
						ui.saveGameFeedback = "Save failed: " + err.Error()
					} else {
						ui.saveGameFeedback = "Saved!"
						ui.saveGameName = ""
						ui.saveGameNameFocused = false
					}
					ui.saveGameFeedbackAt = time.Now()
				},
				saves,
				func(path string) {
					// Load
					if err := game.sim.LoadGame(path); err != nil {
						ui.saveGameFeedback = "Load failed: " + err.Error()
						ui.saveGameFeedbackAt = time.Now()
						return
					}
					ui.saveGameDropdown.Close()
					ui.saveGameNameFocused = false
					game.selectedCreatureID = -1
					game.histHead = 0
					game.histCount = 0
					game.currentSnapshot = nil
					game.world.ResetAnimations()
				},
				func(path string) {
					// Overwrite
					if err := game.sim.SaveGameTo(path); err != nil {
						ui.saveGameFeedback = "Save failed: " + err.Error()
					} else {
						ui.saveGameFeedback = "Saved!"
					}
					ui.saveGameFeedbackAt = time.Now()
				},
			)
		}
		ui.saveGameDropdown.Toggle()
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

	tierBtn := &components.Button{W: 90, H: 24, Label: "Tier: All", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	tierBtn.OnClick = func() {
		tierBtn.Label = game.world.CycleTierFilter()
	}

	foodBtn := &components.Button{W: 60, H: 24, Label: "Food", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	climateBtn := &components.Button{W: 80, H: 24, Label: "Climate", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	spawnBtn := &components.Button{W: 80, H: 24, Label: "Spawning", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	ui.foodBtn = foodBtn
	ui.climateBtn = climateBtn
	ui.spawnBtn = spawnBtn

	mb := &components.MenuBar{
		H:       menuBarH,
		Padding: menuBarPad,
		Spacing: menuBarSpacing,
		Color:   ColorMenuBar,
	}
	mb.AddButton(pauseBtn)
	mb.AddButton(saveGameBtn)
	mb.AddButton(restartBtn)
	mb.AddButton(themeBtn)
	mb.AddButton(spawnRandomBtn)
	mb.AddButton(createGenomeBtn)
	mb.AddButton(tierBtn)
	mb.AddButton(foodBtn)
	mb.AddButton(climateBtn)
	mb.AddButton(spawnBtn)

	// Speed controls (right-aligned): < [speed] >
	speedDownBtn := &components.Button{W: 24, H: 24, Label: "<", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	speedDownBtn.OnClick = func() {
		game.simStepsPerTick = nextSimRate(game.simStepsPerTick, -1)
	}
	speedLabelBtn := &components.Button{W: 52, H: 24, Label: "1x", Color: color.RGBA{0, 0, 0, 0}, LabelColor: color.White, Font: font}
	speedUpBtn := &components.Button{W: 24, H: 24, Label: ">", Color: components.ColorDefault, LabelColor: color.White, Font: font}
	speedUpBtn.OnClick = func() {
		game.simStepsPerTick = nextSimRate(game.simStepsPerTick, 1)
	}
	// Target Energy slider (right-aligned, left of speed controls).
	// TrackOffX wide enough for "Tgt E: 30000k" (~95px); TrackW wider for easier dragging.
	targetESlider := &components.Slider{
		W: 270, H: 24,
		TrackOffX: 100, TrackW: 170,
		Font: font, LabelColor: ColorLabelTargetE,
		Min: 0, Max: 30_000_000,
		Value: sim.TargetEnergy(),
		FormatFunc: func(v float64) string {
			return fmt.Sprintf("Tgt E: %.0fk", v/1000)
		},
		OnChange: func(v float64) { sim.SetTargetEnergy(v) },
	}
	mb.AddSliderRight(targetESlider)

	mb.AddButtonRight(speedDownBtn)
	mb.AddButtonRight(speedLabelBtn)
	mb.AddButtonRight(speedUpBtn)
	ui.speedLabel = speedLabelBtn

	ui.menuBar = mb

	ui.foodDropdown = newFoodDropdown(font, foodBtn, sim)
	foodBtn.OnClick = func() {
		ui.climateDropdown.Close()
		ui.spawnDropdown.Close()
		ui.saveGameDropdown.Close()
		ui.foodDropdown.Toggle()
	}

	ui.climateDropdown = newClimateDropdown(font, climateBtn, sim)
	climateBtn.OnClick = func() {
		ui.foodDropdown.Close()
		ui.spawnDropdown.Close()
		ui.saveGameDropdown.Close()
		ui.climateDropdown.Toggle()
	}

	ui.spawnDropdown = newSpawnDropdown(font, spawnBtn, sim, func() {
		ui.spawnDropdown.Close()
		game.savedGenomesPanel.Open()
	})
	spawnBtn.OnClick = func() {
		ui.foodDropdown.Close()
		ui.climateDropdown.Close()
		ui.saveGameDropdown.Close()
		ui.spawnDropdown.Toggle()
	}

	// ── Left panel stack ──────────────────────────────────────────────────────
	ui.leftStack = &LeftPanelStack{
		X:       leftStackX,
		StartY:  leftStackStartY,
		Spacing: leftStackSpacing,
	}

	ui.histStatsIdx = ui.leftStack.Register(nil) // stats panel, set each frame in Draw

	histGraph := &HistoryGraph{
		getCount:  func() int { return game.histCount },
		getHead:   func() int { return game.histHead },
		getSample: func(i int) histSample { return game.history[i] },
	}
	ui.histGraph = histGraph
	ui.histIdx = ui.leftStack.Register(histGraph)
	ui.detailIdx = ui.leftStack.Register(nil)
	ui.nnIdx = ui.leftStack.Register(nil)

	// ── Modals ────────────────────────────────────────────────────────────────
	ui.genomeEditor = newGenomeEditor(font, smallFont, func(genome *simulation.Genome, name string) {
		game.sim.SpawnGenome(genome, 1.0)
		simulation.SaveCreatureToFileNamed(genome, 1.0, name) //nolint:errcheck
	})
	game.genomeEditor = ui.genomeEditor

	ui.savedGenomesPanel = newSavedGenomesPanel(func(genome *simulation.Genome, generation float32) {
		game.sim.SpawnGenome(genome, generation)
	})
	game.savedGenomesPanel = ui.savedGenomesPanel

	// Wire load-from-saved: opens the saved genomes panel in load mode, result goes into the editor.
	ui.genomeEditor.onLoadSaved = func() {
		ui.savedGenomesPanel.OpenForLoad(func(g *simulation.Genome, _ float32) {
			ui.genomeEditor.LoadGenome(g)
			ui.savedGenomesPanel.visible = false
		})
	}

	return ui
}

// AnySliderDragging reports whether any non-menubar slider is currently being dragged.
func (ui *UserInterface) AnySliderDragging() bool {
	return (ui.foodDropdown != nil && ui.foodDropdown.AnyDragging()) ||
		(ui.climateDropdown != nil && ui.climateDropdown.AnyDragging()) ||
		(ui.spawnDropdown != nil && ui.spawnDropdown.AnyDragging()) ||
		(ui.saveGameDropdown != nil && ui.saveGameDropdown.AnyDragging())
}

// HandleClick processes a mouse-down event; returns true if consumed.
func (ui *UserInterface) HandleClick(mx, my int) bool {
	if ui.menuBar.HandleClick(mx, my) {
		return true
	}
	if ui.foodDropdown != nil && ui.foodDropdown.HandleClick(mx, my) {
		return true
	}
	if ui.climateDropdown != nil && ui.climateDropdown.HandleClick(mx, my) {
		return true
	}
	if ui.spawnDropdown != nil && ui.spawnDropdown.HandleClick(mx, my) {
		return true
	}
	if ui.saveGameDropdown != nil && ui.saveGameDropdown.HandleClick(mx, my) {
		return true
	}
	// Save name input field
	if ui.currentSaveInput != nil && ui.currentSaveInput.Contains(float32(mx), float32(my)) {
		ui.saveNameFocused = true
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

// HandleContinuousInput forwards drag events to the menu bar and dropdown sliders.
func (ui *UserInterface) HandleContinuousInput() {
	mx, _ := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		ui.menuBar.HandleDrag(mx)
		if ui.foodDropdown != nil {
			ui.foodDropdown.HandleDrag(mx)
		}
		if ui.climateDropdown != nil {
			ui.climateDropdown.HandleDrag(mx)
		}
		if ui.spawnDropdown != nil {
			ui.spawnDropdown.HandleDrag(mx)
		}
		if ui.saveGameDropdown != nil {
			ui.saveGameDropdown.HandleDrag(mx)
		}
	} else {
		ui.menuBar.HandleRelease()
		if ui.foodDropdown != nil {
			ui.foodDropdown.HandleRelease()
		}
		if ui.climateDropdown != nil {
			ui.climateDropdown.HandleRelease()
		}
		if ui.spawnDropdown != nil {
			ui.spawnDropdown.HandleRelease()
		}
		if ui.saveGameDropdown != nil {
			ui.saveGameDropdown.HandleRelease()
		}
	}
}

// HandleSaveNameKeyInput processes keyboard input for creature and game save name fields.
func (ui *UserInterface) HandleSaveNameKeyInput() {
	if ui.saveGameNameFocused {
		runes := ebiten.AppendInputChars([]rune(ui.saveGameName))
		ui.saveGameName = string(runes)
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len([]rune(ui.saveGameName)) > 0 {
			r := []rune(ui.saveGameName)
			ui.saveGameName = string(r[:len(r)-1])
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			ui.saveGameNameFocused = false
		}
		return
	}
	if !ui.saveNameFocused {
		return
	}
	runes := ebiten.AppendInputChars([]rune(ui.saveCreatureName))
	ui.saveCreatureName = string(runes)
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len([]rune(ui.saveCreatureName)) > 0 {
		r := []rune(ui.saveCreatureName)
		ui.saveCreatureName = string(r[:len(r)-1])
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		ui.saveNameFocused = false
	}
}

// Draw renders the entire HUD layer.
func (ui *UserInterface) Draw(screen *ebiten.Image, state UIDrawState, game *Game) {
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()

	// Update creature detail and NN slots; build genome panel for side-by-side draw.
	var genomePanel *components.Panel
	if state.selectedID != -1 {
		if detail, ok := ui.sim.CreatureDetail(state.selectedID); ok {
			detailPanel := ui.buildDetailPanel(detail, state.selectedID, game)
			ui.leftStack.Set(ui.detailIdx, detailPanel, true)

			genomePanel = ui.buildGenomePanel(detail)

			nnGraph := &NeuralNetGraph{Font: ui.font, Data: detail}
			ui.leftStack.Set(ui.nnIdx, nnGraph, true)
		} else {
			ui.leftStack.Set(ui.detailIdx, nil, false)
			ui.leftStack.Set(ui.nnIdx, nil, false)
			ui.currentSaveBtn = nil
			ui.currentEditBtn = nil
			ui.currentSaveInput = nil
		}
	} else {
		ui.leftStack.Set(ui.detailIdx, nil, false)
		ui.leftStack.Set(ui.nnIdx, nil, false)
		ui.currentSaveBtn = nil
		ui.currentEditBtn = nil
		ui.currentSaveInput = nil
	}

	ui.leftStack.Set(ui.histStatsIdx, ui.buildHistStatsPanel(), true)

	if ui.speedLabel != nil {
		ui.speedLabel.Label = fmt.Sprintf("%dx", state.simStepsPerTick)
	}
	if ui.foodBtn != nil {
		ui.foodBtn.Active = ui.foodDropdown != nil && ui.foodDropdown.IsOpen()
	}
	if ui.climateBtn != nil {
		ui.climateBtn.Active = ui.climateDropdown != nil && ui.climateDropdown.IsOpen()
	}
	if ui.spawnBtn != nil {
		ui.spawnBtn.Active = ui.spawnDropdown != nil && ui.spawnDropdown.IsOpen()
	}
	if ui.saveGameBtn != nil {
		ui.saveGameBtn.Active = ui.saveGameDropdown != nil && ui.saveGameDropdown.IsOpen()
	}
	ui.menuBar.Draw(screen)
	ui.leftStack.Draw(screen)
	// Dropdowns drawn after leftStack so they render on top of all panels.
	if ui.foodDropdown != nil {
		ui.foodDropdown.Draw(screen)
	}
	if ui.climateDropdown != nil {
		ui.climateDropdown.Draw(screen)
	}
	if ui.spawnDropdown != nil {
		ui.spawnDropdown.Draw(screen)
	}
	if ui.saveGameDropdown != nil {
		ui.saveGameDropdown.Draw(screen)
	}

	// Genome panel — drawn to the right of the detail panel.
	if genomePanel != nil {
		genomeX := leftStackX + detailPanelW + leftStackSpacing
		genomeY := ui.leftStack.SlotY(ui.detailIdx)
		genomePanel.Draw(screen, genomeX, genomeY)
	}

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
		vector.StrokeLine(screen, cx-arm, cy, cx+arm, cy, 1.5, ColorCrosshair, false)
		vector.StrokeLine(screen, cx, cy-arm, cx, cy+arm, 1.5, ColorCrosshair, false)
		vector.StrokeCircle(screen, cx, cy, arm*0.6, 1, ColorCrosshairCircle, false)
	}

	// Top-right stats — start below the menu bar using font metrics for spacing.
	if ui.font != nil {
		m := ui.font.Metrics()
		lineH := int(m.HAscent+m.HDescent) + 6
		x := sw - 210
		y := int(menuBarH) + 6
		drawText(screen, fmt.Sprintf("Population: %d", ui.sim.PopulationCount()), ui.font, x, y, color.White)
		y += lineH
		drawText(screen, fmt.Sprintf("Foliage: %d", ui.sim.FoliageCount()), ui.font, x, y, color.White)
		y += lineH
		drawText(screen, fmt.Sprintf("Fungi: %d", ui.sim.FungiCount()), ui.font, x, y, color.White)
		y += lineH
		drawText(screen, fmt.Sprintf("Avg Age: %.0f", ui.sim.AverageAge()), ui.font, x, y, color.White)
		y += lineH
		drawText(screen, fmt.Sprintf("Avg Gen: %.1f", ui.sim.AverageGeneration()), ui.font, x, y, color.White)
		if state.tickDuration > 0 {
			y += lineH
			simRate := float64(state.simStepsPerTick) / state.tickDuration.Seconds()
			drawText(screen, fmt.Sprintf("Sim: %.0f/s (%dx)", simRate, state.simStepsPerTick), ui.font, x, y, color.White)
		}
	}

	// Save feedback (creature genome)
	if ui.saveFeedback != "" && time.Since(ui.saveFeedbackAt) < 2*time.Second {
		if ui.font != nil {
			drawText(screen, ui.saveFeedback, ui.font, sw/2-30, sh-40, ColorSaveFeedback)
		}
	}
	// Save game feedback
	if ui.saveGameFeedback != "" && time.Since(ui.saveGameFeedbackAt) < 2*time.Second {
		if ui.font != nil {
			drawText(screen, ui.saveGameFeedback, ui.font, sw/2-40, sh-60, ColorSaveFeedback)
		}
	}

	// Modals — savedGenomesPanel always drawn on top of genomeEditor
	ui.genomeEditor.Draw(screen, ui.font)
	ui.savedGenomesPanel.Draw(screen, ui.font)
}

// buildDetailPanel constructs a Panel containing all creature stat rows.
// It is rebuilt each frame so data is always fresh.
func (ui *UserInterface) buildDetailPanel(d simulation.CreatureDetailView, creatureID int, game *Game) *components.Panel {
	if creatureID != ui.saveCreatureID {
		ui.saveCreatureID = creatureID
		ui.saveCreatureName = ""
		ui.saveNameFocused = false
	}

	innerW := detailPanelW - detailPad*2

	p := &components.Panel{
		W:         detailPanelW,
		Padding:   detailPad,
		Spacing:   detailSpacing,
		BaseColor: ColorDetailPanelBG,
		Border:    ColorDetailPanelBorder,
	}

	// Title
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Creature #%d", d.ID),
		Font:  ui.font,
		Color: ColorLabelPrimary,
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
		MaxColor: ColorEnergyHigh,
		MinColor: ColorEnergyLow,
		Width:    innerW,
	})

	// Generation / Age
	p.Add(&components.Label{Text: fmt.Sprintf("Generation: %.2f (Tier: %d)", d.Generation, d.Tier), Font: ui.font, Color: ColorLabelInfo})
	juvenileStr := "Adult"
	if d.IsJuvenile {
		juvenileStr = fmt.Sprintf("Juvenile(%d)", d.JuvenilePeriod-d.Age)
	}
	p.Add(&components.Label{Text: fmt.Sprintf("Age: %d, %s", d.Age, juvenileStr), Font: ui.font, Color: color.White})

	// Last action
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Actions: %s", d.LastAction),
		Font:  ui.font,
		Color: ColorLabelGreen,
	})

	// Mass
	p.Add(&components.Label{Text: fmt.Sprintf("Mass: %.0f", d.CurrentMass), Font: ui.font, Color: color.White})

	// Stomach
	p.Add(&components.Label{Text: fmt.Sprintf("Stomach: %.0f/%.0f", d.Stomach, d.StomachCapacity), Font: ui.font, Color: color.White})
	p.Add(&components.EnergyBar{
		Value:    float32(d.Stomach),
		Max:      float32(d.StomachCapacity),
		MaxColor: ColorEnergyHigh,
		MinColor: ColorEnergyLow,
		Width:    innerW,
	})

	// Digestion Efficiency (proportional)
	p.AddRow(
		&components.Label{Text: fmt.Sprintf("%.0f%%", d.FoliageEfficiency*100), Font: ui.font, Color: ColorFoliage},
		&components.Label{Text: fmt.Sprintf("%.0f%%", d.FungiEfficiency*100), Font: ui.font, Color: ColorFungi},
		&components.Label{Text: fmt.Sprintf("%.0f%%", d.MeatEfficiency*100), Font: ui.font, Color: ColorMeat},
	)
	p.Add(&components.ProportionBar{
		Segments: []components.ProportionSegment{
			{Value: d.FoliageEfficiency, Color: ColorFoliage},
			{Value: d.FungiEfficiency, Color: ColorFungi},
			{Value: d.MeatEfficiency, Color: ColorMeat},
		},
		Width: innerW,
	})

	// Dopamine
	p.Add(&components.Label{Text: fmt.Sprintf("Dopamine: %.02f", d.Dopamine), Font: ui.font, Color: color.White})
	p.Add(&components.EnergyBar{
		Value:    d.Dopamine,
		Max:      float32(1.2),
		MaxColor: ColorDopamineHigh,
		MinColor: ColorDopamineLow,
		Width:    innerW,
		Centered: true,
	})

	// Responsiveness
	p.Add(&components.Label{Text: fmt.Sprintf("Responsiveness: %.02f", d.Responsiveness), Font: ui.font, Color: color.White})
	p.Add(&components.EnergyBar{
		Value:    d.Responsiveness,
		Max:      float32(1),
		MaxColor: ColorResponHigh,
		MinColor: ColorResponLow,
		Width:    innerW,
	})

	// Sight
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Sight: %.f  FOV: %.f°", d.VisionRadius, d.FieldOfView),
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
	reproColor := color.Color(ColorReproAsexual)
	if d.ReproductionType != 0 {
		reproStr = "Sexual"
		reproColor = ColorReproSexual
	}
	p.Add(&components.Label{
		Text:  fmt.Sprintf("Reproduction: %s", reproStr),
		Font:  ui.font,
		Color: reproColor,
	})

	// Phenotype chart
	p.Add(&PhenotypeChart{Font: ui.font, Data: d})

	// Save name input
	saveInput := &components.TextInputField{
		W:           innerW,
		H:           22,
		Text:        ui.saveCreatureName,
		Focused:     ui.saveNameFocused,
		Placeholder: "(save name...)",
		Font:        ui.font,
	}
	p.Add(saveInput)
	ui.currentSaveInput = saveInput

	// Save button
	saveBtn := &components.Button{
		W: innerW, H: 22,
		Label:      "Save Genome",
		Color:      ColorBtnSave,
		LabelColor: color.White,
		Font:       ui.font,
	}
	saveBtn.OnClick = func() {
		if err := game.sim.SaveCreature(creatureID, ui.saveCreatureName); err != nil {
			ui.saveFeedback = "Save failed"
		} else {
			ui.saveFeedback = "Saved!"
			ui.saveCreatureName = ""
			ui.saveNameFocused = false
		}
		ui.saveFeedbackAt = time.Now()
	}
	p.Add(saveBtn)
	ui.currentSaveBtn = saveBtn

	// Edit button
	editBtn := &components.Button{
		W: innerW, H: 22,
		Label:      "Edit Genome",
		Color:      ColorBtnEdit,
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

// buildGenomePanel constructs a compact Panel showing all raw genome trait bytes.
// Uses smallFont and tight spacing to keep it smaller than the detail panel.
// It is rebuilt each frame so values stay current.
func (ui *UserInterface) buildGenomePanel(d simulation.CreatureDetailView) *components.Panel {
	const (
		genomePad     = float32(5)
		genomeSpacing = float32(2)
	)
	p := &components.Panel{
		W:         detailPanelW,
		Padding:   genomePad,
		Spacing:   genomeSpacing,
		BaseColor: ColorDetailPanelBG,
		Border:    ColorDetailPanelBorder,
	}
	p.Add(&components.Label{
		Text:  "Genome",
		Font:  ui.font,
		Color: ColorLabelPrimary,
	})
	g := d.Genome
	barW := detailPanelW - genomePad*2
	type trait struct {
		name   string
		val    byte
		binary bool
	}
	traits := []trait{
		{"OscPeriod", g.OscPeriod, false},
		{"VisionRadius", g.VisionRadius, false},
		{"FieldOfView", g.FieldOfView, false},
		{"Responsiveness", g.Responsiveness, false},
		{"MutationRate", g.MutationRate, false},
		{"BodyMass", g.BodyMass, false},
		{"SurvivalMass", g.SurvivalMass, false},
		{"ReproductionType", g.ReproductionType, true},
		{"CognitiveBreadth", g.CognitiveBreadth, false},
		{"SynapticDensity", g.SynapticDensity, false},
		{"JuvenilePeriod", g.JuvenilePeriod, false},
		{"MetabolicRate", g.MetabolicRate, false},
		{"StomachSize", g.StomachSize, false},
		{"Neuroplasticity", g.Neuroplasticity, false},
		{"LearningThreshold", g.LearningThreshold, false},
		{"MassSplitRatio", g.MassSplitRatio, false},
		{"FoliageDigestion", g.FoliageDigestionEfficiency, false},
		{"FungiDigestion", g.FungiDigestionEfficiency, false},
		{"MeatDigestion", g.MeatDigestionEfficiency, false},
	}
	for _, t := range traits {
		p.Add(&components.GenomeBar{
			Name:   t.name,
			Val:    t.val,
			Font:   ui.smallFont,
			W:      barW,
			Binary: t.binary,
		})
	}
	return p
}

// buildHistStatsPanel constructs a compact Panel showing current world stats.
// Rebuilt each frame so values stay current.
func (ui *UserInterface) buildHistStatsPanel() *components.Panel {
	p := &components.Panel{
		W:         histGraphW,
		Padding:   histGraphPad,
		Spacing:   2,
		BaseColor: ColorStatPanelBG,
		Border:    ColorStatPanelBorder,
	}
	p.Add(&components.Label{Text: fmt.Sprintf("Pop: %d", ui.sim.PopulationCount()), Font: ui.font, Color: ColorInfoBlue})
	p.Add(&components.Label{Text: fmt.Sprintf("Foliage: %.0f", ui.sim.FoliageEnergy()), Font: ui.font, Color: ColorLabelGreen})
	p.Add(&components.Label{Text: fmt.Sprintf("Fungi: %.0f", ui.sim.FungiEnergy()), Font: ui.font, Color: ColorFungi})
	p.Add(&components.Label{Text: fmt.Sprintf("Meat: %.0f", ui.sim.MeatEnergy()), Font: ui.font, Color: ColorLabelMeatRed})
	p.Add(&components.Label{Text: fmt.Sprintf("Energy: %.2f%%", ui.sim.TotalEnergy()/ui.sim.TargetEnergy()*100), Font: ui.font, Color: ColorLabelTargetE})
	return p
}
