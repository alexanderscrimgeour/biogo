package ui

import (
	"gopop/v2/simulation"

	"github.com/hajimehoshi/ebiten/v2"
	"golang.org/x/image/colornames"
)

type Game struct {
	Simulation *simulation.Simulation
	Grid       *Grid
}

var BlockSize int = 1

func NewGame(sim *simulation.Simulation) *Game {
	g := Game{
		Simulation: sim,
		Grid:       NewGrid(0, 0, int(simulation.Params.GridWidth), int(simulation.Params.GridHeight), BlockSize),
	}
	for _, creature := range g.Simulation.Population.Creatures {
		img := g.Grid.AddBlob(BlockSize)
		img.Translate(float64(creature.Loc.X*int(BlockSize)), float64(creature.Loc.Y*int(BlockSize)))
	}
	return &g
}

func (g *Game) Update() error {
	lastGeneration := g.Simulation.Generation
	g.Simulation.Update()
	if g.Simulation.Generation != lastGeneration {
		g.Grid.blobs = []*Blob{}
		for _, creature := range g.Simulation.Population.Creatures {
			img := g.Grid.AddBlob(BlockSize)
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
	screen.Fill(colornames.White)
	g.Grid.DrawGrid(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return 320, 320
}
