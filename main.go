package main

import (
	"gopop/v2/simulation"
	"gopop/v2/ui"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	sim := simulation.New()
	// for i := 0; i < 500*simulation.Params.MaxAge; i++ {
	// 	sim.Update()
	// }

	game := ui.NewGame(sim)

	ebiten.SetWindowSize(int(simulation.Params.GridWidth*5), int(simulation.Params.GridHeight*5))
	ebiten.SetWindowTitle("Genetic Simulation")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal()
	}
}
