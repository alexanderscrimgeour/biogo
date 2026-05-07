package ui

import (
	"biogo/v2/simulation"
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
	CreatureViews() []simulation.CreatureView
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

const (
	fovBtnX = 10
	fovBtnY = 10
	fovBtnW = 70
	fovBtnH = 24

	pauseBtnX = 90
	pauseBtnY = 10
	pauseBtnW = 80
	pauseBtnH = 24

	newGameBtnX = 180
	newGameBtnY = 10
	newGameBtnW = 90
	newGameBtnH = 24

	themeBtnX = 280
	themeBtnY = 10
	themeBtnW = 100
	themeBtnH = 24

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
	spawnMutRateMax = 0.05

	// Creature detail panel
	detailPanelX   = 10
	detailPanelY   = 44
	detailPanelW   = 210
	detailPanelH   = 215
	detailTpad     = 8
	detailSaveBtnH = 22
)

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
	showFOV            bool
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
	}

	wallThickness := 10.0
	cx := float64(sim.GridWidth()) / 2
	bs := float64(BlockSize)

	g.renderGrid.AddLine(
		(cx-wallThickness/2)*bs, float64(sim.GridHeight())/4*bs,
		(cx+wallThickness/2)*bs, (float64(sim.GridHeight())/4+float64(sim.GridHeight())/2)*bs,
	)
	return g
}

