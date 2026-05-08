package main

import (
	"biogo/v2/simulation"
	"biogo/v2/ui"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	params := simulation.DefaultParams()
	sim := simulation.New(params)

	game := ui.NewGame(sim)

	ebiten.SetWindowSize(int(params.GridWidth)*2, int(params.GridHeight)*2)
	ebiten.SetWindowTitle("Genetic Simulation")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal()
	}
}
