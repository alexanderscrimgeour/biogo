package main

import (
	"biogo/v2/simulation"
	"biogo/v2/ui"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	g := simulation.MakeRandomGenome()
	fmt.Println(g.BinaryString())

	sim := simulation.New()
	for i := 0; i < 500*simulation.Params.MaxAge; i++ {
		start := time.Now()
		sim.Update()
		end := time.Now()
		if sim.Tick != 0 && sim.Tick%simulation.Params.MaxAge == 0 {
			fmt.Printf("\tStep took : %s\n", end.Sub(start))
		}
	}

	game := ui.NewGame(sim)

	ebiten.SetWindowSize(simulation.Params.GridWidth*2, simulation.Params.GridHeight*2)
	ebiten.SetWindowTitle("Genetic Simulation")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal()
	}
}
