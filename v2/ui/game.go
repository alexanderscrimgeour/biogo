package ui

import (
	"biogo/v2/simulation"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

// SimulationState is the interface the UI requires from a simulation.
// Keeping this narrow prevents the UI from reaching into simulation internals.
type SimulationState interface {
	Update()
	CreatureViews() []simulation.CreatureView
	GridWidth() int
	GridHeight() int
	CurrentGeneration() int
	PopulationCount() int
}

type Game struct {
	sim        SimulationState
	renderGrid *RenderGrid
	blobsByID  map[int]*Blob
	statFont   font.Face
	lastGen    int
}

var BlockSize int = 2

func NewGame(sim SimulationState) *Game {
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	statFont, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})

	g := &Game{
		sim:        sim,
		renderGrid: NewRenderGrid(0, 0, BlockSize),
		blobsByID:  make(map[int]*Blob),
		statFont:   statFont,
		lastGen:    sim.CurrentGeneration(),
	}

	for _, cv := range sim.CreatureViews() {
		c := color.RGBA{R: cv.R, G: cv.G, B: cv.B, A: cv.A}
		blob := g.renderGrid.AddBlob(BlockSize, c)
		blob.Translate(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
		g.blobsByID[cv.ID] = blob
	}

	width := 5
	centerX := sim.GridWidth() / 2
	minX := centerX - width/2
	maxX := centerX + width/2
	minY := sim.GridHeight() / 4
	maxY := minY + sim.GridHeight()/2
	g.renderGrid.AddLine(float64(minX*BlockSize), float64(minY*BlockSize), float64(maxX*BlockSize), float64(maxY*BlockSize))

	return g
}

func (g *Game) Update() error {
	g.sim.Update()

	currentGen := g.sim.CurrentGeneration()
	views := g.sim.CreatureViews()

	if currentGen != g.lastGen {
		g.lastGen = currentGen
		g.renderGrid.blobs = []*Blob{}
		g.blobsByID = make(map[int]*Blob)
		for _, cv := range views {
			c := color.RGBA{R: cv.R, G: cv.G, B: cv.B, A: cv.A}
			blob := g.renderGrid.AddBlob(BlockSize, c)
			blob.Translate(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
			g.blobsByID[cv.ID] = blob
		}
		return nil
	}

	for _, cv := range views {
		if blob, ok := g.blobsByID[cv.ID]; ok {
			blob.Move(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 15, 255})
	g.renderGrid.DrawGrid(screen)
	g.addStatLine(screen, "Population", g.sim.PopulationCount(), 1)
	g.addStatLine(screen, "Generation", g.sim.CurrentGeneration(), 2)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (g *Game) addStatLine(img *ebiten.Image, description string, value int, row int) {
	x := g.sim.GridWidth()*BlockSize - 200
	y := 20*row + 3
	text.Draw(img, fmt.Sprintf("%s: %d", description, value), g.statFont, x, y, color.White)
}
