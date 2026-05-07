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

// SimulationState is the interface the UI requires from a simulation.
// Keeping this narrow prevents the UI from reaching into simulation internals.
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
<<<<<<< Updated upstream
	CreatureMinSize() byte
	CreatureMaxSize() byte
=======
	CreatureMinMass() byte
	CreatureMaxMass() byte
	SaveCreature(id int) error
	Reset()
	CreatureDetail(id int) (simulation.CreatureDetailView, bool)
	SetSpawnMutationRate(rate float32)
>>>>>>> Stashed changes
}

var foodColor = color.RGBA{R: 50, G: 200, B: 60, A: 255}

const (
	fovBtnX = 10
	fovBtnY = 10
	fovBtnW = 70
	fovBtnH = 24
<<<<<<< Updated upstream
=======

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

	// Spawn mutation rate slider (positioned to the right of the theme button).
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

	// Creature detail panel dimensions (also used for hit-detection in Update).
	detailPanelX   = 10
	detailPanelY   = 44
	detailPanelW   = 210
	detailPanelH   = 215
	detailTpad     = 8
	detailSaveBtnH = 22
>>>>>>> Stashed changes
)

// creatureAnim holds the screen-space state of a creature across one
// simulation tick so Draw() can lerp positions and render triangles.
type creatureAnim struct {
	prevX, prevY float64 // position before the last sim step (pixels)
	curX, curY   float64 // position after the last sim step (pixels)
	r, g, b, a   uint8
	dirX, dirY   int
	size         byte
}

type Game struct {
<<<<<<< Updated upstream
	sim             SimulationState
	renderGrid      *RenderGrid
	foodBlobsByKey  map[string]*Blob
	corpseBlobsByID map[int]*Blob
	statFont        font.Face
	showFOV         bool
	whiteImage      *ebiten.Image
	animByID        map[int]*creatureAnim
	lastTickTime    time.Time
	tickDuration    time.Duration
	minCreatureSize byte
	maxCreatureSize byte
=======
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
	minCreatureSize    byte
	maxCreatureSize    byte
	saveFeedback       string
	saveFeedbackAt     time.Time
	isDarkBackground   bool
	selectedCreatureID int
	paused             bool
	spawnMutRate       float32
	spawnMutDragging   bool
>>>>>>> Stashed changes
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
		sim:             sim,
		renderGrid:      NewRenderGrid(0, 0, BlockSize),
		foodBlobsByKey:  make(map[string]*Blob),
		corpseBlobsByID: make(map[int]*Blob),
		statFont:        statFont,
		whiteImage:      wImg,
		animByID:        make(map[int]*creatureAnim),
		minCreatureSize:    sim.CreatureMinMass(),
		maxCreatureSize:    sim.CreatureMaxMass(),
		selectedCreatureID: -1,
		spawnMutRate:       0.01,
	}

	// Dirty fix, fix with rendering from sim.walls
	width := 5
	centerX := sim.GridWidth() / 2
	centerY := sim.GridHeight() / 2 // Added centerY

	// 1. Vertical Bar
	minX_V := centerX - width/2
	maxX_V := centerX + width/2
	minY_V := sim.GridHeight() / 4
	maxY_V := minY_V + sim.GridHeight()/2
	g.renderGrid.AddLine(float64(minX_V*BlockSize), float64(minY_V*BlockSize), float64(maxX_V*BlockSize), float64(maxY_V*BlockSize))

	// 2. Horizontal Bar
	minX_H := sim.GridWidth() / 4
	maxX_H := minX_H + sim.GridWidth()/2
	minY_H := centerY - width/2
	maxY_H := centerY + width/2
	g.renderGrid.AddLine(float64(minX_H*BlockSize), float64(minY_H*BlockSize), float64(maxX_H*BlockSize), float64(maxY_H*BlockSize))
	return g
}