func (g *Game) Update() error {
	mx, my := ebiten.CursorPosition()
	now := time.Now()

	if !g.paused {
		prevByID := make(map[int][2]float64, len(g.animByID))
		for id, anim := range g.animByID {
			prevByID[id] = [2]float64{anim.curX, anim.curY}
		}

		g.sim.Update()

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

	// UI Interaction
	if g.spawnMutDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.applySpawnMutSlider(mx)
		} else {
			g.spawnMutDragging = false
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		clickedButton := false
		if mx >= spawnMutSliderX && mx < spawnMutSliderX+spawnMutSliderW &&
			my >= spawnMutSliderY && my < spawnMutSliderY+spawnMutSliderH {
			g.spawnMutDragging = true
			g.applySpawnMutSlider(mx)
			clickedButton = true
		}
		if mx >= fovBtnX && mx < fovBtnX+fovBtnW && my >= fovBtnY && my < fovBtnY+fovBtnH {
			g.showFOV = !g.showFOV
			clickedButton = true
		}
		if mx >= pauseBtnX && mx < pauseBtnX+pauseBtnW && my >= pauseBtnY && my < pauseBtnY+pauseBtnH {
			g.paused = !g.paused
			clickedButton = true
		}
		if mx >= newGameBtnX && mx < newGameBtnX+newGameBtnW && my >= newGameBtnY && my < newGameBtnY+newGameBtnH {
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
			clickedButton = true
		}
		if mx >= themeBtnX && mx < themeBtnX+themeBtnW && my >= themeBtnY && my < themeBtnY+themeBtnH {
			g.isDarkBackground = !g.isDarkBackground
			clickedButton = true
		}
		if g.selectedCreatureID != -1 {
			saveBtnX, saveBtnY := detailPanelX+detailTpad, detailPanelY+detailPanelH-detailTpad-detailSaveBtnH
			if mx >= saveBtnX && mx < saveBtnX+(detailPanelW-detailTpad*2) && my >= saveBtnY && my < saveBtnY+detailSaveBtnH {
				if err := g.sim.SaveCreature(g.selectedCreatureID); err != nil {
					g.saveFeedback = "Save failed"
				} else {
					g.saveFeedback = "Saved!"
				}
				g.saveFeedbackAt = time.Now()
				clickedButton = true
			}
		}
		if !clickedButton {
			g.trySelectCreature(mx, my)
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if g.isDarkBackground {
		screen.Fill(color.RGBA{5, 5, 10, 255})
	} else {
		screen.Fill(color.RGBA{15, 15, 15, 255})
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

	if g.showFOV {
		g.drawFOVCones(screen, g.sim.CreatureViews(), t)
	}
	if g.selectedCreatureID != -1 {
		g.drawSelectionHighlight(screen)
		if detail, ok := g.sim.CreatureDetail(g.selectedCreatureID); ok {
			g.drawCreatureDetail(screen, detail)
		} else {
			g.selectedCreatureID = -1
		}
	}

	g.drawFOVButton(screen)
	g.drawPauseButton(screen)
	g.drawNewGameButton(screen)
	g.drawThemeButton(screen)
	g.drawSpawnMutSlider(screen)

	g.addStatLine(screen, "Population", fmt.Sprintf("%d", g.sim.PopulationCount()), 1)
	g.addStatLine(screen, "Food", fmt.Sprintf("%d", g.sim.FoodCount()), 2)
	g.addStatLine(screen, "Avg Age", fmt.Sprintf("%.0f", g.sim.AverageAge()), 3)
	if g.tickDuration > 0 {
		tickRate := 1.0 / g.tickDuration.Seconds()
		g.addStatLine(screen, "Tick Rate", fmt.Sprintf("%.0f/s", tickRate), 4)
	}
}

func (g *Game) drawPauseButton(screen *ebiten.Image) {
	var bg color.RGBA
	label := "Pause"
	if g.paused {
		bg = color.RGBA{50, 140, 80, 220}
		label = "Resume"
	} else {
		bg = color.RGBA{80, 80, 50, 220}
	}
	vector.DrawFilledRect(screen, pauseBtnX, pauseBtnY, pauseBtnW, pauseBtnH, bg, false)
	text.Draw(screen, label, g.statFont, pauseBtnX+10, pauseBtnY+17, color.White)
}

func (g *Game) drawNewGameButton(screen *ebiten.Image) {
	bg := color.RGBA{80, 50, 50, 220}
	vector.DrawFilledRect(screen, newGameBtnX, newGameBtnY, newGameBtnW, newGameBtnH, bg, false)
	text.Draw(screen, "New Game", g.statFont, newGameBtnX+5, newGameBtnY+17, color.White)
}

func (g *Game) drawThemeButton(screen *ebiten.Image) {
	bg := color.RGBA{60, 60, 60, 220}
	vector.DrawFilledRect(screen, themeBtnX, themeBtnY, themeBtnW, themeBtnH, bg, false)
	text.Draw(screen, "Theme", g.statFont, themeBtnX+12, themeBtnY+17, color.White)
}

func (g *Game) drawFOVButton(screen *ebiten.Image) {
	bg := color.RGBA{50, 50, 70, 220}
	if g.showFOV {
		bg = color.RGBA{80, 120, 200, 220}
	}
	vector.DrawFilledRect(screen, fovBtnX, fovBtnY, fovBtnW, fovBtnH, bg, false)
	text.Draw(screen, "FOV", g.statFont, fovBtnX+8, fovBtnY+17, color.White)
}

func (g *Game) drawFOVCones(screen *ebiten.Image, views []simulation.CreatureView, t float64) {
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

// drawCreatureDetail renders the inspector panel for the selected creature.
func (g *Game) drawCreatureDetail(screen *ebiten.Image, d simulation.CreatureDetailView) {
	const tpad = detailTpad
	px := float32(detailPanelX)
	py := float32(detailPanelY)
	pw := float32(detailPanelW)
	ph := float32(detailPanelH)

	vector.DrawFilledRect(screen, px, py, pw, ph, color.RGBA{8, 10, 22, 215}, false)
	vector.StrokeRect(screen, px, py, pw, ph, 1, color.RGBA{90, 90, 150, 255}, false)

	tx := int(px) + tpad
	ty := int(py) + 16

	// Header
	text.Draw(screen, fmt.Sprintf("Creature #%d", d.ID), g.statFont, tx, ty, color.RGBA{R: 255, G: 220, B: 80, A: 255})
	ty += 22

	// Energy label then bar
	frac := d.Energy / float32(d.MaxEnergy)
	if frac < 0 {
		frac = 0
	} else if frac > 1 {
		frac = 1
	}
	text.Draw(screen, fmt.Sprintf("Energy  %.0f / %.0f", d.Energy, float32(d.MaxEnergy)), g.statFont, tx, ty, color.White)
	barX := px + float32(tpad)
	barW := pw - float32(tpad)*2
	barY := float32(ty + 2)
	vector.DrawFilledRect(screen, barX, barY, barW, 6, color.RGBA{35, 35, 35, 255}, false)
	vector.DrawFilledRect(screen, barX, barY, barW*frac, 6, creatureEnergyBarColor(frac), false)
	ty += 26

	// Age
	ageStatus := "Adult"
	if d.IsJuvenile {
		ageStatus = fmt.Sprintf("Juv (%d left)", d.JuvenilePeriod-d.Age)
	}
	text.Draw(screen, fmt.Sprintf("Age  %d  %s", d.Age, ageStatus), g.statFont, tx, ty, color.White)
	ty += 18

	// Action
	text.Draw(screen, fmt.Sprintf("Action  %s", d.LastAction), g.statFont, tx, ty, color.RGBA{R: 160, G: 220, B: 160, A: 255})
	ty += 20

	// Divider
	vector.DrawFilledRect(screen, px+4, float32(ty)-6, pw-8, 1, color.RGBA{70, 70, 100, 255}, false)

	// Mass
	text.Draw(screen, fmt.Sprintf("Mass  %.0f / %d", d.CurrentMass, d.AdultMass), g.statFont, tx, ty, color.White)
	ty += 18

	// Sight + FOV
	text.Draw(screen, fmt.Sprintf("Sight %d  FOV %d°", d.SightDistance, d.FieldOfView), g.statFont, tx, ty, color.White)
	ty += 18

	// Brain
	text.Draw(screen, fmt.Sprintf("Layers %d  Genes %d", d.NeuronCount, d.BrainLength), g.statFont, tx, ty, color.White)
	ty += 18

	// Mutation
	text.Draw(screen, fmt.Sprintf("Mutation  %.2f%%", d.MutationPct), g.statFont, tx, ty, color.White)

	// Save button at the bottom of the panel.
	saveBtnY := float32(detailPanelY + detailPanelH - detailTpad - detailSaveBtnH)
	vector.DrawFilledRect(screen, px+float32(tpad), saveBtnY, pw-float32(tpad)*2, float32(detailSaveBtnH),
		color.RGBA{R: 40, G: 100, B: 60, A: 220}, false)
	text.Draw(screen, "Save", g.statFont, tx+int(pw/2)-16, int(saveBtnY)+17, color.White)

	// Feedback from the last save attempt.
	if !g.saveFeedbackAt.IsZero() && time.Since(g.saveFeedbackAt) < 3*time.Second {
		text.Draw(screen, g.saveFeedback, g.statFont, tx, int(saveBtnY)+detailSaveBtnH+14, color.White)
	}
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

func creatureEnergyBarColor(frac float32) color.RGBA {
	if frac > 0.65 {
		return color.RGBA{55, 185, 55, 255}
	}
	if frac > 0.33 {
		return color.RGBA{190, 175, 45, 255}
	}
	return color.RGBA{190, 55, 55, 255}
}
