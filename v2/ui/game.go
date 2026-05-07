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
	FoodViews() []simulation.FoodView
	CorpseViews() []simulation.CorpseView
	GridWidth() int
	GridHeight() int
	PopulationCount() int
	FoodCount() int
	AverageAge() float64
}

var foodColor = color.RGBA{R: 50, G: 200, B: 60, A: 255}

type Game struct {
	sim              SimulationState
	renderGrid       *RenderGrid
	blobsByID        map[int]*Blob
	foodBlobsByKey   map[string]*Blob
	corpseBlobsByID  map[int]*Blob
	statFont         font.Face
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
		sim:             sim,
		renderGrid:      NewRenderGrid(0, 0, BlockSize),
		blobsByID:       make(map[int]*Blob),
		foodBlobsByKey:  make(map[string]*Blob),
		corpseBlobsByID: make(map[int]*Blob),
		statFont:        statFont,
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

	// Reconcile creature blobs with current simulation state.
	views := g.sim.CreatureViews()
	currentIDs := make(map[int]bool, len(views))
	for _, cv := range views {
		currentIDs[cv.ID] = true
		if blob, ok := g.blobsByID[cv.ID]; ok {
			blob.Move(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
		} else {
			c := color.RGBA{R: cv.R, G: cv.G, B: cv.B, A: cv.A}
			blob := g.renderGrid.AddBlob(BlockSize, c)
			blob.Translate(float64(cv.X*BlockSize), float64(cv.Y*BlockSize))
			g.blobsByID[cv.ID] = blob
		}
	}
	for id, blob := range g.blobsByID {
		if !currentIDs[id] {
			g.renderGrid.RemoveBlob(blob)
			delete(g.blobsByID, id)
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

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 15, 255})
	g.renderGrid.DrawGrid(screen)
	g.addStatLine(screen, "Population", fmt.Sprintf("%d", g.sim.PopulationCount()), 1)
	g.addStatLine(screen, "Food", fmt.Sprintf("%d", g.sim.FoodCount()), 2)
	g.addStatLine(screen, "Avg Age", fmt.Sprintf("%.0f", g.sim.AverageAge()), 3)
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
