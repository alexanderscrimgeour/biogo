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
	prevX, prevY float64
	curX, curY   float64
	r, g, b, a   uint8
	heading      float64
	size         byte
}

type Game struct {
	sim             SimulationState
	renderGrid      *RenderGrid
	foodBlobsByID   map[int]*Blob
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
		foodBlobsByID:   make(map[int]*Blob),
		corpseBlobsByID: make(map[int]*Blob),
		statFont:        statFont,
		whiteImage:      wImg,
		animByID:        make(map[int]*creatureAnim),
		minCreatureSize: sim.CreatureMinSize(),
		maxCreatureSize: sim.CreatureMaxSize(),
	}

	// Draw cross-wall lines matching the world's wall layout.
	wallThickness := 10.0
	cx := float64(sim.GridWidth()) / 2
	cy := float64(sim.GridHeight()) / 2
	bs := float64(BlockSize)

	// Vertical bar
	g.renderGrid.AddLine(
		(cx-wallThickness/2)*bs, float64(sim.GridHeight())/4*bs,
		(cx+wallThickness/2)*bs, (float64(sim.GridHeight())/4+float64(sim.GridHeight())/2)*bs,
	)
	// Horizontal bar
	g.renderGrid.AddLine(
		float64(sim.GridWidth())/4*bs, (cy-wallThickness/2)*bs,
		(float64(sim.GridWidth())/4+float64(sim.GridWidth())/2)*bs, (cy+wallThickness/2)*bs,
	)
	return g
}

func (g *Game) Update() error {
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
		screenX := cv.X * bs
		screenY := cv.Y * bs

		if anim, ok := g.animByID[cv.ID]; ok {
			if prev, ok := prevByID[cv.ID]; ok {
				anim.prevX, anim.prevY = prev[0], prev[1]
			} else {
				anim.prevX, anim.prevY = screenX, screenY
			}
			anim.curX, anim.curY = screenX, screenY
			anim.r, anim.g, anim.b, anim.a = cv.R, cv.G, cv.B, cv.A
			anim.heading = cv.Heading
			anim.size = cv.Size
		} else {
			g.animByID[cv.ID] = &creatureAnim{
				prevX: screenX, prevY: screenY,
				curX: screenX, curY: screenY,
				r: cv.R, g: cv.G, b: cv.B, a: cv.A,
				heading: cv.Heading,
				size:    cv.Size,
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
	currentFood := make(map[int]bool, len(foodViews))
	for _, fv := range foodViews {
		currentFood[fv.ID] = true
		if _, ok := g.foodBlobsByID[fv.ID]; !ok {
			blob := g.renderGrid.AddFoodBlob(BlockSize, foodColor)
			blob.Translate(fv.X*bs, fv.Y*bs)
			g.foodBlobsByID[fv.ID] = blob
		}
	}
	for id, blob := range g.foodBlobsByID {
		if !currentFood[id] {
			g.renderGrid.RemoveFoodBlob(blob)
			delete(g.foodBlobsByID, id)
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
			blob.Move(cv.X*bs, cv.Y*bs)
			blob.SetColor(corpseColor)
		} else {
			blob := g.renderGrid.AddBlob(BlockSize, corpseColor)
			blob.Translate(cv.X*bs, cv.Y*bs)
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

	now := time.Now()
	if !g.lastTickTime.IsZero() {
		g.tickDuration = now.Sub(g.lastTickTime)
	}
	g.lastTickTime = now
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 15, 255})
	g.renderGrid.DrawBackground(screen)

	t := 1.0
	if g.tickDuration > 0 {
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

		angle := anim.heading

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

	bs := float64(BlockSize)
	half := float32(BlockSize) / 2
	for _, cv := range views {
		if cv.SightDistance == 0 || cv.FieldOfView == 0 {
			continue
		}

		cx := float32(cv.X*bs) + half
		cy := float32(cv.Y*bs) + half
		if anim, ok := g.animByID[cv.ID]; ok {
			cx = float32(anim.prevX+(anim.curX-anim.prevX)*t) + half
			cy = float32(anim.prevY+(anim.curY-anim.prevY)*t) + half
		}
		r := float32(cv.SightDistance) * float32(BlockSize)
		halfFOV := float64(cv.FieldOfView) / 2.0 * math.Pi / 180.0
		startAngle := float32(cv.Heading - halfFOV)
		endAngle := float32(cv.Heading + halfFOV)

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