func (g *Game) Update() error {
	if !g.paused {
		// Snapshot target positions before the sim step for interpolation.
		prevByID := make(map[int][2]float64, len(g.animByID))
		for id, anim := range g.animByID {
			prevByID[id] = [2]float64{anim.curX, anim.curY}
		}

		g.sim.Update()

		// Reconcile creature anims with current simulation state.
		views := g.sim.CreatureViews()
		currentIDs := make(map[int]bool, len(views))
		for _, cv := range views {
			currentIDs[cv.ID] = true
			screenX := float64(cv.X * BlockSize)
			screenY := float64(cv.Y * BlockSize)
			if anim, ok := g.animByID[cv.ID]; ok {
				if prev, ok := prevByID[cv.ID]; ok {
					anim.prevX, anim.prevY = prev[0], prev[1]
				} else {
					anim.prevX, anim.prevY = screenX, screenY
				}
				anim.curX, anim.curY = screenX, screenY
				anim.r, anim.g, anim.b, anim.a = cv.R, cv.G, cv.B, cv.A
				anim.dirX, anim.dirY = cv.DirX, cv.DirY
				anim.size = cv.Mass
			} else {
				g.animByID[cv.ID] = &creatureAnim{
					prevX: screenX, prevY: screenY,
					curX: screenX, curY: screenY,
					r: cv.R, g: cv.G, b: cv.B, a: cv.A,
					dirX: cv.DirX, dirY: cv.DirY,
					size: cv.Mass,
				}
			}
		}
		for id := range g.animByID {
			if !currentIDs[id] {
				delete(g.animByID, id)
			}
		}

		// Reconcile food blobs.
		foodViews := g.sim.FoodViews()
		currentFood := make(map[string]bool, len(foodViews))
		for _, fv := range foodViews {
			key := foodKey(fv.X, fv.Y)
			currentFood[key] = true
			if _, ok := g.foodBlobsByKey[key]; !ok {
				blob := g.renderGrid.AddFoodBlob(BlockSize, foodColor)
				blob.Translate(float64(fv.X*BlockSize), float64(fv.Y*BlockSize))
				g.foodBlobsByKey[key] = blob
			}
		}
		for key, blob := range g.foodBlobsByKey {
			if !currentFood[key] {
				g.renderGrid.RemoveFoodBlob(blob)
				delete(g.foodBlobsByKey, key)
			}
		}

		// Reconcile corpse blobs.
		corpseViews := g.sim.CorpseViews()
		currentCorpses := make(map[int]bool, len(corpseViews))
		for _, cv := range corpseViews {
			currentCorpses[cv.ID] = true
			alpha := uint8(cv.EnergyFraction * 220)
			corpseColor := color.RGBA{R: 120, G: 60, B: 20, A: alpha}
			if blob, ok := g.corpseBlobsByID[cv.ID]; ok {
				blob.Move(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
				blob.SetColor(corpseColor)
			} else {
				blob := g.renderGrid.AddBlob(BlockSize, corpseColor)
				blob.Translate(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
				g.corpseBlobsByID[cv.ID] = blob
			}
		}
		for id, blob := range g.corpseBlobsByID {
			if !currentCorpses[id] {
				g.renderGrid.RemoveBlob(blob)
				delete(g.corpseBlobsByID, id)
			}
		}

		now := time.Now()
		if !g.lastTickTime.IsZero() {
			g.tickDuration = now.Sub(g.lastTickTime)
		}
		g.lastTickTime = now
	}

	mx, my := ebiten.CursorPosition()

	// Slider drag: update while button held, clear when released.
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
<<<<<<< Updated upstream
	}
=======
		if mx >= pauseBtnX && mx < pauseBtnX+pauseBtnW && my >= pauseBtnY && my < pauseBtnY+pauseBtnH {
			g.paused = !g.paused
			clickedButton = true
		}
		if mx >= newGameBtnX && mx < newGameBtnX+newGameBtnW && my >= newGameBtnY && my < newGameBtnY+newGameBtnH {
			g.sim.Reset()
			for _, blob := range g.foodBlobsByKey {
				g.renderGrid.RemoveFoodBlob(blob)
			}
			g.foodBlobsByKey = make(map[string]*Blob)
			for _, blob := range g.corpseBlobsByID {
				g.renderGrid.RemoveBlob(blob)
			}
			g.corpseBlobsByID = make(map[int]*Blob)
			g.animByID = make(map[int]*creatureAnim)
			g.lastTickTime = time.Time{}
			g.selectedCreatureID = -1
			g.paused = false
			clickedButton = true
		}
		if mx >= themeBtnX && mx < themeBtnX+themeBtnW && my >= themeBtnY && my < themeBtnY+themeBtnH {
			g.isDarkBackground = !g.isDarkBackground
			clickedButton = true
		}
		// Save button inside the creature detail panel.
		if g.selectedCreatureID != -1 {
			saveBtnX := detailPanelX + detailTpad
			saveBtnY := detailPanelY + detailPanelH - detailTpad - detailSaveBtnH
			saveBtnW := detailPanelW - detailTpad*2
			if mx >= saveBtnX && mx < saveBtnX+saveBtnW && my >= saveBtnY && my < saveBtnY+detailSaveBtnH {
				if err := g.sim.SaveCreature(g.selectedCreatureID); err != nil {
					g.saveFeedback = "Save failed"
				} else {
					g.saveFeedback = "Saved!"
				}
				g.saveFeedbackAt = time.Now()
				clickedButton = true
			}
		}
>>>>>>> Stashed changes

		if !clickedButton {
			g.trySelectCreature(mx, my)
		}
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 15, 255})

	// Static world elements (walls, food).
	g.renderGrid.DrawBackground(screen)

	// How far through the current simulation tick are we?
	// t=0 → just started the tick (draw at prev position)
	// t=1 → tick is complete   (draw at current position)
	t := 1.0
	if g.tickDuration > 0 {
		elapsed := time.Since(g.lastTickTime)
		t = float64(elapsed) / float64(g.tickDuration)
		if t > 1 {
			t = 1
		}
	}

	// Corpses sit still; draw them at their blob's stored position.
	for _, blob := range g.corpseBlobsByID {
		blob.Draw(screen)
	}

	// Creatures are triangles pointing in their travel direction.
	// Circumradius scales linearly from BlockSize (MinSize) to 3*BlockSize (MaxSize).
	var creatureVs []ebiten.Vertex
	var creatureIs []uint16
	sizeRange := float64(g.maxCreatureSize) - float64(g.minCreatureSize)
	half := float64(BlockSize) / 2
	for _, anim := range g.animByID {
		lerpX := anim.prevX + (anim.curX-anim.prevX)*t
		lerpY := anim.prevY + (anim.curY-anim.prevY)*t
		cx := float32(lerpX + half)
		cy := float32(lerpY + half)

		var r float64
		if sizeRange > 0 {
			sizeT := (float64(anim.size) - float64(g.minCreatureSize)) / sizeRange
			if sizeT < 0 {
				sizeT = 0
			} else if sizeT > 1 {
				sizeT = 1
			}
			r = float64(BlockSize) + sizeT*float64(5*BlockSize)
		} else {
			r = float64(BlockSize)
		}

		var angle float64
		if anim.dirX == 0 && anim.dirY == 0 {
			angle = -math.Pi / 2
		} else {
			angle = math.Atan2(float64(anim.dirY), float64(anim.dirX))
		}

		cr := float32(anim.r) / 255
		cg := float32(anim.g) / 255
		cb := float32(anim.b) / 255
		ca := float32(anim.a) / 255

		baseIdx := uint16(len(creatureVs))
		for _, offset := range [3]float64{0, 2 * math.Pi / 3, -2 * math.Pi / 3} {
			creatureVs = append(creatureVs, ebiten.Vertex{
				DstX:   cx + float32(r)*float32(math.Cos(angle+offset)),
				DstY:   cy + float32(r)*float32(math.Sin(angle+offset)),
				SrcX:   0, SrcY: 0,
				ColorR: cr, ColorG: cg, ColorB: cb, ColorA: ca,
			})
		}
		creatureIs = append(creatureIs, baseIdx, baseIdx+1, baseIdx+2)
	}
	if len(creatureVs) > 0 {
		screen.DrawTriangles(creatureVs, creatureIs, g.whiteImage, &ebiten.DrawTrianglesOptions{
			FillRule: ebiten.FillAll,
		})
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
<<<<<<< Updated upstream
	g.addStatLine(screen, "Population", fmt.Sprintf("%d", g.sim.PopulationCount()), 1)
	g.addStatLine(screen, "Food", fmt.Sprintf("%d", g.sim.FoodCount()), 2)
	g.addStatLine(screen, "Avg Age", fmt.Sprintf("%.0f", g.sim.AverageAge()), 3)
=======
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
	var label string
	if g.paused {
		bg = color.RGBA{R: 50, G: 140, B: 80, A: 220}
		label = "Resume"
	} else {
		bg = color.RGBA{R: 80, G: 80, B: 50, A: 220}
		label = "Pause"
	}
	vector.DrawFilledRect(screen, pauseBtnX, pauseBtnY, pauseBtnW, pauseBtnH, bg, false)
	text.Draw(screen, label, g.statFont, pauseBtnX+10, pauseBtnY+17, color.White)
}

