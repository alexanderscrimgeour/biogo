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
	CreatureMinSize() byte
	CreatureMaxSize() byte
}

var foodColor = color.RGBA{R: 50, G: 200, B: 60, A: 255}

const (
	fovBtnX = 10
	fovBtnY = 10
	fovBtnW = 70
	fovBtnH = 24
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
		minCreatureSize: sim.CreatureMinSize(),
		maxCreatureSize: sim.CreatureMaxSize(),
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
	// Snapshot the target positions from last tick as the interpolation start
	// for this tick. Done before sim.Update() so we capture where creatures were.
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
			anim.size = cv.Size
		} else {
			g.animByID[cv.ID] = &creatureAnim{
				prevX: screenX, prevY: screenY,
				curX: screenX, curY: screenY,
				r: cv.R, g: cv.G, b: cv.B, a: cv.A,
				dirX: cv.DirX, dirY: cv.DirY,
				size: cv.Size,
			}
		}
	}
	for id := range g.animByID {
		if !currentIDs[id] {
			delete(g.animByID, id)
		}
	}

	// Reconcile food blobs with current simulation state.
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

	// Reconcile corpse blobs with current simulation state.
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

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		if mx >= fovBtnX && mx < fovBtnX+fovBtnW && my >= fovBtnY && my < fovBtnY+fovBtnH {
			g.showFOV = !g.showFOV
		}
	}

	// Record time at the very end so Draw() can compute how far into this
	// tick we are. tickDuration measures end-to-end time between ticks.
	now := time.Now()
	if !g.lastTickTime.IsZero() {
		g.tickDuration = now.Sub(g.lastTickTime)
	}
	g.lastTickTime = now

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
			r = float64(BlockSize) + sizeT*float64(2*BlockSize)
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
	g.drawFOVButton(screen)
	g.addStatLine(screen, "Population", fmt.Sprintf("%d", g.sim.PopulationCount()), 1)
	g.addStatLine(screen, "Food", fmt.Sprintf("%d", g.sim.FoodCount()), 2)
	g.addStatLine(screen, "Avg Age", fmt.Sprintf("%.0f", g.sim.AverageAge()), 3)
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
