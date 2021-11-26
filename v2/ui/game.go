package ui

import (
	"fmt"
	"gopop/v2/simulation"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

type Game struct {
	Simulation *simulation.Simulation
	Grid       *Grid
	statLine   *StatLine
}

var BlockSize int = 2

func NewGame(sim *simulation.Simulation) *Game {
	g := Game{
		Simulation: sim,
		Grid:       NewGrid(0, 0, BlockSize),
	}
	for _, creature := range g.Simulation.Population.Creatures {
		red, green, blue, alpha := creature.Genome.ToColor()
		c := color.RGBA{
			R: red,
			G: green,
			B: blue,
			A: alpha,
		}
		img := g.Grid.AddBlob(BlockSize, c)
		img.Translate(float64(creature.Loc.X*int(BlockSize)), float64(creature.Loc.Y*int(BlockSize)))
	}

	width := 5
	center := g.Simulation.Grid.SizeX() / 2
	minX := center - width/2
	maxX := center + width/2
	minY := g.Simulation.Grid.SizeY() / 4
	maxY := minY + g.Simulation.Grid.SizeY()/2
	g.Grid.AddLine(float64(minX*BlockSize), float64(minY*BlockSize), float64(maxX*BlockSize), float64(maxY*BlockSize))
	return &g
}

func (g *Game) Update() error {
	lastGeneration := g.Simulation.Generation
	g.Simulation.Update()
	if g.Simulation.Generation != lastGeneration {
		g.Grid.blobs = []*Blob{}
		for _, creature := range g.Simulation.Population.Creatures {
			red, green, blue, alpha := creature.Genome.ToColor()
			c := color.RGBA{
				R: red,
				G: green,
				B: blue,
				A: alpha,
			}
			img := g.Grid.AddBlob(BlockSize, c)
			img.Translate(float64(creature.Loc.X*int(BlockSize)), float64(creature.Loc.Y*int(BlockSize)))
		}
	}
	for i, creature := range g.Simulation.Population.Creatures {
		img := g.Grid.blobs[i]
		img.Move(float64(creature.Loc.X*int(BlockSize)), float64(creature.Loc.Y*int(BlockSize)))
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.RGBA{15, 15, 15, 255})
	g.Grid.DrawGrid(screen)
	g.AddStatLine(screen, "Population", len(g.Simulation.Population.Creatures), 1)
	g.AddStatLine(screen, "Generation", g.Simulation.Generation, 2)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func (g *Game) AddStatLine(img *ebiten.Image, description string, statLine int, count int) {
	tt, _ := opentype.Parse(fonts.MPlus1pRegular_ttf)
	font, _ := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    16,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	text.Draw(img, fmt.Sprintf("%s: %d", description, statLine), font, g.Simulation.Grid.SizeX()*BlockSize-200, 20*count+3, color.White)
}