func (g *Game) drawNewGameButton(screen *ebiten.Image) {
	bg := color.RGBA{R: 80, G: 50, B: 50, A: 220}
	vector.DrawFilledRect(screen, newGameBtnX, newGameBtnY, newGameBtnW, newGameBtnH, bg, false)
	text.Draw(screen, "New Game", g.statFont, newGameBtnX+5, newGameBtnY+17, color.White)
>>>>>>> Stashed changes
}

func (g *Game) drawFOVButton(screen *ebiten.Image) {
	var bg color.RGBA
	if g.showFOV {
		bg = color.RGBA{R: 80, G: 120, B: 200, A: 220}
	} else {
		bg = color.RGBA{R: 50, G: 50, B: 70, A: 220}
	}
	vector.DrawFilledRect(screen, fovBtnX, fovBtnY, fovBtnW, fovBtnH, bg, false)
	text.Draw(screen, "FOV", g.statFont, fovBtnX+8, fovBtnY+17, color.White)
}

func (g *Game) drawFOVCones(screen *ebiten.Image, views []simulation.CreatureView, t float64) {
	var allVs []ebiten.Vertex
	var allIs []uint16

	flush := func() {
		if len(allVs) == 0 {
			return
		}
		screen.DrawTriangles(allVs, allIs, g.whiteImage, &ebiten.DrawTrianglesOptions{
			FillRule: ebiten.FillAll,
		})
		allVs = allVs[:0]
		allIs = allIs[:0]
	}

	half := float32(BlockSize) / 2
	for _, cv := range views {
		if cv.SightDistance == 0 || cv.FieldOfView == 0 || (cv.DirX == 0 && cv.DirY == 0) {
			continue
		}

		// Default apex at the grid position; override with interpolated position
		// when available so the cone tracks the creature body exactly.
		cx := float32(cv.X*BlockSize) + half
		cy := float32(cv.Y*BlockSize) + half
		if anim, ok := g.animByID[cv.ID]; ok {
			cx = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			cy = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		r := float32(cv.SightDistance) * float32(BlockSize)
		halfFOV := float64(cv.FieldOfView) / 2.0 * math.Pi / 180.0
		dirAngle := math.Atan2(float64(cv.DirY), float64(cv.DirX))
		startAngle := float32(dirAngle - halfFOV)
		endAngle := float32(dirAngle + halfFOV)

		var path vector.Path
		path.MoveTo(cx, cy)
		path.Arc(cx, cy, r, startAngle, endAngle, vector.Clockwise)
		path.Close()

		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		cr := float32(cv.R) / 255
		cg := float32(cv.G) / 255
		cb := float32(cv.B) / 255
		for i := range vs {
			vs[i].ColorR = cr
			vs[i].ColorG = cg
			vs[i].ColorB = cb
			vs[i].ColorA = 0.15
		}

		if len(allVs)+len(vs) > 65535 {
			flush()
		}
		offset := uint16(len(allVs))
		for _, idx := range is {
			allIs = append(allIs, idx+offset)
		}
		allVs = append(allVs, vs...)
	}

	flush()
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (g *Game) addStatLine(img *ebiten.Image, description string, value string, row int) {
	x := g.sim.GridWidth()*BlockSize - 200
	y := 20*row + 3
	text.Draw(img, fmt.Sprintf("%s: %s", description, value), g.statFont, x, y, color.White)
}

func foodKey(x, y int) string {
	return fmt.Sprintf("%d,%d", x, y)
}

// trySelectCreature picks the closest creature within a hit radius of the click.
// Clicking the same creature again, or empty space, deselects.
func (g *Game) trySelectCreature(mx, my int) {
	clickX := float64(mx)
	clickY := float64(my)
	half := float64(BlockSize) / 2
	hitRadius := float64(BlockSize) * 6

	bestID := -1
	bestDist := math.Inf(1)
	for id, anim := range g.animByID {
		cx := anim.curX + half
		cy := anim.curY + half
		dx := clickX - cx
		dy := clickY - cy
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist < hitRadius && dist < bestDist {
			bestDist = dist
			bestID = id
		}
	}

	if bestID == -1 || bestID == g.selectedCreatureID {
		g.selectedCreatureID = -1
	} else {
		g.selectedCreatureID = bestID
	}
}

// drawSelectionHighlight draws a ring around the selected creature.
func (g *Game) drawSelectionHighlight(screen *ebiten.Image) {
	anim, ok := g.animByID[g.selectedCreatureID]
	if !ok {
		return
	}
	half := float32(BlockSize) / 2
	cx := float32(anim.curX) + half
	cy := float32(anim.curY) + half
	vector.StrokeCircle(screen, cx, cy, float32(BlockSize)*5, 1.5, color.RGBA{R: 255, G: 240, B: 80, A: 210}, false)
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
	saveBtnY := float32(detailPanelY+detailPanelH-detailTpad-detailSaveBtnH)
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
	g.spawnMutRate = spawnMutRateMin + t*(spawnMutRateMax-spawnMutRateMin)
	g.sim.SetSpawnMutationRate(g.spawnMutRate)
}

func (g *Game) drawSpawnMutSlider(screen *ebiten.Image) {
	vector.DrawFilledRect(screen, spawnMutSliderX, spawnMutSliderY, spawnMutSliderW, spawnMutSliderH,
		color.RGBA{30, 30, 50, 220}, false)

	// Track
	vector.DrawFilledRect(screen, spawnMutTrackX, spawnMutTrackY, spawnMutTrackW, spawnMutTrackH,
		color.RGBA{60, 60, 80, 255}, false)

	// Fill
	t := (g.spawnMutRate - spawnMutRateMin) / (spawnMutRateMax - spawnMutRateMin)
	vector.DrawFilledRect(screen, spawnMutTrackX, spawnMutTrackY, float32(spawnMutTrackW)*t, spawnMutTrackH,
		color.RGBA{80, 140, 210, 255}, false)

	// Thumb
	thumbX := float32(spawnMutTrackX) + float32(spawnMutTrackW)*t - 3
	vector.DrawFilledRect(screen, thumbX, float32(spawnMutTrackY)-3, 6, float32(spawnMutTrackH)+6,
		color.White, false)

	// Label
	text.Draw(screen, fmt.Sprintf("SpawnMut: %.4f", g.spawnMutRate), g.statFont,
		spawnMutSliderX+5, spawnMutSliderY+17, color.White)
}

func creatureEnergyBarColor(frac float32) color.RGBA {
	if frac > 0.65 {
		return color.RGBA{R: 55, G: 185, B: 55, A: 255}
	}
	if frac > 0.33 {
		return color.RGBA{R: 190, G: 175, B: 45, A: 255}
	}
	return color.RGBA{R: 190, G: 55, B: 55, A: 255}
}
